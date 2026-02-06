// Package models defines the core data structures used across Aegis.
//
// Aegis is the Resilience Engine for Open Cloud Ops, providing
// Kubernetes-native backup, disaster recovery, and resilience management.
// These models represent backup jobs, recovery plans, DR policies, and
// health check results that flow through the system.
package models

import "time"

// BackupStatus represents the current state of a backup job.
type BackupStatus string

const (
	BackupStatusActive   BackupStatus = "active"
	BackupStatusPaused   BackupStatus = "paused"
	BackupStatusDisabled BackupStatus = "disabled"
)

// RecordStatus represents the state of a single backup execution.
type RecordStatus string

const (
	RecordStatusPending   RecordStatus = "pending"
	RecordStatusRunning   RecordStatus = "running"
	RecordStatusCompleted RecordStatus = "completed"
	RecordStatusFailed    RecordStatus = "failed"
)

// RecoveryStrategy defines how resources are restored during recovery.
type RecoveryStrategy string

const (
	RecoveryStrategyInPlace       RecoveryStrategy = "in_place"
	RecoveryStrategyNewNamespace  RecoveryStrategy = "new_namespace"
	RecoveryStrategyCrossCluster  RecoveryStrategy = "cross_cluster"
)

// ExecutionStatus represents the state of a recovery execution.
type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"
	ExecutionStatusRunning    ExecutionStatus = "running"
	ExecutionStatusCompleted  ExecutionStatus = "completed"
	ExecutionStatusFailed     ExecutionStatus = "failed"
	ExecutionStatusPartial    ExecutionStatus = "partial"
)

// HealthStatus represents the health state of a Kubernetes resource.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ConflictStrategy defines how conflicts are handled during recovery.
type ConflictStrategy string

const (
	ConflictStrategySkip      ConflictStrategy = "skip"
	ConflictStrategyOverwrite ConflictStrategy = "overwrite"
)

// BackupJob represents a configured backup job that can be scheduled.
// Each job targets a specific namespace and set of resource types,
// runs on a cron schedule, and stores backups at a configurable location.
type BackupJob struct {
	ID              string       `json:"id" db:"id"`
	Name            string       `json:"name" db:"name"`
	Namespace       string       `json:"namespace" db:"namespace"`
	ResourceTypes   []string     `json:"resource_types" db:"resource_types"`
	Schedule        string       `json:"schedule" db:"schedule"` // Cron expression
	RetentionDays   int          `json:"retention_days" db:"retention_days"`
	StorageLocation string       `json:"storage_location" db:"storage_location"`
	Status          BackupStatus `json:"status" db:"status"`
	LastRun         *time.Time   `json:"last_run,omitempty" db:"last_run"`
	NextRun         *time.Time   `json:"next_run,omitempty" db:"next_run"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
}

// BackupRecord represents a single execution of a backup job.
// It tracks the status, size, duration, and any errors that occurred.
type BackupRecord struct {
	ID            string       `json:"id" db:"id"`
	JobID         string       `json:"job_id" db:"job_id"`
	Status        RecordStatus `json:"status" db:"status"`
	SizeBytes     int64        `json:"size_bytes" db:"size_bytes"`
	DurationMs    int64        `json:"duration_ms" db:"duration_ms"`
	ResourceCount int          `json:"resource_count" db:"resource_count"`
	StoragePath   string       `json:"storage_path" db:"storage_path"`
	ErrorMessage  string       `json:"error_message,omitempty" db:"error_message"`
	StartedAt     time.Time    `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
}

// RecoveryPlan defines a plan for restoring resources from a backup.
// Plans specify the source backup, target namespace, and the strategy
// to use for applying recovered resources.
type RecoveryPlan struct {
	ID              string           `json:"id" db:"id"`
	Name            string           `json:"name" db:"name"`
	Description     string           `json:"description" db:"description"`
	BackupID        string           `json:"backup_id" db:"backup_id"`
	TargetNamespace string           `json:"target_namespace" db:"target_namespace"`
	ResourceFilters []string         `json:"resource_filters" db:"resource_filters"`
	Strategy        RecoveryStrategy `json:"strategy" db:"strategy"`
	ConflictPolicy  ConflictStrategy `json:"conflict_policy" db:"conflict_policy"`
	Status          ExecutionStatus  `json:"status" db:"status"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
}

// RecoveryExecution represents a single execution of a recovery plan.
// It tracks the resources restored, any errors encountered, and timing.
type RecoveryExecution struct {
	ID                string          `json:"id" db:"id"`
	PlanID            string          `json:"plan_id" db:"plan_id"`
	Status            ExecutionStatus `json:"status" db:"status"`
	ResourcesRestored int             `json:"resources_restored" db:"resources_restored"`
	ResourcesSkipped  int             `json:"resources_skipped" db:"resources_skipped"`
	Errors            []string        `json:"errors" db:"errors"`
	DryRun            bool            `json:"dry_run" db:"dry_run"`
	StartedAt         time.Time       `json:"started_at" db:"started_at"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}

