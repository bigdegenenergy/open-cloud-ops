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

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/budget"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/config"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/database"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Provider represents a supported LLM API provider.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
)

// providerBaseURLs maps providers to their API base URLs.
var providerBaseURLs = map[Provider]string{
	ProviderOpenAI:    "https://api.openai.com",
	ProviderAnthropic: "https://api.anthropic.com",
	ProviderGemini:    "https://generativelanguage.googleapis.com",
}

// ProxyHandler manages the reverse proxying of LLM API requests.
type ProxyHandler struct {
	cfg      *config.Config
	db       *database.DB
	enforcer *budget.Enforcer
	client   *http.Client
}

// NewProxyHandler creates a new ProxyHandler instance.
func NewProxyHandler(cfg *config.Config, db *database.DB, enforcer *budget.Enforcer) *ProxyHandler {
	return &ProxyHandler{
		cfg:      cfg,
		db:       db,
		enforcer: enforcer,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// HandleOpenAI proxies requests to the OpenAI API.
func (h *ProxyHandler) HandleOpenAI(c *gin.Context) {
	h.proxyRequest(c, ProviderOpenAI)
}

// HandleAnthropic proxies requests to the Anthropic API.
func (h *ProxyHandler) HandleAnthropic(c *gin.Context) {
	h.proxyRequest(c, ProviderAnthropic)
}

// HandleGemini proxies requests to the Google Gemini API.
func (h *ProxyHandler) HandleGemini(c *gin.Context) {
	h.proxyRequest(c, ProviderGemini)
}

// proxyRequest handles the core proxy logic for any provider.
func (h *ProxyHandler) proxyRequest(c *gin.Context, provider Provider) {
	start := time.Now()
	reqID := uuid.New().String()

	// 1. Read the request body.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()

	// 2. Extract metadata from headers and body.
	agentID := c.GetHeader("X-Agent-ID")
	teamID := c.GetHeader("X-Team-ID")
	orgID := c.GetHeader("X-Org-ID")
	if agentID == "" {
		agentID = "default"
	}
	if teamID == "" {
		teamID = "default"
	}
	if orgID == "" {
		orgID = "default"
	}

	model := extractModel(body, provider)

	// 3. Check budget before forwarding.
	allowed, err := h.enforcer.CheckBudget(budget.ScopeAgent, agentID, 0)
	if err != nil {
		log.Printf("[%s] budget check error: %v", reqID, err)
	}
	if !allowed {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":    "budget limit exceeded",
			"agent_id": agentID,
		})
		return
	}

	// 4. Build and send the upstream request.
	upstreamURL := buildUpstreamURL(provider, c.Request.URL.Path)
	upstreamReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}

	// Forward headers and set auth.
	copyHeaders(c.Request.Header, upstreamReq.Header)
	h.setProviderAuth(upstreamReq, provider, c.Request.Header)

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// 5. Read the response body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read upstream response"})
		return
	}

	latency := time.Since(start).Milliseconds()

	// 6. Extract token usage and calculate cost.
	inputTokens, outputTokens := extractTokenUsage(respBody, provider)
	totalTokens := inputTokens + outputTokens
	costUSD := h.calculateCost(c.Request.Context(), string(provider), model, inputTokens, outputTokens)

	// 7. Record the request in the database.
	apiReq := &models.APIRequest{
		ID:           reqID,
		Provider:     models.LLMProvider(provider),
		Model:        model,
		AgentID:      agentID,
		TeamID:       teamID,
		OrgID:        orgID,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		CostUSD:      costUSD,
		LatencyMs:    latency,
		StatusCode:   resp.StatusCode,
		Timestamp:    time.Now(),
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.db.InsertRequest(ctx, apiReq); err != nil {
			log.Printf("[%s] failed to record request: %v", reqID, err)
		}
		if err := h.enforcer.RecordSpend(budget.ScopeAgent, agentID, costUSD); err != nil {
			log.Printf("[%s] failed to record spend: %v", reqID, err)
		}
	}()

	// 8. Return the response to the client.
	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Header("X-Request-ID", reqID)
	c.Header("X-Cost-USD", fmt.Sprintf("%.6f", costUSD))
	c.Header("X-Latency-Ms", fmt.Sprintf("%d", latency))
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// setProviderAuth sets the appropriate authentication header for each provider.
func (h *ProxyHandler) setProviderAuth(req *http.Request, provider Provider, originalHeaders http.Header) {
	switch provider {
	case ProviderOpenAI:
		if key := originalHeaders.Get("Authorization"); key != "" {
			req.Header.Set("Authorization", key)
		} else if h.cfg.OpenAIKey != "" {
			req.Header.Set("Authorization", "Bearer "+h.cfg.OpenAIKey)
		}
	case ProviderAnthropic:
		if key := originalHeaders.Get("X-API-Key"); key != "" {
			req.Header.Set("X-API-Key", key)
		} else if h.cfg.AnthropicKey != "" {
			req.Header.Set("X-API-Key", h.cfg.AnthropicKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	case ProviderGemini:
		if key := originalHeaders.Get("X-Goog-Api-Key"); key != "" {
			req.Header.Set("X-Goog-Api-Key", key)
		} else if h.cfg.GeminiKey != "" {
			req.Header.Set("X-Goog-Api-Key", h.cfg.GeminiKey)
		}
	}
}

// calculateCost computes the cost of a request based on model pricing.
func (h *ProxyHandler) calculateCost(ctx context.Context, provider, model string, inputTokens, outputTokens int64) float64 {
	pricing, err := h.db.GetModelPricing(ctx, provider, model)
	if err != nil {
		log.Printf("pricing not found for %s/%s, using zero cost", provider, model)
		return 0
	}

	inputCost := float64(inputTokens) * pricing.InputPerMToken / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPerMToken / 1_000_000
	return inputCost + outputCost
}

// buildUpstreamURL constructs the full URL for the upstream provider.
func buildUpstreamURL(provider Provider, requestPath string) string {
	base := providerBaseURLs[provider]

	// Strip the proxy prefix (e.g., /v1/openai/... -> /...)
	parts := strings.SplitN(requestPath, string("/"+provider+"/"), 2)
	if len(parts) == 2 {
		return base + "/" + parts[1]
	}

	// Strip just the provider prefix for /v1/proxy/openai paths
	parts = strings.SplitN(requestPath, "/proxy/"+string(provider), 2)
	if len(parts) == 2 {
		return base + parts[1]
	}

	return base + requestPath
}

// copyHeaders copies relevant headers from the original request.
func copyHeaders(src, dst http.Header) {
	for key, vals := range src {
		lower := strings.ToLower(key)
		// Skip hop-by-hop headers and internal headers
		if lower == "host" || lower == "connection" || strings.HasPrefix(lower, "x-agent") ||
			strings.HasPrefix(lower, "x-team") || strings.HasPrefix(lower, "x-org") {
			continue
		}
		for _, v := range vals {
			dst.Add(key, v)
		}
	}
}

// extractModel pulls the model name from the request body.
func extractModel(body []byte, provider Provider) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "unknown"
	}
	if m, ok := parsed["model"].(string); ok {
		return m
	}
	return "unknown"
}

// extractTokenUsage pulls input/output token counts from the provider response.
func extractTokenUsage(body []byte, provider Provider) (int64, int64) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0
	}

	switch provider {
	case ProviderOpenAI:
		return extractOpenAITokens(parsed)
	case ProviderAnthropic:
		return extractAnthropicTokens(parsed)
	case ProviderGemini:
		return extractGeminiTokens(parsed)
	}
	return 0, 0
}

func extractOpenAITokens(data map[string]interface{}) (int64, int64) {
	usage, ok := data["usage"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	input := int64(toFloat(usage["prompt_tokens"]))
	output := int64(toFloat(usage["completion_tokens"]))
	return input, output
}

func extractAnthropicTokens(data map[string]interface{}) (int64, int64) {
	usage, ok := data["usage"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	input := int64(toFloat(usage["input_tokens"]))
	output := int64(toFloat(usage["output_tokens"]))
	return input, output
}

func extractGeminiTokens(data map[string]interface{}) (int64, int64) {
	meta, ok := data["usageMetadata"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	input := int64(toFloat(meta["promptTokenCount"]))
	output := int64(toFloat(meta["candidatesTokenCount"]))
	return input, output
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
