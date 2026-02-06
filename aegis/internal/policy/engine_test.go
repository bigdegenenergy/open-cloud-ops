package policy

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
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

	client.resources["production/Deployment"] = []models.KubernetesResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "prod-app",
			Namespace:  "production",
			Labels:     map[string]string{"app": "prod"},
			Manifest:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"prod-app"}}`),
		},
	}
	client.resources["production/Service"] = []models.KubernetesResource{
		{
			APIVersion: "v1",
			Kind:       "Service",
			Name:       "prod-service",
			Namespace:  "production",
			Labels:     map[string]string{"app": "prod"},
			Manifest:   []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"prod-service"}}`),
		},
	}
	client.resources["staging/Deployment"] = []models.KubernetesResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "staging-app",
			Namespace:  "staging",
			Labels:     map[string]string{"app": "staging"},
			Manifest:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"staging-app"}}`),
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
	return fmt.Errorf("not implemented")
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

// setupTestPolicyEngine creates a PolicyEngine with a BackupManager for testing.
func setupTestPolicyEngine(t *testing.T) (*PolicyEngine, *backup.BackupManager, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "aegis-policy-test-*")
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
	engine := NewPolicyEngine(backupMgr)

	return engine, backupMgr, tmpDir
}

func TestCreatePolicy(t *testing.T) {
	engine, _, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	t.Run("successful policy creation", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:           "production-dr",
			Description:    "Production disaster recovery policy",
			RPOMinutes:     60,
			RTOMinutes:     30,
			BackupSchedule: "@hourly",
			Namespaces:     []string{"production"},
			Enabled:        true,
		}

		created, err := engine.CreatePolicy(ctx, policy)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if created.ID == "" {
			t.Error("Expected policy ID to be set")
		}
		if created.Name != "production-dr" {
			t.Errorf("Expected name 'production-dr', got %q", created.Name)
		}
		if created.RPOMinutes != 60 {
			t.Errorf("Expected RPO 60 minutes, got %d", created.RPOMinutes)
		}
		if created.RTOMinutes != 30 {
			t.Errorf("Expected RTO 30 minutes, got %d", created.RTOMinutes)
		}
		if created.RetentionDays != 30 {
			t.Errorf("Expected default retention 30 days, got %d", created.RetentionDays)
		}
		if created.Priority != 1 {
			t.Errorf("Expected default priority 1, got %d", created.Priority)
		}
	})

	t.Run("missing name returns error", func(t *testing.T) {
		policy := models.DRPolicy{
			RPOMinutes:     60,
			RTOMinutes:     30,
			BackupSchedule: "@hourly",
			Namespaces:     []string{"production"},
		}
		_, err := engine.CreatePolicy(ctx, policy)
		if err == nil {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("invalid RPO returns error", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:           "test",
			RPOMinutes:     0,
			RTOMinutes:     30,
			BackupSchedule: "@hourly",
			Namespaces:     []string{"production"},
		}
		_, err := engine.CreatePolicy(ctx, policy)
		if err == nil {
			t.Error("Expected error for zero RPO")
		}
	})

	t.Run("invalid RTO returns error", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:           "test",
			RPOMinutes:     60,
			RTOMinutes:     -1,
			BackupSchedule: "@hourly",
			Namespaces:     []string{"production"},
		}
		_, err := engine.CreatePolicy(ctx, policy)
		if err == nil {
			t.Error("Expected error for negative RTO")
		}
	})

	t.Run("empty namespaces returns error", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:           "test",
			RPOMinutes:     60,
			RTOMinutes:     30,
			BackupSchedule: "@hourly",
			Namespaces:     []string{},
		}
		_, err := engine.CreatePolicy(ctx, policy)
		if err == nil {
			t.Error("Expected error for empty namespaces")
		}
	})

	t.Run("missing schedule returns error", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:       "test",
			RPOMinutes: 60,
			RTOMinutes: 30,
			Namespaces: []string{"production"},
		}
		_, err := engine.CreatePolicy(ctx, policy)
		if err == nil {
			t.Error("Expected error for missing schedule")
		}
	})

	t.Run("custom retention and priority preserved", func(t *testing.T) {
		policy := models.DRPolicy{
			Name:           "critical-policy",
			RPOMinutes:     15,
			RTOMinutes:     5,
			BackupSchedule: "*/15 * * * *",
			RetentionDays:  90,
			Priority:       5,
			Namespaces:     []string{"production"},
			Enabled:        true,
		}

		created, err := engine.CreatePolicy(ctx, policy)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if created.RetentionDays != 90 {
			t.Errorf("Expected retention 90, got %d", created.RetentionDays)
		}
		if created.Priority != 5 {
			t.Errorf("Expected priority 5, got %d", created.Priority)
		}
	})
}

func TestEvaluateCompliance_NoBackups(t *testing.T) {
	engine, _, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create a policy for a namespace that has no backups
	policy := models.DRPolicy{
		Name:           "unbackedup-policy",
		RPOMinutes:     60,
		RTOMinutes:     30,
		BackupSchedule: "@hourly",
		Namespaces:     []string{"production"},
		Enabled:        true,
	}

	_, err := engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	if report.OverallStatus != "non_compliant" {
		t.Errorf("Expected overall status 'non_compliant', got %q", report.OverallStatus)
	}
	if report.TotalPolicies != 1 {
		t.Errorf("Expected 1 total policy, got %d", report.TotalPolicies)
	}
	if report.ViolationCount == 0 {
		t.Error("Expected at least one violation for namespace without backups")
	}

	// Check that the violation is of the correct type
	foundMissing := false
	for _, v := range report.Violations {
		if v.ViolationType == "missing_backup" {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Error("Expected 'missing_backup' violation type")
	}
}

func TestEvaluateCompliance_WithBackups(t *testing.T) {
	engine, backupMgr, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create a backup job and execute it for the production namespace
	job := models.BackupJob{
		Name:          "prod-backup",
		Namespace:     "production",
		ResourceTypes: []string{"Deployment", "Service"},
		Schedule:      "@hourly",
		RetentionDays: 30,
	}

	created, err := backupMgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create backup job: %v", err)
	}

	_, err = backupMgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Create a policy that the backup should satisfy
	policy := models.DRPolicy{
		Name:           "prod-policy",
		RPOMinutes:     120, // 2 hours - our backup is fresh
		RTOMinutes:     60,
		BackupSchedule: "@hourly",
		RetentionDays:  30,
		Namespaces:     []string{"production"},
		Enabled:        true,
	}

	_, err = engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	if report.OverallStatus != "compliant" {
		t.Errorf("Expected overall status 'compliant', got %q", report.OverallStatus)
		for _, v := range report.Violations {
			t.Logf("Violation: %s - %s", v.ViolationType, v.Description)
		}
	}
	if report.CompliantCount != 1 {
		t.Errorf("Expected 1 compliant policy, got %d", report.CompliantCount)
	}
}

func TestEvaluateCompliance_RPOViolation(t *testing.T) {
	engine, backupMgr, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create and execute a backup
	job := models.BackupJob{
		Name:          "old-backup",
		Namespace:     "production",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@daily",
		RetentionDays: 30,
	}

	created, err := backupMgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	record, err := backupMgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Manually make the backup appear old (3 hours ago)
	oldTime := time.Now().UTC().Add(-3 * time.Hour)
	record.CompletedAt = &oldTime

	// Create a strict RPO policy (1 hour)
	policy := models.DRPolicy{
		Name:           "strict-rpo",
		RPOMinutes:     60, // 1 hour - backup is 3 hours old
		RTOMinutes:     30,
		BackupSchedule: "@hourly",
		RetentionDays:  30,
		Namespaces:     []string{"production"},
		Enabled:        true,
	}

	_, err = engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	if report.OverallStatus != "non_compliant" {
		t.Errorf("Expected 'non_compliant' status due to RPO violation, got %q", report.OverallStatus)
	}

	// Check for RPO violation
	foundRPO := false
	for _, v := range report.Violations {
		if v.ViolationType == "rpo" {
			foundRPO = true
			break
		}
	}
	if !foundRPO {
		t.Error("Expected 'rpo' violation type")
	}
}

func TestEvaluateCompliance_RetentionViolation(t *testing.T) {
	engine, backupMgr, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create a backup job with short retention
	job := models.BackupJob{
		Name:          "short-retention",
		Namespace:     "production",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@daily",
		RetentionDays: 7, // Short retention
	}

	created, err := backupMgr.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	_, err = backupMgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Create a policy requiring longer retention
	policy := models.DRPolicy{
		Name:           "long-retention-policy",
		RPOMinutes:     1440, // 24 hours - lenient RPO
		RTOMinutes:     60,
		BackupSchedule: "@daily",
		RetentionDays:  90, // Requires 90 days, but job only has 7
		Namespaces:     []string{"production"},
		Enabled:        true,
	}

	_, err = engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	// Should have a retention violation
	foundRetention := false
	for _, v := range report.Violations {
		if v.ViolationType == "retention" {
			foundRetention = true
			break
		}
	}
	if !foundRetention {
		t.Error("Expected 'retention' violation type")
		for _, v := range report.Violations {
			t.Logf("Found violation: %s - %s", v.ViolationType, v.Description)
		}
	}
}

func TestEvaluateCompliance_DisabledPolicySkipped(t *testing.T) {
	engine, _, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create a disabled policy
	policy := models.DRPolicy{
		Name:           "disabled-policy",
		RPOMinutes:     60,
		RTOMinutes:     30,
		BackupSchedule: "@hourly",
		Namespaces:     []string{"production"},
		Enabled:        false, // Disabled
	}

	_, err := engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	if report.TotalPolicies != 0 {
		t.Errorf("Expected 0 evaluated policies (disabled), got %d", report.TotalPolicies)
	}
	if report.OverallStatus != "compliant" {
		t.Errorf("Expected 'compliant' with no policies, got %q", report.OverallStatus)
	}
}

func TestDeletePolicy(t *testing.T) {
	engine, _, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	policy := models.DRPolicy{
		Name:           "to-delete",
		RPOMinutes:     60,
		RTOMinutes:     30,
		BackupSchedule: "@hourly",
		Namespaces:     []string{"production"},
	}

	created, err := engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Verify it exists
	policies, err := engine.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("Expected 1 policy, got %d", len(policies))
	}

	// Delete it
	err = engine.DeletePolicy(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify it's gone
	policies, err = engine.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies after deletion, got %d", len(policies))
	}
}

func TestDeletePolicy_NotFound(t *testing.T) {
	engine, _, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	err := engine.DeletePolicy(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent policy")
	}
}

func TestSeverityForPriority(t *testing.T) {
	tests := []struct {
		priority int
		expected string
	}{
		{1, "info"},
		{2, "info"},
		{3, "warning"},
		{4, "warning"},
		{5, "critical"},
		{10, "critical"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("priority_%d", tt.priority), func(t *testing.T) {
			result := severityForPriority(tt.priority)
			if result != tt.expected {
				t.Errorf("Priority %d: expected %q, got %q", tt.priority, tt.expected, result)
			}
		})
	}
}

func TestMultiplePoliciesMultipleNamespaces(t *testing.T) {
	engine, backupMgr, tmpDir := setupTestPolicyEngine(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create backup for production only
	prodJob := models.BackupJob{
		Name:          "prod-backup",
		Namespace:     "production",
		ResourceTypes: []string{"Deployment"},
		Schedule:      "@hourly",
		RetentionDays: 30,
	}

	created, err := backupMgr.CreateJob(ctx, prodJob)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	_, err = backupMgr.ExecuteBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to execute backup: %v", err)
	}

	// Policy covering both production AND staging
	policy := models.DRPolicy{
		Name:           "multi-ns-policy",
		RPOMinutes:     120,
		RTOMinutes:     60,
		BackupSchedule: "@hourly",
		RetentionDays:  30,
		Namespaces:     []string{"production", "staging"},
		Enabled:        true,
	}

	_, err = engine.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	report, err := engine.EvaluateCompliance(ctx)
	if err != nil {
		t.Fatalf("Failed to evaluate compliance: %v", err)
	}

	// Should be non-compliant because staging has no backups
	if report.OverallStatus != "non_compliant" {
		t.Errorf("Expected 'non_compliant' (staging has no backups), got %q", report.OverallStatus)
	}

	// Should have a missing_backup violation for staging
	foundStagingViolation := false
	for _, v := range report.Violations {
		if v.Namespace == "staging" && v.ViolationType == "missing_backup" {
			foundStagingViolation = true
			break
		}
	}
	if !foundStagingViolation {
		t.Error("Expected missing_backup violation for staging namespace")
	}
}
