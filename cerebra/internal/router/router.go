// Package router implements the smart model routing engine.
//
// The router analyzes incoming LLM requests and determines the optimal
// model to use based on query complexity, cost, and quality requirements.
// Simple queries are routed to cheaper models (e.g., Claude Haiku, GPT-4o Mini)
// while complex queries are sent to more powerful models (e.g., Claude Opus, GPT-4).
package router

// RoutingStrategy defines how requests should be routed across models.
type RoutingStrategy string

const (
	StrategyCostOptimized    RoutingStrategy = "cost_optimized"
	StrategyQualityFirst     RoutingStrategy = "quality_first"
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
	StrategyAdaptive         RoutingStrategy = "adaptive"
)

// ModelTier represents the capability tier of an LLM model.
type ModelTier string

const (
	TierEconomy  ModelTier = "economy"  // e.g., Claude Haiku, GPT-4o Mini
	TierStandard ModelTier = "standard" // e.g., Claude Sonnet, GPT-4o
	TierPremium  ModelTier = "premium"  // e.g., Claude Opus, GPT-4
)

// ModelEntry describes a model with its tier and provider endpoint.
type ModelEntry struct {
	Provider string
	Model    string
	Tier     ModelTier
}

// DefaultModels provides the default model registry.
var DefaultModels = []ModelEntry{
	// OpenAI
	{Provider: "openai", Model: "gpt-4o-mini", Tier: TierEconomy},
	{Provider: "openai", Model: "gpt-4o", Tier: TierStandard},
	{Provider: "openai", Model: "gpt-4-turbo", Tier: TierPremium},
	{Provider: "openai", Model: "o1-mini", Tier: TierStandard},
	{Provider: "openai", Model: "o1", Tier: TierPremium},
	// Anthropic
	{Provider: "anthropic", Model: "claude-haiku-3-20250414", Tier: TierEconomy},
	{Provider: "anthropic", Model: "claude-sonnet-4-20250514", Tier: TierStandard},
	{Provider: "anthropic", Model: "claude-opus-4-20250514", Tier: TierPremium},
	// Gemini
	{Provider: "gemini", Model: "gemini-2.0-flash", Tier: TierEconomy},
	{Provider: "gemini", Model: "gemini-1.5-flash", Tier: TierEconomy},
	{Provider: "gemini", Model: "gemini-2.0-pro", Tier: TierPremium},
}

// Router manages the intelligent routing of LLM requests.
type Router struct {
	Strategy RoutingStrategy
	Models   []ModelEntry
}

// NewRouter creates a new Router with the specified strategy.
func NewRouter(strategy RoutingStrategy) *Router {
	return &Router{
		Strategy: strategy,
		Models:   DefaultModels,
	}
}

// Route determines the optimal model for a given request.
// It analyzes the query complexity and returns the recommended model.
func (r *Router) Route(requestedModel string, inputLength int) *ModelEntry {
	switch r.Strategy {
	case StrategyCostOptimized:
		return r.routeCostOptimized(requestedModel, inputLength)
	case StrategyQualityFirst:
		return r.routeQualityFirst(requestedModel)
	case StrategyLatencyOptimized:
		return r.routeLatencyOptimized()
	default:
		return r.findModel(requestedModel)
	}
}

// routeCostOptimized routes to the cheapest model that can handle the request.
func (r *Router) routeCostOptimized(requestedModel string, inputLength int) *ModelEntry {
	// For short queries (<500 tokens estimate), use economy tier.
	if inputLength < 2000 { // ~500 tokens
		for i := range r.Models {
			if r.Models[i].Tier == TierEconomy {
				return &r.Models[i]
			}
		}
	}

	// For medium queries, use standard tier.
	if inputLength < 8000 { // ~2000 tokens
		for i := range r.Models {
			if r.Models[i].Tier == TierStandard {
				return &r.Models[i]
			}
		}
	}

	// For long/complex queries, keep the requested model.
	return r.findModel(requestedModel)
}

// routeQualityFirst routes to the highest quality model.
func (r *Router) routeQualityFirst(requestedModel string) *ModelEntry {
	// Keep the requested model or upgrade to premium.
	if m := r.findModel(requestedModel); m != nil {
		return m
	}
	for i := range r.Models {
		if r.Models[i].Tier == TierPremium {
			return &r.Models[i]
		}
	}
	return nil
}

// routeLatencyOptimized routes to the fastest responding model.
func (r *Router) routeLatencyOptimized() *ModelEntry {
	// Economy models are generally fastest.
	for i := range r.Models {
		if r.Models[i].Tier == TierEconomy {
			return &r.Models[i]
		}
	}
	return nil
}

// findModel looks up a specific model by name.
func (r *Router) findModel(name string) *ModelEntry {
	for i := range r.Models {
		if r.Models[i].Model == name {
			return &r.Models[i]
		}
	}
	return nil
}
