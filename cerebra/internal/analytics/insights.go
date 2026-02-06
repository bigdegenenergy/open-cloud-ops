// Package analytics implements cost analytics and AI-powered insights.
//
// The analytics engine processes usage data to detect cost spikes,
// identify optimization opportunities, and generate actionable
// recommendations for reducing LLM API spend.
package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	EstimatedSaving float64     `json:"estimated_saving"`
	AffectedEntity  string      `json:"affected_entity"`
	CreatedAt       time.Time   `json:"created_at"`
	Dismissed       bool        `json:"dismissed"`
}

// InsightsEngine generates and manages cost insights.
type InsightsEngine struct {
	pool *pgxpool.Pool
}

// NewInsightsEngine creates a new InsightsEngine.
func NewInsightsEngine(pool *pgxpool.Pool) *InsightsEngine {
	return &InsightsEngine{pool: pool}
}

// DetectSpikes analyzes recent usage data to identify cost spikes.
// A spike is defined as a day where cost exceeds the 7-day rolling average by 2x.
func (e *InsightsEngine) DetectSpikes(ctx context.Context) ([]Insight, error) {
	if e.pool == nil {
		return nil, nil
	}

	rows, err := e.pool.Query(ctx, `
		WITH daily_costs AS (
			SELECT
				DATE(timestamp) AS day,
				SUM(cost_usd) AS daily_cost,
				agent_id
			FROM api_requests
			WHERE timestamp > NOW() - INTERVAL '14 days'
			GROUP BY DATE(timestamp), agent_id
		),
		rolling_avg AS (
			SELECT
				day,
				agent_id,
				daily_cost,
				AVG(daily_cost) OVER (
					PARTITION BY agent_id
					ORDER BY day
					ROWS BETWEEN 7 PRECEDING AND 1 PRECEDING
				) AS avg_cost
			FROM daily_costs
		)
		SELECT day, agent_id, daily_cost, avg_cost
		FROM rolling_avg
		WHERE daily_cost > avg_cost * 2
		  AND avg_cost > 0
		ORDER BY day DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("detecting spikes: %w", err)
	}
	defer rows.Close()

	var insights []Insight
	for rows.Next() {
		var day time.Time
		var agentID string
		var dailyCost, avgCost float64

		if err := rows.Scan(&day, &agentID, &dailyCost, &avgCost); err != nil {
			return nil, fmt.Errorf("scanning spike row: %w", err)
		}

		spikeMultiple := dailyCost / avgCost
		severity := SeverityWarning
		if spikeMultiple > 5 {
			severity = SeverityCritical
		}

		insights = append(insights, Insight{
			ID:       fmt.Sprintf("spike-%s-%s", agentID, day.Format("2006-01-02")),
			Type:     InsightCostSpike,
			Severity: severity,
			Title:    fmt.Sprintf("Cost spike detected for agent %s", agentID),
			Description: fmt.Sprintf(
				"On %s, agent %s spent $%.4f, which is %.1fx the 7-day rolling average of $%.4f.",
				day.Format("Jan 2"), agentID, dailyCost, spikeMultiple, avgCost,
			),
			EstimatedSaving: dailyCost - avgCost,
			AffectedEntity:  agentID,
			CreatedAt:       time.Now(),
		})
	}

	return insights, rows.Err()
}

// RecommendModelSwitches identifies opportunities to use cheaper models.
func (e *InsightsEngine) RecommendModelSwitches(ctx context.Context) ([]Insight, error) {
	if e.pool == nil {
		return nil, nil
	}

	rows, err := e.pool.Query(ctx, `
		SELECT
			model,
			provider,
			COUNT(*) AS request_count,
			SUM(cost_usd) AS total_cost,
			AVG(input_tokens) AS avg_input_tokens
		FROM api_requests
		WHERE timestamp > NOW() - INTERVAL '7 days'
		GROUP BY model, provider
		HAVING SUM(cost_usd) > 1.0
		ORDER BY total_cost DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying model usage: %w", err)
	}
	defer rows.Close()

	// Approximate cost ratios for model downgrades.
	savingsRatios := map[string]struct {
		cheaper string
		ratio   float64
	}{
		"gpt-4o":                   {"gpt-4o-mini", 0.85},
		"gpt-4-turbo":              {"gpt-4o", 0.60},
		"claude-opus-4-20250514":   {"claude-sonnet-4-20250514", 0.80},
		"claude-sonnet-4-20250514": {"claude-haiku-3-20250414", 0.80},
		"gemini-2.0-pro":           {"gemini-2.0-flash", 0.90},
	}

	var insights []Insight
	for rows.Next() {
		var model, provider string
		var reqCount int64
		var totalCost, avgInput float64

		if err := rows.Scan(&model, &provider, &reqCount, &totalCost, &avgInput); err != nil {
			return nil, fmt.Errorf("scanning model usage: %w", err)
		}

		if s, ok := savingsRatios[model]; ok {
			estimatedSaving := totalCost * s.ratio
			insights = append(insights, Insight{
				ID:       fmt.Sprintf("switch-%s-%s", provider, model),
				Type:     InsightModelSwitch,
				Severity: SeverityInfo,
				Title:    fmt.Sprintf("Consider switching %s to %s", model, s.cheaper),
				Description: fmt.Sprintf(
					"You spent $%.2f on %s (%d requests, avg %.0f input tokens). "+
						"Switching to %s for simpler queries could save ~$%.2f/week.",
					totalCost, model, reqCount, avgInput, s.cheaper, estimatedSaving,
				),
				EstimatedSaving: math.Round(estimatedSaving*100) / 100,
				AffectedEntity:  model,
				CreatedAt:       time.Now(),
			})
		}
	}

	return insights, rows.Err()
}

// GenerateReport creates a summary report for a given time period.
func (e *InsightsEngine) GenerateReport(ctx context.Context, from, to time.Time) (*Report, error) {
	if e.pool == nil {
		return nil, nil
	}

	var report Report
	report.From = from
	report.To = to

	err := e.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(cost_usd), 0),
			COUNT(*),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(AVG(latency_ms), 0),
			COALESCE(SUM(savings_usd), 0)
		FROM api_requests
		WHERE timestamp >= $1 AND timestamp <= $2
	`, from, to).Scan(
		&report.TotalCostUSD,
		&report.TotalRequests,
		&report.TotalTokens,
		&report.AvgLatencyMs,
		&report.TotalSavingsUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("generating report: %w", err)
	}

	return &report, nil
}

// Report is a summary of usage and costs over a time period.
type Report struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	TotalCostUSD    float64   `json:"total_cost_usd"`
	TotalRequests   int64     `json:"total_requests"`
	TotalTokens     int64     `json:"total_tokens"`
	AvgLatencyMs    float64   `json:"avg_latency_ms"`
	TotalSavingsUSD float64   `json:"total_savings_usd"`
}
