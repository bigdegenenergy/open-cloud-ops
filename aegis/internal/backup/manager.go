package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// KubeClient defines the interface for Kubernetes API interactions.
// In production, this wraps client-go; in tests, a mock can be provided.
type KubeClient interface {
	// ListResources lists resources of the given type in the given namespace.
	ListResources(ctx context.Context, resourceType, namespace string) ([]models.KubernetesResource, error)

	// ApplyResource creates or updates a resource in the cluster.
	ApplyResource(ctx context.Context, resource models.KubernetesResource) error

	// DeleteResource removes a resource from the cluster.
	DeleteResource(ctx context.Context, resourceType, name, namespace string) error

	// ResourceExists checks whether a resource exists in the cluster.
	ResourceExists(ctx context.Context, resourceType, name, namespace string) (bool, error)
}

// BackupManager orchestrates backup operations including job management,
// execution, scheduling, and retention enforcement.
type BackupManager struct {
	kubeClient   KubeClient
	storage      StorageBackend
	storagePath  string
	retentionDays int

	mu      sync.RWMutex
	jobs    map[string]*models.BackupJob
	records map[string][]*models.BackupRecord // jobID -> records
}

// NewBackupManager creates a new BackupManager with the given dependencies.
func NewBackupManager(kubeClient KubeClient, storage StorageBackend, storagePath string, retentionDays int) *BackupManager {
	return &BackupManager{
		kubeClient:    kubeClient,
		storage:       storage,
		storagePath:   storagePath,
		retentionDays: retentionDays,
		jobs:          make(map[string]*models.BackupJob),
		records:       make(map[string][]*models.BackupRecord),
	}
}

// CreateJob registers a new backup job. It validates the job configuration,
// assigns an ID and timestamps, and stores the job in memory.
// In production, this would also persist to PostgreSQL.
func (m *BackupManager) CreateJob(ctx context.Context, job models.BackupJob) (*models.BackupJob, error) {
	if job.Name == "" {
		return nil, fmt.Errorf("backup: job name is required")
	}
	if job.Namespace == "" {
		return nil, fmt.Errorf("backup: job namespace is required")
	}
	if len(job.ResourceTypes) == 0 {
		return nil, fmt.Errorf("backup: at least one resource type is required")
	}
	if job.Schedule == "" {
		return nil, fmt.Errorf("backup: schedule is required")
	}

	now := time.Now().UTC()

	if job.ID == "" {
		job.ID = fmt.Sprintf("bj-%d", now.UnixNano())
	}
	if job.RetentionDays <= 0 {
		job.RetentionDays = m.retentionDays
	}
	if job.StorageLocation == "" {
		job.StorageLocation = m.storagePath
	}
	if job.Status == "" {
		job.Status = models.BackupStatusActive
	}
	job.CreatedAt = now

	// Calculate next run from schedule
	nextRun := calculateNextRun(job.Schedule, now)
	job.NextRun = &nextRun

	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobs[job.ID] = &job
	m.records[job.ID] = make([]*models.BackupRecord, 0)

	log.Printf("backup: created job %s (%s) for namespace %s", job.ID, job.Name, job.Namespace)
	return &job, nil
}

// GetJob retrieves a backup job by ID.
func (m *BackupManager) GetJob(ctx context.Context, jobID string) (*models.BackupJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("backup: job %q not found", jobID)
	}
	return job, nil
}

