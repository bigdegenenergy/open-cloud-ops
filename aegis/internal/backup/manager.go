package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
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
	kubeClient         KubeClient
	storage            StorageBackend
	store              BackupStore // Optional DB persistence (write-through cache)
	storagePath        string
	retentionDays      int
	encryptionKey      []byte // Optional AES-256 key for encrypting archives at rest
	encryptionRequired bool   // Fail-closed: if true, backups fail when no key is set

	mu      sync.RWMutex
	jobs    map[string]*models.BackupJob
	records map[string][]*models.BackupRecord // jobID -> records
}

// NewBackupManager creates a new BackupManager with the given dependencies.
// If encryptionKeyHex is non-empty, archives will be AES-GCM encrypted at rest.
// The key must be a 64-character hex string (32 bytes / AES-256).
func NewBackupManager(kubeClient KubeClient, storage StorageBackend, storagePath string, retentionDays int) *BackupManager {
	m := &BackupManager{
		kubeClient:    kubeClient,
		storage:       storage,
		storagePath:   storagePath,
		retentionDays: retentionDays,
		jobs:          make(map[string]*models.BackupJob),
		records:       make(map[string][]*models.BackupRecord),
	}

	// Load optional encryption key from environment
	if keyHex := os.Getenv("AEGIS_BACKUP_ENCRYPTION_KEY"); keyHex != "" {
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			log.Printf("backup: WARNING: AEGIS_BACKUP_ENCRYPTION_KEY must be 64 hex chars (AES-256); encryption disabled")
		} else {
			m.encryptionKey = key
			log.Printf("backup: AES-CTR+HMAC-SHA256 encryption enabled for archives")
		}
	}

	// Fail-closed: when AEGIS_ENCRYPTION_REQUIRED=true, refuse to run
	// backups without a valid encryption key.
	if strings.EqualFold(os.Getenv("AEGIS_ENCRYPTION_REQUIRED"), "true") {
		m.encryptionRequired = true
		if m.encryptionKey == nil {
			log.Printf("backup: WARNING: AEGIS_ENCRYPTION_REQUIRED=true but no valid encryption key configured")
		}
	}

	return m
}

// SetStore configures a persistent BackupStore for write-through caching.
// When set, all mutations are persisted to the store in addition to memory.
func (m *BackupManager) SetStore(store BackupStore) {
	m.store = store
}

