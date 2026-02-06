"""
Cost calculation utilities for the Economist module.

Provides currency normalization, cost aggregation, trend analysis,
and simple linear forecasting.
"""

from __future__ import annotations

import logging
from collections import defaultdict
from datetime import date, datetime, timedelta
from typing import Any

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Static exchange rates (relative to 1 USD).  In a production system these
# would be fetched from a live forex API, but we keep a static table here
# so the module works without external dependencies.
# ---------------------------------------------------------------------------
_EXCHANGE_RATES_TO_USD: dict[str, float] = {
    "USD": 1.0,
    "EUR": 1.09,
    "GBP": 1.27,
    "JPY": 0.0067,
    "CAD": 0.74,
    "AUD": 0.65,
    "CHF": 1.13,
    "CNY": 0.14,
    "INR": 0.012,
    "BRL": 0.20,
    "KRW": 0.00075,
    "SGD": 0.75,
    "HKD": 0.13,
    "SEK": 0.096,
    "NOK": 0.094,
    "MXN": 0.058,
}


def normalize_cost(
    amount: float,
    currency: str,
    target_currency: str = "USD",
) -> float:
    """Convert *amount* from *currency* to *target_currency*.

    Uses a static exchange-rate table.  Unknown currencies are treated as
    1:1 with USD and a warning is logged.

    Parameters
    ----------
    amount:
        The monetary amount to convert.
    currency:
        ISO 4217 code of the source currency (e.g. ``"EUR"``).
    target_currency:
        ISO 4217 code of the destination currency.  Defaults to ``"USD"``.

    Returns
    -------
    float
        The converted amount rounded to 6 decimal places.
    """
    currency = currency.upper()
    target_currency = target_currency.upper()

    if currency == target_currency:
        return round(amount, 6)

    # Convert source -> USD
    src_rate = _EXCHANGE_RATES_TO_USD.get(currency)
    if src_rate is None:
        logger.warning("Unknown currency %s; treating as 1:1 with USD", currency)
        src_rate = 1.0

    amount_usd = amount * src_rate

    # Convert USD -> target
    tgt_rate = _EXCHANGE_RATES_TO_USD.get(target_currency)
    if tgt_rate is None:
        logger.warning(
            "Unknown target currency %s; treating as 1:1 with USD",
            target_currency,
        )
        tgt_rate = 1.0

    return round(amount_usd / tgt_rate, 6)


def aggregate_costs(
    costs: list[dict[str, Any]],
    group_by: str = "provider",
) -> dict[str, float]:
    """Aggregate a list of cost dicts by a given dimension.

    Parameters
    ----------
    costs:
        Each dict must have a ``"cost_usd"`` key and the key specified
        by *group_by* (e.g. ``"provider"``, ``"service"``, ``"region"``,
        ``"account_id"``).
    group_by:
        The key to group on.  Common values: ``"provider"``,
        ``"service"``, ``"region"``, ``"account_id"``.

    Returns
    -------
    dict[str, float]
        Mapping of dimension value -> total cost in USD, sorted
        descending by cost.
    """
    totals: dict[str, float] = defaultdict(float)
    for cost in costs:
        key = cost.get(group_by, "unknown")
        if key is None:
            key = "unknown"
        totals[str(key)] += cost.get("cost_usd", 0.0)

    # Sort descending by cost
    return dict(sorted(totals.items(), key=lambda kv: kv[1], reverse=True))