// ListJobs returns all registered backup jobs.
func (m *BackupManager) ListJobs(ctx context.Context) ([]*models.BackupJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*models.BackupJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// ExecuteBackup runs a backup for the specified job. It lists target resources
// from Kubernetes, serializes them to JSON, creates a compressed tar.gz archive,
// stores it via the storage backend, and records metadata.
func (m *BackupManager) ExecuteBackup(ctx context.Context, jobID string) (*models.BackupRecord, error) {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("backup: job %q not found", jobID)
	}
	// Copy job data so we can release the lock
	jobCopy := *job
	m.mu.RUnlock()

	now := time.Now().UTC()
	record := &models.BackupRecord{
		ID:        fmt.Sprintf("br-%d", now.UnixNano()),
		JobID:     jobID,
		Status:    models.RecordStatusRunning,
		StartedAt: now,
	}

	// Store the initial record
	m.mu.Lock()
	m.records[jobID] = append(m.records[jobID], record)
	m.mu.Unlock()

	log.Printf("backup: starting backup execution %s for job %s (%s)", record.ID, jobID, jobCopy.Name)

	// Collect resources from Kubernetes
	var allResources []models.KubernetesResource
	var errors []string

	for _, resourceType := range jobCopy.ResourceTypes {
		resources, err := m.kubeClient.ListResources(ctx, resourceType, jobCopy.Namespace)
		if err != nil {
			errMsg := fmt.Sprintf("failed to list %s in %s: %v", resourceType, jobCopy.Namespace, err)
			errors = append(errors, errMsg)
			log.Printf("backup: %s", errMsg)
			continue
		}
		allResources = append(allResources, resources...)
	}

	if len(allResources) == 0 && len(errors) > 0 {
		// Complete failure: no resources collected and errors occurred
		completedAt := time.Now().UTC()
		record.Status = models.RecordStatusFailed
		record.ErrorMessage = strings.Join(errors, "; ")
		record.CompletedAt = &completedAt
		record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
		return record, fmt.Errorf("backup: failed to collect any resources: %s", record.ErrorMessage)
	}

	// Build the backup manifest
	manifest := models.BackupManifest{
		BackupID:      record.ID,
		JobID:         jobID,
		Namespace:     jobCopy.Namespace,
		ResourceTypes: jobCopy.ResourceTypes,
		Resources:     allResources,
		ResourceCount: len(allResources),
		CreatedAt:     now,
	}

	// Create compressed tar.gz archive
	archiveData, checksum, err := createArchive(manifest)
	if err != nil {
		completedAt := time.Now().UTC()
		record.Status = models.RecordStatusFailed
		record.ErrorMessage = fmt.Sprintf("failed to create archive: %v", err)
		record.CompletedAt = &completedAt
		record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
		return record, fmt.Errorf("backup: %s", record.ErrorMessage)
	}

	manifest.Checksum = checksum

	// Store the archive
	storagePath := fmt.Sprintf("%s/%s/%s.tar.gz", jobID, record.ID, record.ID)
	if err := m.storage.Write(ctx, storagePath, archiveData); err != nil {
		completedAt := time.Now().UTC()
		record.Status = models.RecordStatusFailed
		record.ErrorMessage = fmt.Sprintf("failed to store archive: %v", err)
		record.CompletedAt = &completedAt
		record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
		return record, fmt.Errorf("backup: %s", record.ErrorMessage)
	}

	// Store the manifest alongside the archive
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Printf("backup: warning: failed to marshal manifest: %v", err)
	} else {
		manifestPath := fmt.Sprintf("%s/%s/manifest.json", jobID, record.ID)
		if err := m.storage.Write(ctx, manifestPath, manifestData); err != nil {
			log.Printf("backup: warning: failed to store manifest: %v", err)
		}
	}

	// Update the record with success
	completedAt := time.Now().UTC()
	record.Status = models.RecordStatusCompleted
	record.SizeBytes = int64(len(archiveData))
	record.ResourceCount = len(allResources)
	record.StoragePath = storagePath
	record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
	record.CompletedAt = &completedAt

	// Record partial failures as warnings in the error message
	if len(errors) > 0 {
		record.ErrorMessage = fmt.Sprintf("partial errors: %s", strings.Join(errors, "; "))
	}

	// Update the job's last run time
	m.mu.Lock()
	if j, ok := m.jobs[jobID]; ok {
		j.LastRun = &completedAt
		nextRun := calculateNextRun(j.Schedule, completedAt)
		j.NextRun = &nextRun
	}
	m.mu.Unlock()

	log.Printf("backup: completed backup %s: %d resources, %d bytes, %dms",
		record.ID, record.ResourceCount, record.SizeBytes, record.DurationMs)

	return record, nil
}

// ListBackups returns all backup records for the specified job.
func (m *BackupManager) ListBackups(ctx context.Context, jobID string) ([]*models.BackupRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, exists := m.records[jobID]
	if !exists {
		return nil, fmt.Errorf("backup: job %q not found", jobID)
	}

	// Return a copy to prevent external modification
	result := make([]*models.BackupRecord, len(records))
	copy(result, records)
	return result, nil
}

// ListAllBackups returns all backup records across all jobs.
func (m *BackupManager) ListAllBackups(ctx context.Context) ([]*models.BackupRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []*models.BackupRecord
	for _, records := range m.records {
		all = append(all, records...)
	}
	return all, nil
}

// GetBackupRecord retrieves a specific backup record by ID.
func (m *BackupManager) GetBackupRecord(ctx context.Context, recordID string) (*models.BackupRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, records := range m.records {
		for _, r := range records {
			if r.ID == recordID {
				return r, nil
			}
		}
	}
	return nil, fmt.Errorf("backup: record %q not found", recordID)
}

// DeleteBackup removes a backup record and its associated storage files.
func (m *BackupManager) DeleteBackup(ctx context.Context, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for jobID, records := range m.records {
		for i, r := range records {
			if r.ID == recordID {
				// Delete from storage
				storagePath := fmt.Sprintf("%s/%s", jobID, recordID)
				if err := m.storage.Delete(ctx, storagePath); err != nil {
					log.Printf("backup: warning: failed to delete storage for %s: %v", recordID, err)
				}

				// Remove from records
				m.records[jobID] = append(records[:i], records[i+1:]...)
				log.Printf("backup: deleted backup record %s", recordID)
				return nil
			}
		}
	}

	return fmt.Errorf("backup: record %q not found", recordID)
}

// ScheduleBackups evaluates all active jobs and returns those whose
// next scheduled run time has passed. In production, this would be
// called by a ticker goroutine.
func (m *BackupManager) ScheduleBackups(ctx context.Context) ([]*models.BackupJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now().UTC()
	var due []*models.BackupJob

	for _, job := range m.jobs {
		if job.Status != models.BackupStatusActive {
			continue
		}
		if job.NextRun != nil && !job.NextRun.After(now) {
			due = append(due, job)
		}
	}

	return due, nil
}

