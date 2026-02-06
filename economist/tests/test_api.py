"""
Tests for api/routes.py

Uses FastAPI TestClient with an in-memory SQLite database so tests
run without external services.
"""

from __future__ import annotations

import os
import sys
import uuid
from datetime import date, timedelta

import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

_PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from api.routes import configure_routes, router as api_router
from internal.ingestion.collector import CostCollector
from internal.optimizer.engine import OptimizationEngine
from internal.policy.engine import PolicyEngine
from pkg.database import (
    Base,
    CloudCost,
    GovernancePolicy,
    OptimizationRecommendation,
    PolicyViolation,
    get_session,
)

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def test_app():
    """Create a FastAPI app wired to an in-memory SQLite database."""
    from fastapi import FastAPI

    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    TestSession = sessionmaker(bind=engine)

    def _override_get_session():
        session = TestSession()
        try:
            yield session
        finally:
            session.close()

    app = FastAPI()
    app.include_router(api_router)

    # Dependency override
    app.dependency_overrides[get_session] = _override_get_session

    # Wire engines
    collector = CostCollector()
    optimizer = OptimizationEngine()
    policy_engine = PolicyEngine()
    configure_routes(collector, optimizer, policy_engine)

    yield app, TestSession

    Base.metadata.drop_all(engine)
    engine.dispose()


@pytest.fixture()
def client(test_app):
    app, _ = test_app
    return TestClient(app)


@pytest.fixture()
def session(test_app):
    _, TestSession = test_app
    s = TestSession()
    yield s
    s.close()


def _seed_costs(session, count: int = 30, daily_cost: float = 10.0):
    """Insert sample cost rows spanning the last *count* days."""
    today = date.today()
    for i in range(count):
        session.add(
            CloudCost(
                id=uuid.uuid4(),
                provider="aws",
                service="EC2",
                resource_id=f"i-{i:04d}",
                resource_name=f"instance-{i}",
                cost_usd=daily_cost,
                currency="USD",
                usage_quantity=100.0,
                usage_unit="hours",
                region="us-east-1",
                account_id="123456789012",
                tags={"env": "prod"},
                date=today - timedelta(days=i),
            )
        )
    session.commit()


def _seed_recommendation(session, status: str = "open", savings: float = 50.0):
    rec = OptimizationRecommendation(
        id=uuid.uuid4(),
        provider="aws",
        resource_id="i-over",
        resource_type="ec2:instance",
        recommendation_type="rightsizing",
        title="Rightsize i-over",
        description="Instance is over-provisioned",
        estimated_monthly_savings=savings,
        confidence=0.8,
        status=status,
    )
    session.add(rec)
    session.commit()
    return rec


def _seed_policy(session) -> GovernancePolicy:
    policy = GovernancePolicy(
        id=uuid.uuid4(),
        name="tag-policy",
        description="All resources must have env tag",
        policy_type="tag_compliance",
        rules={"required_tags": ["env"]},
        severity="warning",
        enabled=True,
    )
    session.add(policy)
    session.commit()
    return policy


# ---------------------------------------------------------------------------
# Cost endpoints
# ---------------------------------------------------------------------------


class TestCostSummary:
    def test_empty_db(self, client):
        resp = client.get("/api/v1/costs/summary")
        assert resp.status_code == 200
        data = resp.json()
        assert data["total_cost_usd"] == 0.0
        assert data["record_count"] == 0

    def test_with_data(self, client, session):
        _seed_costs(session, count=10, daily_cost=25.0)
        resp = client.get("/api/v1/costs/summary")
        assert resp.status_code == 200
        data = resp.json()
        assert data["total_cost_usd"] == pytest.approx(250.0)
        assert data["record_count"] == 10

    def test_filter_by_provider(self, client, session):
        _seed_costs(session, count=5)
        resp = client.get("/api/v1/costs/summary?provider=azure")
        data = resp.json()
        assert data["record_count"] == 0

    def test_filter_by_date(self, client, session):
        _seed_costs(session, count=30)
        start = (date.today() - timedelta(days=5)).isoformat()
        end = date.today().isoformat()
        resp = client.get(f"/api/v1/costs/summary?start_date={start}&end_date={end}")
        data = resp.json()
        assert data["record_count"] <= 6  # at most 6 days of data


class TestCostBreakdown:
    def test_by_service(self, client, session):
        _seed_costs(session, count=5)
        resp = client.get("/api/v1/costs/breakdown?dimension=service")
        assert resp.status_code == 200
        data = resp.json()
        assert data["dimension"] == "service"
        assert "EC2" in data["breakdown"]

    def test_by_provider(self, client, session):
        _seed_costs(session, count=5)
        resp = client.get("/api/v1/costs/breakdown?dimension=provider")
        data = resp.json()
        assert "aws" in data["breakdown"]

    def test_invalid_dimension(self, client):
        resp = client.get("/api/v1/costs/breakdown?dimension=invalid")
        assert resp.status_code == 400


class TestCostTrend:
    def test_trend_endpoint(self, client, session):
        _seed_costs(session, count=60)
        resp = client.get("/api/v1/costs/trend?period_days=30")
        assert resp.status_code == 200
        data = resp.json()
        assert "trend" in data
        assert data["period_days"] == 30


