// Package proxy implements the reverse proxy for LLM API providers.
//
// The proxy intercepts requests destined for LLM providers (OpenAI, Anthropic,
// Google Gemini), logs metadata (model, token counts, timestamps, costs),
// and forwards the request to the upstream provider. API keys are passed
// through in-memory and are never persisted to disk or database.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/budget"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

// Provider represents a supported LLM API provider.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
)

// providerUpstreamURLs maps each provider to its base API URL.
var providerUpstreamURLs = map[Provider]string{
	ProviderOpenAI:    "https://api.openai.com",
	ProviderAnthropic: "https://api.anthropic.com",
	ProviderGemini:    "https://generativelanguage.googleapis.com",
}

// ProxyHandler manages the reverse proxying of LLM API requests.
type ProxyHandler struct {
	pool     *pgxpool.Pool
	enforcer *budget.Enforcer
	pricing  map[string]models.ModelPricing // keyed by "provider:model"
	client   *http.Client
}

// NewProxyHandler creates a new ProxyHandler instance with all required dependencies.
func NewProxyHandler(pool *pgxpool.Pool, enforcer *budget.Enforcer, pricing map[string]models.ModelPricing) *ProxyHandler {
	return &ProxyHandler{
		pool:     pool,
		enforcer: enforcer,
		pricing:  pricing,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// HandleRequest processes an incoming LLM API request.
// It extracts metadata, checks budgets, forwards the request to the upstream provider,
// logs usage data, and returns the response unchanged.
func (h *ProxyHandler) HandleRequest(c *gin.Context) {
	startTime := time.Now()

	// 1. Extract provider from URL path (/v1/openai/*, /v1/anthropic/*, /v1/gemini/*)
	provider, upstreamPath, err := extractProviderAndPath(c.Request.URL.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	upstreamBase, ok := providerUpstreamURLs[provider]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported provider: %s", provider)})
		return
	}

	// 2. Read the request body and parse JSON to extract the model name
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	c.Request.Body.Close()

	modelName := extractModelFromRequest(provider, bodyBytes)

	// 3. Extract agent-id, team-id, org-id from headers
	agentID := c.GetHeader("X-Agent-ID")
	teamID := c.GetHeader("X-Team-ID")
	orgID := c.GetHeader("X-Org-ID")

	// 4. Estimate cost and check budget
	estimatedCost := h.estimateCost(string(provider), modelName, len(bodyBytes))

	if h.enforcer != nil {
		// Check budgets at all applicable scopes
		if agentID != "" {
			allowed, err := h.enforcer.CheckBudget(budget.ScopeAgent, agentID, estimatedCost)
			if err != nil {
				log.Printf("proxy: budget check error for agent %s: %v", agentID, err)
			} else if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "budget_exceeded",
					"message": fmt.Sprintf("agent %s has exceeded its budget limit", agentID),
				})
				return
			}
		}
		if teamID != "" {
			allowed, err := h.enforcer.CheckBudget(budget.ScopeTeam, teamID, estimatedCost)
			if err != nil {
				log.Printf("proxy: budget check error for team %s: %v", teamID, err)
			} else if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "budget_exceeded",
					"message": fmt.Sprintf("team %s has exceeded its budget limit", teamID),
				})
				return
			}
		}
		if orgID != "" {
			allowed, err := h.enforcer.CheckBudget(budget.ScopeOrg, orgID, estimatedCost)
			if err != nil {
				log.Printf("proxy: budget check error for org %s: %v", orgID, err)
			} else if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "budget_exceeded",
					"message": fmt.Sprintf("organization %s has exceeded its budget limit", orgID),
				})
				return
			}
		}
	}

	// 5. Build and forward request to the upstream provider
	upstreamURL := upstreamBase + upstreamPath
	if c.Request.URL.RawQuery != "" {
		upstreamURL += "?" + c.Request.URL.RawQuery
	}

	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}

	// Copy headers from the original request, passing through auth headers
	copyRequestHeaders(c.Request, proxyReq, provider)

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// 6. Read response and extract token usage
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read upstream response"})
		return
	}

	latencyMs := time.Since(startTime).Milliseconds()

	inputTokens, outputTokens, totalTokens := extractTokenUsage(provider, respBody)

	// If model was not in the request body (unusual), try to extract from response
	if modelName == "" {
		modelName = extractModelFromResponse(provider, respBody)
	}

	// 7. Calculate cost using ModelPricing
	costUSD := h.calculateCost(string(provider), modelName, inputTokens, outputTokens)

	// 8. Record spend via budget enforcer
	if h.enforcer != nil {
		if agentID != "" {
			if err := h.enforcer.RecordSpend(budget.ScopeAgent, agentID, costUSD); err != nil {
				log.Printf("proxy: failed to record spend for agent %s: %v", agentID, err)
			}
		}
		if teamID != "" {
			if err := h.enforcer.RecordSpend(budget.ScopeTeam, teamID, costUSD); err != nil {
				log.Printf("proxy: failed to record spend for team %s: %v", teamID, err)
			}
		}
		if orgID != "" {
			if err := h.enforcer.RecordSpend(budget.ScopeOrg, orgID, costUSD); err != nil {
				log.Printf("proxy: failed to record spend for org %s: %v", orgID, err)
			}
		}
	}

	// 9. Log APIRequest metadata to database (async via goroutine)
	apiReq := models.APIRequest{
		ID:           uuid.New().String(),
		Provider:     models.LLMProvider(provider),
		Model:        modelName,
		AgentID:      agentID,
		TeamID:       teamID,
		OrgID:        orgID,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		CostUSD:      costUSD,
		LatencyMs:    latencyMs,
		StatusCode:   resp.StatusCode,
		Timestamp:    time.Now(),
	}

	if h.pool != nil {
		go h.logRequest(context.Background(), apiReq)
	}

	// 10. Return original response to client
	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)
}

