"""
Tests for pkg/cost/calculator.py

Covers currency normalization, cost aggregation, trend analysis,
and linear cost forecasting.
"""

from __future__ import annotations

import os
import sys
from datetime import date, timedelta

import pytest

_PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from pkg.cost.calculator import (
    aggregate_costs,
    calculate_trend,
    forecast_cost,
    normalize_cost,
)


# ---------------------------------------------------------------------------
# normalize_cost
# ---------------------------------------------------------------------------


class TestNormalizeCost:
    def test_same_currency_is_noop(self):
        assert normalize_cost(100.0, "USD", "USD") == 100.0

    def test_eur_to_usd(self):
        result = normalize_cost(100.0, "EUR", "USD")
        # 100 EUR * 1.09 = 109 USD
        assert result == pytest.approx(109.0, rel=1e-4)

    def test_usd_to_eur(self):
        result = normalize_cost(109.0, "USD", "EUR")
        # 109 USD / 1.09 = 100 EUR
        assert result == pytest.approx(100.0, rel=1e-2)

    def test_gbp_to_usd(self):
        result = normalize_cost(50.0, "GBP", "USD")
        # 50 * 1.27 = 63.5
        assert result == pytest.approx(63.5, rel=1e-4)

    def test_unknown_currency_treated_as_usd(self):
        result = normalize_cost(42.0, "XYZ", "USD")
        assert result == pytest.approx(42.0, rel=1e-4)

    def test_case_insensitive(self):
        result = normalize_cost(100.0, "eur", "usd")
        assert result == pytest.approx(109.0, rel=1e-4)

    def test_zero_amount(self):
        assert normalize_cost(0.0, "EUR", "USD") == 0.0

    def test_negative_amount(self):
        result = normalize_cost(-50.0, "USD", "USD")
        assert result == -50.0

    def test_jpy_to_usd(self):
        result = normalize_cost(10000.0, "JPY", "USD")
        # 10000 * 0.0067 = 67.0
        assert result == pytest.approx(67.0, rel=1e-4)


# ---------------------------------------------------------------------------
# aggregate_costs
# ---------------------------------------------------------------------------


class TestAggregateCosts:
    @pytest.fixture()
    def sample_costs(self):
        return [
            {"provider": "aws", "service": "EC2", "cost_usd": 100.0, "region": "us-east-1"},
            {"provider": "aws", "service": "S3", "cost_usd": 30.0, "region": "us-east-1"},
            {"provider": "azure", "service": "VMs", "cost_usd": 80.0, "region": "eastus"},
            {"provider": "aws", "service": "EC2", "cost_usd": 50.0, "region": "eu-west-1"},
            {"provider": "gcp", "service": "Compute", "cost_usd": 60.0, "region": "us-central1"},
        ]

    def test_group_by_provider(self, sample_costs):
        result = aggregate_costs(sample_costs, "provider")
        assert result["aws"] == pytest.approx(180.0)
        assert result["azure"] == pytest.approx(80.0)
        assert result["gcp"] == pytest.approx(60.0)

    def test_group_by_service(self, sample_costs):
        result = aggregate_costs(sample_costs, "service")
        assert result["EC2"] == pytest.approx(150.0)
        assert result["S3"] == pytest.approx(30.0)
        assert result["VMs"] == pytest.approx(80.0)

    def test_group_by_region(self, sample_costs):
        result = aggregate_costs(sample_costs, "region")
        assert result["us-east-1"] == pytest.approx(130.0)

    def test_sorted_descending(self, sample_costs):
        result = aggregate_costs(sample_costs, "provider")
        values = list(result.values())
        assert values == sorted(values, reverse=True)

    def test_empty_costs(self):
        result = aggregate_costs([], "provider")
        assert result == {}

    def test_missing_key_uses_unknown(self):
        costs = [{"cost_usd": 10.0}]
        result = aggregate_costs(costs, "provider")
        assert result["unknown"] == 10.0

    def test_none_key_uses_unknown(self):
        costs = [{"provider": None, "cost_usd": 5.0}]
        result = aggregate_costs(costs, "provider")
        assert "None" in result or "unknown" in result


# ---------------------------------------------------------------------------
# calculate_trend
# ---------------------------------------------------------------------------


