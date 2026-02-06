"""
Full API route definitions for the Economist module.

All endpoints live under ``/api/v1/`` and are grouped into:

* **Costs** -- summary, breakdown, trend, forecast, collection trigger.
* **Recommendations** -- list, resolve.
* **Governance** -- policies CRUD, violation listing.
* **Dashboard** -- combined overview for UI consumption.
"""

from __future__ import annotations

import hashlib
import hmac
import logging
from datetime import date, datetime, timedelta
from typing import Any, Optional
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query, Security
from fastapi.security import APIKeyHeader
from pydantic import BaseModel, Field
from sqlalchemy import func, text
from sqlalchemy.orm import Session

from internal.ingestion.collector import CostCollector
from internal.optimizer.engine import OptimizationEngine
from internal.policy.engine import PolicyEngine
from pkg.cost.calculator import (
    calculate_trend,
    forecast_cost,
)
from pkg.database import (
    CloudCost,
    GovernancePolicy,
    OptimizationRecommendation,
    PolicyViolation,
    get_session,
)

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# API Key authentication
# ---------------------------------------------------------------------------
_api_key_header = APIKeyHeader(name="X-API-Key", auto_error=False)


def _hash_api_key(key: str) -> str:
    """SHA-256 hash an API key (consistent with Cerebra/Aegis)."""
    return hashlib.sha256(key.encode()).hexdigest()


async def require_api_key(
    api_key: str | None = Security(_api_key_header),
    db: Session = Depends(get_session),
) -> str:
    """Validate API key against the api_keys table using SHA-256 hash."""
    if not api_key:
        raise HTTPException(
            status_code=401,
            detail="Missing API key. Provide X-API-Key header.",
        )

    key_prefix = api_key[:8] if len(api_key) >= 8 else api_key
    key_hash = _hash_api_key(api_key)

    row = db.execute(
        text(
            "SELECT key_hash FROM api_keys "
            "WHERE key_prefix = :prefix AND revoked = false"
        ),
        {"prefix": key_prefix},
    ).fetchone()

    if row is None or not hmac.compare_digest(row[0], key_hash):
        raise HTTPException(status_code=401, detail="Invalid API key.")

    return api_key


# ---------------------------------------------------------------------------
# Shared singletons -- wired at startup via ``configure_routes``
# ---------------------------------------------------------------------------
_collector: CostCollector | None = None
_optimizer: OptimizationEngine | None = None
_policy_engine: PolicyEngine | None = None


def configure_routes(
    collector: CostCollector,
    optimizer: OptimizationEngine,
    policy_engine: PolicyEngine,
) -> None:
    """Inject runtime dependencies into the route module."""
    global _collector, _optimizer, _policy_engine
    _collector = collector
    _optimizer = optimizer
    _policy_engine = policy_engine


# ---------------------------------------------------------------------------
# Pydantic request / response schemas
# ---------------------------------------------------------------------------


class CostSummaryResponse(BaseModel):
    total_cost_usd: float
    provider_breakdown: dict[str, float]
    service_breakdown: dict[str, float]
    date_range: dict[str, str]
    record_count: int


class CostBreakdownResponse(BaseModel):
    dimension: str
    breakdown: dict[str, float]
    total_cost_usd: float


class CostTrendResponse(BaseModel):
    current_period_cost: float
    previous_period_cost: float
    change_percent: float
    trend: str
    period_days: int


class CostForecastResponse(BaseModel):
    daily_forecasts: list[dict[str, Any]]
    total_forecast: float
    average_daily: float
    forecast_days: int


class CollectRequest(BaseModel):
    start_date: Optional[str] = None
    end_date: Optional[str] = None
    providers: Optional[list[str]] = None


class CollectResponse(BaseModel):
    records_collected: int
    records_stored: int
    providers: list[str]


class RecommendationResponse(BaseModel):
    id: str
    provider: str
    resource_id: str
    resource_type: str
    recommendation_type: str
    title: str
    description: Optional[str] = None
    estimated_monthly_savings: float
    confidence: float
    status: str
    created_at: Optional[str] = None
    resolved_at: Optional[str] = None