// extractProviderAndPath parses the request URL to determine the provider and
// the remaining upstream path. Expected format: /v1/{provider}/...
func extractProviderAndPath(urlPath string) (Provider, string, error) {
	// Normalize path: remove leading slash, split by "/"
	trimmed := strings.TrimPrefix(urlPath, "/")
	parts := strings.SplitN(trimmed, "/", 3)

	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid path: expected /v1/{provider}/..., got %s", urlPath)
	}

	if parts[0] != "v1" {
		return "", "", fmt.Errorf("invalid API version in path: expected /v1/..., got /%s/...", parts[0])
	}

	providerStr := strings.ToLower(parts[1])
	var provider Provider
	switch providerStr {
	case "openai":
		provider = ProviderOpenAI
	case "anthropic":
		provider = ProviderAnthropic
	case "gemini":
		provider = ProviderGemini
	default:
		return "", "", fmt.Errorf("unsupported provider: %s", providerStr)
	}

	// Build the upstream path from the remaining segments
	upstreamPath := "/"
	if len(parts) == 3 && parts[2] != "" {
		upstreamPath = "/" + parts[2]
	}

	return provider, upstreamPath, nil
}

// extractModelFromRequest parses the request body JSON to extract the model name.
// Different providers use different JSON structures.
func extractModelFromRequest(provider Provider, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	// All three providers use "model" as the top-level key for the model name
	if model, ok := parsed["model"].(string); ok {
		return model
	}

	return ""
}

// extractModelFromResponse tries to extract the model name from the API response.
func extractModelFromResponse(provider Provider, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	if model, ok := parsed["model"].(string); ok {
		return model
	}

	return ""
}

// extractTokenUsage parses the response body to extract token counts.
// Each provider has a different response structure for usage data.
func extractTokenUsage(provider Provider, body []byte) (inputTokens, outputTokens, totalTokens int64) {
	if len(body) == 0 {
		return 0, 0, 0
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0, 0
	}

	switch provider {
	case ProviderOpenAI:
		// OpenAI: {"usage": {"prompt_tokens": N, "completion_tokens": N, "total_tokens": N}}
		if usage, ok := parsed["usage"].(map[string]interface{}); ok {
			inputTokens = jsonToInt64(usage["prompt_tokens"])
			outputTokens = jsonToInt64(usage["completion_tokens"])
			totalTokens = jsonToInt64(usage["total_tokens"])
		}

	case ProviderAnthropic:
		// Anthropic: {"usage": {"input_tokens": N, "output_tokens": N}}
		if usage, ok := parsed["usage"].(map[string]interface{}); ok {
			inputTokens = jsonToInt64(usage["input_tokens"])
			outputTokens = jsonToInt64(usage["output_tokens"])
			totalTokens = inputTokens + outputTokens
		}

	case ProviderGemini:
		// Gemini: {"usageMetadata": {"promptTokenCount": N, "candidatesTokenCount": N, "totalTokenCount": N}}
		if usage, ok := parsed["usageMetadata"].(map[string]interface{}); ok {
			inputTokens = jsonToInt64(usage["promptTokenCount"])
			outputTokens = jsonToInt64(usage["candidatesTokenCount"])
			totalTokens = jsonToInt64(usage["totalTokenCount"])
		}
	}

	// Fallback: compute total if not explicitly provided
	if totalTokens == 0 && (inputTokens > 0 || outputTokens > 0) {
		totalTokens = inputTokens + outputTokens
	}

	return inputTokens, outputTokens, totalTokens
}

