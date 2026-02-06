// Package router implements the smart model routing engine.
//
// The router analyzes incoming LLM requests and determines the optimal
// model to use based on query complexity, cost, and quality requirements.
// Simple queries are routed to cheaper models (e.g., Claude Haiku, GPT-4o Mini)
// while complex queries are sent to more powerful models (e.g., Claude Opus, GPT-4).
package router

import (
	"math"
	"sort"
	"strings"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

// RoutingStrategy defines how requests should be routed across models.
type RoutingStrategy string

const (
	// StrategyCostOptimized routes to the cheapest model that meets quality thresholds.
	StrategyCostOptimized RoutingStrategy = "cost_optimized"

	// StrategyQualityFirst routes to the highest quality model within budget.
	StrategyQualityFirst RoutingStrategy = "quality_first"

	// StrategyLatencyOptimized routes to the fastest responding model.
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"

	// StrategyAdaptive uses a scoring function to dynamically select the best model.
	StrategyAdaptive RoutingStrategy = "adaptive"
)

// ModelTier represents the capability tier of an LLM model.
type ModelTier string

const (
	TierEconomy  ModelTier = "economy"  // e.g., Claude Haiku, GPT-4o Mini
	TierStandard ModelTier = "standard" // e.g., Claude Sonnet, GPT-4o
	TierPremium  ModelTier = "premium"  // e.g., Claude Opus, GPT-4
)

// ModelInfo describes a model available for routing.
type ModelInfo struct {
	Provider     models.LLMProvider
	Model        string
	Tier         ModelTier
	QualityScore float64 // 0.0-1.0 quality rating
	AvgLatencyMs int64   // Average latency in milliseconds
	InputPerM    float64 // Cost per 1M input tokens
	OutputPerM   float64 // Cost per 1M output tokens
}

// RouteRequest contains the information needed to make a routing decision.
type RouteRequest struct {
	OriginalModel   string             // The model originally requested (may be empty for auto-routing)
	OriginalProvider models.LLMProvider // The provider originally requested (may be empty)
	InputText       string             // The input text/prompt for complexity analysis
	HasSystemPrompt bool               // Whether the request includes a system prompt
	MaxBudgetUSD    float64            // Maximum budget available for this request
	PreferredTier   ModelTier          // Optional preferred tier
	AgentID         string             // Agent making the request
	TeamID          string             // Team the agent belongs to
}

// RouteResult contains the routing decision.
type RouteResult struct {
	SelectedModel    string
	SelectedProvider models.LLMProvider
	ProviderEndpoint string
	Tier             ModelTier
	EstimatedCostUSD float64
	WasRerouted      bool   // True if the model was changed from the original request
	Reason           string // Explanation for the routing decision
}

// ComplexityLevel represents the assessed complexity of a query.
type ComplexityLevel int

const (
	ComplexityLow    ComplexityLevel = 1
	ComplexityMedium ComplexityLevel = 2
	ComplexityHigh   ComplexityLevel = 3
)

// QualityMetrics tracks runtime quality data for models.
type QualityMetrics struct {
	SuccessRate   float64
	AvgLatencyMs  float64
	ErrorRate     float64
	RequestCount  int64
}

// Router manages the intelligent routing of LLM requests.
type Router struct {
	Strategy      RoutingStrategy
	modelRegistry map[string]ModelInfo            // keyed by "provider:model"
	pricing       map[string]models.ModelPricing  // keyed by "provider:model"
	metrics       map[string]*QualityMetrics      // keyed by "provider:model"
}

// NewRouter creates a new Router with the specified strategy and pricing data.
func NewRouter(strategy RoutingStrategy, pricing map[string]models.ModelPricing) *Router {
	r := &Router{
		Strategy:      strategy,
		pricing:       pricing,
		metrics:       make(map[string]*QualityMetrics),
		modelRegistry: buildDefaultModelRegistry(),
	}
	return r
}

// Route determines the optimal model for a given request.
// It analyzes the query complexity and returns the recommended model.
func (r *Router) Route(req RouteRequest) RouteResult {
	// If a specific model is requested and no auto-routing is needed, honor it
	if req.OriginalModel != "" && req.PreferredTier == "" {
		key := string(req.OriginalProvider) + ":" + req.OriginalModel
		if info, ok := r.modelRegistry[key]; ok {
			return RouteResult{
				SelectedModel:    req.OriginalModel,
				SelectedProvider: req.OriginalProvider,
				ProviderEndpoint: providerEndpoint(req.OriginalProvider),
				Tier:             info.Tier,
				WasRerouted:      false,
				Reason:           "using explicitly requested model",
			}
		}
	}

	// 1. Analyze query complexity
	complexity := analyzeComplexity(req)

	// 2. Determine the minimum tier needed based on complexity
	minTier := complexityToMinTier(complexity)

	// 3. Get candidate models that meet the minimum tier
	candidates := r.getCandidates(minTier, req.MaxBudgetUSD, req.OriginalProvider)

	if len(candidates) == 0 {
		// Fallback: return the original model if no candidates found
		return RouteResult{
			SelectedModel:    req.OriginalModel,
			SelectedProvider: req.OriginalProvider,
			ProviderEndpoint: providerEndpoint(req.OriginalProvider),
			WasRerouted:      false,
			Reason:           "no suitable candidates found, using original model",
		}
	}

	// 4. Select model based on strategy
	var selected ModelInfo
	var reason string

	switch r.Strategy {
	case StrategyCostOptimized:
		selected, reason = r.selectCostOptimized(candidates, minTier)
	case StrategyQualityFirst:
		selected, reason = r.selectQualityFirst(candidates, req.MaxBudgetUSD)
	case StrategyLatencyOptimized:
		selected, reason = r.selectLatencyOptimized(candidates)
	case StrategyAdaptive:
		selected, reason = r.selectAdaptive(candidates, complexity, req.MaxBudgetUSD)
	default:
		selected, reason = r.selectCostOptimized(candidates, minTier)
	}

	// Estimate cost (rough: assume 1000 input tokens, 500 output tokens as baseline)
	estimatedTokens := estimateTokenCount(req.InputText)
	estimatedCost := (float64(estimatedTokens) / 1_000_000.0) * selected.InputPerM

	wasRerouted := req.OriginalModel != "" && req.OriginalModel != selected.Model

	return RouteResult{
		SelectedModel:    selected.Model,
		SelectedProvider: selected.Provider,
		ProviderEndpoint: providerEndpoint(selected.Provider),
		Tier:             selected.Tier,
		EstimatedCostUSD: estimatedCost,
		WasRerouted:      wasRerouted,
		Reason:           reason,
	}
}

// analyzeComplexity assesses the complexity of a request based on heuristics.
func analyzeComplexity(req RouteRequest) ComplexityLevel {
	score := 0

	// Factor 1: Input length (token count heuristic)
	tokenEstimate := estimateTokenCount(req.InputText)
	if tokenEstimate > 4000 {
		score += 3
	} else if tokenEstimate > 1000 {
		score += 2
	} else {
		score += 1
	}

	// Factor 2: Presence of system prompt (indicates structured/complex task)
	if req.HasSystemPrompt {
		score += 1
	}

	// Factor 3: Content complexity indicators
	lowerInput := strings.ToLower(req.InputText)

	// Complex task indicators
	complexIndicators := []string{
		"analyze", "compare", "evaluate", "synthesize", "critique",
		"write code", "implement", "debug", "refactor", "architect",
		"translate", "summarize the following", "explain in detail",
		"step by step", "reasoning", "proof", "mathematical",
	}
	for _, indicator := range complexIndicators {
		if strings.Contains(lowerInput, indicator) {
			score += 1
			break
		}
	}

	// Simple task indicators (reduce score)
	simpleIndicators := []string{
		"hello", "hi", "thanks", "yes", "no",
		"what is", "define", "list", "name",
	}
	for _, indicator := range simpleIndicators {
		if strings.Contains(lowerInput, indicator) && tokenEstimate < 200 {
			score -= 1
			break
		}
	}

	// Clamp to valid range
	if score <= 2 {
		return ComplexityLow
	} else if score <= 4 {
		return ComplexityMedium
	}
	return ComplexityHigh
}

// estimateTokenCount provides a rough token count estimate from text.
// Uses the ~4 characters per token heuristic for English.
func estimateTokenCount(text string) int {
	if len(text) == 0 {
		return 0
	}
	// Rough approximation: 1 token per 4 characters for English
	return len(text) / 4
}

// complexityToMinTier maps complexity to the minimum model tier needed.
func complexityToMinTier(complexity ComplexityLevel) ModelTier {
	switch complexity {
	case ComplexityLow:
		return TierEconomy
	case ComplexityMedium:
		return TierStandard
	case ComplexityHigh:
		return TierPremium
	default:
		return TierStandard
	}
}

// tierRank returns a numeric rank for tier comparison.
func tierRank(tier ModelTier) int {
	switch tier {
	case TierEconomy:
		return 1
	case TierStandard:
		return 2
	case TierPremium:
		return 3
	default:
		return 0
	}
}

// getCandidates returns models that meet the minimum tier and are within budget.
func (r *Router) getCandidates(minTier ModelTier, maxBudget float64, preferredProvider models.LLMProvider) []ModelInfo {
	var candidates []ModelInfo
	minRank := tierRank(minTier)

	for _, info := range r.modelRegistry {
		// Must meet minimum tier
		if tierRank(info.Tier) < minRank {
			continue
		}

		// If a preferred provider is set, only include models from that provider
		if preferredProvider != "" && info.Provider != preferredProvider {
			continue
		}

		candidates = append(candidates, info)
	}

	return candidates
}

// selectCostOptimized picks the cheapest model that meets the minimum tier threshold.
func (r *Router) selectCostOptimized(candidates []ModelInfo, minTier ModelTier) (ModelInfo, string) {
	// Sort by input cost (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].InputPerM < candidates[j].InputPerM
	})

	minRank := tierRank(minTier)
	for _, c := range candidates {
		if tierRank(c.Tier) >= minRank {
			return c, "cost_optimized: selected cheapest model meeting quality threshold"
		}
	}

	// Fallback to absolute cheapest
	return candidates[0], "cost_optimized: selected cheapest available model"
}

