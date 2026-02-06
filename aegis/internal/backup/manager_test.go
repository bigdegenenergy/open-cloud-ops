package backup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// mockKubeClient is a test implementation of the KubeClient interface.
type mockKubeClient struct {
	resources map[string][]models.KubernetesResource
}

func newMockKubeClient() *mockKubeClient {
	client := &mockKubeClient{
		resources: make(map[string][]models.KubernetesResource),
	}

	// Add some test resources
	client.resources["default/Deployment"] = []models.KubernetesResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-app",
			Namespace:  "default",
			Labels:     map[string]string{"app": "test"},
			Manifest:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-app","namespace":"default"}}`),
		},
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-worker",
			Namespace:  "default",
			Labels:     map[string]string{"app": "worker"},
			Manifest:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-worker","namespace":"default"}}`),
		},
	}

	client.resources["default/Service"] = []models.KubernetesResource{
		{
			APIVersion: "v1",
			Kind:       "Service",
			Name:       "test-service",
			Namespace:  "default",
			Labels:     map[string]string{"app": "test"},
			Manifest:   []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"test-service","namespace":"default"}}`),
		},
	}

	client.resources["default/ConfigMap"] = []models.KubernetesResource{
		{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "test-config",
			Namespace:  "default",
			Labels:     map[string]string{"app": "test"},
			Manifest:   []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"test-config","namespace":"default"}}`),
		},
	}

	return client
}

func (m *mockKubeClient) ListResources(ctx context.Context, resourceType, namespace string) ([]models.KubernetesResource, error) {
	key := namespace + "/" + resourceType
	resources, exists := m.resources[key]
	if !exists {
		return []models.KubernetesResource{}, nil
	}
	return resources, nil
}

func (m *mockKubeClient) ApplyResource(ctx context.Context, resource models.KubernetesResource) error {
	key := resource.Namespace + "/" + resource.Kind
	m.resources[key] = append(m.resources[key], resource)
	return nil
}

func (m *mockKubeClient) DeleteResource(ctx context.Context, resourceType, name, namespace string) error {
	key := namespace + "/" + resourceType
	resources := m.resources[key]
	for i, r := range resources {
		if r.Name == name {
			m.resources[key] = append(resources[:i], resources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("resource not found")
}

func (m *mockKubeClient) ResourceExists(ctx context.Context, resourceType, name, namespace string) (bool, error) {
	key := namespace + "/" + resourceType
	for _, r := range m.resources[key] {
		if r.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// setupTestManager creates a BackupManager with a temporary directory for testing.
func setupTestManager(t *testing.T) (*BackupManager, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "aegis-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	kubeClient := newMockKubeClient()
	manager := NewBackupManager(kubeClient, storage, tmpDir, 30)

	return manager, tmpDir
}

func TestCreateJob(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	t.Run("successful job creation", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "test-backup",
			Namespace:     "default",
			ResourceTypes: []string{"Deployment", "Service"},
			Schedule:      "@daily",
		}

		created, err := mgr.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if created.ID == "" {
			t.Error("Expected job ID to be set")
		}
		if created.Name != "test-backup" {
			t.Errorf("Expected name 'test-backup', got %q", created.Name)
		}
		if created.Status != models.BackupStatusActive {
			t.Errorf("Expected status 'active', got %q", created.Status)
		}
		if created.RetentionDays != 30 {
			t.Errorf("Expected default retention of 30 days, got %d", created.RetentionDays)
		}
		if created.NextRun == nil {
			t.Error("Expected NextRun to be set")
		}
	})

	t.Run("missing name returns error", func(t *testing.T) {
		job := models.BackupJob{
			Namespace:     "default",
			ResourceTypes: []string{"Deployment"},
			Schedule:      "@daily",
		}

		_, err := mgr.CreateJob(ctx, job)
		if err == nil {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "test",
			ResourceTypes: []string{"Deployment"},
			Schedule:      "@daily",
		}

		_, err := mgr.CreateJob(ctx, job)
		if err == nil {
			t.Error("Expected error for missing namespace")
		}
	})

	t.Run("empty resource types returns error", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "test",
			Namespace:     "default",
			ResourceTypes: []string{},
			Schedule:      "@daily",
		}

		_, err := mgr.CreateJob(ctx, job)
		if err == nil {
			t.Error("Expected error for empty resource types")
		}
	})

	t.Run("missing schedule returns error", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "test",
			Namespace:     "default",
			ResourceTypes: []string{"Deployment"},
		}

		_, err := mgr.CreateJob(ctx, job)
		if err == nil {
			t.Error("Expected error for missing schedule")
		}
	})

	t.Run("custom retention days preserved", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "custom-retention",
			Namespace:     "default",
			ResourceTypes: []string{"Deployment"},
			Schedule:      "@hourly",
			RetentionDays: 90,
		}

		created, err := mgr.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if created.RetentionDays != 90 {
			t.Errorf("Expected retention of 90 days, got %d", created.RetentionDays)
		}
	})
}