// jsonToInt64 safely converts a JSON number (float64) to int64.
func jsonToInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

// copyRequestHeaders copies relevant headers from the client request to the
// upstream proxy request. It passes through authentication headers and content-type
// but strips Cerebra-specific tracking headers (X-Agent-ID, X-Team-ID, X-Org-ID).
// The client's Authorization and provider-specific auth headers (e.g., x-api-key
// for Anthropic) are preserved so that upstream providers receive valid credentials.
func copyRequestHeaders(src *http.Request, dst *http.Request, provider Provider) {
	// Copy standard headers
	for key, values := range src.Header {
		lowerKey := strings.ToLower(key)

		// Skip hop-by-hop headers and Cerebra-specific tracking headers only
		if lowerKey == "x-agent-id" || lowerKey == "x-team-id" || lowerKey == "x-org-id" ||
			lowerKey == "host" || lowerKey == "connection" {
			continue
		}

		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}

	// Ensure content-type is set
	if dst.Header.Get("Content-Type") == "" {
		dst.Header.Set("Content-Type", "application/json")
	}
}

// estimateCost provides a rough cost estimate for budget pre-check purposes.
// This uses a rough character-to-token ratio to estimate input tokens.
func (h *ProxyHandler) estimateCost(provider, model string, bodyLen int) float64 {
	// Rough estimate: ~4 characters per token for English text
	estimatedInputTokens := float64(bodyLen) / 4.0

	key := provider + ":" + model
	if pricing, ok := h.pricing[key]; ok {
		return (estimatedInputTokens / 1_000_000.0) * pricing.InputPerMToken
	}

	// Default fallback: assume $3 per 1M tokens as a conservative estimate
	return (estimatedInputTokens / 1_000_000.0) * 3.0
}

// calculateCost computes the actual cost based on token usage and model pricing.
func (h *ProxyHandler) calculateCost(provider, model string, inputTokens, outputTokens int64) float64 {
	key := provider + ":" + model
	pricing, ok := h.pricing[key]
	if !ok {
		// If no pricing data found, return zero rather than guessing
		log.Printf("proxy: no pricing data for %s, cost will be recorded as $0", key)
		return 0
	}

	inputCost := (float64(inputTokens) / 1_000_000.0) * pricing.InputPerMToken
	outputCost := (float64(outputTokens) / 1_000_000.0) * pricing.OutputPerMToken

	return inputCost + outputCost
}

// logRequest asynchronously persists API request metadata to the database.
// This runs in a goroutine to avoid blocking the response to the client.
func (h *ProxyHandler) logRequest(ctx context.Context, req models.APIRequest) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO api_requests (
			id, provider, model, agent_id, team_id, org_id,
			input_tokens, output_tokens, total_tokens,
			cost_usd, latency_ms, status_code,
			was_routed, original_model, routed_model, savings_usd,
			timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := h.pool.Exec(ctx, query,
		req.ID, string(req.Provider), req.Model, req.AgentID, req.TeamID, req.OrgID,
		req.InputTokens, req.OutputTokens, req.TotalTokens,
		req.CostUSD, req.LatencyMs, req.StatusCode,
		req.WasRouted, req.OriginalModel, req.RoutedModel, req.SavingsUSD,
		req.Timestamp,
	)
	if err != nil {
		log.Printf("proxy: failed to log API request %s: %v", req.ID, err)
	}
}

// GetPricing returns the current pricing map (used by tests and other components).
func (h *ProxyHandler) GetPricing() map[string]models.ModelPricing {
	return h.pricing
}
