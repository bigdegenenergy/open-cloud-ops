// Package recovery implements disaster recovery orchestration for Kubernetes resources.
//
// The recovery manager handles creating recovery plans, executing them against
// backup data, performing dry runs for validation, and tracking execution results.
// It integrates with the backup storage layer to load archived resources and
// the Kubernetes client interface to apply them to target clusters/namespaces.
package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// RecoveryManager orchestrates disaster recovery operations including
// plan management, execution, dry runs, and validation.
type RecoveryManager struct {
	kubeClient    backup.KubeClient
	backupManager *backup.BackupManager
	storage       backup.StorageBackend

	mu         sync.RWMutex
	plans      map[string]*models.RecoveryPlan
	executions map[string][]*models.RecoveryExecution // planID -> executions
}

// NewRecoveryManager creates a new RecoveryManager with the given dependencies.
func NewRecoveryManager(kubeClient backup.KubeClient, backupManager *backup.BackupManager, storage backup.StorageBackend) *RecoveryManager {
	return &RecoveryManager{
		kubeClient:    kubeClient,
		backupManager: backupManager,
		storage:       storage,
		plans:         make(map[string]*models.RecoveryPlan),
		executions:    make(map[string][]*models.RecoveryExecution),
	}
}

// CreatePlan creates a new recovery plan after validating its configuration.
// The referenced backup must exist and be in a completed state.
func (m *RecoveryManager) CreatePlan(ctx context.Context, plan models.RecoveryPlan) (*models.RecoveryPlan, error) {
	if plan.Name == "" {
		return nil, fmt.Errorf("recovery: plan name is required")
	}
	if plan.BackupID == "" {
		return nil, fmt.Errorf("recovery: backup_id is required")
	}
	if plan.TargetNamespace == "" {
		return nil, fmt.Errorf("recovery: target_namespace is required")
	}

	// Validate that the referenced backup exists
	record, err := m.backupManager.GetBackupRecord(ctx, plan.BackupID)
	if err != nil {
		return nil, fmt.Errorf("recovery: referenced backup not found: %w", err)
	}
	if record.Status != models.RecordStatusCompleted {
		return nil, fmt.Errorf("recovery: backup %s is not in completed state (current: %s)", plan.BackupID, record.Status)
	}

	now := time.Now().UTC()

	if plan.ID == "" {
		plan.ID = fmt.Sprintf("rp-%d", now.UnixNano())
	}
	if plan.Strategy == "" {
		plan.Strategy = models.RecoveryStrategyInPlace
	}
	if plan.ConflictPolicy == "" {
		plan.ConflictPolicy = models.ConflictStrategySkip
	}
	if plan.Status == "" {
		plan.Status = models.ExecutionStatusPending
	}
	plan.CreatedAt = now

	m.mu.Lock()
	defer m.mu.Unlock()

	m.plans[plan.ID] = &plan
	m.executions[plan.ID] = make([]*models.RecoveryExecution, 0)

	log.Printf("recovery: created plan %s (%s) targeting namespace %s from backup %s",
		plan.ID, plan.Name, plan.TargetNamespace, plan.BackupID)
	return &plan, nil
}

// GetPlan retrieves a recovery plan by ID.
func (m *RecoveryManager) GetPlan(ctx context.Context, planID string) (*models.RecoveryPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plan, exists := m.plans[planID]
	if !exists {
		return nil, fmt.Errorf("recovery: plan %q not found", planID)
	}
	return plan, nil
}

// ListPlans returns all registered recovery plans.
func (m *RecoveryManager) ListPlans(ctx context.Context) ([]*models.RecoveryPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plans := make([]*models.RecoveryPlan, 0, len(m.plans))
	for _, plan := range m.plans {
		plans = append(plans, plan)
	}
	return plans, nil
}

// ExecuteRecovery runs a recovery plan. It loads the backup data, deserializes
// resources, applies filters, and restores resources to the target namespace.
// The execution tracks progress and handles conflicts according to the plan's
// conflict policy.
func (m *RecoveryManager) ExecuteRecovery(ctx context.Context, planID string) (*models.RecoveryExecution, error) {
	return m.executeRecoveryInternal(ctx, planID, false)
}

// DryRun simulates a recovery execution without actually applying resources.
// It performs all the same steps as a real recovery (loading backup, filtering,
// conflict detection) but skips the actual Kubernetes apply step. This is useful
// for validating a recovery plan before executing it.
func (m *RecoveryManager) DryRun(ctx context.Context, planID string) (*models.RecoveryExecution, error) {
	return m.executeRecoveryInternal(ctx, planID, true)
}