// LoadFromStore populates the in-memory cache from the persistent store.
// Call this on startup after SetStore to recover state from a previous run.
func (m *BackupManager) LoadFromStore(ctx context.Context) error {
	if m.store == nil {
		return nil
	}

	jobs, err := m.store.ListJobs(ctx)
	if err != nil {
		return fmt.Errorf("backup: failed to load jobs from store: %w", err)
	}

	allRecords, err := m.store.ListAllRecords(ctx)
	if err != nil {
		return fmt.Errorf("backup: failed to load records from store: %w", err)
	}

	// Group records by job ID
	recordsByJob := make(map[string][]*models.BackupRecord)
	for _, r := range allRecords {
		recordsByJob[r.JobID] = append(recordsByJob[r.JobID], r)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range jobs {
		m.jobs[job.ID] = job
		m.records[job.ID] = recordsByJob[job.ID]
		if m.records[job.ID] == nil {
			m.records[job.ID] = make([]*models.BackupRecord, 0)
		}
	}

	log.Printf("backup: loaded %d jobs and %d records from store", len(jobs), len(allRecords))
	return nil
}

// CreateJob registers a new backup job. It validates the job configuration,
// assigns an ID and timestamps, and persists to both memory and the store.
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
	nextRun, err := calculateNextRun(job.Schedule, now)
	if err != nil {
		return nil, fmt.Errorf("backup: invalid schedule for job %q: %w", job.Name, err)
	}
	job.NextRun = &nextRun

	m.mu.Lock()
	m.jobs[job.ID] = &job
	m.records[job.ID] = make([]*models.BackupRecord, 0)
	m.mu.Unlock()

	if m.store != nil {
		logStoreErr("SaveJob", m.store.SaveJob(ctx, &job))
	}

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

	if m.store != nil {
		logStoreErr("SaveRecord(initial)", m.store.SaveRecord(ctx, record))
	}

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

	// Create compressed tar.gz archive (streamed to temp file)
	archivePath, checksum, err := createArchive(manifest)
	if err != nil {
		completedAt := time.Now().UTC()
		record.Status = models.RecordStatusFailed
		record.ErrorMessage = fmt.Sprintf("failed to create archive: %v", err)
		record.CompletedAt = &completedAt
		record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
		return record, fmt.Errorf("backup: %s", record.ErrorMessage)
	}
	defer os.Remove(archivePath)

	manifest.Checksum = checksum

	// Fail-closed: refuse to store unencrypted backups when encryption is required
	if m.encryptionRequired && m.encryptionKey == nil {
		completedAt := time.Now().UTC()
		record.Status = models.RecordStatusFailed
		record.ErrorMessage = "encryption required but no valid key configured (AEGIS_ENCRYPTION_REQUIRED=true)"
		record.CompletedAt = &completedAt
		record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
		os.Remove(archivePath)
		return record, fmt.Errorf("backup: %s", record.ErrorMessage)
	}

	// Encrypt archive at rest if an encryption key is configured
	if m.encryptionKey != nil {
		encPath, encErr := m.encryptFile(archivePath)
		if encErr != nil {
			completedAt := time.Now().UTC()
			record.Status = models.RecordStatusFailed
			record.ErrorMessage = fmt.Sprintf("failed to encrypt archive: %v", encErr)
			record.CompletedAt = &completedAt
			record.DurationMs = completedAt.Sub(record.StartedAt).Milliseconds()
			return record, fmt.Errorf("backup: %s", record.ErrorMessage)
		}
		// Replace the plain archive with the encrypted one
		os.Remove(archivePath)
		archivePath = encPath
		defer os.Remove(encPath)
	}

	// Store the archive by streaming from temp file (avoids loading into memory)
	storagePath := fmt.Sprintf("%s/%s/%s.tar.gz", jobID, record.ID, record.ID)
	if err := m.storage.WriteFromFile(ctx, storagePath, archivePath); err != nil {
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
	// Get archive file size from disk (archiveData was removed in streaming refactor)
	if fi, statErr := os.Stat(archivePath); statErr == nil {
		record.SizeBytes = fi.Size()
	}
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
		nextRun, schedErr := calculateNextRun(j.Schedule, completedAt)
		if schedErr != nil {
			log.Printf("backup: warning: could not compute next run for job %s: %v", jobID, schedErr)
		} else {
			j.NextRun = &nextRun
		}
		if m.store != nil {
			jCopy := *j
			go func() { logStoreErr("SaveJob(lastRun)", m.store.SaveJob(ctx, &jCopy)) }()
		}
	}
	m.mu.Unlock()

	// Persist the completed record
	if m.store != nil {
		logStoreErr("SaveRecord(completed)", m.store.SaveRecord(ctx, record))
	}

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
				if m.store != nil {
					logStoreErr("DeleteRecord", m.store.DeleteRecord(ctx, recordID))
				}
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
// and returns the temp file path and its SHA-256 checksum.
// The caller is responsible for removing the temp file when done.
func createArchive(manifest models.BackupManifest) (string, string, error) {
	tmpFile, err := os.CreateTemp("", "aegis-backup-*.tar.gz")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Restrict permissions so only the owner can read backup archives
	// (they may contain Kubernetes Secrets and ConfigMaps)
	os.Chmod(tmpPath, 0600)
	defer tmpFile.Close()

	// Hash writer to compute checksum while writing
	hashWriter := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hashWriter)

	// Create gzip writer
	gzWriter := gzip.NewWriter(multiWriter)
	gzWriter.Comment = fmt.Sprintf("Aegis backup %s", manifest.BackupID)
	gzWriter.ModTime = manifest.CreatedAt

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	// Add each resource as a separate file in the archive
	for i, resource := range manifest.Resources {
		resourceData, err := json.MarshalIndent(resource, "", "  ")
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal resource %s/%s: %w", resource.Kind, resource.Name, err)
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
			return "", "", fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := tarWriter.Write(resourceData); err != nil {
			return "", "", fmt.Errorf("failed to write tar data: %w", err)
		}
	}

	// Add manifest as a special file
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestHeader := &tar.Header{
		Name:    "manifest.json",
		Size:    int64(len(manifestData)),
		Mode:    0644,
		ModTime: manifest.CreatedAt,
	}
	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return "", "", fmt.Errorf("failed to write manifest header: %w", err)
	}
	if _, err := tarWriter.Write(manifestData); err != nil {
		return "", "", fmt.Errorf("failed to write manifest data: %w", err)
	}

	// Close writers in order
	if err := tarWriter.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Calculate checksum from the hash writer
	checksum := hex.EncodeToString(hashWriter.Sum(nil))

	return tmpPath, checksum, nil
}