class PolicyCreateRequest(BaseModel):
    name: str
    description: str = ""
    policy_type: str
    rules: dict[str, Any]
    severity: str = "warning"


class PolicyUpdateRequest(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    rules: Optional[dict[str, Any]] = None
    severity: Optional[str] = None
    enabled: Optional[bool] = None


class PolicyResponse(BaseModel):
    id: str
    name: str
    description: Optional[str] = None
    policy_type: str
    rules: dict[str, Any]
    severity: str
    enabled: bool
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class ViolationResponse(BaseModel):
    id: str
    policy_id: str
    resource_id: str
    provider: str
    description: Optional[str] = None
    severity: str
    detected_at: Optional[str] = None
    resolved_at: Optional[str] = None


class DashboardOverview(BaseModel):
    total_cost_usd: float
    cost_trend: CostTrendResponse
    top_services: dict[str, float]
    top_providers: dict[str, float]
    recommendation_count: int
    total_potential_savings: float
    active_policies: int
    open_violations: int


# ---------------------------------------------------------------------------
# Router
# ---------------------------------------------------------------------------
router = APIRouter(
    prefix="/api/v1",
    tags=["economist"],
    dependencies=[Depends(require_api_key)],
)


# ===== COSTS ==============================================================


@router.get("/costs/summary", response_model=CostSummaryResponse)
async def get_cost_summary(
    provider: Optional[str] = Query(None, description="Filter by provider"),
    service: Optional[str] = Query(None, description="Filter by service"),
    start_date: Optional[str] = Query(None, description="Start date (YYYY-MM-DD)"),
    end_date: Optional[str] = Query(None, description="End date (YYYY-MM-DD)"),
    db: Session = Depends(get_session),
) -> CostSummaryResponse:
    """Return an aggregated cost summary with optional filters.

    Uses SQL-level aggregation to avoid loading all rows into memory.
    """
    start = _parse_date_param(start_date, default_days_back=30)
    end = _parse_date_param(end_date, default_days_back=0)

    base_filter = [CloudCost.date >= start, CloudCost.date <= end]
    if provider:
        base_filter.append(CloudCost.provider == provider)
    if service:
        base_filter.append(CloudCost.service == service)

    # Total cost + record count via SQL aggregation
    totals = (
        db.query(
            func.coalesce(func.sum(CloudCost.cost_usd), 0),
            func.count(CloudCost.id),
        )
        .filter(*base_filter)
        .one()
    )
    total_cost = float(totals[0])
    record_count = totals[1]

    # Provider breakdown via SQL GROUP BY
    by_provider_rows = (
        db.query(CloudCost.provider, func.sum(CloudCost.cost_usd))
        .filter(*base_filter)
        .group_by(CloudCost.provider)
        .order_by(func.sum(CloudCost.cost_usd).desc())
        .all()
    )
    by_provider = {row[0]: float(row[1]) for row in by_provider_rows}

    # Service breakdown via SQL GROUP BY
    by_service_rows = (
        db.query(CloudCost.service, func.sum(CloudCost.cost_usd))
        .filter(*base_filter)
        .group_by(CloudCost.service)
        .order_by(func.sum(CloudCost.cost_usd).desc())
        .all()
    )
    by_service = {row[0]: float(row[1]) for row in by_service_rows}

    return CostSummaryResponse(
        total_cost_usd=round(total_cost, 2),
        provider_breakdown=by_provider,
        service_breakdown=by_service,
        date_range={"start": start.isoformat(), "end": end.isoformat()},
        record_count=record_count,
    )


@router.get("/costs/breakdown", response_model=CostBreakdownResponse)
async def get_cost_breakdown(
    dimension: str = Query(
        "service",
        description="Dimension to break down by (provider, service, region, account_id)",
    ),
    provider: Optional[str] = Query(None),
    start_date: Optional[str] = Query(None),
    end_date: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000, description="Max rows to return"),
    db: Session = Depends(get_session),
) -> CostBreakdownResponse:
    """Return a detailed cost breakdown by a single dimension.

    Uses SQL GROUP BY for aggregation to avoid loading all rows into memory.
    """
    _valid_dimension_cols = {
        "provider": CloudCost.provider,
        "service": CloudCost.service,
        "region": CloudCost.region,
        "account_id": CloudCost.account_id,
    }
    if dimension not in _valid_dimension_cols:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid dimension '{dimension}'. Must be one of: {sorted(_valid_dimension_cols)}",
        )

    dim_col = _valid_dimension_cols[dimension]
    start = _parse_date_param(start_date, default_days_back=30)
    end = _parse_date_param(end_date, default_days_back=0)

    base_filter = [CloudCost.date >= start, CloudCost.date <= end]
    if provider:
        base_filter.append(CloudCost.provider == provider)

    rows = (
        db.query(dim_col, func.sum(CloudCost.cost_usd))
        .filter(*base_filter)
        .group_by(dim_col)
        .order_by(func.sum(CloudCost.cost_usd).desc())
        .limit(limit)
        .all()
    )
    breakdown = {str(row[0] or "unknown"): float(row[1]) for row in rows}
    total = sum(breakdown.values())

    return CostBreakdownResponse(
        dimension=dimension,
        breakdown=breakdown,
        total_cost_usd=round(total, 2),
    )


