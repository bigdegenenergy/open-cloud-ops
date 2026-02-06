// Package analytics implements cost analytics and AI-powered insights.
//
// The analytics engine processes usage data to detect cost spikes,
// identify optimization opportunities, and generate actionable
// recommendations for reducing LLM API spend.
package analytics

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

// InsightType categorizes the kind of insight generated.
type InsightType string

const (
	InsightCostSpike       InsightType = "cost_spike"
	InsightModelSwitch     InsightType = "model_switch"
	InsightBudgetWarning   InsightType = "budget_warning"
	InsightAnomalyDetected InsightType = "anomaly_detected"
	InsightSavingsFound    InsightType = "savings_found"
)

// Severity indicates the urgency of an insight.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Insight represents an actionable recommendation or alert.
type Insight struct {
	ID              string      `json:"id"`
	Type            InsightType `json:"type"`
	Severity        Severity    `json:"severity"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	EstimatedSaving float64     `json:"estimated_saving"` // Potential monthly savings in USD
	AffectedEntity  string      `json:"affected_entity"`  // Agent, team, or model affected
	CreatedAt       time.Time   `json:"created_at"`
	Dismissed       bool        `json:"dismissed"`
}

// SpikeThreshold is the multiplier above the rolling average that triggers a spike alert.
const SpikeThreshold = 2.0

// premiumModelAlternatives maps premium models to cheaper alternatives for recommendation.
var premiumModelAlternatives = map[string]string{
	"gpt-4-turbo":            "gpt-4o",
	"gpt-4":                  "gpt-4o",
	"o1":                     "gpt-4o",
	"claude-3-opus-20240229": "claude-3-5-sonnet-20241022",
	"gemini-ultra":           "gemini-1.5-pro",
}

// InsightsEngine generates and manages cost insights.
type InsightsEngine struct {
	pool *pgxpool.Pool
}

// NewInsightsEngine creates a new InsightsEngine.
func NewInsightsEngine(pool *pgxpool.Pool) *InsightsEngine {
	return &InsightsEngine{
		pool: pool,
	}
}

// DetectSpikes analyzes recent usage data to identify cost spikes.
// It compares the cost in the recent lookbackHours to the rolling average
// of the prior 7 days, and flags any period where cost exceeds 2x the average.
func (e *InsightsEngine) DetectSpikes(lookbackHours int) ([]Insight, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if lookbackHours <= 0 {
		lookbackHours = 1
	}

	var insights []Insight

	// Query recent hourly costs grouped by agent/team, then compare to rolling average
	query := `
		WITH recent_costs AS (
			SELECT
				agent_id,
				team_id,
				SUM(cost_usd) AS recent_cost,
				COUNT(*) AS recent_requests
			FROM api_requests
			WHERE timestamp > NOW() - INTERVAL '1 hour' * $1
			GROUP BY agent_id, team_id
		),
		baseline_costs AS (
			SELECT
				agent_id,
				team_id,
				SUM(cost_usd) / GREATEST(EXTRACT(EPOCH FROM (NOW() - MIN(timestamp))) / 3600, 1) * $1 AS avg_cost_per_period,
				COUNT(*) AS baseline_requests
			FROM api_requests
			WHERE timestamp > NOW() - INTERVAL '7 days'
			  AND timestamp <= NOW() - INTERVAL '1 hour' * $1
			GROUP BY agent_id, team_id
		)
		SELECT
			r.agent_id,
			r.team_id,
			r.recent_cost,
			r.recent_requests,
			COALESCE(b.avg_cost_per_period, 0) AS avg_cost,
			COALESCE(b.baseline_requests, 0) AS baseline_requests
		FROM recent_costs r
		LEFT JOIN baseline_costs b ON r.agent_id = b.agent_id AND r.team_id = b.team_id
		WHERE COALESCE(b.avg_cost_per_period, 0) > 0
		  AND r.recent_cost > COALESCE(b.avg_cost_per_period, 0) * $2
	`

	rows, err := e.pool.Query(ctx, query, lookbackHours, SpikeThreshold)
	if err != nil {
		return nil, fmt.Errorf("analytics: failed to detect spikes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agentID, teamID string
		var recentCost, avgCost float64
		var recentRequests, baselineRequests int64

		if err := rows.Scan(&agentID, &teamID, &recentCost, &recentRequests, &avgCost, &baselineRequests); err != nil {
			log.Printf("analytics: error scanning spike row: %v", err)
			continue
		}

		multiplier := recentCost / avgCost
		affectedEntity := agentID
		if affectedEntity == "" {
			affectedEntity = teamID
		}

		severity := SeverityWarning
		if multiplier >= 5.0 {
			severity = SeverityCritical
		}

		insight := Insight{
			ID:       uuid.New().String(),
			Type:     InsightCostSpike,
			Severity: severity,
			Title:    fmt.Sprintf("Cost spike detected: %.1fx above average", multiplier),
			Description: fmt.Sprintf(
				"Entity %s spent $%.4f in the last %d hour(s), which is %.1fx the rolling average of $%.4f. "+
					"Recent requests: %d, baseline requests: %d.",
				affectedEntity, recentCost, lookbackHours, multiplier, avgCost,
				recentRequests, baselineRequests,
			),
			EstimatedSaving: recentCost - avgCost,
			AffectedEntity:  affectedEntity,
			CreatedAt:       time.Now(),
		}

		insights = append(insights, insight)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("analytics: error iterating spike results: %w", err)
	}

	return insights, nil
}

// RecommendModelSwitches identifies opportunities to use cheaper models.
// It looks for agents/teams using premium-tier models that could potentially
// use standard or economy tier models instead, and estimates the savings.
func (e *InsightsEngine) RecommendModelSwitches() ([]Insight, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var insights []Insight

	// Find agents/models using premium models with high request volumes
	query := `
		SELECT
			agent_id,
			team_id,
			model,
			provider,
			COUNT(*) AS request_count,
			SUM(cost_usd) AS total_cost,
			AVG(input_tokens) AS avg_input_tokens,
			AVG(output_tokens) AS avg_output_tokens
		FROM api_requests
		WHERE timestamp > NOW() - INTERVAL '30 days'
		  AND model != ''
		GROUP BY agent_id, team_id, model, provider
		HAVING COUNT(*) >= 10
		ORDER BY SUM(cost_usd) DESC
		LIMIT 50
	`

	rows, err := e.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("analytics: failed to query model usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agentID, teamID, model, provider string
		var requestCount int64
		var totalCost, avgInputTokens, avgOutputTokens float64

		if err := rows.Scan(&agentID, &teamID, &model, &provider, &requestCount, &totalCost, &avgInputTokens, &avgOutputTokens); err != nil {
			log.Printf("analytics: error scanning model switch row: %v", err)
			continue
		}

		// Check if there is a cheaper alternative for this model
		alternative, hasAlternative := premiumModelAlternatives[model]
		if !hasAlternative {
			continue
		}

		// Estimate savings: assume the alternative costs ~60% less on average
		estimatedSavings := totalCost * 0.60

		affectedEntity := agentID
		if affectedEntity == "" {
			affectedEntity = teamID
		}

		insight := Insight{
			ID:       uuid.New().String(),
			Type:     InsightModelSwitch,
			Severity: SeverityInfo,
			Title:    fmt.Sprintf("Consider switching from %s to %s", model, alternative),
			Description: fmt.Sprintf(
				"Entity %s made %d requests to %s (%s) in the last 30 days, costing $%.2f total. "+
					"Average request size: %.0f input / %.0f output tokens. "+
					"Switching to %s could save approximately $%.2f/month.",
				affectedEntity, requestCount, model, provider, totalCost,
				avgInputTokens, avgOutputTokens,
				alternative, estimatedSavings,
			),
			EstimatedSaving: estimatedSavings,
			AffectedEntity:  affectedEntity,
			CreatedAt:       time.Now(),
		}

		insights = append(insights, insight)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("analytics: error iterating model switch results: %w", err)
	}

	return insights, nil
}

// GenerateReport creates a summary report for a given time period.
// It aggregates cost data into CostSummary objects by multiple dimensions
// (agent, team, model, provider).
func (e *InsightsEngine) GenerateReport(from, to time.Time) ([]models.CostSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var summaries []models.CostSummary

	// Generate summaries for each dimension
	dimensions := []struct {
		name    string
		idCol   string
		nameCol string
	}{
		{"agent", "agent_id", "agent_id"},
		{"team", "team_id", "team_id"},
		{"model", "model", "model"},
		{"provider", "provider", "provider"},
	}

	// Allowlist of valid column names to prevent SQL injection
	validColumns := map[string]bool{
		"agent_id": true, "team_id": true, "model": true, "provider": true,
	}

	for _, dim := range dimensions {
		if !validColumns[dim.idCol] || !validColumns[dim.nameCol] {
			log.Printf("analytics: skipping invalid dimension column: %s", dim.idCol)
			continue
		}

		query := fmt.Sprintf(`
			SELECT
				%s AS dimension_id,
				%s AS dimension_name,
				COALESCE(SUM(cost_usd), 0) AS total_cost,
				COUNT(*) AS total_requests,
				COALESCE(SUM(total_tokens), 0) AS total_tokens,
				COALESCE(AVG(latency_ms), 0) AS avg_latency,
				COALESCE(SUM(savings_usd), 0) AS total_savings
			FROM api_requests
			WHERE timestamp >= $1 AND timestamp <= $2
			  AND %s != ''
			GROUP BY %s
			ORDER BY total_cost DESC
			LIMIT 100
		`, dim.idCol, dim.nameCol, dim.idCol, dim.idCol)

		rows, err := e.pool.Query(ctx, query, from, to)
		if err != nil {
			log.Printf("analytics: failed to generate report for dimension %s: %v", dim.name, err)
			continue
		}

		for rows.Next() {
			var summary models.CostSummary
			var dimID, dimName string
			if err := rows.Scan(&dimID, &dimName, &summary.TotalCostUSD, &summary.TotalRequests,
				&summary.TotalTokens, &summary.AvgLatencyMs, &summary.TotalSavings); err != nil {
				log.Printf("analytics: error scanning report row for %s: %v", dim.name, err)
				continue
			}

			summary.Dimension = dim.name
			summary.DimensionID = dimID
			summary.DimensionName = dimName

			summaries = append(summaries, summary)
		}
		rows.Close()
	}

	return summaries, nil
}

// GetTopSpenders returns the top N entities by cost for a given time window.
func (e *InsightsEngine) GetTopSpenders(ctx context.Context, dimension string, limit int, since time.Time) ([]models.CostSummary, error) {
	if limit <= 0 {
		limit = 10
	}

	var colName string
	switch dimension {
	case "agent":
		colName = "agent_id"
	case "team":
		colName = "team_id"
	case "model":
		colName = "model"
	case "provider":
		colName = "provider"
	default:
		return nil, fmt.Errorf("analytics: unsupported dimension: %s", dimension)
	}

	query := fmt.Sprintf(`
		SELECT
			%s AS dimension_id,
			%s AS dimension_name,
			COALESCE(SUM(cost_usd), 0) AS total_cost,
			COUNT(*) AS total_requests,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(AVG(latency_ms), 0) AS avg_latency,
			COALESCE(SUM(savings_usd), 0) AS total_savings
		FROM api_requests
		WHERE timestamp >= $1
		  AND %s != ''
		GROUP BY %s
		ORDER BY total_cost DESC
		LIMIT $2
	`, colName, colName, colName, colName)

	rows, err := e.pool.Query(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics: failed to get top spenders: %w", err)
	}
	defer rows.Close()

	var summaries []models.CostSummary
	for rows.Next() {
		var summary models.CostSummary
		var dimID, dimName string
		if err := rows.Scan(&dimID, &dimName, &summary.TotalCostUSD, &summary.TotalRequests,
			&summary.TotalTokens, &summary.AvgLatencyMs, &summary.TotalSavings); err != nil {
			return nil, fmt.Errorf("analytics: error scanning top spenders row: %w", err)
		}

		summary.Dimension = dimension
		summary.DimensionID = dimID
		summary.DimensionName = dimName
		summaries = append(summaries, summary)
	}

	return summaries, nil
}
