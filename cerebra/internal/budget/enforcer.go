// Package budget implements hard budget controls for LLM API usage.
//
// Budget enforcement ensures that agents and teams do not exceed their
// allocated spending limits. When a budget limit is reached, subsequent
// requests are automatically blocked until the next billing period.
package budget

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/cache"
)

// BudgetScope defines the entity to which a budget applies.
type BudgetScope string

const (
	ScopeAgent BudgetScope = "agent"
	ScopeTeam  BudgetScope = "team"
	ScopeUser  BudgetScope = "user"
	ScopeOrg   BudgetScope = "org"
)

// AlertThreshold defines a budget usage percentage that triggers an alert.
type AlertThreshold struct {
	Percentage float64 // e.g., 0.80 for 80%
	Label      string  // e.g., "80%"
}

// DefaultAlertThresholds are the standard thresholds for budget warnings.
var DefaultAlertThresholds = []AlertThreshold{
	{Percentage: 0.80, Label: "80%"},
	{Percentage: 0.90, Label: "90%"},
	{Percentage: 0.95, Label: "95%"},
	{Percentage: 1.00, Label: "100%"},
}

// Budget represents a spending limit for a specific scope.
type Budget struct {
	ID        string
	Scope     BudgetScope
	EntityID  string  // The ID of the agent, team, user, or org
	LimitUSD  float64 // Monthly budget limit in USD
	SpentUSD  float64 // Current spend in the billing period
	Period    time.Duration
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BudgetStatus represents the current state of a budget for API responses.
type BudgetStatus struct {
	Scope       BudgetScope `json:"scope"`
	EntityID    string      `json:"entity_id"`
	LimitUSD    float64     `json:"limit_usd"`
	SpentUSD    float64     `json:"spent_usd"`
	RemainingUSD float64    `json:"remaining_usd"`
	UsagePercent float64    `json:"usage_percent"`
	IsExhausted bool        `json:"is_exhausted"`
}

// Enforcer manages budget checks and enforcement.
type Enforcer struct {
	pool            *pgxpool.Pool
	cache           *cache.Cache
	alertThresholds []AlertThreshold
	// alertsSent tracks which alert thresholds have already been fired
	// to avoid duplicate notifications. Key: "scope:entityID:threshold"
	alertsSent map[string]bool
}

// NewEnforcer creates a new budget Enforcer.
func NewEnforcer(pool *pgxpool.Pool, cache *cache.Cache) *Enforcer {
	return &Enforcer{
		pool:            pool,
		cache:           cache,
		alertThresholds: DefaultAlertThresholds,
		alertsSent:      make(map[string]bool),
	}
}

// CheckBudget verifies whether the given entity has remaining budget.
// It checks Redis first for fast lookups, then falls back to the database.
// Returns true if the request is allowed, false if the budget is exhausted.
func (e *Enforcer) CheckBudget(scope BudgetScope, entityID string, estimatedCostUSD float64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Try to get the budget limit from the database
	budgetLimit, err := e.getBudgetLimit(ctx, scope, entityID)
	if err != nil {
		return false, fmt.Errorf("budget: failed to get limit for %s/%s: %w", scope, entityID, err)
	}

	// If no budget is configured, allow the request (no enforcement)
	if budgetLimit <= 0 {
		return true, nil
	}

	// Step 2: Get current spend from Redis (fast path)
	currentSpend, err := e.getCurrentSpend(ctx, scope, entityID)
	if err != nil {
		// Redis error - fall back to database
		log.Printf("budget: Redis error for %s/%s, falling back to DB: %v", scope, entityID, err)
		currentSpend, err = e.getSpendFromDB(ctx, scope, entityID)
		if err != nil {
			return false, fmt.Errorf("budget: failed to get spend for %s/%s: %w", scope, entityID, err)
		}
	}

	// Step 3: Compare current spend + estimated cost against the limit
	projectedSpend := currentSpend + estimatedCostUSD
	if projectedSpend > budgetLimit {
		log.Printf("budget: BLOCKED request for %s/%s: projected spend $%.4f exceeds limit $%.2f",
			scope, entityID, projectedSpend, budgetLimit)
		return false, nil
	}

	return true, nil
}

// RecordSpend updates the spend for a given entity after a successful request.
// It updates both the database and Redis cache, then checks alert thresholds.
func (e *Enforcer) RecordSpend(scope BudgetScope, entityID string, costUSD float64) error {
	if costUSD <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Update the spend in the database
	err := e.updateSpendInDB(ctx, scope, entityID, costUSD)
	if err != nil {
		return fmt.Errorf("budget: failed to update DB spend for %s/%s: %w", scope, entityID, err)
	}

	// Step 2: Update the cached spend in Redis for fast lookups
	if e.cache != nil {
		newSpend, err := e.cache.IncrBudgetSpend(ctx, string(scope), entityID, costUSD)
		if err != nil {
			log.Printf("budget: failed to update Redis spend for %s/%s: %v", scope, entityID, err)
		} else {
			// Step 3: Check if spend is approaching the limit and send warnings
			e.checkAlertThresholds(ctx, scope, entityID, newSpend)
		}
	}

	return nil
}

// GetBudgetStatus returns the current budget details for a given scope and entity.
func (e *Enforcer) GetBudgetStatus(scope BudgetScope, entityID string) (*BudgetStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	budgetLimit, err := e.getBudgetLimit(ctx, scope, entityID)
	if err != nil {
		return nil, fmt.Errorf("budget: failed to get limit for %s/%s: %w", scope, entityID, err)
	}

	currentSpend, err := e.getCurrentSpend(ctx, scope, entityID)
	if err != nil {
		// Fall back to DB
		currentSpend, err = e.getSpendFromDB(ctx, scope, entityID)
		if err != nil {
			return nil, fmt.Errorf("budget: failed to get spend for %s/%s: %w", scope, entityID, err)
		}
	}

	remaining := budgetLimit - currentSpend
	if remaining < 0 {
		remaining = 0
	}

	var usagePercent float64
	if budgetLimit > 0 {
		usagePercent = (currentSpend / budgetLimit) * 100.0
	}

	return &BudgetStatus{
		Scope:        scope,
		EntityID:     entityID,
		LimitUSD:     budgetLimit,
		SpentUSD:     currentSpend,
		RemainingUSD: remaining,
		UsagePercent: usagePercent,
		IsExhausted:  currentSpend >= budgetLimit && budgetLimit > 0,
	}, nil
}

// ResetBudgets resets all budget spend counters for a new billing period.
// This should be called monthly (e.g., via a cron job or scheduler).
func (e *Enforcer) ResetBudgets() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Reset all budget spends in the database
	_, err := e.pool.Exec(ctx, `
		UPDATE budgets SET spent_usd = 0, updated_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("budget: failed to reset budgets in DB: %w", err)
	}

	// Clear all budget spend keys in Redis
	// We query all budgets and reset their Redis keys
	rows, err := e.pool.Query(ctx, `SELECT scope, entity_id FROM budgets`)
	if err != nil {
		return fmt.Errorf("budget: failed to query budgets for Redis reset: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var scope, entityID string
		if err := rows.Scan(&scope, &entityID); err != nil {
			log.Printf("budget: error scanning budget row during reset: %v", err)
			continue
		}

		if e.cache != nil {
			if err := e.cache.SetBudgetSpend(ctx, scope, entityID, 0); err != nil {
				log.Printf("budget: failed to reset Redis spend for %s/%s: %v", scope, entityID, err)
			}
		}
	}

	// Clear sent alerts so they can fire again in the new period
	e.alertsSent = make(map[string]bool)

	log.Println("budget: all budgets reset for new billing period")
	return nil
}

// getBudgetLimit retrieves the budget limit from the database for a scope/entity pair.
func (e *Enforcer) getBudgetLimit(ctx context.Context, scope BudgetScope, entityID string) (float64, error) {
	var limitUSD float64
	err := e.pool.QueryRow(ctx,
		`SELECT limit_usd FROM budgets WHERE scope = $1 AND entity_id = $2`,
		string(scope), entityID,
	).Scan(&limitUSD)

	if err == pgx.ErrNoRows {
		// No budget configured - return 0 to indicate no enforcement
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return limitUSD, nil
}

// getCurrentSpend gets the current spend from Redis (fast path).
func (e *Enforcer) getCurrentSpend(ctx context.Context, scope BudgetScope, entityID string) (float64, error) {
	if e.cache == nil {
		return e.getSpendFromDB(ctx, scope, entityID)
	}
	return e.cache.GetBudgetSpend(ctx, string(scope), entityID)
}

// getSpendFromDB retrieves the current spend from the database (slow path / fallback).
func (e *Enforcer) getSpendFromDB(ctx context.Context, scope BudgetScope, entityID string) (float64, error) {
	var spentUSD float64
	err := e.pool.QueryRow(ctx,
		`SELECT spent_usd FROM budgets WHERE scope = $1 AND entity_id = $2`,
		string(scope), entityID,
	).Scan(&spentUSD)

	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return spentUSD, nil
}

// updateSpendInDB atomically increments the spend in the database.
func (e *Enforcer) updateSpendInDB(ctx context.Context, scope BudgetScope, entityID string, costUSD float64) error {
	result, err := e.pool.Exec(ctx, `
		UPDATE budgets
		SET spent_usd = spent_usd + $1, updated_at = NOW()
		WHERE scope = $2 AND entity_id = $3
	`, costUSD, string(scope), entityID)

	if err != nil {
		return err
	}

	// If no rows were updated, the budget entry does not exist yet - create it
	if result.RowsAffected() == 0 {
		_, err = e.pool.Exec(ctx, `
			INSERT INTO budgets (id, scope, entity_id, limit_usd, spent_usd, created_at, updated_at)
			VALUES ($1, $2, $3, 0, $4, NOW(), NOW())
			ON CONFLICT (scope, entity_id) DO UPDATE SET spent_usd = budgets.spent_usd + $4, updated_at = NOW()
		`, fmt.Sprintf("%s-%s", scope, entityID), string(scope), entityID, costUSD)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkAlertThresholds evaluates whether the current spend has crossed any
// alert thresholds and logs warnings accordingly.
func (e *Enforcer) checkAlertThresholds(ctx context.Context, scope BudgetScope, entityID string, currentSpend float64) {
	budgetLimit, err := e.getBudgetLimit(ctx, scope, entityID)
	if err != nil || budgetLimit <= 0 {
		return
	}

	usagePercent := currentSpend / budgetLimit

	for _, threshold := range e.alertThresholds {
		alertKey := fmt.Sprintf("%s:%s:%s", scope, entityID, threshold.Label)

		if usagePercent >= threshold.Percentage && !e.alertsSent[alertKey] {
			e.alertsSent[alertKey] = true

			severity := "WARNING"
			if threshold.Percentage >= 1.0 {
				severity = "CRITICAL"
			}

			log.Printf("budget: [%s] %s/%s has reached %s of budget ($%.2f / $%.2f)",
				severity, scope, entityID, threshold.Label, currentSpend, budgetLimit)

			// In production, this would trigger notifications (email, Slack, webhook, etc.)
			// For now, we log the alert and record it.
		}
	}
}