@router.get("/costs/trend", response_model=CostTrendResponse)
async def get_cost_trend(
    period_days: int = Query(30, ge=1, le=365),
    provider: Optional[str] = Query(None),
    db: Session = Depends(get_session),
) -> CostTrendResponse:
    """Return cost trend analysis over two consecutive periods."""
    lookback = period_days * 2
    start = date.today() - timedelta(days=lookback)

    query = db.query(CloudCost).filter(CloudCost.date >= start)
    if provider:
        query = query.filter(CloudCost.provider == provider)

    rows = query.all()
    cost_dicts = [r.to_dict() for r in rows]
    trend_data = calculate_trend(cost_dicts, period_days)

    return CostTrendResponse(**trend_data)


@router.get("/costs/forecast", response_model=CostForecastResponse)
async def get_cost_forecast(
    forecast_days: int = Query(30, ge=1, le=365),
    history_days: int = Query(90, ge=7, le=730),
    provider: Optional[str] = Query(None),
    db: Session = Depends(get_session),
) -> CostForecastResponse:
    """Forecast future costs based on historical data."""
    start = date.today() - timedelta(days=history_days)

    query = db.query(CloudCost).filter(CloudCost.date >= start)
    if provider:
        query = query.filter(CloudCost.provider == provider)

    rows = query.all()
    cost_dicts = [r.to_dict() for r in rows]
    forecast_data = forecast_cost(cost_dicts, forecast_days)

    return CostForecastResponse(**forecast_data)


@router.post("/costs/collect", response_model=CollectResponse)
async def trigger_cost_collection(
    body: CollectRequest | None = None,
    db: Session = Depends(get_session),
) -> CollectResponse:
    """Trigger an ad-hoc cost collection run."""
    if _collector is None:
        raise HTTPException(status_code=503, detail="Collector not initialized")

    body = body or CollectRequest()

    end = _parse_date_param(body.end_date, default_days_back=0)
    start = _parse_date_param(body.start_date, default_days_back=7)

    # Collect
    if body.providers:
        all_costs: list[dict[str, Any]] = []
        for name in body.providers:
            try:
                costs = await _collector.collect_from(name, start, end)
                all_costs.extend(costs)
            except ValueError as exc:
                raise HTTPException(status_code=400, detail=str(exc)) from exc
    else:
        all_costs = await _collector.collect_all(start, end)

    # Store
    stored = _collector.store_costs(all_costs, db)

    return CollectResponse(
        records_collected=len(all_costs),
        records_stored=stored,
        providers=body.providers or _collector.provider_names,
    )


# ===== RECOMMENDATIONS =====================================================


