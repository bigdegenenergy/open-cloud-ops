// Package api implements the HTTP API handlers for the Aegis Resilience Engine.
//
// All endpoints are versioned under /api/v1 and follow RESTful conventions.
// Handlers delegate to the appropriate manager (backup, recovery, policy, health)
// and return JSON responses with appropriate HTTP status codes.
package api

import (
	"net/http"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/health"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/policy"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/recovery"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
	"github.com/gin-gonic/gin"
)

// Handler holds references to all managers and provides HTTP handler methods.
type Handler struct {
	backupManager   *backup.BackupManager
	recoveryManager *recovery.RecoveryManager
	policyEngine    *policy.PolicyEngine
	healthChecker   *health.HealthChecker
	startTime       time.Time
}

// NewHandler creates a new Handler with all required manager dependencies.
func NewHandler(
	backupManager *backup.BackupManager,
	recoveryManager *recovery.RecoveryManager,
	policyEngine *policy.PolicyEngine,
	healthChecker *health.HealthChecker,
) *Handler {
	return &Handler{
		backupManager:   backupManager,
		recoveryManager: recoveryManager,
		policyEngine:    policyEngine,
		healthChecker:   healthChecker,
		startTime:       time.Now().UTC(),
	}
}

// APIKeyAuth is a simple Gin middleware that requires a non-empty X-API-Key header.
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing API key. Provide X-API-Key header.",
			})
			c.Abort()
			return
		}
		if len(apiKey) < 16 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid API key format.",
			})
			c.Abort()
			return
		}
		c.Set("api_key", apiKey)
		c.Next()
	}
}

// RegisterRoutes sets up all API routes on the given Gin engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Service health endpoint (unauthenticated)
	r.GET("/health", h.ServiceHealth)

	// API v1 routes (require API key)
	v1 := r.Group("/api/v1")
	v1.Use(APIKeyAuth())
	{
		// Backup job management
		backups := v1.Group("/backups")
		{
			backups.GET("/jobs", h.ListBackupJobs)
			backups.POST("/jobs", h.CreateBackupJob)
			backups.GET("/jobs/:id", h.GetBackupJob)
			backups.POST("/jobs/:id/execute", h.ExecuteBackup)
			backups.GET("/records", h.ListBackupRecords)
			backups.GET("/records/:id", h.GetBackupRecord)
			backups.DELETE("/records/:id", h.DeleteBackup)
		}

		// Recovery plan management
		recoveryGroup := v1.Group("/recovery")
		{
			recoveryGroup.GET("/plans", h.ListRecoveryPlans)
			recoveryGroup.POST("/plans", h.CreateRecoveryPlan)
			recoveryGroup.GET("/plans/:id", h.GetRecoveryPlan)
			recoveryGroup.POST("/plans/:id/execute", h.ExecuteRecovery)
			recoveryGroup.POST("/plans/:id/dry-run", h.DryRunRecovery)
			recoveryGroup.GET("/plans/:id/executions", h.ListRecoveryExecutions)
			recoveryGroup.POST("/validate/:id", h.ValidateBackup)
		}

		// DR policy management
		policies := v1.Group("/policies")
		{
			policies.GET("", h.ListPolicies)
			policies.POST("", h.CreatePolicy)
			policies.GET("/:id", h.GetPolicy)
			policies.DELETE("/:id", h.DeletePolicy)
			policies.GET("/compliance", h.GetComplianceReport)
			policies.POST("/remediate", h.AutoRemediate)
		}

		// Health monitoring
		healthGroup := v1.Group("/health")
		{
			healthGroup.GET("/summary", h.GetHealthSummary)
			healthGroup.GET("/namespace/:namespace", h.CheckNamespace)
			healthGroup.GET("/resource/:type/:namespace/:name", h.CheckResource)
		}
	}
}

// ServiceHealth returns the overall health of the Aegis service.
func (h *Handler) ServiceHealth(c *gin.Context) {
	uptime := time.Since(h.startTime)
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "aegis",
		"version": "1.0.0",
		"uptime":  uptime.String(),
	})
}

// --- Backup Handlers ---

// ListBackupJobs returns all registered backup jobs.
func (h *Handler) ListBackupJobs(c *gin.Context) {
	jobs, err := h.backupManager.ListJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs, "count": len(jobs)})
}

// CreateBackupJob creates a new backup job from the request body.
func (h *Handler) CreateBackupJob(c *gin.Context) {
	var job models.BackupJob
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	created, err := h.backupManager.CreateJob(c.Request.Context(), job)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetBackupJob returns a single backup job by ID.
func (h *Handler) GetBackupJob(c *gin.Context) {
	jobID := c.Param("id")
	job, err := h.backupManager.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

// ExecuteBackup triggers an immediate backup execution for the given job.
func (h *Handler) ExecuteBackup(c *gin.Context) {
	jobID := c.Param("id")
	record, err := h.backupManager.ExecuteBackup(c.Request.Context(), jobID)
	if err != nil {
		// Check if the error is a job not found or execution failure
		if record == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			// Execution started but had errors
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err.Error(),
				"record": record,
			})
		}
		return
	}
	c.JSON(http.StatusOK, record)
}

