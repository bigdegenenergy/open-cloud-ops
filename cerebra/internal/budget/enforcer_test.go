package budget

import (
	"testing"
)

func TestNewEnforcer_NilRedis(t *testing.T) {
	e := NewEnforcer(nil, true)
	if e == nil {
		t.Fatal("expected non-nil enforcer")
	}
}

func TestCheckBudget_NilRedis_AllowsAll(t *testing.T) {
	e := NewEnforcer(nil, true)

	allowed, err := e.CheckBudget(ScopeAgent, "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed with nil Redis")
	}
}

func TestCheckBudget_NilRedis_FailClosed(t *testing.T) {
	e := NewEnforcer(nil, false)

	allowed, err := e.CheckBudget(ScopeAgent, "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected request to be blocked with nil Redis and fail-closed")
	}
}

func TestRecordSpend_NilRedis_NoError(t *testing.T) {
	e := NewEnforcer(nil, true)

	err := e.RecordSpend(ScopeAgent, "test-agent", 5.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordSpend_ZeroCost_NoError(t *testing.T) {
	e := NewEnforcer(nil, true)

	err := e.RecordSpend(ScopeAgent, "test-agent", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetBudget_NilRedis_NoError(t *testing.T) {
	e := NewEnforcer(nil, true)

	err := e.SetBudget(ScopeAgent, "test-agent", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSpent_NilRedis_ReturnsZero(t *testing.T) {
	e := NewEnforcer(nil, true)

	spent, err := e.GetSpent(ScopeAgent, "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spent != 0 {
		t.Errorf("expected 0 spent, got %f", spent)
	}
}

func TestResetSpend_NilRedis_NoError(t *testing.T) {
	e := NewEnforcer(nil, true)

	err := e.ResetSpend(ScopeAgent, "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBudgetScopes(t *testing.T) {
	if ScopeAgent != "agent" {
		t.Errorf("expected ScopeAgent = agent, got %s", ScopeAgent)
	}
	if ScopeTeam != "team" {
		t.Errorf("expected ScopeTeam = team, got %s", ScopeTeam)
	}
	if ScopeUser != "user" {
		t.Errorf("expected ScopeUser = user, got %s", ScopeUser)
	}
	if ScopeOrg != "org" {
		t.Errorf("expected ScopeOrg = org, got %s", ScopeOrg)
	}
}
