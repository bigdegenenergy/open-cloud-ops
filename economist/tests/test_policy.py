"""
Tests for internal/policy/engine.py

Covers all four built-in policy types: budget_limit, tag_compliance,
region_restriction, and service_allowlist.
"""

from __future__ import annotations

import os
import sys
import uuid
from datetime import date, timedelta

import pytest

_PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from internal.policy.engine import PolicyEngine
from pkg.database import GovernancePolicy


@pytest.fixture()
def engine():
    return PolicyEngine()


def _make_policy(
    policy_type: str,
    rules: dict,
    severity: str = "warning",
    name: str | None = None,
) -> GovernancePolicy:
    """Create a GovernancePolicy instance in memory (no DB)."""
    return GovernancePolicy(
        id=uuid.uuid4(),
        name=name or f"test-{policy_type}",
        description="test policy",
        policy_type=policy_type,
        rules=rules,
        severity=severity,
        enabled=True,
    )


# ---------------------------------------------------------------------------
# Policy creation
# ---------------------------------------------------------------------------


class TestCreatePolicy:
    def test_create_valid_policy(self, engine):
        policy = engine.create_policy(
            name="test-budget",
            description="budget test",
            policy_type="budget_limit",
            rules={"monthly_budget_usd": 1000},
            severity="critical",
        )
        assert policy.name == "test-budget"
        assert policy.policy_type == "budget_limit"
        assert policy.rules == {"monthly_budget_usd": 1000}

    def test_create_invalid_type_raises(self, engine):
        with pytest.raises(ValueError, match="Unknown policy_type"):
            engine.create_policy(
                name="bad",
                description="",
                policy_type="nonexistent",
                rules={},
            )

    def test_create_with_db(self, engine, db_session):
        policy = engine.create_policy(
            name="db-policy",
            description="persisted",
            policy_type="tag_compliance",
            rules={"required_tags": ["env"]},
            db=db_session,
        )
        # Should be persisted
        from pkg.database import GovernancePolicy as GP

        row = db_session.query(GP).filter(GP.name == "db-policy").first()
        assert row is not None
        assert row.id == policy.id


# ---------------------------------------------------------------------------
# budget_limit
# ---------------------------------------------------------------------------


