package router

import (
	"testing"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

func newTestRouter(strategy RoutingStrategy) *Router {
	return NewRouter(strategy, nil)
}

func TestNewRouter(t *testing.T) {
	r := newTestRouter(StrategyCostOptimized)
	if r.Strategy != StrategyCostOptimized {
		t.Errorf("expected strategy %q, got %q", StrategyCostOptimized, r.Strategy)
	}
	if len(r.modelRegistry) == 0 {
		t.Error("expected non-empty model registry")
	}
}

func TestRouteExplicitModel(t *testing.T) {
	r := newTestRouter(StrategyCostOptimized)

	req := RouteRequest{
		OriginalModel:    "gpt-4o",
		OriginalProvider: models.ProviderOpenAI,
	}

	result := r.Route(req)
	if result.SelectedModel != "gpt-4o" {
		t.Errorf("expected model %q, got %q", "gpt-4o", result.SelectedModel)
	}
	if result.WasRerouted {
		t.Error("expected WasRerouted to be false for explicit model request")
	}
	if result.SelectedProvider != models.ProviderOpenAI {
		t.Errorf("expected provider %q, got %q", models.ProviderOpenAI, result.SelectedProvider)
	}
}

func TestRouteCostOptimized_SimpleQuery(t *testing.T) {
	r := newTestRouter(StrategyCostOptimized)

	req := RouteRequest{
		InputText:        "hello, how are you?",
		OriginalProvider: models.ProviderOpenAI,
	}

	result := r.Route(req)
	// For a simple query with cost optimization, should select economy tier
	if result.Tier != TierEconomy {
		t.Errorf("expected economy tier for simple query, got %q", result.Tier)
	}
	if result.SelectedModel != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini for simple cost-optimized query, got %q", result.SelectedModel)
	}
}

func TestRouteCostOptimized_ComplexQuery(t *testing.T) {
	r := newTestRouter(StrategyCostOptimized)

	// Create a complex query that should trigger standard/premium tier
	req := RouteRequest{
		InputText:        "Please analyze the following code and implement a comprehensive refactoring strategy with step by step reasoning for each change. " + longText(5000),
		HasSystemPrompt:  true,
		OriginalProvider: models.ProviderOpenAI,
	}

	result := r.Route(req)
	// For a complex query, should select at least standard tier
	if tierRank(result.Tier) < tierRank(TierStandard) {
		t.Errorf("expected at least standard tier for complex query, got %q", result.Tier)
	}
}

func TestRouteQualityFirst(t *testing.T) {
	r := newTestRouter(StrategyQualityFirst)

	req := RouteRequest{
		InputText:        "Write a simple greeting.",
		OriginalProvider: models.ProviderAnthropic,
	}

	result := r.Route(req)
	// Quality first should select the highest quality model for the provider
	if result.SelectedProvider != models.ProviderAnthropic {
		t.Errorf("expected Anthropic provider, got %q", result.SelectedProvider)
	}
}

func TestRouteLatencyOptimized(t *testing.T) {
	r := newTestRouter(StrategyLatencyOptimized)

	req := RouteRequest{
		InputText:        "Quick question: what is 2+2?",
		OriginalProvider: models.ProviderGemini,
	}

	result := r.Route(req)
	// Latency optimized should prefer models with lower AvgLatencyMs
	if result.SelectedProvider != models.ProviderGemini {
		t.Errorf("expected Gemini provider, got %q", result.SelectedProvider)
	}
}

func TestRouteAdaptive(t *testing.T) {
	r := newTestRouter(StrategyAdaptive)

	req := RouteRequest{
		InputText:        "hello",
		OriginalProvider: models.ProviderOpenAI,
	}

	result := r.Route(req)
	if result.SelectedModel == "" {
		t.Error("expected a model to be selected")
	}
	if result.Reason == "" {
		t.Error("expected a reason to be provided")
	}
}

func TestAnalyzeComplexity_Low(t *testing.T) {
	req := RouteRequest{
		InputText: "hello",
	}
	complexity := analyzeComplexity(req)
	if complexity != ComplexityLow {
		t.Errorf("expected ComplexityLow for simple input, got %d", complexity)
	}
}