@router.get("/recommendations", response_model=list[RecommendationResponse])
async def list_recommendations(
    status: Optional[str] = Query(
        None, description="Filter by status (open, resolved)"
    ),
    recommendation_type: Optional[str] = Query(None),
    provider: Optional[str] = Query(None),
    min_savings: Optional[float] = Query(None, ge=0),
    limit: int = Query(100, ge=1, le=1000, description="Max results to return"),
    offset: int = Query(0, ge=0, description="Number of results to skip"),
    db: Session = Depends(get_session),
) -> list[RecommendationResponse]:
    """List optimization recommendations with optional filters and pagination."""
    query = db.query(OptimizationRecommendation)

    if status:
        query = query.filter(OptimizationRecommendation.status == status)
    if recommendation_type:
        query = query.filter(
            OptimizationRecommendation.recommendation_type == recommendation_type
        )
    if provider:
        query = query.filter(OptimizationRecommendation.provider == provider)
    if min_savings is not None:
        query = query.filter(
            OptimizationRecommendation.estimated_monthly_savings >= min_savings
        )

    rows = (
        query.order_by(OptimizationRecommendation.estimated_monthly_savings.desc())
        .offset(offset)
        .limit(limit)
        .all()
    )

    return [RecommendationResponse(**r.to_dict()) for r in rows]


@router.post(
    "/recommendations/{recommendation_id}/resolve",
    response_model=RecommendationResponse,
)
async def resolve_recommendation(
    recommendation_id: str,
    db: Session = Depends(get_session),
) -> RecommendationResponse:
    """Mark a recommendation as resolved."""
    rec = (
        db.query(OptimizationRecommendation)
        .filter(OptimizationRecommendation.id == recommendation_id)
        .first()
    )
    if rec is None:
        raise HTTPException(status_code=404, detail="Recommendation not found")

    rec.status = "resolved"
    rec.resolved_at = datetime.utcnow()
    db.commit()
    db.refresh(rec)

    return RecommendationResponse(**rec.to_dict())


# ===== GOVERNANCE ==========================================================


@router.get("/governance/policies", response_model=list[PolicyResponse])
async def list_policies(
    enabled_only: bool = Query(True),
    db: Session = Depends(get_session),
) -> list[PolicyResponse]:
    """List governance policies."""
    if _policy_engine is None:
        raise HTTPException(status_code=503, detail="Policy engine not initialized")
    policies = _policy_engine.list_policies(db, enabled_only=enabled_only)
    return [PolicyResponse(**p.to_dict()) for p in policies]


@router.post("/governance/policies", response_model=PolicyResponse, status_code=201)
async def create_policy(
    body: PolicyCreateRequest,
    db: Session = Depends(get_session),
) -> PolicyResponse:
    """Create a new governance policy."""
    if _policy_engine is None:
        raise HTTPException(status_code=503, detail="Policy engine not initialized")
    try:
        policy = _policy_engine.create_policy(
            name=body.name,
            description=body.description,
            policy_type=body.policy_type,
            rules=body.rules,
            severity=body.severity,
            db=db,
        )
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc

    return PolicyResponse(**policy.to_dict())


@router.put(
    "/governance/policies/{policy_id}",
    response_model=PolicyResponse,
)
async def update_policy(
    policy_id: str,
    body: PolicyUpdateRequest,
    db: Session = Depends(get_session),
) -> PolicyResponse:
    """Update an existing governance policy."""
    if _policy_engine is None:
        raise HTTPException(status_code=503, detail="Policy engine not initialized")

    updates = body.model_dump(exclude_unset=True)
    policy = _policy_engine.update_policy(policy_id, updates, db)
    if policy is None:
        raise HTTPException(status_code=404, detail="Policy not found")

    return PolicyResponse(**policy.to_dict())