class TestBudgetLimit:
    def test_violation_when_over_budget(self, engine):
        policy = _make_policy(
            "budget_limit",
            {"monthly_budget_usd": 500, "scope": "all"},
            severity="critical",
        )
        today = date.today()
        costs = [
            {"cost_usd": 200.0, "date": today.isoformat(), "provider": "aws"},
            {"cost_usd": 400.0, "date": today.isoformat(), "provider": "azure"},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 1
        assert "exceeds budget" in violations[0].description

    def test_no_violation_under_budget(self, engine):
        policy = _make_policy(
            "budget_limit",
            {"monthly_budget_usd": 10000, "scope": "all"},
        )
        today = date.today()
        costs = [
            {"cost_usd": 50.0, "date": today.isoformat(), "provider": "aws"},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 0

    def test_scope_filters_provider(self, engine):
        policy = _make_policy(
            "budget_limit",
            {"monthly_budget_usd": 100, "scope": "aws"},
        )
        today = date.today()
        costs = [
            {"cost_usd": 200.0, "date": today.isoformat(), "provider": "aws"},
            {"cost_usd": 9999.0, "date": today.isoformat(), "provider": "azure"},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 1
        assert "$200" in violations[0].description

    def test_ignores_old_costs(self, engine):
        policy = _make_policy(
            "budget_limit",
            {"monthly_budget_usd": 100, "scope": "all"},
        )
        old = date.today().replace(day=1) - timedelta(days=1)
        costs = [
            {"cost_usd": 9999.0, "date": old.isoformat(), "provider": "aws"},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 0


# ---------------------------------------------------------------------------
# tag_compliance
# ---------------------------------------------------------------------------


class TestTagCompliance:
    def test_violation_for_missing_tags(self, engine):
        policy = _make_policy(
            "tag_compliance",
            {"required_tags": ["environment", "owner"]},
        )
        resources = [
            {"resource_id": "r-1", "provider": "aws", "tags": {"environment": "prod"}},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 1
        assert "owner" in violations[0].description

    def test_no_violation_when_compliant(self, engine):
        policy = _make_policy(
            "tag_compliance",
            {"required_tags": ["environment", "owner"]},
        )
        resources = [
            {
                "resource_id": "r-2",
                "provider": "aws",
                "tags": {"environment": "prod", "owner": "team-a"},
            },
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 0

    def test_no_tags_at_all(self, engine):
        policy = _make_policy(
            "tag_compliance",
            {"required_tags": ["env"]},
        )
        resources = [
            {"resource_id": "r-3", "provider": "aws", "tags": None},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 1

    def test_empty_required_tags_no_violations(self, engine):
        policy = _make_policy(
            "tag_compliance",
            {"required_tags": []},
        )
        resources = [
            {"resource_id": "r-4", "provider": "aws", "tags": {}},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 0


# ---------------------------------------------------------------------------
# region_restriction
# ---------------------------------------------------------------------------


class TestRegionRestriction:
    def test_violation_for_disallowed_region(self, engine):
        policy = _make_policy(
            "region_restriction",
            {"allowed_regions": ["us-east-1", "eu-west-1"]},
        )
        resources = [
            {"resource_id": "r-1", "provider": "aws", "region": "ap-southeast-1"},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 1
        assert "ap-southeast-1" in violations[0].description

    def test_no_violation_in_allowed_region(self, engine):
        policy = _make_policy(
            "region_restriction",
            {"allowed_regions": ["us-east-1"]},
        )
        resources = [
            {"resource_id": "r-2", "provider": "aws", "region": "us-east-1"},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 0

    def test_multiple_violations(self, engine):
        policy = _make_policy(
            "region_restriction",
            {"allowed_regions": ["us-east-1"]},
        )
        resources = [
            {"resource_id": "r-1", "provider": "aws", "region": "ap-southeast-1"},
            {"resource_id": "r-2", "provider": "aws", "region": "eu-central-1"},
            {"resource_id": "r-3", "provider": "aws", "region": "us-east-1"},
        ]
        violations = engine.detect_violations(resources, [], [policy])
        assert len(violations) == 2


# ---------------------------------------------------------------------------
# service_allowlist
# ---------------------------------------------------------------------------


class TestServiceAllowlist:
    def test_violation_for_disallowed_service(self, engine):
        policy = _make_policy(
            "service_allowlist",
            {"allowed_services": ["EC2", "S3"]},
        )
        costs = [
            {"service": "EC2", "provider": "aws", "cost_usd": 10, "date": date.today().isoformat()},
            {"service": "Redshift", "provider": "aws", "cost_usd": 20, "date": date.today().isoformat()},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 1
        assert "Redshift" in violations[0].description

    def test_no_violation_when_all_allowed(self, engine):
        policy = _make_policy(
            "service_allowlist",
            {"allowed_services": ["EC2", "S3"]},
        )
        costs = [
            {"service": "EC2", "provider": "aws", "cost_usd": 10, "date": date.today().isoformat()},
            {"service": "S3", "provider": "aws", "cost_usd": 5, "date": date.today().isoformat()},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 0

    def test_multiple_disallowed_services(self, engine):
        policy = _make_policy(
            "service_allowlist",
            {"allowed_services": ["EC2"]},
        )
        costs = [
            {"service": "S3", "provider": "aws", "cost_usd": 5, "date": date.today().isoformat()},
            {"service": "Redshift", "provider": "aws", "cost_usd": 20, "date": date.today().isoformat()},
        ]
        violations = engine.detect_violations([], costs, [policy])
        assert len(violations) == 2


# ---------------------------------------------------------------------------
# evaluate_policies (DB-backed)
# ---------------------------------------------------------------------------


class TestEvaluatePolicies:
    def test_evaluate_stores_violations(self, engine, db_session):
        """Create a policy in DB, evaluate it, check violations are persisted."""
        engine.create_policy(
            name="tag-check",
            description="enforce tags",
            policy_type="tag_compliance",
            rules={"required_tags": ["env"]},
            severity="warning",
            db=db_session,
        )

        resources = [
            {"resource_id": "r-no-tags", "provider": "aws", "tags": {}},
        ]

        violations = engine.evaluate_policies(resources, [], db_session)
        assert len(violations) == 1

        # Verify persisted
        from pkg.database import PolicyViolation as PV

        stored = db_session.query(PV).all()
        assert len(stored) == 1
        assert stored[0].resource_id == "r-no-tags"

    def test_evaluate_no_policies(self, engine, db_session):
        violations = engine.evaluate_policies([], [], db_session)
        assert violations == []
