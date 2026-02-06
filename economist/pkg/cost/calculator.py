"""
Cost calculation utilities for the Economist module.

Provides currency normalization, cost aggregation, trend analysis,
and simple linear forecasting. All monetary calculations use
``decimal.Decimal`` to avoid floating-point precision errors.
"""

from __future__ import annotations

import json
import logging
import os
from collections import defaultdict
from datetime import date, datetime, timedelta
from decimal import ROUND_HALF_UP, Decimal
from typing import Any

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Exchange rates (relative to 1 USD).
#
# Default static table used when no external rates are provided.  Override at
# runtime via the ``EXCHANGE_RATES_JSON`` environment variable, which should
# contain a JSON object mapping currency codes to their USD multipliers,
# e.g. ``{"EUR": "1.09", "GBP": "1.27"}``.
# ---------------------------------------------------------------------------
_DEFAULT_RATES: dict[str, Decimal] = {
    "USD": Decimal("1.0"),
    "EUR": Decimal("1.09"),
    "GBP": Decimal("1.27"),
    "JPY": Decimal("0.0067"),
    "CAD": Decimal("0.74"),
    "AUD": Decimal("0.65"),
    "CHF": Decimal("1.13"),
    "CNY": Decimal("0.14"),
    "INR": Decimal("0.012"),
    "BRL": Decimal("0.20"),
    "KRW": Decimal("0.00075"),
    "SGD": Decimal("0.75"),
    "HKD": Decimal("0.13"),
    "SEK": Decimal("0.096"),
    "NOK": Decimal("0.094"),
    "MXN": Decimal("0.058"),
}


def _load_exchange_rates() -> dict[str, Decimal]:
    """Load exchange rates, preferring env-var overrides over defaults."""
    rates = dict(_DEFAULT_RATES)
    env_json = os.environ.get("EXCHANGE_RATES_JSON", "").strip()
    if env_json:
        try:
            overrides = json.loads(env_json)
            for code, value in overrides.items():
                rates[code.upper()] = Decimal(str(value))
            logger.info(
                "Loaded %d exchange rate overrides from EXCHANGE_RATES_JSON",
                len(overrides),
            )
        except (json.JSONDecodeError, Exception) as exc:
            logger.warning(
                "Failed to parse EXCHANGE_RATES_JSON; using defaults: %s", exc
            )
    return rates


_EXCHANGE_RATES_TO_USD: dict[str, Decimal] = _load_exchange_rates()

_SIX_PLACES = Decimal("0.000001")
_TWO_PLACES = Decimal("0.01")


def normalize_cost(
    amount: float | Decimal,
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
    amt = Decimal(str(amount))

    if currency == target_currency:
        return float(amt.quantize(_SIX_PLACES, rounding=ROUND_HALF_UP))

    # Convert source -> USD
    src_rate = _EXCHANGE_RATES_TO_USD.get(currency)
    if src_rate is None:
        logger.warning("Unknown currency %s; treating as 1:1 with USD", currency)
        src_rate = Decimal("1.0")

    amount_usd = amt * src_rate

    # Convert USD -> target
    tgt_rate = _EXCHANGE_RATES_TO_USD.get(target_currency)
    if tgt_rate is None:
        logger.warning(
            "Unknown target currency %s; treating as 1:1 with USD",
            target_currency,
        )
        tgt_rate = Decimal("1.0")

    result = amount_usd / tgt_rate
    return float(result.quantize(_SIX_PLACES, rounding=ROUND_HALF_UP))


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
    totals: dict[str, Decimal] = defaultdict(lambda: Decimal("0"))
    for cost in costs:
        key = cost.get(group_by, "unknown")
        if key is None:
            key = "unknown"
        totals[str(key)] += Decimal(str(cost.get("cost_usd", 0.0)))

    # Sort descending by cost, convert back to float for API compatibility
    return dict(
        sorted(
            ((k, float(v)) for k, v in totals.items()),
            key=lambda kv: kv[1],
            reverse=True,
        )
    )


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

    current_total = Decimal("0")
    previous_total = Decimal("0")

    for cost in costs:
        cost_date = _parse_date(cost.get("date"))
        if cost_date is None:
            continue
        amount = Decimal(str(cost.get("cost_usd", 0.0)))

        if current_start <= cost_date <= today:
            current_total += amount
        elif previous_start <= cost_date < current_start:
            previous_total += amount

    if previous_total > 0:
        change_pct = ((current_total - previous_total) / previous_total) * Decimal(
            "100"
        )
    elif current_total > 0:
        change_pct = Decimal("100")
    else:
        change_pct = Decimal("0")

    if change_pct > Decimal("2"):
        trend = "increasing"
    elif change_pct < Decimal("-2"):
        trend = "decreasing"
    else:
        trend = "stable"

    return {
        "current_period_cost": float(
            current_total.quantize(_TWO_PLACES, rounding=ROUND_HALF_UP)
        ),
        "previous_period_cost": float(
            previous_total.quantize(_TWO_PLACES, rounding=ROUND_HALF_UP)
        ),
        "change_percent": float(
            change_pct.quantize(_TWO_PLACES, rounding=ROUND_HALF_UP)
        ),
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
    # Aggregate costs by day using Decimal for precision
    daily_dec: dict[date, Decimal] = defaultdict(lambda: Decimal("0"))
    for cost in historical_costs:
        d = _parse_date(cost.get("date"))
        if d is None:
            continue
        daily_dec[d] += Decimal(str(cost.get("cost_usd", 0.0)))
    # Convert to float for linear regression (acceptable for forecasting)
    daily: dict[date, float] = {k: float(v) for k, v in daily_dec.items()}

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


def _linear_regression(xs: list[float], ys: list[float], n: int) -> tuple[float, float]:
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