// selectQualityFirst picks the highest quality model that fits within budget.
func (r *Router) selectQualityFirst(candidates []ModelInfo, maxBudget float64) (ModelInfo, string) {
	// Sort by quality score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].QualityScore > candidates[j].QualityScore
	})

	// If we have a budget constraint, filter
	if maxBudget > 0 {
		for _, c := range candidates {
			// Rough budget check: assume 10k tokens, see if it fits
			estimatedCost := (10000.0 / 1_000_000.0) * (c.InputPerM + c.OutputPerM)
			if estimatedCost <= maxBudget {
				return c, "quality_first: selected highest quality model within budget"
			}
		}
	}

	// Return highest quality regardless
	return candidates[0], "quality_first: selected highest quality model"
}

// selectLatencyOptimized picks the model with the lowest average latency.
func (r *Router) selectLatencyOptimized(candidates []ModelInfo) (ModelInfo, string) {
	// Sort by latency (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].AvgLatencyMs < candidates[j].AvgLatencyMs
	})

	return candidates[0], "latency_optimized: selected fastest responding model"
}

// selectAdaptive uses a weighted scoring function to balance cost, quality, and latency.
func (r *Router) selectAdaptive(candidates []ModelInfo, complexity ComplexityLevel, maxBudget float64) (ModelInfo, string) {
	if len(candidates) == 0 {
		return ModelInfo{}, "adaptive: no candidates available"
	}

	// Determine weights based on complexity
	var costWeight, qualityWeight, latencyWeight float64
	switch complexity {
	case ComplexityLow:
		costWeight = 0.6
		qualityWeight = 0.1
		latencyWeight = 0.3
	case ComplexityMedium:
		costWeight = 0.3
		qualityWeight = 0.4
		latencyWeight = 0.3
	case ComplexityHigh:
		costWeight = 0.1
		qualityWeight = 0.7
		latencyWeight = 0.2
	}

	// Find min/max for normalization
	var maxCost, minCost float64
	var maxLatency, minLatency float64
	maxCost = 0
	minCost = math.MaxFloat64
	maxLatency = 0
	minLatency = math.MaxFloat64

	for _, c := range candidates {
		cost := c.InputPerM + c.OutputPerM
		if cost > maxCost {
			maxCost = cost
		}
		if cost < minCost {
			minCost = cost
		}
		lat := float64(c.AvgLatencyMs)
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	costRange := maxCost - minCost
	if costRange == 0 {
		costRange = 1 // Avoid division by zero
	}
	latencyRange := maxLatency - minLatency
	if latencyRange == 0 {
		latencyRange = 1
	}

	// Score each candidate
	bestScore := -1.0
	bestIdx := 0

	for i, c := range candidates {
		cost := c.InputPerM + c.OutputPerM

		// Normalize cost: lower is better, so invert
		normalizedCost := 1.0 - ((cost - minCost) / costRange)

		// Quality is already 0-1
		normalizedQuality := c.QualityScore

		// Normalize latency: lower is better, so invert
		normalizedLatency := 1.0 - ((float64(c.AvgLatencyMs) - minLatency) / latencyRange)

		score := (costWeight * normalizedCost) +
			(qualityWeight * normalizedQuality) +
			(latencyWeight * normalizedLatency)

		// Check runtime metrics if available
		key := string(c.Provider) + ":" + c.Model
		if metrics, ok := r.metrics[key]; ok {
			// Penalize models with high error rates
			score *= (1.0 - metrics.ErrorRate)
		}

		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	return candidates[bestIdx], "adaptive: selected model via weighted scoring (cost/quality/latency)"
}

// UpdateMetrics updates the runtime quality metrics for a model.
// This should be called after each request to refine routing decisions.
func (r *Router) UpdateMetrics(provider models.LLMProvider, model string, latencyMs int64, success bool) {
	key := string(provider) + ":" + model

	metrics, ok := r.metrics[key]
	if !ok {
		metrics = &QualityMetrics{}
		r.metrics[key] = metrics
	}

	metrics.RequestCount++

	// Exponential moving average for latency
	alpha := 0.1
	if metrics.RequestCount == 1 {
		metrics.AvgLatencyMs = float64(latencyMs)
	} else {
		metrics.AvgLatencyMs = alpha*float64(latencyMs) + (1-alpha)*metrics.AvgLatencyMs
	}

	// Update success/error rate
	if success {
		metrics.SuccessRate = (metrics.SuccessRate*float64(metrics.RequestCount-1) + 1.0) / float64(metrics.RequestCount)
	} else {
		metrics.SuccessRate = (metrics.SuccessRate * float64(metrics.RequestCount-1)) / float64(metrics.RequestCount)
	}
	metrics.ErrorRate = 1.0 - metrics.SuccessRate
}

// GetModelRegistry returns the current model registry for inspection.
func (r *Router) GetModelRegistry() map[string]ModelInfo {
	return r.modelRegistry
}

// providerEndpoint maps a provider to its base API endpoint.
func providerEndpoint(provider models.LLMProvider) string {
	switch provider {
	case models.ProviderOpenAI:
		return "https://api.openai.com"
	case models.ProviderAnthropic:
		return "https://api.anthropic.com"
	case models.ProviderGemini:
		return "https://generativelanguage.googleapis.com"
	default:
		return ""
	}
}

// buildDefaultModelRegistry creates the default set of models available for routing.
func buildDefaultModelRegistry() map[string]ModelInfo {
	registry := map[string]ModelInfo{
		// OpenAI models
		"openai:gpt-4o-mini": {
			Provider: models.ProviderOpenAI, Model: "gpt-4o-mini",
			Tier: TierEconomy, QualityScore: 0.65, AvgLatencyMs: 400,
			InputPerM: 0.15, OutputPerM: 0.60,
		},
		"openai:gpt-4o": {
			Provider: models.ProviderOpenAI, Model: "gpt-4o",
			Tier: TierStandard, QualityScore: 0.85, AvgLatencyMs: 800,
			InputPerM: 2.50, OutputPerM: 10.00,
		},
		"openai:gpt-4-turbo": {
			Provider: models.ProviderOpenAI, Model: "gpt-4-turbo",
			Tier: TierPremium, QualityScore: 0.90, AvgLatencyMs: 1200,
			InputPerM: 10.00, OutputPerM: 30.00,
		},
		"openai:o1": {
			Provider: models.ProviderOpenAI, Model: "o1",
			Tier: TierPremium, QualityScore: 0.95, AvgLatencyMs: 2000,
			InputPerM: 15.00, OutputPerM: 60.00,
		},

		// Anthropic models
		"anthropic:claude-3-haiku-20240307": {
			Provider: models.ProviderAnthropic, Model: "claude-3-haiku-20240307",
			Tier: TierEconomy, QualityScore: 0.70, AvgLatencyMs: 350,
			InputPerM: 0.25, OutputPerM: 1.25,
		},
		"anthropic:claude-3-5-sonnet-20241022": {
			Provider: models.ProviderAnthropic, Model: "claude-3-5-sonnet-20241022",
			Tier: TierStandard, QualityScore: 0.88, AvgLatencyMs: 700,
			InputPerM: 3.00, OutputPerM: 15.00,
		},
		"anthropic:claude-3-opus-20240229": {
			Provider: models.ProviderAnthropic, Model: "claude-3-opus-20240229",
			Tier: TierPremium, QualityScore: 0.92, AvgLatencyMs: 1500,
			InputPerM: 15.00, OutputPerM: 75.00,
		},

		// Gemini models
		"gemini:gemini-1.5-flash": {
			Provider: models.ProviderGemini, Model: "gemini-1.5-flash",
			Tier: TierEconomy, QualityScore: 0.60, AvgLatencyMs: 300,
			InputPerM: 0.075, OutputPerM: 0.30,
		},
		"gemini:gemini-1.5-pro": {
			Provider: models.ProviderGemini, Model: "gemini-1.5-pro",
			Tier: TierStandard, QualityScore: 0.82, AvgLatencyMs: 900,
			InputPerM: 1.25, OutputPerM: 5.00,
		},
		"gemini:gemini-ultra": {
			Provider: models.ProviderGemini, Model: "gemini-ultra",
			Tier: TierPremium, QualityScore: 0.88, AvgLatencyMs: 1800,
			InputPerM: 10.00, OutputPerM: 30.00,
		},
	}

	return registry
}
