package router

import (
	"testing"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter(StrategyCostOptimized)
	if r.Strategy != StrategyCostOptimized {
		t.Errorf("expected strategy %s, got %s", StrategyCostOptimized, r.Strategy)
	}
	if len(r.Models) == 0 {
		t.Error("expected default models to be populated")
	}
}

func TestRouteCostOptimized_ShortQuery(t *testing.T) {
	r := NewRouter(StrategyCostOptimized)
	result := r.Route("gpt-4-turbo", 500)
	if result == nil {
		t.Fatal("expected a model, got nil")
	}
	if result.Tier != TierEconomy {
		t.Errorf("expected economy tier for short query, got %s (%s)", result.Tier, result.Model)
	}
}

func TestRouteCostOptimized_MediumQuery(t *testing.T) {
	r := NewRouter(StrategyCostOptimized)
	result := r.Route("gpt-4-turbo", 5000)
	if result == nil {
		t.Fatal("expected a model, got nil")
	}
	if result.Tier != TierStandard {
		t.Errorf("expected standard tier for medium query, got %s (%s)", result.Tier, result.Model)
	}
}

func TestRouteCostOptimized_LongQuery(t *testing.T) {
	r := NewRouter(StrategyCostOptimized)
	result := r.Route("gpt-4-turbo", 10000)
	if result == nil {
		t.Fatal("expected a model, got nil")
	}
	if result.Model != "gpt-4-turbo" {
		t.Errorf("expected requested model for long query, got %s", result.Model)
	}
}

func TestRouteQualityFirst(t *testing.T) {
	r := NewRouter(StrategyQualityFirst)
	result := r.Route("gpt-4o", 100)
	if result == nil {
		t.Fatal("expected a model, got nil")
	}
	if result.Model != "gpt-4o" {
		t.Errorf("expected requested model gpt-4o, got %s", result.Model)
	}
}

func TestRouteQualityFirst_UnknownModel(t *testing.T) {
	r := NewRouter(StrategyQualityFirst)
	result := r.Route("nonexistent-model", 100)
	if result == nil {
		t.Fatal("expected a premium fallback, got nil")
	}
	if result.Tier != TierPremium {
		t.Errorf("expected premium tier fallback, got %s", result.Tier)
	}
}

func TestRouteLatencyOptimized(t *testing.T) {
	r := NewRouter(StrategyLatencyOptimized)
	result := r.Route("gpt-4-turbo", 100)
	if result == nil {
		t.Fatal("expected a model, got nil")
	}
	if result.Tier != TierEconomy {
		t.Errorf("expected economy tier for latency optimization, got %s", result.Tier)
	}
}

func TestFindModel(t *testing.T) {
	r := NewRouter(StrategyCostOptimized)

	found := r.findModel("gpt-4o")
	if found == nil {
		t.Fatal("expected to find gpt-4o")
	}
	if found.Provider != "openai" {
		t.Errorf("expected openai provider, got %s", found.Provider)
	}

	notFound := r.findModel("nonexistent")
	if notFound != nil {
		t.Errorf("expected nil for nonexistent model, got %v", notFound)
	}
}
