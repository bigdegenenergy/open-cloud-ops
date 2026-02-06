"""
Full API route definitions for the Economist module.

All endpoints live under ``/api/v1/`` and are grouped into:

* **Costs** -- summary, breakdown, trend, forecast, collection trigger.
* **Recommendations** -- list, resolve.
* **Governance** -- policies CRUD, violation listing.
* **Dashboard** -- combined overview for UI consumption.
"""

from __future__ import annotations

import logging
from datetime import date, datetime, timedelta
from typing import Any, Optional
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query, Security
from fastapi.security import APIKeyHeader
from pydantic import BaseModel, Field
from sqlalchemy import func
from sqlalchemy.orm import Session

from internal.ingestion.collector import CostCollector
from internal.optimizer.engine import OptimizationEngine
from internal.policy.engine import PolicyEngine
from pkg.cost.calculator import (
    aggregate_costs,
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
router = APIRouter(prefix="/api/v1", tags=["economist"])


# ===== COSTS ==============================================================


@router.get("/costs/summary", response_model=CostSummaryResponse)
async def get_cost_summary(
    provider: Optional[str] = Query(None, description="Filter by provider"),
    service: Optional[str] = Query(None, description="Filter by service"),
    start_date: Optional[str] = Query(None, description="Start date (YYYY-MM-DD)"),
    end_date: Optional[str] = Query(None, description="End date (YYYY-MM-DD)"),
    db: Session = Depends(get_session),
) -> CostSummaryResponse:
    """Return an aggregated cost summary with optional filters."""
    query = db.query(CloudCost)

    if provider:
        query = query.filter(CloudCost.provider == provider)
    if service:
        query = query.filter(CloudCost.service == service)

    start = _parse_date_param(start_date, default_days_back=30)
    end = _parse_date_param(end_date, default_days_back=0)
    query = query.filter(CloudCost.date >= start, CloudCost.date <= end)

    rows = query.all()
    cost_dicts = [r.to_dict() for r in rows]

    total = sum(c.get("cost_usd", 0) for c in cost_dicts)
    by_provider = aggregate_costs(cost_dicts, "provider")
    by_service = aggregate_costs(cost_dicts, "service")

    return CostSummaryResponse(
        total_cost_usd=round(total, 2),
        provider_breakdown=by_provider,
        service_breakdown=by_service,
        date_range={"start": start.isoformat(), "end": end.isoformat()},
        record_count=len(cost_dicts),
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
    db: Session = Depends(get_session),
) -> CostBreakdownResponse:
    """Return a detailed cost breakdown by a single dimension."""
    valid_dimensions = {"provider", "service", "region", "account_id"}
    if dimension not in valid_dimensions:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid dimension '{dimension}'. Must be one of: {sorted(valid_dimensions)}",
        )

    query = db.query(CloudCost)
    if provider:
        query = query.filter(CloudCost.provider == provider)

    start = _parse_date_param(start_date, default_days_back=30)
    end = _parse_date_param(end_date, default_days_back=0)
    query = query.filter(CloudCost.date >= start, CloudCost.date <= end)

    rows = query.all()
    cost_dicts = [r.to_dict() for r in rows]
    breakdown = aggregate_costs(cost_dicts, dimension)
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
    db: Session = Depends(get_session),
) -> list[RecommendationResponse]:
    """List optimization recommendations with optional filters."""
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

    rows = query.order_by(
        OptimizationRecommendation.estimated_monthly_savings.desc()
    ).all()

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
    db: Session = Depends(get_session),
) -> list[ViolationResponse]:
    """List policy violations with optional filters."""
    query = db.query(PolicyViolation)

    if severity:
        query = query.filter(PolicyViolation.severity == severity)
    if provider:
        query = query.filter(PolicyViolation.provider == provider)
    if resolved is True:
        query = query.filter(PolicyViolation.resolved_at.isnot(None))
    elif resolved is False:
        query = query.filter(PolicyViolation.resolved_at.is_(None))

    rows = query.order_by(PolicyViolation.detected_at.desc()).all()
    return [ViolationResponse(**v.to_dict()) for v in rows]


# ===== DASHBOARD ===========================================================


@router.get("/dashboard/overview", response_model=DashboardOverview)
async def get_dashboard_overview(
    db: Session = Depends(get_session),
) -> DashboardOverview:
    """Combined dashboard data endpoint."""
    # Total cost (last 30 days)
    thirty_days_ago = date.today() - timedelta(days=30)
    cost_rows = db.query(CloudCost).filter(CloudCost.date >= thirty_days_ago).all()
    cost_dicts = [r.to_dict() for r in cost_rows]
    total_cost = sum(c.get("cost_usd", 0) for c in cost_dicts)

    # Trend
    sixty_days_ago = date.today() - timedelta(days=60)
    trend_rows = db.query(CloudCost).filter(CloudCost.date >= sixty_days_ago).all()
    trend_dicts = [r.to_dict() for r in trend_rows]
    trend_data = calculate_trend(trend_dicts, 30)

    # Top services and providers
    top_services = aggregate_costs(cost_dicts, "service")
    top_services = dict(list(top_services.items())[:10])
    top_providers = aggregate_costs(cost_dicts, "provider")

    # Recommendations
    open_recs = (
        db.query(OptimizationRecommendation)
        .filter(OptimizationRecommendation.status == "open")
        .all()
    )
    rec_count = len(open_recs)
    total_savings = sum(r.estimated_monthly_savings for r in open_recs)

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
