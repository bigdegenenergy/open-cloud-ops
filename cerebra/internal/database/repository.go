package database

import (
	"context"
	"fmt"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

// InsertRequest stores an API request record.
func (db *DB) InsertRequest(ctx context.Context, req *models.APIRequest) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO api_requests (
			id, provider, model, agent_id, team_id, org_id,
			input_tokens, output_tokens, total_tokens, cost_usd,
			latency_ms, status_code, was_routed, original_model,
			routed_model, savings_usd, timestamp
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`, req.ID, req.Provider, req.Model, req.AgentID, req.TeamID, req.OrgID,
		req.InputTokens, req.OutputTokens, req.TotalTokens, req.CostUSD,
		req.LatencyMs, req.StatusCode, req.WasRouted, req.OriginalModel,
		req.RoutedModel, req.SavingsUSD, req.Timestamp)
	if err != nil {
		return fmt.Errorf("inserting request: %w", err)
	}
	return nil
}

// GetCostSummary returns aggregated cost data grouped by a given dimension.
// Only whitelisted dimension values are accepted; all SQL identifiers are
// derived from the whitelisted map to prevent SQL injection.
func (db *DB) GetCostSummary(ctx context.Context, dimension string, from, to time.Time) ([]models.CostSummary, error) {
	// Whitelist: maps user-facing dimension names to SQL column identifiers.
	allowed := map[string]string{
		"agent":    "agent_id",
		"team":     "team_id",
		"model":    "model",
		"provider": "provider",
	}
	col, ok := allowed[dimension]
	if !ok {
		return nil, fmt.Errorf("unsupported dimension: %s", dimension)
	}

	// Use the whitelisted column name as the label too (not the raw input).
	query := fmt.Sprintf(`
		SELECT
			'%s' AS dimension,
			%s AS dimension_id,
			%s AS dimension_name,
			COALESCE(SUM(cost_usd), 0) AS total_cost_usd,
			COUNT(*) AS total_requests,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(AVG(latency_ms), 0) AS avg_latency_ms,
			COALESCE(SUM(savings_usd), 0) AS total_savings_usd
		FROM api_requests
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY %s
		ORDER BY total_cost_usd DESC
	`, col, col, col, col)

	rows, err := db.Pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying cost summary: %w", err)
	}
	defer rows.Close()

	var results []models.CostSummary
	for rows.Next() {
		var cs models.CostSummary
		if err := rows.Scan(
			&cs.Dimension, &cs.DimensionID, &cs.DimensionName,
			&cs.TotalCostUSD, &cs.TotalRequests, &cs.TotalTokens,
			&cs.AvgLatencyMs, &cs.TotalSavings,
		); err != nil {
			return nil, fmt.Errorf("scanning cost summary: %w", err)
		}
		results = append(results, cs)
	}
	return results, rows.Err()
}

// GetBudget retrieves a budget by scope and entity ID.
func (db *DB) GetBudget(ctx context.Context, scope, entityID string) (*models.Budget, error) {
	var b models.Budget
	err := db.Pool.QueryRow(ctx, `
		SELECT id, scope, entity_id, limit_usd, spent_usd, period_days, created_at, updated_at
		FROM budgets WHERE scope = $1 AND entity_id = $2
	`, scope, entityID).Scan(
		&b.ID, &b.Scope, &b.EntityID, &b.LimitUSD, &b.SpentUSD,
		&b.PeriodDays, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// UpsertBudget creates or updates a budget.
func (db *DB) UpsertBudget(ctx context.Context, b *models.Budget) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO budgets (id, scope, entity_id, limit_usd, spent_usd, period_days)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (scope, entity_id) DO UPDATE
		SET limit_usd = EXCLUDED.limit_usd,
		    updated_at = NOW()
	`, b.ID, b.Scope, b.EntityID, b.LimitUSD, b.SpentUSD, b.PeriodDays)
	return err
}

// DeleteBudget removes a budget by scope and entity ID.
func (db *DB) DeleteBudget(ctx context.Context, scope, entityID string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM budgets WHERE scope = $1 AND entity_id = $2`, scope, entityID)
	return err
}

// UpdateBudgetSpend atomically increments the spent amount for a budget.
func (db *DB) UpdateBudgetSpend(ctx context.Context, scope, entityID string, amount float64) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE budgets SET spent_usd = spent_usd + $1, updated_at = NOW()
		WHERE scope = $2 AND entity_id = $3
	`, amount, scope, entityID)
	return err
}

// GetModelPricing returns pricing for a specific provider and model.
func (db *DB) GetModelPricing(ctx context.Context, provider, model string) (*models.ModelPricing, error) {
	var mp models.ModelPricing
	err := db.Pool.QueryRow(ctx, `
		SELECT provider, model, input_per_m_token, output_per_m_token, updated_at
		FROM model_pricing WHERE provider = $1 AND model = $2
	`, provider, model).Scan(
		&mp.Provider, &mp.Model, &mp.InputPerMToken, &mp.OutputPerMToken, &mp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

// ListBudgets returns all budgets, optionally filtered by scope.
func (db *DB) ListBudgets(ctx context.Context, scope string) ([]models.Budget, error) {
	var query string
	var args []interface{}

	if scope != "" {
		query = `SELECT id, scope, entity_id, limit_usd, spent_usd, period_days, created_at, updated_at FROM budgets WHERE scope = $1 ORDER BY updated_at DESC`
		args = []interface{}{scope}
	} else {
		query = `SELECT id, scope, entity_id, limit_usd, spent_usd, period_days, created_at, updated_at FROM budgets ORDER BY updated_at DESC`
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying budgets: %w", err)
	}
	defer rows.Close()

	var results []models.Budget
	for rows.Next() {
		var b models.Budget
		if err := rows.Scan(&b.ID, &b.Scope, &b.EntityID, &b.LimitUSD, &b.SpentUSD, &b.PeriodDays, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning budget: %w", err)
		}
		results = append(results, b)
	}
	return results, rows.Err()
}

// GetRecentRequests returns the most recent N API requests.
func (db *DB) GetRecentRequests(ctx context.Context, limit int) ([]models.APIRequest, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, provider, model, agent_id, team_id, org_id,
		       input_tokens, output_tokens, total_tokens, cost_usd,
		       latency_ms, status_code, was_routed, original_model,
		       routed_model, savings_usd, timestamp
		FROM api_requests ORDER BY timestamp DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent requests: %w", err)
	}
	defer rows.Close()

	var results []models.APIRequest
	for rows.Next() {
		var r models.APIRequest
		if err := rows.Scan(
			&r.ID, &r.Provider, &r.Model, &r.AgentID, &r.TeamID, &r.OrgID,
			&r.InputTokens, &r.OutputTokens, &r.TotalTokens, &r.CostUSD,
			&r.LatencyMs, &r.StatusCode, &r.WasRouted, &r.OriginalModel,
			&r.RoutedModel, &r.SavingsUSD, &r.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scanning request: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
