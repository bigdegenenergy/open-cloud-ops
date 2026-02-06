// Package budget implements hard budget controls for LLM API usage.
//
// Budget enforcement ensures that agents and teams do not exceed their
// allocated spending limits. When a budget limit is reached, subsequent
// requests are automatically blocked until the next billing period.
package budget

import "time"

// BudgetScope defines the entity to which a budget applies.
type BudgetScope string

const (
	ScopeAgent BudgetScope = "agent"
	ScopeTeam  BudgetScope = "team"
	ScopeUser  BudgetScope = "user"
	ScopeOrg   BudgetScope = "org"
)

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

// Enforcer manages budget checks and enforcement.
type Enforcer struct {
	// TODO: Add fields for database, cache, notification channels, etc.
}

// NewEnforcer creates a new budget Enforcer.
func NewEnforcer() *Enforcer {
	return &Enforcer{}
}

// CheckBudget verifies whether the given entity has remaining budget.
// Returns true if the request is allowed, false if the budget is exhausted.
func (e *Enforcer) CheckBudget(scope BudgetScope, entityID string, estimatedCostUSD float64) (bool, error) {
	// TODO: Implement budget check logic:
	// 1. Look up the budget for the given scope and entity.
	// 2. Compare current spend + estimated cost against the limit.
	// 3. If within budget, return true.
	// 4. If over budget, return false and trigger an alert.
	return true, nil
}

// RecordSpend updates the spend for a given entity after a successful request.
func (e *Enforcer) RecordSpend(scope BudgetScope, entityID string, costUSD float64) error {
	// TODO: Implement spend recording:
	// 1. Update the spend in the database.
	// 2. Update the cached spend in Redis for fast lookups.
	// 3. Check if spend is approaching the limit and send warnings.
	return nil
}