// DRPolicy defines a disaster recovery policy with RPO and RTO targets.
// Policies are evaluated against actual backup state to determine compliance.
type DRPolicy struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	RPOMinutes     int       `json:"rpo_minutes" db:"rpo_minutes"`   // Recovery Point Objective
	RTOMinutes     int       `json:"rto_minutes" db:"rto_minutes"`   // Recovery Time Objective
	BackupSchedule string    `json:"backup_schedule" db:"backup_schedule"` // Required cron schedule
	RetentionDays  int       `json:"retention_days" db:"retention_days"`
	Priority       int       `json:"priority" db:"priority"` // Higher = more critical
	Namespaces     []string  `json:"namespaces" db:"namespaces"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// HealthCheck represents a point-in-time health assessment of a
// Kubernetes resource. Health checks track status transitions over time.
type HealthCheck struct {
	ID           string            `json:"id" db:"id"`
	ResourceType string            `json:"resource_type" db:"resource_type"`
	ResourceName string            `json:"resource_name" db:"resource_name"`
	Namespace    string            `json:"namespace" db:"namespace"`
	Status       HealthStatus      `json:"status" db:"status"`
	LastCheck    time.Time         `json:"last_check" db:"last_check"`
	Details      map[string]string `json:"details" db:"details"`
}

// ComplianceViolation represents a single violation of a DR policy.
type ComplianceViolation struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	Namespace   string `json:"namespace"`
	ViolationType string `json:"violation_type"` // "rpo", "rto", "retention", "missing_backup"
	Description string `json:"description"`
	Severity    string `json:"severity"` // "critical", "warning", "info"
}

// ComplianceReport is the result of evaluating all DR policies
// against the current backup state.
type ComplianceReport struct {
	GeneratedAt     time.Time             `json:"generated_at"`
	TotalPolicies   int                   `json:"total_policies"`
	CompliantCount  int                   `json:"compliant_count"`
	ViolationCount  int                   `json:"violation_count"`
	Violations      []ComplianceViolation `json:"violations"`
	OverallStatus   string                `json:"overall_status"` // "compliant", "non_compliant"
}

// HealthSummary provides an aggregated view of cluster health.
type HealthSummary struct {
	Timestamp      time.Time             `json:"timestamp"`
	TotalResources int                   `json:"total_resources"`
	Healthy        int                   `json:"healthy"`
	Degraded       int                   `json:"degraded"`
	Unhealthy      int                   `json:"unhealthy"`
	Unknown        int                   `json:"unknown"`
	ByNamespace    map[string]*NamespaceHealth `json:"by_namespace"`
}

// NamespaceHealth summarizes health for a single Kubernetes namespace.
type NamespaceHealth struct {
	Namespace string `json:"namespace"`
	Healthy   int    `json:"healthy"`
	Degraded  int    `json:"degraded"`
	Unhealthy int    `json:"unhealthy"`
	Unknown   int    `json:"unknown"`
}

// KubernetesResource represents a generic Kubernetes resource
// captured during backup. It stores the full resource manifest as JSON.
type KubernetesResource struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Labels     map[string]string `json:"labels,omitempty"`
	Manifest   []byte            `json:"manifest"` // Full JSON manifest
}

// BackupManifest is the metadata file stored with each backup archive.
// It describes the contents of the backup for validation and recovery.
type BackupManifest struct {
	BackupID      string               `json:"backup_id"`
	JobID         string               `json:"job_id"`
	Namespace     string               `json:"namespace"`
	ResourceTypes []string             `json:"resource_types"`
	Resources     []KubernetesResource `json:"resources"`
	ResourceCount int                  `json:"resource_count"`
	CreatedAt     time.Time            `json:"created_at"`
	Checksum      string               `json:"checksum"` // SHA-256 of the archive
}
