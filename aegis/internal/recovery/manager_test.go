package recovery

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// mockKubeClient is a test implementation of the KubeClient interface.
type mockKubeClient struct {
	resources   map[string][]models.KubernetesResource
	applyErrors map[string]error // resource name -> error
}

func newMockKubeClient() *mockKubeClient {
	client := &mockKubeClient{
		resources:   make(map[string][]models.KubernetesResource),
		applyErrors: make(map[string]error),
	}

	// Populate with test resources
	client.resources["default/Deployment"] = []models.KubernetesResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-app",
			Namespace:  "default",
			Labels:     map[string]string{"app": "test"},
			Manifest:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-app"}}`),
		},
	}
	client.resources["default/Service"] = []models.KubernetesResource{
		{
			APIVersion: "v1",
			Kind:       "Service",
			Name:       "test-service",
			Namespace:  "default",
			Labels:     map[string]string{"app": "test"},
			Manifest:   []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"test-service"}}`),
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
	if err, ok := m.applyErrors[resource.Name]; ok {
		return err
	}
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

// setupTestManagers creates both a BackupManager and RecoveryManager for testing.
func setupTestManagers(t *testing.T) (*backup.BackupManager, *RecoveryManager, *mockKubeClient, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "aegis-recovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	storage, err := backup.NewLocalStorage(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	kubeClient := newMockKubeClient()
	backupMgr := backup.NewBackupManager(kubeClient, storage, tmpDir, 30)
	recoveryMgr := NewRecoveryManager(kubeClient, backupMgr, storage)

	return backupMgr, recoveryMgr, kubeClient, tmpDir
}

// createTestBackup creates a backup job, executes it, and returns the record.
func createTestBackup(t *testing.T, ctx context.Context, backupMgr *backup.BackupManager) *models.BackupRecord {
	t.Helper()

	job := models.BackupJob{
		Name:          "recovery-test-backup",
		Namespace:     "default",
		ResourceTypes: []string{"Deployment", "Service"},
		Schedule:      "@daily",
	}

	created, err := backupMgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create backup job: %v", err)
	}

	record, err := backupMgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	return record
}

func TestCreatePlan(t *testing.T) {
	backupMgr, recoveryMgr, _, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	t.Run("successful plan creation", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:            "test-recovery",
			BackupID:        record.ID,
			TargetNamespace: "recovery-ns",
		}

		created, err := recoveryMgr.CreatePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if created.ID == "" {
			t.Error("Expected plan ID to be set")
		}
		if created.Strategy != models.RecoveryStrategyInPlace {
			t.Errorf("Expected default strategy 'in_place', got %q", created.Strategy)
		}
		if created.ConflictPolicy != models.ConflictStrategySkip {
			t.Errorf("Expected default conflict policy 'skip', got %q", created.ConflictPolicy)
		}
		if created.Status != models.ExecutionStatusPending {
			t.Errorf("Expected status 'pending', got %q", created.Status)
		}
	})

	t.Run("missing name returns error", func(t *testing.T) {
		plan := models.RecoveryPlan{
			BackupID:        record.ID,
			TargetNamespace: "recovery-ns",
		}
		_, err := recoveryMgr.CreatePlan(ctx, plan)
		if err == nil {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("missing backup ID returns error", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:            "test",
			TargetNamespace: "recovery-ns",
		}
		_, err := recoveryMgr.CreatePlan(ctx, plan)
		if err == nil {
			t.Error("Expected error for missing backup ID")
		}
	})

	t.Run("nonexistent backup returns error", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:            "test",
			BackupID:        "nonexistent-backup",
			TargetNamespace: "recovery-ns",
		}
		_, err := recoveryMgr.CreatePlan(ctx, plan)
		if err == nil {
			t.Error("Expected error for nonexistent backup")
		}
	})

	t.Run("missing target namespace returns error", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:     "test",
			BackupID: record.ID,
		}
		_, err := recoveryMgr.CreatePlan(ctx, plan)
		if err == nil {
			t.Error("Expected error for missing target namespace")
		}
	})

	t.Run("custom strategy preserved", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:            "cross-cluster-test",
			BackupID:        record.ID,
			TargetNamespace: "new-cluster-ns",
			Strategy:        models.RecoveryStrategyCrossCluster,
			ConflictPolicy:  models.ConflictStrategyOverwrite,
		}

		created, err := recoveryMgr.CreatePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if created.Strategy != models.RecoveryStrategyCrossCluster {
			t.Errorf("Expected strategy 'cross_cluster', got %q", created.Strategy)
		}
		if created.ConflictPolicy != models.ConflictStrategyOverwrite {
			t.Errorf("Expected conflict policy 'overwrite', got %q", created.ConflictPolicy)
		}
	})
}

func TestExecuteRecovery(t *testing.T) {
	backupMgr, recoveryMgr, _, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	t.Run("successful recovery execution", func(t *testing.T) {
		plan := models.RecoveryPlan{
			Name:            "exec-recovery",
			BackupID:        record.ID,
			TargetNamespace: "restored-ns",
		}

		created, err := recoveryMgr.CreatePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to create plan: %v", err)
		}

		execution, err := recoveryMgr.ExecuteRecovery(ctx, created.ID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if execution.Status != models.ExecutionStatusCompleted {
			t.Errorf("Expected status 'completed', got %q", execution.Status)
		}
		if execution.ResourcesRestored != 2 { // 1 deployment + 1 service
			t.Errorf("Expected 2 resources restored, got %d", execution.ResourcesRestored)
		}
		if execution.DryRun {
			t.Error("Expected DryRun to be false")
		}
		if execution.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}
	})

	t.Run("recovery for nonexistent plan returns error", func(t *testing.T) {
		_, err := recoveryMgr.ExecuteRecovery(ctx, "nonexistent-plan")
		if err == nil {
			t.Error("Expected error for nonexistent plan")
		}
	})
}

func TestDryRun(t *testing.T) {
	backupMgr, recoveryMgr, _, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	plan := models.RecoveryPlan{
		Name:            "dry-run-test",
		BackupID:        record.ID,
		TargetNamespace: "dry-run-ns",
	}

	created, err := recoveryMgr.CreatePlan(ctx, plan)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	execution, err := recoveryMgr.DryRun(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !execution.DryRun {
		t.Error("Expected DryRun to be true")
	}
	if execution.Status != models.ExecutionStatusCompleted {
		t.Errorf("Expected status 'completed', got %q", execution.Status)
	}
	if execution.ResourcesRestored != 2 {
		t.Errorf("Expected 2 resources in dry run, got %d", execution.ResourcesRestored)
	}
}

func TestValidateBackup(t *testing.T) {
	backupMgr, recoveryMgr, _, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	t.Run("valid backup passes validation", func(t *testing.T) {
		err := recoveryMgr.ValidateBackup(ctx, record.ID)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("nonexistent backup fails validation", func(t *testing.T) {
		err := recoveryMgr.ValidateBackup(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent backup")
		}
	})
}

func TestFilterResources(t *testing.T) {
	resources := []models.KubernetesResource{
		{Kind: "Deployment", Name: "app-1"},
		{Kind: "Service", Name: "svc-1"},
		{Kind: "ConfigMap", Name: "cm-1"},
		{Kind: "Deployment", Name: "app-2"},
		{Kind: "Secret", Name: "secret-1"},
	}

	t.Run("empty filter returns all resources", func(t *testing.T) {
		result := filterResources(resources, nil)
		if len(result) != 5 {
			t.Errorf("Expected 5 resources, got %d", len(result))
		}
	})

	t.Run("single filter", func(t *testing.T) {
		result := filterResources(resources, []string{"Deployment"})
		if len(result) != 2 {
			t.Errorf("Expected 2 Deployments, got %d", len(result))
		}
	})

	t.Run("multiple filters", func(t *testing.T) {
		result := filterResources(resources, []string{"Deployment", "Service"})
		if len(result) != 3 {
			t.Errorf("Expected 3 resources (2 Deployments + 1 Service), got %d", len(result))
		}
	})

	t.Run("case insensitive filter", func(t *testing.T) {
		result := filterResources(resources, []string{"deployment"})
		if len(result) != 2 {
			t.Errorf("Expected 2 Deployments (case insensitive), got %d", len(result))
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		result := filterResources(resources, []string{"StatefulSet"})
		if len(result) != 0 {
			t.Errorf("Expected 0 resources, got %d", len(result))
		}
	})
}

func TestRecoveryWithConflictSkip(t *testing.T) {
	backupMgr, recoveryMgr, kubeClient, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	// Pre-populate the target namespace with a conflicting resource
	kubeClient.resources["restored-ns/Deployment"] = []models.KubernetesResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-app",
			Namespace:  "restored-ns",
		},
	}

	plan := models.RecoveryPlan{
		Name:            "conflict-skip-test",
		BackupID:        record.ID,
		TargetNamespace: "restored-ns",
		ConflictPolicy:  models.ConflictStrategySkip,
	}

	created, err := recoveryMgr.CreatePlan(ctx, plan)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	execution, err := recoveryMgr.ExecuteRecovery(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The test-app Deployment should be skipped because it already exists
	if execution.ResourcesSkipped != 1 {
		t.Errorf("Expected 1 skipped resource, got %d", execution.ResourcesSkipped)
	}
	// test-service should still be restored
	if execution.ResourcesRestored != 1 {
		t.Errorf("Expected 1 restored resource, got %d", execution.ResourcesRestored)
	}
}

func TestListExecutions(t *testing.T) {
	backupMgr, recoveryMgr, _, tmpDir := setupTestManagers(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	record := createTestBackup(t, ctx, backupMgr)

	plan := models.RecoveryPlan{
		Name:            "executions-list-test",
		BackupID:        record.ID,
		TargetNamespace: "list-test-ns",
	}

	created, err := recoveryMgr.CreatePlan(ctx, plan)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Execute recovery twice
	_, err = recoveryMgr.ExecuteRecovery(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed first recovery: %v", err)
	}

	_, err = recoveryMgr.DryRun(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed dry run: %v", err)
	}

	executions, err := recoveryMgr.ListExecutions(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to list executions: %v", err)
	}

	if len(executions) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(executions))
	}

	// Verify one is a dry run and one is not
	dryRunCount := 0
	for _, exec := range executions {
		if exec.DryRun {
			dryRunCount++
		}
	}
	if dryRunCount != 1 {
		t.Errorf("Expected 1 dry run execution, got %d", dryRunCount)
	}
}