@router.get("/governance/violations", response_model=list[ViolationResponse])
async def list_violations(
    severity: Optional[str] = Query(None),
    provider: Optional[str] = Query(None),
    resolved: Optional[bool] = Query(None),
    limit: int = Query(100, ge=1, le=1000, description="Max results to return"),
    offset: int = Query(0, ge=0, description="Number of results to skip"),
    db: Session = Depends(get_session),
) -> list[ViolationResponse]:
    """List policy violations with optional filters and pagination."""
    query = db.query(PolicyViolation)

    if severity:
        query = query.filter(PolicyViolation.severity == severity)
    if provider:
        query = query.filter(PolicyViolation.provider == provider)
    if resolved is True:
        query = query.filter(PolicyViolation.resolved_at.isnot(None))
    elif resolved is False:
        query = query.filter(PolicyViolation.resolved_at.is_(None))

    rows = (
        query.order_by(PolicyViolation.detected_at.desc())
        .offset(offset)
        .limit(limit)
        .all()
    )
    return [ViolationResponse(**v.to_dict()) for v in rows]


# ===== DASHBOARD ===========================================================


@router.get("/dashboard/overview", response_model=DashboardOverview)
async def get_dashboard_overview(
    db: Session = Depends(get_session),
) -> DashboardOverview:
    """Combined dashboard data endpoint.

    Uses SQL-level aggregation to avoid loading all cost rows into memory.
    """
    thirty_days_ago = date.today() - timedelta(days=30)
    sixty_days_ago = date.today() - timedelta(days=60)

    # Total cost (last 30 days) via SQL
    total_cost_row = (
        db.query(func.coalesce(func.sum(CloudCost.cost_usd), 0))
        .filter(CloudCost.date >= thirty_days_ago)
        .scalar()
    )
    total_cost = float(total_cost_row)

    # Trend: we still need per-record date info for the two-period comparison,
    # but only pull the minimal columns needed (cost + date).
    trend_rows = (
        db.query(CloudCost.cost_usd, CloudCost.date)
        .filter(CloudCost.date >= sixty_days_ago)
        .all()
    )
    trend_dicts = [{"cost_usd": float(r[0]), "date": r[1]} for r in trend_rows]
    trend_data = calculate_trend(trend_dicts, 30)

    # Top services (SQL GROUP BY, top 10)
    top_svc_rows = (
        db.query(CloudCost.service, func.sum(CloudCost.cost_usd))
        .filter(CloudCost.date >= thirty_days_ago)
        .group_by(CloudCost.service)
        .order_by(func.sum(CloudCost.cost_usd).desc())
        .limit(10)
        .all()
    )
    top_services = {row[0]: float(row[1]) for row in top_svc_rows}

    # Top providers (SQL GROUP BY)
    top_prov_rows = (
        db.query(CloudCost.provider, func.sum(CloudCost.cost_usd))
        .filter(CloudCost.date >= thirty_days_ago)
        .group_by(CloudCost.provider)
        .order_by(func.sum(CloudCost.cost_usd).desc())
        .all()
    )
    top_providers = {row[0]: float(row[1]) for row in top_prov_rows}

    # Recommendations: aggregate via SQL instead of loading all rows
    rec_agg = (
        db.query(
            func.count(OptimizationRecommendation.id),
            func.coalesce(
                func.sum(OptimizationRecommendation.estimated_monthly_savings), 0
            ),
        )
        .filter(OptimizationRecommendation.status == "open")
        .one()
    )
    rec_count = rec_agg[0]
    total_savings = float(rec_agg[1])

    # Policies & violations
    active_policies = (
        db.query(GovernancePolicy).filter(GovernancePolicy.enabled.is_(True)).count()
    )
    open_violations = (
        db.query(PolicyViolation).filter(PolicyViolation.resolved_at.is_(None)).count()
    )

    return DashboardOverview(
        total_cost_usd=round(total_cost, 2),
        cost_trend=CostTrendResponse(**trend_data),
        top_services=top_services,
        top_providers=top_providers,
        recommendation_count=rec_count,
        total_potential_savings=round(total_savings, 2),
        active_policies=active_policies,
        open_violations=open_violations,
    )


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _parse_date_param(
    value: str | None,
    default_days_back: int = 30,
) -> date:
    """Parse an optional date string, falling back to *N* days ago."""
    if value:
        try:
            return date.fromisoformat(value)
        except ValueError as exc:
            raise HTTPException(
                status_code=400,
                detail=f"Invalid date format: {value}. Use YYYY-MM-DD.",
            ) from exc
    return date.today() - timedelta(days=default_days_back)