// EnforceRetention deletes backups that exceed the retention period for
// the specified job. Returns the number of backups deleted.
func (m *BackupManager) EnforceRetention(ctx context.Context, jobID string) (int, error) {
	m.mu.Lock()
	job, exists := m.jobs[jobID]
	if !exists {
		m.mu.Unlock()
		return 0, fmt.Errorf("backup: job %q not found", jobID)
	}
	retentionDays := job.RetentionDays
	records := m.records[jobID]
	m.mu.Unlock()

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	var toDelete []string

	for _, r := range records {
		if r.CompletedAt != nil && r.CompletedAt.Before(cutoff) {
			toDelete = append(toDelete, r.ID)
		}
	}

	deleted := 0
	for _, recordID := range toDelete {
		if err := m.DeleteBackup(ctx, recordID); err != nil {
			log.Printf("backup: retention: failed to delete %s: %v", recordID, err)
			continue
		}
		deleted++
	}

	log.Printf("backup: retention enforcement for job %s: deleted %d/%d expired backups",
		jobID, deleted, len(toDelete))
	return deleted, nil
}

// LoadManifest retrieves and parses the backup manifest for a given record.
func (m *BackupManager) LoadManifest(ctx context.Context, recordID string) (*models.BackupManifest, error) {
	record, err := m.GetBackupRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}

	// The manifest is stored alongside the archive
	manifestPath := fmt.Sprintf("%s/%s/manifest.json", record.JobID, recordID)
	data, err := m.storage.Read(ctx, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("backup: failed to read manifest for %s: %w", recordID, err)
	}

	var manifest models.BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("backup: failed to parse manifest for %s: %w", recordID, err)
	}

	return &manifest, nil
}

// createArchive builds a tar.gz archive containing the serialized resources
// and returns the archive bytes and its SHA-256 checksum.
func createArchive(manifest models.BackupManifest) ([]byte, string, error) {
	var buf bytes.Buffer

	// Create gzip writer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Comment = fmt.Sprintf("Aegis backup %s", manifest.BackupID)
	gzWriter.ModTime = manifest.CreatedAt

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	// Add each resource as a separate file in the archive
	for i, resource := range manifest.Resources {
		resourceData, err := json.MarshalIndent(resource, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal resource %s/%s: %w", resource.Kind, resource.Name, err)
		}

		fileName := fmt.Sprintf("%s/%s_%s_%d.json",
			resource.Kind,
			resource.Namespace,
			resource.Name,
			i,
		)

		header := &tar.Header{
			Name:    fileName,
			Size:    int64(len(resourceData)),
			Mode:    0644,
			ModTime: manifest.CreatedAt,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, "", fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := tarWriter.Write(resourceData); err != nil {
			return nil, "", fmt.Errorf("failed to write tar data: %w", err)
		}
	}

	// Add manifest as a special file
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestHeader := &tar.Header{
		Name:    "manifest.json",
		Size:    int64(len(manifestData)),
		Mode:    0644,
		ModTime: manifest.CreatedAt,
	}
	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return nil, "", fmt.Errorf("failed to write manifest header: %w", err)
	}
	if _, err := tarWriter.Write(manifestData); err != nil {
		return nil, "", fmt.Errorf("failed to write manifest data: %w", err)
	}

	// Close writers in order
	if err := tarWriter.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	archiveData := buf.Bytes()

	// Calculate checksum
	hash := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(hash[:])

	return archiveData, checksum, nil
}

// calculateNextRun computes the next run time from a cron-like schedule string.
// This is a simplified implementation that supports common intervals:
//   - "@hourly"  -> next hour
//   - "@daily"   -> next day at midnight
//   - "@weekly"  -> next Monday at midnight
//   - "*/N * * * *" -> every N minutes
//
// A full cron parser (e.g., robfig/cron) should be used in production.
func calculateNextRun(schedule string, from time.Time) time.Time {
	switch strings.ToLower(strings.TrimSpace(schedule)) {
	case "@hourly":
		return from.Truncate(time.Hour).Add(time.Hour)
	case "@daily":
		next := from.Truncate(24 * time.Hour).Add(24 * time.Hour)
		return next
	case "@weekly":
		// Next Monday at midnight
		daysUntilMonday := (8 - int(from.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		return from.Truncate(24 * time.Hour).AddDate(0, 0, daysUntilMonday)
	default:
		// Parse simple "*/N * * * *" format (every N minutes)
		if strings.HasPrefix(schedule, "*/") {
			parts := strings.Fields(schedule)
			if len(parts) >= 1 {
				intervalStr := strings.TrimPrefix(parts[0], "*/")
				var interval int
				fmt.Sscanf(intervalStr, "%d", &interval)
				if interval > 0 {
					return from.Add(time.Duration(interval) * time.Minute)
				}
			}
		}
		// Default: run in 1 hour
		return from.Add(time.Hour)
	}
}
