// Package api implements the REST API endpoints for the Cerebra dashboard.
package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/budget"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/database"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handlers provides REST API endpoint handlers.
type Handlers struct {
	db       *database.DB
	enforcer *budget.Enforcer
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *database.DB, enforcer *budget.Enforcer) *Handlers {
	return &Handlers{db: db, enforcer: enforcer}
}

// HealthCheck returns the service health status.
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "cerebra",
		"version": "0.1.0",
	})
}

// requireDB returns true if the database is available, or sends a 503 and returns false.
func (h *Handlers) requireDB(c *gin.Context) bool {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return false
	}
	return true
}

// GetCostSummary returns aggregated cost data.
// Query params: dimension (agent|team|model|provider), from, to
func (h *Handlers) GetCostSummary(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	dimension := c.DefaultQuery("dimension", "model")
	fromStr := c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format(time.RFC3339))
	toStr := c.DefaultQuery("to", time.Now().Format(time.RFC3339))

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date format, use RFC3339"})
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date format, use RFC3339"})
		return
	}

	summaries, err := h.db.GetCostSummary(c.Request.Context(), dimension, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dimension": dimension,
		"from":      from,
		"to":        to,
		"data":      summaries,
	})
}

// GetRecentRequests returns the most recent API requests.
func (h *Handlers) GetRecentRequests(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 50
	}

	requests, err := h.db.GetRecentRequests(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(requests),
		"data":  requests,
	})
}

// ListBudgets returns all configured budgets.
func (h *Handlers) ListBudgets(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	scope := c.Query("scope")

	budgets, err := h.db.ListBudgets(c.Request.Context(), scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(budgets),
		"data":  budgets,
	})
}

// CreateBudgetRequest represents the request body for creating a budget.
type CreateBudgetRequest struct {
	Scope      string  `json:"scope" binding:"required"`
	EntityID   string  `json:"entity_id" binding:"required"`
	LimitUSD   float64 `json:"limit_usd" binding:"required"`
	PeriodDays int     `json:"period_days"`
}

// CreateBudget creates or updates a budget.
func (h *Handlers) CreateBudget(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PeriodDays <= 0 {
		req.PeriodDays = 30
	}

	b := &models.Budget{
		ID:         uuid.New().String(),
		Scope:      req.Scope,
		EntityID:   req.EntityID,
		LimitUSD:   req.LimitUSD,
		PeriodDays: req.PeriodDays,
	}

	if err := h.db.UpsertBudget(c.Request.Context(), b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also set the budget in Redis for fast enforcement.
	if err := h.enforcer.SetBudget(budget.BudgetScope(req.Scope), req.EntityID, req.LimitUSD); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "budget saved to DB but failed to sync to cache"})
		return
	}

	c.JSON(http.StatusCreated, b)
}

// GetBudget retrieves a specific budget.
func (h *Handlers) GetBudget(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	scope := c.Param("scope")
	entityID := c.Param("entity_id")

	b, err := h.db.GetBudget(c.Request.Context(), scope, entityID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "budget not found"})
		return
	}

	// Enrich with real-time spend from Redis.
	spent, err := h.enforcer.GetSpent(budget.BudgetScope(scope), entityID)
	if err != nil {
		log.Printf("failed to get spend from Redis for %s/%s: %v", scope, entityID, err)
		// Fall through with DB-stored spend value.
	} else if spent > 0 {
		b.SpentUSD = spent
	}

	c.JSON(http.StatusOK, b)
}