class TestCostForecast:
    def test_forecast_endpoint(self, client, session):
        _seed_costs(session, count=60)
        resp = client.get("/api/v1/costs/forecast?forecast_days=14&history_days=60")
        assert resp.status_code == 200
        data = resp.json()
        assert data["forecast_days"] == 14
        assert len(data["daily_forecasts"]) == 14


class TestCostCollect:
    def test_collect_no_providers(self, client):
        """With no real providers registered, should collect 0 records."""
        resp = client.post("/api/v1/costs/collect", json={})
        assert resp.status_code == 200
        data = resp.json()
        assert data["records_collected"] == 0


# ---------------------------------------------------------------------------
# Recommendation endpoints
# ---------------------------------------------------------------------------


class TestRecommendations:
    def test_list_empty(self, client):
        resp = client.get("/api/v1/recommendations")
        assert resp.status_code == 200
        assert resp.json() == []

    def test_list_with_data(self, client, session):
        _seed_recommendation(session, status="open", savings=100.0)
        resp = client.get("/api/v1/recommendations")
        data = resp.json()
        assert len(data) == 1
        assert data[0]["status"] == "open"

    def test_filter_by_status(self, client, session):
        _seed_recommendation(session, status="open")
        _seed_recommendation(session, status="resolved")
        # Need unique resource_ids; override by just seeding twice
        resp = client.get("/api/v1/recommendations?status=open")
        data = resp.json()
        for item in data:
            assert item["status"] == "open"

    def test_resolve_recommendation(self, client, session):
        rec = _seed_recommendation(session)
        resp = client.post(f"/api/v1/recommendations/{rec.id}/resolve")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "resolved"
        assert data["resolved_at"] is not None

    def test_resolve_not_found(self, client):
        fake_id = str(uuid.uuid4())
        resp = client.post(f"/api/v1/recommendations/{fake_id}/resolve")
        assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Governance endpoints
# ---------------------------------------------------------------------------


class TestGovernancePolicies:
    def test_list_empty(self, client):
        resp = client.get("/api/v1/governance/policies")
        assert resp.status_code == 200
        assert resp.json() == []

    def test_create_policy(self, client):
        body = {
            "name": "budget-test",
            "description": "test budget",
            "policy_type": "budget_limit",
            "rules": {"monthly_budget_usd": 5000, "scope": "all"},
            "severity": "critical",
        }
        resp = client.post("/api/v1/governance/policies", json=body)
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "budget-test"
        assert data["policy_type"] == "budget_limit"

    def test_create_invalid_type(self, client):
        body = {
            "name": "bad",
            "policy_type": "not_real",
            "rules": {},
        }
        resp = client.post("/api/v1/governance/policies", json=body)
        assert resp.status_code == 400

    def test_update_policy(self, client, session):
        policy = _seed_policy(session)
        resp = client.put(
            f"/api/v1/governance/policies/{policy.id}",
            json={"severity": "critical", "enabled": False},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["severity"] == "critical"
        assert data["enabled"] is False

    def test_update_not_found(self, client):
        fake_id = str(uuid.uuid4())
        resp = client.put(
            f"/api/v1/governance/policies/{fake_id}",
            json={"severity": "info"},
        )
        assert resp.status_code == 404


class TestGovernanceViolations:
    def test_list_empty(self, client):
        resp = client.get("/api/v1/governance/violations")
        assert resp.status_code == 200
        assert resp.json() == []

    def test_list_with_violations(self, client, session):
        policy = _seed_policy(session)
        session.add(
            PolicyViolation(
                id=uuid.uuid4(),
                policy_id=policy.id,
                resource_id="r-bad",
                provider="aws",
                description="missing tags",
                severity="warning",
            )
        )
        session.commit()

        resp = client.get("/api/v1/governance/violations")
        data = resp.json()
        assert len(data) == 1
        assert data[0]["resource_id"] == "r-bad"

    def test_filter_severity(self, client, session):
        policy = _seed_policy(session)
        session.add(
            PolicyViolation(
                id=uuid.uuid4(),
                policy_id=policy.id,
                resource_id="r-1",
                provider="aws",
                description="test",
                severity="critical",
            )
        )
        session.add(
            PolicyViolation(
                id=uuid.uuid4(),
                policy_id=policy.id,
                resource_id="r-2",
                provider="aws",
                description="test",
                severity="warning",
            )
        )
        session.commit()

        resp = client.get("/api/v1/governance/violations?severity=critical")
        data = resp.json()
        assert all(v["severity"] == "critical" for v in data)


# ---------------------------------------------------------------------------
# Dashboard
# ---------------------------------------------------------------------------


class TestDashboard:
    def test_overview_empty(self, client):
        resp = client.get("/api/v1/dashboard/overview")
        assert resp.status_code == 200
        data = resp.json()
        assert data["total_cost_usd"] == 0.0
        assert data["recommendation_count"] == 0

    def test_overview_with_data(self, client, session):
        _seed_costs(session, count=30, daily_cost=20.0)
        _seed_recommendation(session, savings=75.0)
        _seed_policy(session)

        resp = client.get("/api/v1/dashboard/overview")
        assert resp.status_code == 200
        data = resp.json()
        assert data["total_cost_usd"] > 0
        assert data["recommendation_count"] == 1
        assert data["total_potential_savings"] == pytest.approx(75.0)
        assert data["active_policies"] >= 1