// encryptFile encrypts a file using AES-256-CTR + HMAC-SHA256 in a streaming
// fashion, avoiding loading the entire file into memory. The output format is:
//
//	[16-byte IV] [encrypted data] [32-byte HMAC]
//
// The HMAC covers (IV || ciphertext) for authenticated encryption.
// The caller is responsible for removing the output file.
func (m *BackupManager) encryptFile(plainPath string) (string, error) {
	src, err := os.Open(plainPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for encryption: %w", err)
	}
	defer src.Close()

	encFile, err := os.CreateTemp("", "aegis-backup-enc-*.tar.gz.enc")
	if err != nil {
		return "", fmt.Errorf("failed to create encrypted temp file: %w", err)
	}
	os.Chmod(encFile.Name(), 0600)

	// Derive separate encryption and MAC keys from the master key using
	// HMAC-SHA256 with distinct info labels.  This prevents related-key
	// attacks when using the same master secret for both AES-CTR and HMAC.
	encKey, macKey := deriveKeys(m.encryptionKey)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		encFile.Close()
		os.Remove(encFile.Name())
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate random IV (one per file, never reused)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		encFile.Close()
		os.Remove(encFile.Name())
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	// Write IV first
	if _, err := encFile.Write(iv); err != nil {
		encFile.Close()
		os.Remove(encFile.Name())
		return "", fmt.Errorf("failed to write IV: %w", err)
	}

	// Stream encrypt with AES-CTR and compute HMAC over IV+ciphertext
	mac := hmac.New(sha256.New, macKey)
	mac.Write(iv)

	stream := cipher.NewCTR(block, iv)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			encrypted := make([]byte, n)
			stream.XORKeyStream(encrypted, buf[:n])
			mac.Write(encrypted)
			if _, err := encFile.Write(encrypted); err != nil {
				encFile.Close()
				os.Remove(encFile.Name())
				return "", fmt.Errorf("failed to write encrypted data: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			encFile.Close()
			os.Remove(encFile.Name())
			return "", fmt.Errorf("failed to read source file: %w", readErr)
		}
	}

	// Append HMAC for authentication
	if _, err := encFile.Write(mac.Sum(nil)); err != nil {
		encFile.Close()
		os.Remove(encFile.Name())
		return "", fmt.Errorf("failed to write HMAC: %w", err)
	}

	encFile.Close()
	return encFile.Name(), nil
}

// deriveKeys derives separate encryption and MAC keys from a master key using
// HMAC-SHA256 with distinct info labels (a simplified HKDF-Expand).
// This ensures the AES-CTR key and HMAC-SHA256 key are cryptographically
// independent, preventing related-key attacks.
func deriveKeys(masterKey []byte) (encKey, macKey []byte) {
	encDeriver := hmac.New(sha256.New, masterKey)
	encDeriver.Write([]byte("aegis-backup-enc-v1"))
	encKey = encDeriver.Sum(nil)

	macDeriver := hmac.New(sha256.New, masterKey)
	macDeriver.Write([]byte("aegis-backup-mac-v1"))
	macKey = macDeriver.Sum(nil)

	return encKey, macKey
}