// executeRecoveryInternal is the shared implementation for both real and dry-run
// recovery executions.
func (m *RecoveryManager) executeRecoveryInternal(ctx context.Context, planID string, dryRun bool) (*models.RecoveryExecution, error) {
	m.mu.RLock()
	plan, exists := m.plans[planID]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("recovery: plan %q not found", planID)
	}
	planCopy := *plan
	m.mu.RUnlock()

	now := time.Now().UTC()
	execution := &models.RecoveryExecution{
		ID:        fmt.Sprintf("re-%d", now.UnixNano()),
		PlanID:    planID,
		Status:    models.ExecutionStatusRunning,
		DryRun:    dryRun,
		StartedAt: now,
	}

	// Store the initial execution
	m.mu.Lock()
	m.executions[planID] = append(m.executions[planID], execution)
	m.mu.Unlock()

	mode := "execution"
	if dryRun {
		mode = "dry-run"
	}
	log.Printf("recovery: starting %s %s for plan %s (%s)", mode, execution.ID, planID, planCopy.Name)

	// Load the backup manifest
	manifest, err := m.backupManager.LoadManifest(ctx, planCopy.BackupID)
	if err != nil {
		return m.completeExecution(execution, models.ExecutionStatusFailed,
			fmt.Sprintf("failed to load backup manifest: %v", err))
	}

	// Filter resources according to the plan
	filteredResources := filterResources(manifest.Resources, planCopy.ResourceFilters)

	if len(filteredResources) == 0 {
		return m.completeExecution(execution, models.ExecutionStatusCompleted,
			"no resources matched the filter criteria")
	}

	// Apply resources
	var errors []string
	restored := 0
	skipped := 0

	for _, resource := range filteredResources {
		select {
		case <-ctx.Done():
			execution.Errors = append(errors, "context cancelled")
			return m.completeExecution(execution, models.ExecutionStatusFailed, "context cancelled")
		default:
		}

		// Update the namespace according to the recovery strategy
		targetResource := resource
		targetResource.Namespace = planCopy.TargetNamespace

		// Check for conflicts
		exists, err := m.kubeClient.ResourceExists(ctx, targetResource.Kind, targetResource.Name, targetResource.Namespace)
		if err != nil {
			errMsg := fmt.Sprintf("failed to check existence of %s/%s in %s: %v",
				targetResource.Kind, targetResource.Name, targetResource.Namespace, err)
			errors = append(errors, errMsg)
			continue
		}

		if exists {
			switch planCopy.ConflictPolicy {
			case models.ConflictStrategySkip:
				log.Printf("recovery: skipping existing resource %s/%s in %s",
					targetResource.Kind, targetResource.Name, targetResource.Namespace)
				skipped++
				continue
			case models.ConflictStrategyOverwrite:
				log.Printf("recovery: overwriting existing resource %s/%s in %s",
					targetResource.Kind, targetResource.Name, targetResource.Namespace)
				// Proceed with apply, which will overwrite
			}
		}

		if dryRun {
			log.Printf("recovery: [dry-run] would apply %s/%s to namespace %s",
				targetResource.Kind, targetResource.Name, targetResource.Namespace)
			restored++
			continue
		}

		// Apply the resource
		if err := m.kubeClient.ApplyResource(ctx, targetResource); err != nil {
			errMsg := fmt.Sprintf("failed to apply %s/%s to %s: %v",
				targetResource.Kind, targetResource.Name, targetResource.Namespace, err)
			errors = append(errors, errMsg)
			log.Printf("recovery: %s", errMsg)
			continue
		}

		restored++
		log.Printf("recovery: restored %s/%s to namespace %s",
			targetResource.Kind, targetResource.Name, targetResource.Namespace)
	}

	execution.ResourcesRestored = restored
	execution.ResourcesSkipped = skipped
	execution.Errors = errors

	// Determine final status
	status := models.ExecutionStatusCompleted
	if len(errors) > 0 && restored == 0 {
		status = models.ExecutionStatusFailed
	} else if len(errors) > 0 {
		status = models.ExecutionStatusPartial
	}

	result, _ := m.completeExecution(execution, status, "")

	log.Printf("recovery: %s %s complete: %d restored, %d skipped, %d errors",
		mode, execution.ID, restored, skipped, len(errors))

	return result, nil
}