def calculate_trend(
    costs: list[dict[str, Any]],
    period_days: int = 30,
) -> dict[str, Any]:
    """Calculate cost trend over two consecutive periods.

    Compares the total cost of the most recent *period_days* with the
    preceding *period_days* and returns the percentage change.

    Parameters
    ----------
    costs:
        Each dict must contain ``"cost_usd"`` (float) and ``"date"``
        (``date`` or ISO-format string).
    period_days:
        Length of each comparison period in days.

    Returns
    -------
    dict
        Keys: ``current_period_cost``, ``previous_period_cost``,
        ``change_percent``, ``trend`` (``"increasing"`` /
        ``"decreasing"`` / ``"stable"``), ``period_days``.
    """
    today = date.today()
    current_start = today - timedelta(days=period_days)
    previous_start = current_start - timedelta(days=period_days)

    current_total = 0.0
    previous_total = 0.0

    for cost in costs:
        cost_date = _parse_date(cost.get("date"))
        if cost_date is None:
            continue
        amount = cost.get("cost_usd", 0.0)

        if current_start <= cost_date <= today:
            current_total += amount
        elif previous_start <= cost_date < current_start:
            previous_total += amount

    if previous_total > 0:
        change_pct = ((current_total - previous_total) / previous_total) * 100.0
    elif current_total > 0:
        change_pct = 100.0
    else:
        change_pct = 0.0

    if change_pct > 2.0:
        trend = "increasing"
    elif change_pct < -2.0:
        trend = "decreasing"
    else:
        trend = "stable"

    return {
        "current_period_cost": round(current_total, 2),
        "previous_period_cost": round(previous_total, 2),
        "change_percent": round(change_pct, 2),
        "trend": trend,
        "period_days": period_days,
    }


def forecast_cost(
    historical_costs: list[dict[str, Any]],
    forecast_days: int = 30,
) -> dict[str, Any]:
    """Produce a simple linear cost forecast.

    Fits a least-squares line to daily cost totals and projects
    *forecast_days* into the future.

    Parameters
    ----------
    historical_costs:
        Each dict must contain ``"cost_usd"`` and ``"date"``.
    forecast_days:
        Number of days to forecast.

    Returns
    -------
    dict
        Keys: ``daily_forecasts`` (list of ``{date, cost}``),
        ``total_forecast``, ``average_daily``, ``forecast_days``.
    """
    # Aggregate costs by day
    daily: dict[date, float] = defaultdict(float)
    for cost in historical_costs:
        d = _parse_date(cost.get("date"))
        if d is None:
            continue
        daily[d] += cost.get("cost_usd", 0.0)

    if not daily:
        return {
            "daily_forecasts": [],
            "total_forecast": 0.0,
            "average_daily": 0.0,
            "forecast_days": forecast_days,
        }

    sorted_dates = sorted(daily.keys())
    base_date = sorted_dates[0]

    # Build x (days from base) / y (cost) arrays
    xs: list[float] = []
    ys: list[float] = []
    for d in sorted_dates:
        xs.append(float((d - base_date).days))
        ys.append(daily[d])

    n = len(xs)
    slope, intercept = _linear_regression(xs, ys, n)

    # Project forward
    last_date = sorted_dates[-1]
    forecasts: list[dict[str, Any]] = []
    total = 0.0

    for i in range(1, forecast_days + 1):
        forecast_date = last_date + timedelta(days=i)
        x_val = float((forecast_date - base_date).days)
        predicted = max(slope * x_val + intercept, 0.0)
        forecasts.append(
            {"date": forecast_date.isoformat(), "cost": round(predicted, 2)}
        )
        total += predicted

    avg_daily = total / forecast_days if forecast_days > 0 else 0.0

    return {
        "daily_forecasts": forecasts,
        "total_forecast": round(total, 2),
        "average_daily": round(avg_daily, 2),
        "forecast_days": forecast_days,
    }


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


def _parse_date(value: Any) -> date | None:
    """Best-effort parse of a date-like value."""
    if value is None:
        return None
    if isinstance(value, datetime):
        return value.date()
    if isinstance(value, date):
        return value
    if isinstance(value, str):
        try:
            return date.fromisoformat(value[:10])
        except ValueError:
            return None
    return None


def _linear_regression(
    xs: list[float], ys: list[float], n: int
) -> tuple[float, float]:
    """Return (slope, intercept) via simple ordinary least squares."""
    if n == 0:
        return 0.0, 0.0
    if n == 1:
        return 0.0, ys[0]

    sum_x = sum(xs)
    sum_y = sum(ys)
    sum_xy = sum(x * y for x, y in zip(xs, ys))
    sum_x2 = sum(x * x for x in xs)

    denom = n * sum_x2 - sum_x * sum_x
    if denom == 0:
        return 0.0, sum_y / n

    slope = (n * sum_xy - sum_x * sum_y) / denom
    intercept = (sum_y - slope * sum_x) / n
    return slope, intercept
