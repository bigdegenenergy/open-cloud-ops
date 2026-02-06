"""
Tests for internal/optimizer/engine.py

Covers idle-resource detection, right-sizing, reserved-capacity
recommendations, and spot opportunity analysis.
"""

from __future__ import annotations

import os
import sys
from datetime import date, timedelta

import pytest

_PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from internal.optimizer.engine import OptimizationEngine


@pytest.fixture()
def engine():
    return OptimizationEngine()


# ---------------------------------------------------------------------------
# analyze_idle_resources
# ---------------------------------------------------------------------------


class TestIdleResources:
    def _make_costs(self, rid: str, daily_cost: float, daily_usage: float, days: int):
        today = date.today()
        return [
            {
                "resource_id": rid,
                "provider": "aws",
                "service": "EC2",
                "cost_usd": daily_cost,
                "usage_quantity": daily_usage,
                "date": (today - timedelta(days=i)).isoformat(),
            }
            for i in range(1, days + 1)
        ]

    def test_detects_idle_low_cost_low_usage(self, engine):
        costs = self._make_costs("i-idle1", 0.50, 0.01, 30)
        recs = engine.analyze_idle_resources(costs)
        assert len(recs) == 1
        assert recs[0].recommendation_type == "idle_resource"
        assert recs[0].resource_id == "i-idle1"

    def test_ignores_active_resources(self, engine):
        costs = self._make_costs("i-active", 50.0, 1000.0, 30)
        recs = engine.analyze_idle_resources(costs)
        assert len(recs) == 0

    def test_ignores_zero_cost(self, engine):
        costs = self._make_costs("i-free", 0.0, 0.0, 30)
        recs = engine.analyze_idle_resources(costs)
        assert len(recs) == 0

    def test_savings_estimate_positive(self, engine):
        costs = self._make_costs("i-idle2", 0.80, 0.001, 30)
        recs = engine.analyze_idle_resources(costs)
        assert len(recs) == 1
        assert recs[0].estimated_monthly_savings > 0

    def test_multiple_idle_resources(self, engine):
        costs = (
            self._make_costs("i-idle-a", 0.30, 0.01, 30)
            + self._make_costs("i-idle-b", 0.20, 0.001, 30)
        )
        recs = engine.analyze_idle_resources(costs)
        assert len(recs) == 2


# ---------------------------------------------------------------------------
# analyze_rightsizing
# ---------------------------------------------------------------------------


class TestRightsizing:
    def test_detects_over_provisioned_cpu(self, engine):
        resources = [
            {
                "resource_id": "i-over",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "instance_type": "m5.4xlarge",
                "cpu_avg": 0.05,
                "memory_avg": 0.50,
            }
        ]
        recs = engine.analyze_rightsizing(resources)
        assert len(recs) == 1
        assert recs[0].recommendation_type == "rightsizing"

    def test_detects_over_provisioned_memory(self, engine):
        resources = [
            {
                "resource_id": "i-mem",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "instance_type": "r5.2xlarge",
                "cpu_avg": 0.50,
                "memory_avg": 0.10,
            }
        ]
        recs = engine.analyze_rightsizing(resources)
        assert len(recs) == 1

    def test_ignores_well_utilised(self, engine):
        resources = [
            {
                "resource_id": "i-ok",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "instance_type": "m5.large",
                "cpu_avg": 0.60,
                "memory_avg": 0.55,
            }
        ]
        recs = engine.analyze_rightsizing(resources)
        assert len(recs) == 0

    def test_skips_resources_without_metrics(self, engine):
        resources = [
            {
                "resource_id": "i-nometrics",
                "resource_type": "ec2:instance",
                "provider": "aws",
            }
        ]
        recs = engine.analyze_rightsizing(resources)
        assert len(recs) == 0

    def test_savings_positive(self, engine):
        resources = [
            {
                "resource_id": "i-over2",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "instance_type": "c5.9xlarge",
                "cpu_avg": 0.03,
                "memory_avg": 0.04,
            }
        ]
        recs = engine.analyze_rightsizing(resources)
        assert recs[0].estimated_monthly_savings > 0


# ---------------------------------------------------------------------------
# analyze_reserved_capacity
# ---------------------------------------------------------------------------