// completeExecution finalizes an execution record with the given status.
func (m *RecoveryManager) completeExecution(exec *models.RecoveryExecution, status models.ExecutionStatus, errMsg string) (*models.RecoveryExecution, error) {
	completedAt := time.Now().UTC()
	exec.Status = status
	exec.CompletedAt = &completedAt

	if errMsg != "" {
		exec.Errors = append(exec.Errors, errMsg)
	}

	if status == models.ExecutionStatusFailed {
		return exec, fmt.Errorf("recovery: execution failed: %s", strings.Join(exec.Errors, "; "))
	}
	return exec, nil
}

// ValidateBackup verifies the integrity of a backup by loading it from
// storage, parsing its manifest, and verifying the checksum.
func (m *RecoveryManager) ValidateBackup(ctx context.Context, backupID string) error {
	record, err := m.backupManager.GetBackupRecord(ctx, backupID)
	if err != nil {
		return fmt.Errorf("recovery: backup record not found: %w", err)
	}

	if record.Status != models.RecordStatusCompleted {
		return fmt.Errorf("recovery: backup %s is not completed (status: %s)", backupID, record.Status)
	}

	// Load the manifest
	manifest, err := m.backupManager.LoadManifest(ctx, backupID)
	if err != nil {
		return fmt.Errorf("recovery: failed to load manifest: %w", err)
	}

	// Verify the archive exists and checksum matches
	archiveData, err := m.storage.Read(ctx, record.StoragePath)
	if err != nil {
		return fmt.Errorf("recovery: failed to read archive: %w", err)
	}

	hash := sha256.Sum256(archiveData)
	actualChecksum := hex.EncodeToString(hash[:])

	if manifest.Checksum != "" && actualChecksum != manifest.Checksum {
		return fmt.Errorf("recovery: checksum mismatch for backup %s: expected %s, got %s",
			backupID, manifest.Checksum, actualChecksum)
	}

	// Verify resource count
	if manifest.ResourceCount != len(manifest.Resources) {
		return fmt.Errorf("recovery: resource count mismatch: manifest says %d, found %d",
			manifest.ResourceCount, len(manifest.Resources))
	}

	// Validate each resource has required fields
	for i, resource := range manifest.Resources {
		if resource.Kind == "" {
			return fmt.Errorf("recovery: resource %d has empty Kind", i)
		}
		if resource.Name == "" {
			return fmt.Errorf("recovery: resource %d has empty Name", i)
		}
	}

	log.Printf("recovery: backup %s validated successfully: %d resources, checksum OK",
		backupID, manifest.ResourceCount)
	return nil
}

// ListExecutions returns all executions for a given plan.
func (m *RecoveryManager) ListExecutions(ctx context.Context, planID string) ([]*models.RecoveryExecution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	executions, exists := m.executions[planID]
	if !exists {
		return nil, fmt.Errorf("recovery: plan %q not found", planID)
	}

	result := make([]*models.RecoveryExecution, len(executions))
	copy(result, executions)
	return result, nil
}

// filterResources applies resource type filters to a list of resources.
// If filters is empty, all resources are returned. Filters are matched
// case-insensitively against the resource Kind.
func filterResources(resources []models.KubernetesResource, filters []string) []models.KubernetesResource {
	if len(filters) == 0 {
		return resources
	}

	// Build a set of allowed kinds for O(1) lookup
	allowed := make(map[string]bool, len(filters))
	for _, f := range filters {
		allowed[strings.ToLower(strings.TrimSpace(f))] = true
	}

	var filtered []models.KubernetesResource
	for _, r := range resources {
		if allowed[strings.ToLower(r.Kind)] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// extractResourcesFromManifest is a helper to deserialize resource manifests.
// Each resource's Manifest field contains the full JSON representation.
func extractResourcesFromManifest(manifest *models.BackupManifest) ([]models.KubernetesResource, error) {
	resources := make([]models.KubernetesResource, 0, len(manifest.Resources))
	for _, r := range manifest.Resources {
		if r.Manifest != nil {
			var full models.KubernetesResource
			if err := json.Unmarshal(r.Manifest, &full); err != nil {
				return nil, fmt.Errorf("failed to deserialize resource %s/%s: %w", r.Kind, r.Name, err)
			}
			resources = append(resources, full)
		} else {
			resources = append(resources, r)
		}
	}
	return resources, nil
}