func TestAnalyzeComplexity_Medium(t *testing.T) {
	req := RouteRequest{
		InputText:       "Please explain in detail how to implement a binary search tree.",
		HasSystemPrompt: true,
	}
	complexity := analyzeComplexity(req)
	if complexity < ComplexityMedium {
		t.Errorf("expected at least ComplexityMedium, got %d", complexity)
	}
}

func TestAnalyzeComplexity_High(t *testing.T) {
	req := RouteRequest{
		InputText:       "Please analyze and implement a comprehensive " + longText(20000),
		HasSystemPrompt: true,
	}
	complexity := analyzeComplexity(req)
	if complexity != ComplexityHigh {
		t.Errorf("expected ComplexityHigh for long complex input, got %d", complexity)
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty", "", 0},
		{"short", "hello", 1},
		{"medium", "This is a longer piece of text that should have more tokens.", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimateTokenCount(tt.text)
			if count != tt.expected {
				t.Errorf("estimateTokenCount(%q) = %d, expected %d", tt.text, count, tt.expected)
			}
		})
	}
}

func TestTierRank(t *testing.T) {
	if tierRank(TierEconomy) >= tierRank(TierStandard) {
		t.Error("economy should rank below standard")
	}
	if tierRank(TierStandard) >= tierRank(TierPremium) {
		t.Error("standard should rank below premium")
	}
}

func TestComplexityToMinTier(t *testing.T) {
	if complexityToMinTier(ComplexityLow) != TierEconomy {
		t.Error("low complexity should map to economy tier")
	}
	if complexityToMinTier(ComplexityMedium) != TierStandard {
		t.Error("medium complexity should map to standard tier")
	}
	if complexityToMinTier(ComplexityHigh) != TierPremium {
		t.Error("high complexity should map to premium tier")
	}
}

func TestDefaultModelRegistry(t *testing.T) {
	registry := buildDefaultModelRegistry()

	expectedModels := []string{
		"openai:gpt-4o-mini",
		"openai:gpt-4o",
		"openai:gpt-4-turbo",
		"anthropic:claude-3-haiku-20240307",
		"anthropic:claude-3-5-sonnet-20241022",
		"anthropic:claude-3-opus-20240229",
		"gemini:gemini-1.5-flash",
		"gemini:gemini-1.5-pro",
	}

	for _, key := range expectedModels {
		if _, ok := registry[key]; !ok {
			t.Errorf("expected model %q in registry", key)
		}
	}
}

func TestUpdateMetrics(t *testing.T) {
	r := newTestRouter(StrategyCostOptimized)

	r.UpdateMetrics(models.ProviderOpenAI, "gpt-4o", 500, true)
	r.UpdateMetrics(models.ProviderOpenAI, "gpt-4o", 600, true)
	r.UpdateMetrics(models.ProviderOpenAI, "gpt-4o", 700, false)

	metrics, ok := r.metrics["openai:gpt-4o"]
	if !ok {
		t.Fatal("expected metrics for openai:gpt-4o")
	}
	if metrics.RequestCount != 3 {
		t.Errorf("expected 3 requests, got %d", metrics.RequestCount)
	}
	if metrics.SuccessRate < 0.5 || metrics.SuccessRate > 0.8 {
		t.Errorf("expected success rate ~0.67, got %.2f", metrics.SuccessRate)
	}
	if metrics.ErrorRate < 0.2 || metrics.ErrorRate > 0.5 {
		t.Errorf("expected error rate ~0.33, got %.2f", metrics.ErrorRate)
	}
}

func TestProviderEndpoint(t *testing.T) {
	tests := []struct {
		provider models.LLMProvider
		expected string
	}{
		{models.ProviderOpenAI, "https://api.openai.com"},
		{models.ProviderAnthropic, "https://api.anthropic.com"},
		{models.ProviderGemini, "https://generativelanguage.googleapis.com"},
		{models.LLMProvider("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := providerEndpoint(tt.provider)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// longText generates a string of approximately n characters.
func longText(n int) string {
	base := "This is a sample text for testing purposes. "
	result := ""
	for len(result) < n {
		result += base
	}
	return result[:n]
}