class TestCalculateTrend:
    def test_increasing_trend(self):
        today = date.today()
        costs = []
        # Previous period: $10/day for 30 days = $300
        for i in range(60, 30, -1):
            costs.append({
                "cost_usd": 10.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })
        # Current period: $20/day for 30 days = $600
        for i in range(30, 0, -1):
            costs.append({
                "cost_usd": 20.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })

        result = calculate_trend(costs, period_days=30)
        assert result["trend"] == "increasing"
        assert result["change_percent"] > 0
        assert result["current_period_cost"] > result["previous_period_cost"]

    def test_decreasing_trend(self):
        today = date.today()
        costs = []
        # Previous: $20/day
        for i in range(60, 30, -1):
            costs.append({
                "cost_usd": 20.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })
        # Current: $10/day
        for i in range(30, 0, -1):
            costs.append({
                "cost_usd": 10.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })

        result = calculate_trend(costs, period_days=30)
        assert result["trend"] == "decreasing"
        assert result["change_percent"] < 0

    def test_stable_trend(self):
        today = date.today()
        costs = []
        for i in range(60, 0, -1):
            costs.append({
                "cost_usd": 10.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })

        result = calculate_trend(costs, period_days=30)
        assert result["trend"] == "stable"
        assert abs(result["change_percent"]) <= 2.0

    def test_empty_costs(self):
        result = calculate_trend([], period_days=30)
        assert result["current_period_cost"] == 0.0
        assert result["previous_period_cost"] == 0.0
        assert result["trend"] == "stable"

    def test_period_days_preserved(self):
        result = calculate_trend([], period_days=14)
        assert result["period_days"] == 14


# ---------------------------------------------------------------------------
# forecast_cost
# ---------------------------------------------------------------------------


class TestForecastCost:
    def test_linear_forecast(self):
        today = date.today()
        costs = []
        # Constant $10/day for 30 days
        for i in range(30, 0, -1):
            costs.append({
                "cost_usd": 10.0,
                "date": (today - timedelta(days=i)).isoformat(),
            })

        result = forecast_cost(costs, forecast_days=30)

        assert result["forecast_days"] == 30
        assert len(result["daily_forecasts"]) == 30
        # With constant data, forecast should be close to $10/day
        assert result["average_daily"] == pytest.approx(10.0, abs=2.0)
        assert result["total_forecast"] == pytest.approx(300.0, abs=60.0)

    def test_empty_history(self):
        result = forecast_cost([], forecast_days=30)
        assert result["daily_forecasts"] == []
        assert result["total_forecast"] == 0.0
        assert result["average_daily"] == 0.0

    def test_single_day_history(self):
        today = date.today()
        costs = [{"cost_usd": 50.0, "date": today.isoformat()}]
        result = forecast_cost(costs, forecast_days=7)
        assert len(result["daily_forecasts"]) == 7
        # Single-point regression uses intercept = 50, slope = 0
        assert result["average_daily"] == pytest.approx(50.0, abs=1.0)

    def test_forecast_dates_are_sequential(self):
        today = date.today()
        costs = [
            {"cost_usd": 10.0, "date": (today - timedelta(days=i)).isoformat()}
            for i in range(10, 0, -1)
        ]
        result = forecast_cost(costs, forecast_days=5)
        dates = [fc["date"] for fc in result["daily_forecasts"]]
        assert len(dates) == 5
        # Each date should be later than the previous
        for i in range(1, len(dates)):
            assert dates[i] > dates[i - 1]

    def test_increasing_trend_forecast(self):
        today = date.today()
        costs = []
        # Linearly increasing: day 1 = $1, day 2 = $2, ...
        for i in range(30, 0, -1):
            d = today - timedelta(days=i)
            costs.append({"cost_usd": float(31 - i), "date": d.isoformat()})

        result = forecast_cost(costs, forecast_days=10)
        # Forecast should continue the upward trend
        daily_costs = [fc["cost"] for fc in result["daily_forecasts"]]
        assert daily_costs[-1] > daily_costs[0]

    def test_forecast_non_negative(self):
        today = date.today()
        # Rapidly decreasing costs
        costs = []
        for i in range(30, 0, -1):
            costs.append({
                "cost_usd": max(100.0 - i * 10, 0),
                "date": (today - timedelta(days=i)).isoformat(),
            })

        result = forecast_cost(costs, forecast_days=60)
        for fc in result["daily_forecasts"]:
            assert fc["cost"] >= 0, "Forecast should never be negative"
