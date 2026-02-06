// Package policy implements disaster recovery policy management and compliance evaluation.
//
// The policy engine allows organizations to define RPO (Recovery Point Objective)
// and RTO (Recovery Time Objective) requirements for their Kubernetes namespaces.
// It continuously evaluates backup state against these policies and generates
// compliance reports identifying violations. Auto-remediation can schedule
// missing backups to bring namespaces back into compliance.
package policy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// PolicyEngine manages DR policies and evaluates compliance against actual
// backup state. It integrates with the BackupManager to check backup
// frequency, retention, and recency.
type PolicyEngine struct {
	backupManager *backup.BackupManager

	mu       sync.RWMutex
	policies map[string]*models.DRPolicy
}

// NewPolicyEngine creates a new PolicyEngine with the given BackupManager.
func NewPolicyEngine(backupManager *backup.BackupManager) *PolicyEngine {
	return &PolicyEngine{
		backupManager: backupManager,
		policies:      make(map[string]*models.DRPolicy),
	}
}

// CreatePolicy registers a new DR policy after validating its configuration.
// RPO and RTO values must be positive, and at least one namespace must be specified.
func (e *PolicyEngine) CreatePolicy(ctx context.Context, policy models.DRPolicy) (*models.DRPolicy, error) {
	if policy.Name == "" {
		return nil, fmt.Errorf("policy: name is required")
	}
	if policy.RPOMinutes <= 0 {
		return nil, fmt.Errorf("policy: rpo_minutes must be positive")
	}
	if policy.RTOMinutes <= 0 {
		return nil, fmt.Errorf("policy: rto_minutes must be positive")
	}
	if len(policy.Namespaces) == 0 {
		return nil, fmt.Errorf("policy: at least one namespace is required")
	}
	if policy.BackupSchedule == "" {
		return nil, fmt.Errorf("policy: backup_schedule is required")
	}

	now := time.Now().UTC()

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("dp-%d", now.UnixNano())
	}
	if policy.RetentionDays <= 0 {
		policy.RetentionDays = 30 // Default retention
	}
	if policy.Priority <= 0 {
		policy.Priority = 1
	}
	policy.CreatedAt = now

	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies[policy.ID] = &policy

	log.Printf("policy: created policy %s (%s) with RPO=%dm, RTO=%dm for namespaces %v",
		policy.ID, policy.Name, policy.RPOMinutes, policy.RTOMinutes, policy.Namespaces)
	return &policy, nil
}

// GetPolicy retrieves a DR policy by ID.
func (e *PolicyEngine) GetPolicy(ctx context.Context, policyID string) (*models.DRPolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, exists := e.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy: %q not found", policyID)
	}
	return policy, nil
}

// ListPolicies returns all registered DR policies.
func (e *PolicyEngine) ListPolicies(ctx context.Context) ([]*models.DRPolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policies := make([]*models.DRPolicy, 0, len(e.policies))
	for _, p := range e.policies {
		policies = append(policies, p)
	}
	return policies, nil
}

// DeletePolicy removes a DR policy by ID.
func (e *PolicyEngine) DeletePolicy(ctx context.Context, policyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.policies[policyID]; !exists {
		return fmt.Errorf("policy: %q not found", policyID)
	}

	delete(e.policies, policyID)
	log.Printf("policy: deleted policy %s", policyID)
	return nil
}