class TestReservedCapacity:
    def _make_steady_costs(self, service: str, daily_cost: float, days: int):
        today = date.today()
        return [
            {
                "resource_id": f"aws:{service}",
                "provider": "aws",
                "service": service,
                "cost_usd": daily_cost,
                "date": (today - timedelta(days=i)).isoformat(),
            }
            for i in range(1, days + 1)
        ]

    def test_recommends_ri_for_steady_high_spend(self, engine):
        costs = self._make_steady_costs("EC2", 20.0, 55)
        recs = engine.analyze_reserved_capacity(costs)
        assert len(recs) == 1
        assert recs[0].recommendation_type == "reserved_capacity"

    def test_no_rec_for_low_spend(self, engine):
        costs = self._make_steady_costs("CloudWatch", 1.0, 55)
        recs = engine.analyze_reserved_capacity(costs)
        assert len(recs) == 0

    def test_no_rec_for_insufficient_data(self, engine):
        costs = self._make_steady_costs("EC2", 50.0, 10)
        recs = engine.analyze_reserved_capacity(costs)
        assert len(recs) == 0

    def test_savings_around_thirty_percent(self, engine):
        costs = self._make_steady_costs("EC2", 100.0, 55)
        recs = engine.analyze_reserved_capacity(costs)
        assert len(recs) == 1
        # 100 * 30 * 0.3 = 900
        assert recs[0].estimated_monthly_savings == pytest.approx(900.0, rel=0.05)


# ---------------------------------------------------------------------------
# analyze_spot_opportunities
# ---------------------------------------------------------------------------


class TestSpotOpportunities:
    def test_ec2_instance_eligible(self, engine):
        resources = [
            {
                "resource_id": "i-spot1",
                "resource_type": "ec2:instance",
                "provider": "aws",
            }
        ]
        recs = engine.analyze_spot_opportunities(resources)
        assert len(recs) == 1
        assert recs[0].recommendation_type == "spot_instance"

    def test_already_spot_skipped(self, engine):
        resources = [
            {
                "resource_id": "i-spot2",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "lifecycle": "spot",
            }
        ]
        recs = engine.analyze_spot_opportunities(resources)
        assert len(recs) == 0

    def test_ineligible_type_skipped(self, engine):
        resources = [
            {
                "resource_id": "s3-bucket",
                "resource_type": "s3:bucket",
                "provider": "aws",
            }
        ]
        recs = engine.analyze_spot_opportunities(resources)
        assert len(recs) == 0

    def test_compute_instance_eligible(self, engine):
        resources = [
            {
                "resource_id": "gcp-vm-1",
                "resource_type": "compute:instance",
                "provider": "gcp",
            }
        ]
        recs = engine.analyze_spot_opportunities(resources)
        assert len(recs) == 1


# ---------------------------------------------------------------------------
# generate_recommendations (orchestrator)
# ---------------------------------------------------------------------------


class TestGenerateRecommendations:
    def test_orchestrates_all_analyses(self, engine):
        today = date.today()
        costs = [
            {
                "resource_id": "i-idle",
                "provider": "aws",
                "service": "EC2",
                "cost_usd": 0.50,
                "usage_quantity": 0.01,
                "date": (today - timedelta(days=i)).isoformat(),
            }
            for i in range(1, 31)
        ]
        resources = [
            {
                "resource_id": "i-over",
                "resource_type": "ec2:instance",
                "provider": "aws",
                "cpu_avg": 0.05,
                "memory_avg": 0.50,
                "instance_type": "m5.4xlarge",
            },
            {
                "resource_id": "i-spot-candidate",
                "resource_type": "ec2:instance",
                "provider": "aws",
            },
        ]

        recs = engine.generate_recommendations(costs, resources)
        types = {r.recommendation_type for r in recs}
        assert "idle_resource" in types
        assert "rightsizing" in types
        assert "spot_instance" in types

    def test_deduplication(self, engine):
        today = date.today()
        costs = [
            {
                "resource_id": "i-dup",
                "provider": "aws",
                "service": "EC2",
                "cost_usd": 0.50,
                "usage_quantity": 0.01,
                "date": (today - timedelta(days=i)).isoformat(),
            }
            for i in range(1, 31)
        ] * 2  # duplicate records

        recs = engine.generate_recommendations(costs, [])
        # Duplicates of (provider, resource_id, type) should be merged
        idle_recs = [r for r in recs if r.resource_id == "i-dup"]
        assert len(idle_recs) == 1

    def test_empty_data(self, engine):
        recs = engine.generate_recommendations([], [])
        assert recs == []