func TestExecuteBackup(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	t.Run("successful backup execution", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "exec-test",
			Namespace:     "default",
			ResourceTypes: []string{"Deployment", "Service", "ConfigMap"},
			Schedule:      "@daily",
		}

		created, err := mgr.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		record, err := mgr.ExecuteBackup(ctx, created.ID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if record.Status != models.RecordStatusCompleted {
			t.Errorf("Expected status 'completed', got %q", record.Status)
		}
		if record.ResourceCount != 4 { // 2 deployments + 1 service + 1 configmap
			t.Errorf("Expected 4 resources, got %d", record.ResourceCount)
		}
		if record.SizeBytes <= 0 {
			t.Errorf("Expected positive size, got %d", record.SizeBytes)
		}
		if record.DurationMs < 0 {
			t.Errorf("Expected non-negative duration, got %d", record.DurationMs)
		}
		if record.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}
		if record.StoragePath == "" {
			t.Error("Expected StoragePath to be set")
		}
	})

	t.Run("backup for nonexistent job returns error", func(t *testing.T) {
		_, err := mgr.ExecuteBackup(ctx, "nonexistent-job")
		if err == nil {
			t.Error("Expected error for nonexistent job")
		}
	})

	t.Run("backup updates job last run", func(t *testing.T) {
		job := models.BackupJob{
			Name:          "lastrun-test",
			Namespace:     "default",
			ResourceTypes: []string{"Deployment"},
			Schedule:      "@daily",
		}

		created, err := mgr.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		if created.LastRun != nil {
			t.Error("Expected LastRun to be nil before execution")
		}

		_, err = mgr.ExecuteBackup(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to execute backup: %v", err)
		}

		updated, err := mgr.GetJob(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}

		if updated.LastRun == nil {
			t.Error("Expected LastRun to be set after execution")
		}
	})
}

func TestListBackups(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	job := models.BackupJob{
		Name:          "list-test",
		Namespace:     "default",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@daily",
	}

	created, err := mgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Execute two backups
	_, err = mgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute first backup: %v", err)
	}

	_, err = mgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute second backup: %v", err)
	}

	records, err := mgr.ListBackups(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 backup records, got %d", len(records))
	}
}

func TestDeleteBackup(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	job := models.BackupJob{
		Name:          "delete-test",
		Namespace:     "default",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@daily",
	}

	created, err := mgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	record, err := mgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Verify record exists
	records, err := mgr.ListBackups(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	// Delete the backup
	err = mgr.DeleteBackup(ctx, record.ID)
	if err != nil {
		t.Fatalf("Failed to delete backup: %v", err)
	}

	// Verify record is gone
	records, err = mgr.ListBackups(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to list backups after delete: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected 0 records after delete, got %d", len(records))
	}
}

func TestEnforceRetention(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	job := models.BackupJob{
		Name:          "retention-test",
		Namespace:     "default",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@daily",
		RetentionDays: 7,
	}

	created, err := mgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Execute a backup
	record, err := mgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Manually set the completed time to 10 days ago (exceeds 7-day retention)
	oldTime := time.Now().UTC().AddDate(0, 0, -10)
	record.CompletedAt = &oldTime

	// Enforce retention
	deleted, err := mgr.EnforceRetention(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to enforce retention: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 backup deleted, got %d", deleted)
	}

	// Verify record is gone
	records, err := mgr.ListBackups(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected 0 records after retention, got %d", len(records))
	}
}

func TestScheduleBackups(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create a job with a schedule that is already past due
	job := models.BackupJob{
		Name:          "schedule-test",
		Namespace:     "default",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@hourly",
	}

	created, err := mgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Manually set NextRun to the past
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	mgr.mu.Lock()
	mgr.jobs[created.ID].NextRun = &pastTime
	mgr.mu.Unlock()

	due, err := mgr.ScheduleBackups(ctx)
	if err != nil {
		t.Fatalf("Failed to check schedule: %v", err)
	}

	if len(due) != 1 {
		t.Errorf("Expected 1 due job, got %d", len(due))
	}
}

func TestCreateArchive(t *testing.T) {
	manifest := models.BackupManifest{
		BackupID:  "test-archive",
		JobID:     "test-job",
		Namespace: "default",
		ResourceTypes: []string{"Deployment"},
		Resources: []models.KubernetesResource{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-app",
				Namespace:  "default",
				Manifest:   []byte(`{"kind":"Deployment","name":"test-app"}`),
			},
		},
		ResourceCount: 1,
		CreatedAt:     time.Now().UTC(),
	}

	data, checksum, err := createArchive(manifest)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty archive data")
	}
	if checksum == "" {
		t.Error("Expected non-empty checksum")
	}
	if len(checksum) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("Expected checksum length 64, got %d", len(checksum))
	}
}

func TestCalculateNextRun(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) // Monday

	tests := []struct {
		schedule string
		expected time.Time
	}{
		{"@hourly", time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)},
		{"@daily", time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)},
		{"*/5 * * * *", now.Add(5 * time.Minute)},
		{"*/30 * * * *", now.Add(30 * time.Minute)},
		{"unknown", now.Add(time.Hour)}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.schedule, func(t *testing.T) {
			result := calculateNextRun(tt.schedule, now)
			if !result.Equal(tt.expected) {
				t.Errorf("Schedule %q: expected %v, got %v", tt.schedule, tt.expected, result)
			}
		})
	}
}