// calculateNextRun computes the next run time from a cron-like schedule string.
//
// Supported schedule formats:
//
//	Shorthand aliases:
//	  @hourly   -> next hour boundary
//	  @daily    -> next midnight (00:00 UTC)
//	  @weekly   -> next Monday at midnight
//	  @monthly  -> 1st of next month at midnight
//	  @every_Nm -> every N minutes (e.g. @every_15m)
//	  @every_Nh -> every N hours   (e.g. @every_6h)
//
//	Cron-like patterns (minute and hour fields only):
//	  */N * * * *  -> every N minutes
//	  M  */N * * * -> at minute M, every N hours
//	  M  H  * * *  -> daily at H:M
//
// For full 5-field cron semantics (day-of-month, month, day-of-week),
// use a production cron library such as github.com/robfig/cron/v3.
//
// Returns (nextRun, nil) on success, or (zero, error) for unrecognised schedules.
func calculateNextRun(schedule string, from time.Time) (time.Time, error) {
	s := strings.ToLower(strings.TrimSpace(schedule))

	// --- Shorthand aliases ---
	switch s {
	case "@hourly":
		return from.Truncate(time.Hour).Add(time.Hour), nil
	case "@daily":
		next := from.Truncate(24 * time.Hour).Add(24 * time.Hour)
		return next, nil
	case "@weekly":
		daysUntilMonday := (8 - int(from.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		return from.Truncate(24*time.Hour).AddDate(0, 0, daysUntilMonday), nil
	case "@monthly":
		y, m, _ := from.Date()
		return time.Date(y, m+1, 1, 0, 0, 0, 0, from.Location()), nil
	}

	// --- @every_Nm / @every_Nh ---
	if strings.HasPrefix(s, "@every_") {
		suffix := strings.TrimPrefix(s, "@every_")
		if len(suffix) < 2 {
			return time.Time{}, fmt.Errorf("backup: invalid schedule %q: @every_ requires a value like @every_15m", schedule)
		}
		unit := suffix[len(suffix)-1]
		var n int
		if _, err := fmt.Sscanf(suffix[:len(suffix)-1], "%d", &n); err != nil || n <= 0 {
			return time.Time{}, fmt.Errorf("backup: invalid schedule %q: could not parse interval", schedule)
		}
		switch unit {
		case 'm':
			return from.Add(time.Duration(n) * time.Minute), nil
		case 'h':
			return from.Add(time.Duration(n) * time.Hour), nil
		default:
			return time.Time{}, fmt.Errorf("backup: invalid schedule %q: unit must be 'm' or 'h'", schedule)
		}
	}

	// --- Cron-like 5-field patterns (partial support) ---
	parts := strings.Fields(s)
	if len(parts) == 5 {
		minute, hour := parts[0], parts[1]

		// "*/N * * * *" -> every N minutes
		if strings.HasPrefix(minute, "*/") {
			var n int
			if _, err := fmt.Sscanf(strings.TrimPrefix(minute, "*/"), "%d", &n); err == nil && n > 0 {
				return from.Add(time.Duration(n) * time.Minute), nil
			}
		}

		// "M */N * * *" -> at minute M, every N hours
		if strings.HasPrefix(hour, "*/") {
			var m, n int
			if _, err := fmt.Sscanf(minute, "%d", &m); err == nil && m >= 0 && m < 60 {
				if _, err := fmt.Sscanf(strings.TrimPrefix(hour, "*/"), "%d", &n); err == nil && n > 0 {
					next := from.Truncate(time.Hour)
					next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), m, 0, 0, next.Location())
					if !next.After(from) {
						next = next.Add(time.Duration(n) * time.Hour)
					}
					return next, nil
				}
			}
		}

		// "M H * * *" -> daily at H:M
		var m, h int
		if _, err := fmt.Sscanf(minute, "%d", &m); err == nil && m >= 0 && m < 60 {
			if _, err := fmt.Sscanf(hour, "%d", &h); err == nil && h >= 0 && h < 24 {
				next := time.Date(from.Year(), from.Month(), from.Day(), h, m, 0, 0, from.Location())
				if !next.After(from) {
					next = next.AddDate(0, 0, 1)
				}
				return next, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("backup: unrecognised schedule %q; use @hourly, @daily, @weekly, @monthly, @every_Nm, @every_Nh, or a 5-field cron pattern", schedule)
}
