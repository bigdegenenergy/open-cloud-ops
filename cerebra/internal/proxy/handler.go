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

// Default size limits for proxy request/response bodies.
// Request limit is higher to accommodate multimodal payloads (base64 images/PDFs).
// Response limit is lower since LLM text responses are typically well under 1MB;
// streaming responses bypass this limit entirely.
const (
	defaultMaxRequestBodySize  = 50 << 20 // 50 MB (multimodal: images, PDFs)
	defaultMaxResponseBodySize = 10 << 20 // 10 MB
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
	cfg                 *config.Config
	db                  *database.DB
	enforcer            *budget.Enforcer
	client              *http.Client
	maxRequestBodySize  int64
	maxResponseBodySize int64
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
		maxRequestBodySize:  defaultMaxRequestBodySize,
		maxResponseBodySize: defaultMaxResponseBodySize,
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

	// 1. Read the request body (with size limit to prevent OOM).
	// Read limit+1 bytes so we can distinguish "exactly at limit" from "over limit".
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, h.maxRequestBodySize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()
	if int64(len(body)) > h.maxRequestBodySize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
		return
	}

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

	// 3. Check budget before forwarding (atomically reserves estimated cost).
	const estimatedCost = 0.01 // Conservative default; reconciled after response.
	allowed, err := h.enforcer.CheckBudget(budget.ScopeAgent, agentID, estimatedCost)
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

	// releaseReservation undoes the estimated cost reservation on proxy failure.
	releaseReservation := func() {
		if err := h.enforcer.AdjustReservation(budget.ScopeAgent, agentID, -estimatedCost); err != nil {
			log.Printf("[%s] failed to release budget reservation: %v", reqID, err)
		}
	}

	// 4. Build and send the upstream request (preserving query parameters).
	upstreamURL := buildUpstreamURL(provider, c.Request.URL.Path)
	if c.Request.URL.RawQuery != "" {
		upstreamURL += "?" + c.Request.URL.RawQuery
	}
	upstreamReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		releaseReservation()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}

	// Forward headers and set auth.
	copyHeaders(c.Request.Header, upstreamReq.Header)
	h.setProviderAuth(upstreamReq, provider, c.Request.Header)

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		releaseReservation()
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("upstream request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// 5. Check if the response is a streaming response (SSE).
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		h.streamResponse(c, resp, reqID, provider, model, agentID, teamID, orgID, start, estimatedCost)
		return
	}

	// 6. Read the response body (with size limit to prevent OOM).
	// Read limit+1 to distinguish "exactly at limit" from "over limit".
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, h.maxResponseBodySize+1))
	if err != nil {
		releaseReservation()
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read upstream response"})
		return
	}

	// Check if response exceeded the size limit.
	if int64(len(respBody)) > h.maxResponseBodySize {
		releaseReservation()
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream response too large"})
		return
	}

	latency := time.Since(start).Milliseconds()

	// 7. Extract token usage and calculate cost.
	inputTokens, outputTokens := extractTokenUsage(respBody, provider)
	totalTokens := inputTokens + outputTokens
	costUSD := h.calculateCost(c.Request.Context(), string(provider), model, inputTokens, outputTokens)

	// 8. Record the request in the database (if available).
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
		if h.db != nil {
			if err := h.db.InsertRequest(ctx, apiReq); err != nil {
				log.Printf("[%s] failed to record request: %v", reqID, err)
			}
		}
		// Reconcile the reservation: adjust by (actual - estimated) cost.
		if diff := costUSD - estimatedCost; diff != 0 {
			if err := h.enforcer.AdjustReservation(budget.ScopeAgent, agentID, diff); err != nil {
				log.Printf("[%s] failed to adjust budget reservation: %v", reqID, err)
			}
		}
	}()

	// 9. Return the response to the client.
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

