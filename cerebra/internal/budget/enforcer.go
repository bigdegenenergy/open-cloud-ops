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

	"github.com/redis/go-redis/v9"
)

// BudgetScope defines the entity to which a budget applies.
type BudgetScope string

const (
	ScopeAgent BudgetScope = "agent"
	ScopeTeam  BudgetScope = "team"
	ScopeUser  BudgetScope = "user"
	ScopeOrg   BudgetScope = "org"
)

// Enforcer manages budget checks and enforcement using Redis for fast lookups
// and PostgreSQL for persistence.
type Enforcer struct {
	rdb *redis.Client
	dsn string // for direct DB access when needed
}

// NewEnforcer creates a new budget Enforcer.
func NewEnforcer(rdb *redis.Client) *Enforcer {
	return &Enforcer{rdb: rdb}
}

// CheckBudget verifies whether the given entity has remaining budget.
// Returns true if the request is allowed, false if the budget is exhausted.
func (e *Enforcer) CheckBudget(scope BudgetScope, entityID string, estimatedCostUSD float64) (bool, error) {
	if e.rdb == nil {
		// No Redis available — allow all requests (graceful degradation).
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	limitKey := fmt.Sprintf("budget:%s:%s:limit", scope, entityID)
	spentKey := fmt.Sprintf("budget:%s:%s:spent", scope, entityID)

	limit, err := e.rdb.Get(ctx, limitKey).Float64()
	if err == redis.Nil {
		// No budget configured — allow the request.
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("checking budget limit: %w", err)
	}

	spent, err := e.rdb.Get(ctx, spentKey).Float64()
	if err == redis.Nil {
		spent = 0
	} else if err != nil {
		return true, fmt.Errorf("checking budget spent: %w", err)
	}

	if spent+estimatedCostUSD > limit {
		log.Printf("budget exceeded for %s/%s: spent=%.4f + est=%.4f > limit=%.4f",
			scope, entityID, spent, estimatedCostUSD, limit)
		return false, nil
	}

	return true, nil
}

// RecordSpend updates the spend for a given entity after a successful request.
func (e *Enforcer) RecordSpend(scope BudgetScope, entityID string, costUSD float64) error {
	if e.rdb == nil || costUSD <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	spentKey := fmt.Sprintf("budget:%s:%s:spent", scope, entityID)
	return e.rdb.IncrByFloat(ctx, spentKey, costUSD).Err()
}

// SetBudget configures a budget limit in Redis for fast enforcement.
func (e *Enforcer) SetBudget(scope BudgetScope, entityID string, limitUSD float64) error {
	if e.rdb == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	limitKey := fmt.Sprintf("budget:%s:%s:limit", scope, entityID)
	return e.rdb.Set(ctx, limitKey, limitUSD, 30*24*time.Hour).Err()
}

// GetSpent returns the current spend for an entity from Redis.
func (e *Enforcer) GetSpent(scope BudgetScope, entityID string) (float64, error) {
	if e.rdb == nil {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	spentKey := fmt.Sprintf("budget:%s:%s:spent", scope, entityID)
	spent, err := e.rdb.Get(ctx, spentKey).Float64()
	if err == redis.Nil {
		return 0, nil
	}
	return spent, err
}

// ResetSpend resets the spend counter for a budget period.
func (e *Enforcer) ResetSpend(scope BudgetScope, entityID string) error {
	if e.rdb == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	spentKey := fmt.Sprintf("budget:%s:%s:spent", scope, entityID)
	return e.rdb.Del(ctx, spentKey).Err()
}