// ListBackupRecords returns backup records. If a job_id query parameter is
// provided, only records for that job are returned; otherwise all records.
func (h *Handler) ListBackupRecords(c *gin.Context) {
	jobID := c.Query("job_id")

	if jobID != "" {
		records, err := h.backupManager.ListBackups(c.Request.Context(), jobID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records, "count": len(records)})
		return
	}

	records, err := h.backupManager.ListAllBackups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records, "count": len(records)})
}

// GetBackupRecord returns a single backup record by ID.
func (h *Handler) GetBackupRecord(c *gin.Context) {
	recordID := c.Param("id")
	record, err := h.backupManager.GetBackupRecord(c.Request.Context(), recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, record)
}

// DeleteBackup deletes a backup record and its storage.
func (h *Handler) DeleteBackup(c *gin.Context) {
	recordID := c.Param("id")
	if err := h.backupManager.DeleteBackup(c.Request.Context(), recordID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "backup deleted", "id": recordID})
}

// --- Recovery Handlers ---

// ListRecoveryPlans returns all registered recovery plans.
func (h *Handler) ListRecoveryPlans(c *gin.Context) {
	plans, err := h.recoveryManager.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans, "count": len(plans)})
}

// CreateRecoveryPlan creates a new recovery plan from the request body.
func (h *Handler) CreateRecoveryPlan(c *gin.Context) {
	var plan models.RecoveryPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	created, err := h.recoveryManager.CreatePlan(c.Request.Context(), plan)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetRecoveryPlan returns a single recovery plan by ID.
func (h *Handler) GetRecoveryPlan(c *gin.Context) {
	planID := c.Param("id")
	plan, err := h.recoveryManager.GetPlan(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// ExecuteRecovery triggers a recovery execution for the given plan.
func (h *Handler) ExecuteRecovery(c *gin.Context) {
	planID := c.Param("id")
	execution, err := h.recoveryManager.ExecuteRecovery(c.Request.Context(), planID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if execution == nil {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{
			"error":     err.Error(),
			"execution": execution,
		})
		return
	}
	c.JSON(http.StatusOK, execution)
}

// DryRunRecovery performs a simulated recovery for the given plan.
func (h *Handler) DryRunRecovery(c *gin.Context) {
	planID := c.Param("id")
	execution, err := h.recoveryManager.DryRun(c.Request.Context(), planID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if execution == nil {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{
			"error":     err.Error(),
			"execution": execution,
		})
		return
	}
	c.JSON(http.StatusOK, execution)
}

// ListRecoveryExecutions returns all executions for a given plan.
func (h *Handler) ListRecoveryExecutions(c *gin.Context) {
	planID := c.Param("id")
	executions, err := h.recoveryManager.ListExecutions(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"executions": executions, "count": len(executions)})
}

// ValidateBackup validates the integrity of a backup.
func (h *Handler) ValidateBackup(c *gin.Context) {
	backupID := c.Param("id")
	if err := h.recoveryManager.ValidateBackup(c.Request.Context(), backupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"error":  err.Error(),
			"backup": backupID,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"backup": backupID,
	})
}

// --- Policy Handlers ---

// ListPolicies returns all registered DR policies.
func (h *Handler) ListPolicies(c *gin.Context) {
	policies, err := h.policyEngine.ListPolicies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies, "count": len(policies)})
}

// CreatePolicy creates a new DR policy from the request body.
func (h *Handler) CreatePolicy(c *gin.Context) {
	var pol models.DRPolicy
	if err := c.ShouldBindJSON(&pol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	created, err := h.policyEngine.CreatePolicy(c.Request.Context(), pol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetPolicy returns a single DR policy by ID.
func (h *Handler) GetPolicy(c *gin.Context) {
	policyID := c.Param("id")
	pol, err := h.policyEngine.GetPolicy(c.Request.Context(), policyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pol)
}

// DeletePolicy removes a DR policy by ID.
func (h *Handler) DeletePolicy(c *gin.Context) {
	policyID := c.Param("id")
	if err := h.policyEngine.DeletePolicy(c.Request.Context(), policyID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted", "id": policyID})
}

// GetComplianceReport evaluates all policies and returns the compliance report.
func (h *Handler) GetComplianceReport(c *gin.Context) {
	report, err := h.policyEngine.GetComplianceReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// AutoRemediate triggers automatic remediation of compliance violations.
func (h *Handler) AutoRemediate(c *gin.Context) {
	triggered, err := h.policyEngine.AutoRemediate(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":           "auto-remediation complete",
		"backups_triggered": triggered,
	})
}

// --- Health Handlers ---

// GetHealthSummary returns an aggregate health summary across all monitored resources.
func (h *Handler) GetHealthSummary(c *gin.Context) {
	summary, err := h.healthChecker.GetHealthSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// CheckNamespace performs health checks on all resources in the given namespace.
func (h *Handler) CheckNamespace(c *gin.Context) {
	namespace := c.Param("namespace")
	checks, err := h.healthChecker.CheckNamespace(c.Request.Context(), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checks": checks, "count": len(checks), "namespace": namespace})
}

// CheckResource performs a health check on a specific resource.
func (h *Handler) CheckResource(c *gin.Context) {
	resourceType := c.Param("type")
	namespace := c.Param("namespace")
	name := c.Param("name")

	check, err := h.healthChecker.CheckResource(c.Request.Context(), resourceType, name, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, check)
}