// streamResponse streams SSE responses to the client while capturing data for
// post-stream cost tracking. A capped buffer accumulates the streamed data so
// token usage can be extracted from the final SSE chunks (providers typically
// include usage metadata in their final message_delta / usage chunk).
func (h *ProxyHandler) streamResponse(c *gin.Context, resp *http.Response, reqID string, provider Provider, model, agentID, teamID, orgID string, start time.Time, estimatedCost float64) {
	// Forward response headers.
	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Header("X-Request-ID", reqID)
	c.Status(resp.StatusCode)

	// Capture the tail of the streamed data for usage metadata extraction.
	// Usage info is in the final SSE chunks, so we keep a rolling tail buffer.
	// Truncation happens on SSE event boundaries ("\n\n") to avoid splitting
	// JSON payloads mid-object, which would cause unmarshal failures.
	const maxCapture = 1 << 20 // 1 MB
	var captured bytes.Buffer

	// Stream data to client while tee-ing to captured buffer.
	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 4096)
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := w.Write(buf[:n]); wErr != nil {
				// Client disconnected — stop reading upstream to avoid
				// wasting tokens and keeping the goroutine alive.
				return false
			}
			captured.Write(buf[:n])
			// When buffer exceeds cap, discard leading events to keep
			// memory bounded while retaining complete trailing SSE events.
			if captured.Len() > maxCapture {
				b := captured.Bytes()
				half := len(b) / 2
				// Find the first SSE event boundary ("\n\n") after the midpoint
				// so we only discard complete events, never partial JSON.
				cut := bytes.Index(b[half:], []byte("\n\n"))
				if cut >= 0 {
					half += cut + 2 // skip past the "\n\n"
				}
				captured.Reset()
				captured.Write(b[half:])
			}
		}
		return err == nil
	})

	latency := time.Since(start).Milliseconds()

	// Parse token usage from captured stream data.
	inputTokens, outputTokens := extractStreamTokenUsage(captured.Bytes(), provider)
	totalTokens := inputTokens + outputTokens
	costUSD := h.calculateCost(context.Background(), string(provider), model, inputTokens, outputTokens)

	// Record the streaming request and reconcile budget reservation.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
		if h.db != nil {
			if err := h.db.InsertRequest(ctx, apiReq); err != nil {
				log.Printf("[%s] failed to record streaming request: %v", reqID, err)
			}
		}
		// Reconcile the reservation: adjust by (actual - estimated) cost.
		if diff := costUSD - estimatedCost; diff != 0 {
			if err := h.enforcer.AdjustReservation(budget.ScopeAgent, agentID, diff); err != nil {
				log.Printf("[%s] failed to adjust streaming budget reservation: %v", reqID, err)
			}
		}
	}()
}

// extractStreamTokenUsage parses SSE stream data to find usage metadata.
// LLM providers include token usage in their final streaming chunks:
//   - OpenAI: data: {"usage":{"prompt_tokens":N,"completion_tokens":N}} (when stream_options.include_usage is set)
//   - Anthropic: data: {"type":"message_delta","usage":{"output_tokens":N}}
func extractStreamTokenUsage(data []byte, provider Provider) (int64, int64) {
	// Scan SSE lines for "data: " prefixed JSON containing usage info.
	lines := strings.Split(string(data), "\n")
	var inputTokens, outputTokens int64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "[DONE]" {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
			continue
		}

		switch provider {
		case ProviderOpenAI:
			if usage, ok := parsed["usage"].(map[string]interface{}); ok {
				if v := toFloat(usage["prompt_tokens"]); v > 0 {
					inputTokens = int64(v)
				}
				if v := toFloat(usage["completion_tokens"]); v > 0 {
					outputTokens = int64(v)
				}
			}
		case ProviderAnthropic:
			// Anthropic sends input_tokens in message_start, output_tokens in message_delta.
			if msg, ok := parsed["message"].(map[string]interface{}); ok {
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if v := toFloat(usage["input_tokens"]); v > 0 {
						inputTokens = int64(v)
					}
				}
			}
			if usage, ok := parsed["usage"].(map[string]interface{}); ok {
				if v := toFloat(usage["output_tokens"]); v > 0 {
					outputTokens = int64(v)
				}
			}
		case ProviderGemini:
			if meta, ok := parsed["usageMetadata"].(map[string]interface{}); ok {
				if v := toFloat(meta["promptTokenCount"]); v > 0 {
					inputTokens = int64(v)
				}
				if v := toFloat(meta["candidatesTokenCount"]); v > 0 {
					outputTokens = int64(v)
				}
			}
		}
	}

	return inputTokens, outputTokens
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
	if h.db == nil {
		return 0
	}
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
// Auth headers (Authorization, X-API-Key, X-Goog-Api-Key) are stripped here
// and set explicitly by setProviderAuth to prevent credential leakage across providers.
func copyHeaders(src, dst http.Header) {
	for key, vals := range src {
		lower := strings.ToLower(key)
		// Skip hop-by-hop headers, internal headers, and auth headers
		// (auth is set explicitly per-provider by setProviderAuth)
		if lower == "host" || lower == "connection" ||
			lower == "authorization" || lower == "x-api-key" || lower == "x-goog-api-key" ||
			strings.HasPrefix(lower, "x-agent") ||
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
