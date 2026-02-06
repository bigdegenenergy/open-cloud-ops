// Package models defines the core data structures used across Cerebra.
package models

import "time"

// LLMProvider represents a supported LLM API provider.
type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderGemini    LLMProvider = "gemini"
)

// APIRequest represents a single LLM API request and its metadata.
// This is the core data structure for cost tracking.
// Note: Prompt content and response content are NEVER stored.
type APIRequest struct {
	ID            string      `json:"id" db:"id"`
	Provider      LLMProvider `json:"provider" db:"provider"`
	Model         string      `json:"model" db:"model"`
	AgentID       string      `json:"agent_id" db:"agent_id"`
	TeamID        string      `json:"team_id" db:"team_id"`
	OrgID         string      `json:"org_id" db:"org_id"`
	InputTokens   int64       `json:"input_tokens" db:"input_tokens"`
	OutputTokens  int64       `json:"output_tokens" db:"output_tokens"`
	TotalTokens   int64       `json:"total_tokens" db:"total_tokens"`
	CostUSD       float64     `json:"cost_usd" db:"cost_usd"`
	LatencyMs     int64       `json:"latency_ms" db:"latency_ms"`
	StatusCode    int         `json:"status_code" db:"status_code"`
	WasRouted     bool        `json:"was_routed" db:"was_routed"`
	OriginalModel string      `json:"original_model,omitempty" db:"original_model"`
	RoutedModel   string      `json:"routed_model,omitempty" db:"routed_model"`
	SavingsUSD    float64     `json:"savings_usd" db:"savings_usd"`
	Timestamp     time.Time   `json:"timestamp" db:"timestamp"`
}

// Agent represents an AI agent that makes LLM API calls.
type Agent struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	TeamID    string    `json:"team_id" db:"team_id"`
	Tags      []string  `json:"tags" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Team represents a group of agents and users.
type Team struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	OrgID     string    `json:"org_id" db:"org_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Organization represents a top-level entity.
type Organization struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ModelPricing defines the cost per token for a specific LLM model.
type ModelPricing struct {
	Provider        LLMProvider `json:"provider" db:"provider"`
	Model           string      `json:"model" db:"model"`
	InputPerMToken  float64     `json:"input_per_m_token" db:"input_per_m_token"`   // Cost per 1M input tokens
	OutputPerMToken float64     `json:"output_per_m_token" db:"output_per_m_token"` // Cost per 1M output tokens
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// CostSummary provides aggregated cost data for a given dimension and period.
type CostSummary struct {
	Dimension     string  `json:"dimension"` // e.g., "agent", "team", "model", "provider"
	DimensionID   string  `json:"dimension_id"`
	DimensionName string  `json:"dimension_name"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	TotalSavings  float64 `json:"total_savings_usd"`
}

// Budget represents a spending limit for a specific scope and entity.
type Budget struct {
	ID         string    `json:"id" db:"id"`
	Scope      string    `json:"scope" db:"scope"`
	EntityID   string    `json:"entity_id" db:"entity_id"`
	LimitUSD   float64   `json:"limit_usd" db:"limit_usd"`
	SpentUSD   float64   `json:"spent_usd" db:"spent_usd"`
	PeriodDays int       `json:"period_days" db:"period_days"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
