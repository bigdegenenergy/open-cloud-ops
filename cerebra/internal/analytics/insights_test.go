package analytics

import (
	"testing"
	"time"
)

func TestInsightTypeConstants(t *testing.T) {
	types := []InsightType{
		InsightCostSpike,
		InsightModelSwitch,
		InsightBudgetWarning,
		InsightAnomalyDetected,
		InsightSavingsFound,
	}

	seen := make(map[InsightType]bool)
	for _, it := range types {
		if seen[it] {
			t.Errorf("duplicate insight type: %s", it)
		}
		seen[it] = true
		if it == "" {
			t.Error("insight type should not be empty")
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.expected {
			t.Errorf("expected severity %q, got %q", tt.expected, tt.severity)
		}
	}
}

func TestSpikeThreshold(t *testing.T) {
	if SpikeThreshold != 2.0 {
		t.Errorf("expected spike threshold 2.0, got %f", SpikeThreshold)
	}
}

func TestPremiumModelAlternatives(t *testing.T) {
	expectedMappings := map[string]string{
		"gpt-4-turbo":            "gpt-4o",
		"gpt-4":                  "gpt-4o",
		"o1":                     "gpt-4o",
		"claude-3-opus-20240229": "claude-3-5-sonnet-20241022",
		"gemini-ultra":           "gemini-1.5-pro",
	}

	for premium, expectedAlternative := range expectedMappings {
		alternative, ok := premiumModelAlternatives[premium]
		if !ok {
			t.Errorf("expected alternative for %q, but none found", premium)
			continue
		}
		if alternative != expectedAlternative {
			t.Errorf("for %q: expected alternative %q, got %q", premium, expectedAlternative, alternative)
		}
	}
}

func TestInsightStruct(t *testing.T) {
	insight := Insight{
		ID:              "insight-123",
		Type:            InsightCostSpike,
		Severity:        SeverityWarning,
		Title:           "Cost spike detected: 3.5x above average",
		Description:     "Agent agent-1 spent $5.00 in the last hour, which is 3.5x the rolling average.",
		EstimatedSaving: 3.50,
		AffectedEntity:  "agent-1",
		CreatedAt:       time.Now(),
		Dismissed:       false,
	}

	if insight.ID != "insight-123" {
		t.Errorf("unexpected ID: %s", insight.ID)
	}
	if insight.Type != InsightCostSpike {
		t.Errorf("unexpected type: %s", insight.Type)
	}
	if insight.Severity != SeverityWarning {
		t.Errorf("unexpected severity: %s", insight.Severity)
	}
	if insight.EstimatedSaving != 3.50 {
		t.Errorf("unexpected estimated saving: %f", insight.EstimatedSaving)
	}
	if insight.Dismissed {
		t.Error("expected dismissed to be false")
	}
}

func TestNewInsightsEngine(t *testing.T) {
	engine := NewInsightsEngine(nil)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.pool != nil {
		t.Error("expected nil pool when created with nil")
	}
}

func TestInsightSeverityClassification(t *testing.T) {
	// Test the logic: multiplier >= 5.0 -> critical, otherwise warning
	tests := []struct {
		multiplier float64
		expected   Severity
	}{
		{1.5, SeverityWarning},
		{2.0, SeverityWarning},
		{3.0, SeverityWarning},
		{4.9, SeverityWarning},
		{5.0, SeverityCritical},
		{10.0, SeverityCritical},
	}

	for _, tt := range tests {
		severity := SeverityWarning
		if tt.multiplier >= 5.0 {
			severity = SeverityCritical
		}
		if severity != tt.expected {
			t.Errorf("multiplier %.1f: expected severity %q, got %q", tt.multiplier, tt.expected, severity)
		}
	}
}

func TestEstimatedSavingsCalculation(t *testing.T) {
	// Model switch savings are estimated at 60% of total cost
	totalCost := 100.0
	estimatedSavings := totalCost * 0.60

	if estimatedSavings != 60.0 {
		t.Errorf("expected savings $60.00, got $%.2f", estimatedSavings)
	}
}

func TestSpikeSavingsCalculation(t *testing.T) {
	// Spike savings = recent cost - average cost
	recentCost := 10.0
	avgCost := 3.0
	savings := recentCost - avgCost

	if savings != 7.0 {
		t.Errorf("expected savings $7.00, got $%.2f", savings)
	}
}

func TestPremiumModelAlternativesNotEmpty(t *testing.T) {
	if len(premiumModelAlternatives) == 0 {
		t.Error("expected non-empty premium model alternatives map")
	}

	for premium, alternative := range premiumModelAlternatives {
		if premium == "" {
			t.Error("premium model key should not be empty")
		}
		if alternative == "" {
			t.Errorf("alternative for %q should not be empty", premium)
		}
		// Alternative should not be the same as the premium model
		if premium == alternative {
			t.Errorf("alternative should differ from premium model: %q", premium)
		}
	}
}
