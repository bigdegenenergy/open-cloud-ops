package budget

import (
	"testing"
	"time"
)

func TestBudgetScopeConstants(t *testing.T) {
	if ScopeAgent != "agent" {
		t.Errorf("expected ScopeAgent to be %q, got %q", "agent", ScopeAgent)
	}
	if ScopeTeam != "team" {
		t.Errorf("expected ScopeTeam to be %q, got %q", "team", ScopeTeam)
	}
	if ScopeUser != "user" {
		t.Errorf("expected ScopeUser to be %q, got %q", "user", ScopeUser)
	}
	if ScopeOrg != "org" {
		t.Errorf("expected ScopeOrg to be %q, got %q", "org", ScopeOrg)
	}
}

func TestDefaultAlertThresholds(t *testing.T) {
	if len(DefaultAlertThresholds) != 4 {
		t.Fatalf("expected 4 default thresholds, got %d", len(DefaultAlertThresholds))
	}

	expectedPercentages := []float64{0.80, 0.90, 0.95, 1.00}
	for i, threshold := range DefaultAlertThresholds {
		if threshold.Percentage != expectedPercentages[i] {
			t.Errorf("threshold %d: expected %.2f, got %.2f", i, expectedPercentages[i], threshold.Percentage)
		}
	}
}

func TestBudgetStruct(t *testing.T) {
	b := Budget{
		ID:        "budget-1",
		Scope:     ScopeAgent,
		EntityID:  "agent-123",
		LimitUSD:  100.00,
		SpentUSD:  45.50,
		Period:    30 * 24 * time.Hour,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if b.ID != "budget-1" {
		t.Errorf("unexpected ID: %s", b.ID)
	}
	if b.Scope != ScopeAgent {
		t.Errorf("unexpected scope: %s", b.Scope)
	}
	if b.LimitUSD != 100.00 {
		t.Errorf("unexpected limit: %f", b.LimitUSD)
	}
	if b.SpentUSD != 45.50 {
		t.Errorf("unexpected spent: %f", b.SpentUSD)
	}
}

func TestBudgetStatusFields(t *testing.T) {
	status := BudgetStatus{
		Scope:        ScopeTeam,
		EntityID:     "team-456",
		LimitUSD:     200.00,
		SpentUSD:     150.00,
		RemainingUSD: 50.00,
		UsagePercent: 75.0,
		IsExhausted:  false,
	}

	if status.RemainingUSD != 50.00 {
		t.Errorf("expected remaining $50.00, got $%.2f", status.RemainingUSD)
	}
	if status.UsagePercent != 75.0 {
		t.Errorf("expected 75%% usage, got %.1f%%", status.UsagePercent)
	}
	if status.IsExhausted {
		t.Error("expected IsExhausted to be false when under limit")
	}
}

func TestBudgetStatusExhausted(t *testing.T) {
	status := BudgetStatus{
		Scope:        ScopeAgent,
		EntityID:     "agent-789",
		LimitUSD:     50.00,
		SpentUSD:     55.00,
		RemainingUSD: 0,
		UsagePercent: 110.0,
		IsExhausted:  true,
	}

	if !status.IsExhausted {
		t.Error("expected IsExhausted to be true when over limit")
	}
	if status.RemainingUSD != 0 {
		t.Errorf("expected remaining $0.00, got $%.2f", status.RemainingUSD)
	}
}

func TestNewEnforcerNilDeps(t *testing.T) {
	// NewEnforcer should handle nil cache (falls back to DB-only mode)
	enforcer := NewEnforcer(nil, nil)
	if enforcer == nil {
		t.Fatal("expected non-nil enforcer")
	}
	if enforcer.alertsSent == nil {
		t.Error("expected alertsSent map to be initialized")
	}
	if len(enforcer.alertThresholds) != 4 {
		t.Errorf("expected 4 default alert thresholds, got %d", len(enforcer.alertThresholds))
	}
}

func TestAlertThresholdLabels(t *testing.T) {
	expectedLabels := []string{"80%", "90%", "95%", "100%"}
	for i, threshold := range DefaultAlertThresholds {
		if threshold.Label != expectedLabels[i] {
			t.Errorf("threshold %d: expected label %q, got %q", i, expectedLabels[i], threshold.Label)
		}
	}
}

func TestBudgetScopeValues(t *testing.T) {
	scopes := []BudgetScope{ScopeAgent, ScopeTeam, ScopeUser, ScopeOrg}
	seen := make(map[BudgetScope]bool)

	for _, scope := range scopes {
		if seen[scope] {
			t.Errorf("duplicate scope value: %s", scope)
		}
		seen[scope] = true

		if scope == "" {
			t.Error("scope should not be empty")
		}
	}
}