// EvaluateCompliance checks all enabled policies against the current backup
// state and returns a compliance report. For each policy, it verifies:
//   - Backup frequency meets RPO requirements
//   - Estimated recovery time is within RTO
//   - Retention requirements are satisfied
//   - Backups exist for all covered namespaces
func (e *PolicyEngine) EvaluateCompliance(ctx context.Context) (*models.ComplianceReport, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	now := time.Now().UTC()
	report := &models.ComplianceReport{
		GeneratedAt:   now,
		TotalPolicies: 0,
		Violations:    make([]models.ComplianceViolation, 0),
	}

	// Get all backup jobs for cross-referencing
	allJobs, err := e.backupManager.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("policy: failed to list backup jobs: %w", err)
	}

	// Build a map of namespace -> jobs for quick lookup
	namespaceJobs := make(map[string][]*models.BackupJob)
	for _, job := range allJobs {
		namespaceJobs[job.Namespace] = append(namespaceJobs[job.Namespace], job)
	}

	compliant := 0

	for _, policy := range e.policies {
		if !policy.Enabled {
			continue
		}
		report.TotalPolicies++

		policyCompliant := true

		for _, namespace := range policy.Namespaces {
			jobs := namespaceJobs[namespace]

			// Check 1: Verify backups exist for the namespace
			if len(jobs) == 0 {
				report.Violations = append(report.Violations, models.ComplianceViolation{
					PolicyID:      policy.ID,
					PolicyName:    policy.Name,
					Namespace:     namespace,
					ViolationType: "missing_backup",
					Description:   fmt.Sprintf("No backup job configured for namespace %q", namespace),
					Severity:      severityForPriority(policy.Priority),
				})
				policyCompliant = false
				continue
			}

			// Check 2: Verify backup frequency meets RPO
			rpoDuration := time.Duration(policy.RPOMinutes) * time.Minute
			latestBackup := findLatestBackup(ctx, e.backupManager, jobs)

			if latestBackup == nil {
				report.Violations = append(report.Violations, models.ComplianceViolation{
					PolicyID:      policy.ID,
					PolicyName:    policy.Name,
					Namespace:     namespace,
					ViolationType: "rpo",
					Description:   fmt.Sprintf("No completed backups found for namespace %q", namespace),
					Severity:      "critical",
				})
				policyCompliant = false
				continue
			}

			if latestBackup.CompletedAt != nil {
				timeSinceBackup := now.Sub(*latestBackup.CompletedAt)
				if timeSinceBackup > rpoDuration {
					report.Violations = append(report.Violations, models.ComplianceViolation{
						PolicyID:      policy.ID,
						PolicyName:    policy.Name,
						Namespace:     namespace,
						ViolationType: "rpo",
						Description: fmt.Sprintf(
							"Last backup for namespace %q was %s ago (RPO: %s)",
							namespace,
							timeSinceBackup.Round(time.Minute),
							rpoDuration.Round(time.Minute),
						),
						Severity: severityForPriority(policy.Priority),
					})
					policyCompliant = false
				}
			}

			// Check 3: Estimate recovery time against RTO
			// Estimation based on backup size: ~100MB/minute restore rate (conservative)
			if latestBackup.SizeBytes > 0 {
				estimatedMinutes := float64(latestBackup.SizeBytes) / (100 * 1024 * 1024) // 100 MB/min
				if estimatedMinutes < 1 {
					estimatedMinutes = 1
				}
				if int(estimatedMinutes) > policy.RTOMinutes {
					report.Violations = append(report.Violations, models.ComplianceViolation{
						PolicyID:      policy.ID,
						PolicyName:    policy.Name,
						Namespace:     namespace,
						ViolationType: "rto",
						Description: fmt.Sprintf(
							"Estimated recovery time for namespace %q is %d minutes (RTO: %d minutes)",
							namespace, int(estimatedMinutes), policy.RTOMinutes,
						),
						Severity: "warning",
					})
					policyCompliant = false
				}
			}

			// Check 4: Verify retention compliance
			for _, job := range jobs {
				if job.RetentionDays < policy.RetentionDays {
					report.Violations = append(report.Violations, models.ComplianceViolation{
						PolicyID:      policy.ID,
						PolicyName:    policy.Name,
						Namespace:     namespace,
						ViolationType: "retention",
						Description: fmt.Sprintf(
							"Backup job %q has %d day retention (policy requires %d days)",
							job.Name, job.RetentionDays, policy.RetentionDays,
						),
						Severity: "warning",
					})
					policyCompliant = false
				}
			}
		}

		if policyCompliant {
			compliant++
		}
	}

	report.CompliantCount = compliant
	report.ViolationCount = len(report.Violations)

	if report.ViolationCount > 0 {
		report.OverallStatus = "non_compliant"
	} else {
		report.OverallStatus = "compliant"
	}

	log.Printf("policy: compliance evaluation complete: %d/%d policies compliant, %d violations",
		report.CompliantCount, report.TotalPolicies, report.ViolationCount)

	return report, nil
}

// GetComplianceReport is a convenience method that calls EvaluateCompliance
// and returns the full compliance report.
func (e *PolicyEngine) GetComplianceReport(ctx context.Context) (*models.ComplianceReport, error) {
	return e.EvaluateCompliance(ctx)
}

// AutoRemediate evaluates compliance and automatically schedules backup
// executions for namespaces that are missing backups or have stale backups
// that violate RPO requirements. Returns the number of backups triggered.
func (e *PolicyEngine) AutoRemediate(ctx context.Context) (int, error) {
	report, err := e.EvaluateCompliance(ctx)
	if err != nil {
		return 0, fmt.Errorf("policy: auto-remediation failed during evaluation: %w", err)
	}

	if report.OverallStatus == "compliant" {
		log.Printf("policy: auto-remediation: all policies compliant, no action needed")
		return 0, nil
	}

	// Collect namespaces that need immediate backup
	namespacesNeedingBackup := make(map[string]bool)
	for _, violation := range report.Violations {
		if violation.ViolationType == "rpo" || violation.ViolationType == "missing_backup" {
			namespacesNeedingBackup[violation.Namespace] = true
		}
	}

	// Find backup jobs for affected namespaces and trigger them
	allJobs, err := e.backupManager.ListJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("policy: failed to list jobs for remediation: %w", err)
	}

	triggered := 0
	for _, job := range allJobs {
		if namespacesNeedingBackup[job.Namespace] && job.Status == models.BackupStatusActive {
			log.Printf("policy: auto-remediation: triggering backup for job %s (namespace: %s)", job.ID, job.Namespace)
			if _, err := e.backupManager.ExecuteBackup(ctx, job.ID); err != nil {
				log.Printf("policy: auto-remediation: failed to execute backup for job %s: %v", job.ID, err)
				continue
			}
			triggered++
			delete(namespacesNeedingBackup, job.Namespace)
		}
	}

	log.Printf("policy: auto-remediation complete: triggered %d backups", triggered)
	return triggered, nil
}

// findLatestBackup finds the most recent completed backup across all jobs
// for a set of backup jobs.
func findLatestBackup(ctx context.Context, mgr *backup.BackupManager, jobs []*models.BackupJob) *models.BackupRecord {
	var latest *models.BackupRecord

	for _, job := range jobs {
		records, err := mgr.ListBackups(ctx, job.ID)
		if err != nil {
			continue
		}

		for _, r := range records {
			if r.Status != models.RecordStatusCompleted {
				continue
			}
			if r.CompletedAt == nil {
				continue
			}
			if latest == nil || r.CompletedAt.After(*latest.CompletedAt) {
				latest = r
			}
		}
	}

	return latest
}

// severityForPriority maps a policy priority level to a violation severity.
// Higher priority policies produce more severe violations.
func severityForPriority(priority int) string {
	switch {
	case priority >= 5:
		return "critical"
	case priority >= 3:
		return "warning"
	default:
		return "info"
	}
}
