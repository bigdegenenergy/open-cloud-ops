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

// checkBudgetScript atomically checks budget and increments spend if allowed.
// Returns: 1 if allowed (and spend incremented), 0 if budget exceeded, -1 if no limit set.
var checkBudgetScript = redis.NewScript(`
local limit = redis.call('GET', KEYS[1])
if not limit then
	return -1
end
limit = tonumber(limit)
local spent = tonumber(redis.call('GET', KEYS[2]) or '0')
local cost = tonumber(ARGV[1])
if spent + cost > limit then
	return 0
end
if cost > 0 then
	redis.call('INCRBYFLOAT', KEYS[2], ARGV[1])
	local ttl = redis.call('TTL', KEYS[2])
	if ttl < 0 then
		redis.call('EXPIRE', KEYS[2], ARGV[2])
	end
end
return 1
`)

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

	// Default spend TTL: 30 days (matches budget period).
	spendTTLSec := int64(30 * 24 * 3600)

	result, err := checkBudgetScript.Run(ctx, e.rdb,
		[]string{limitKey, spentKey},
		estimatedCostUSD, spendTTLSec,
	).Int64()
	if err != nil {
		return true, fmt.Errorf("checking budget: %w", err)
	}

	switch result {
	case -1:
		// No budget configured — allow the request.
		return true, nil
	case 0:
		log.Printf("budget exceeded for %s/%s", scope, entityID)
		return false, nil
	default:
		return true, nil
	}
}

// RecordSpend updates the spend for a given entity after a successful request.
func (e *Enforcer) RecordSpend(scope BudgetScope, entityID string, costUSD float64) error {
	if e.rdb == nil || costUSD <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	spentKey := fmt.Sprintf("budget:%s:%s:spent", scope, entityID)
	if err := e.rdb.IncrByFloat(ctx, spentKey, costUSD).Err(); err != nil {
		return err
	}

	// Ensure spend key has a TTL (30 days) so it auto-resets each period.
	ttl, err := e.rdb.TTL(ctx, spentKey).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		return e.rdb.Expire(ctx, spentKey, 30*24*time.Hour).Err()
	}
	return nil
}

// SetBudget configures a budget limit in Redis for fast enforcement.
func (e *Enforcer) SetBudget(scope BudgetScope, entityID string, limitUSD float64) error {
	if e.rdb == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	limitKey := fmt.Sprintf("budget:%s:%s:limit", scope, entityID)
	// No TTL on limit keys — they persist until explicitly changed or deleted.
	// (Spend keys have TTL for auto-reset; limits are managed via the API.)
	return e.rdb.Set(ctx, limitKey, limitUSD, 0).Err()
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
