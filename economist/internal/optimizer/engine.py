"""
Cost optimization recommendation engine.

Analyses cloud costs and resources to generate actionable
optimization recommendations covering idle resources, right-sizing,
reserved capacity, and spot/preemptible opportunities.
"""

from __future__ import annotations

import logging
import uuid
from collections import defaultdict
from datetime import date, datetime, timedelta
from typing import Any

from pkg.database import OptimizationRecommendation

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Thresholds (tunable)
# ---------------------------------------------------------------------------
IDLE_COST_THRESHOLD_USD = 1.0  # daily cost below which a resource may be idle
IDLE_USAGE_THRESHOLD = 0.05  # usage qty below which a resource is considered idle
RIGHTSIZING_CPU_THRESHOLD = 0.20  # average CPU below this -> over-provisioned
RIGHTSIZING_MEM_THRESHOLD = 0.25
RESERVED_CAPACITY_COVERAGE_THRESHOLD = 0.70  # on-demand % above which RI is suggested
SPOT_ELIGIBLE_TYPES = {"ec2:instance", "compute:instance", "batch", "emr"}


class OptimizationEngine:
    """Generates cost optimization recommendations from cost / resource data."""

    # ------------------------------------------------------------------
    # Idle resources
    # ------------------------------------------------------------------

    def analyze_idle_resources(
        self,
        costs: list[dict[str, Any]],
    ) -> list[OptimizationRecommendation]:
        """Detect resources whose cost or usage is negligibly low.

        Parameters
        ----------
        costs:
            Cost records (dicts with at least ``resource_id``,
            ``cost_usd``, ``usage_quantity``, ``provider``, ``service``,
            ``date``).

        Returns
        -------
        list[OptimizationRecommendation]
        """
        # Aggregate costs per resource over last 30 days
        cutoff = date.today() - timedelta(days=30)
        resource_totals: dict[str, dict[str, Any]] = defaultdict(
            lambda: {"cost": 0.0, "usage": 0.0, "days": 0, "provider": "", "service": ""}
        )

        for cost in costs:
            cost_date = self._parse_date(cost.get("date"))
            if cost_date and cost_date >= cutoff:
                rid = cost.get("resource_id", "")
                bucket = resource_totals[rid]
                bucket["cost"] += cost.get("cost_usd", 0.0)
                bucket["usage"] += cost.get("usage_quantity", 0.0) or 0.0
                bucket["days"] += 1
                bucket["provider"] = cost.get("provider", "")
                bucket["service"] = cost.get("service", "")

        recommendations: list[OptimizationRecommendation] = []

        for rid, data in resource_totals.items():
            days = max(data["days"], 1)
            avg_daily_cost = data["cost"] / days
            avg_daily_usage = data["usage"] / days

            if (
                avg_daily_cost < IDLE_COST_THRESHOLD_USD
                and avg_daily_usage < IDLE_USAGE_THRESHOLD
                and data["cost"] > 0
            ):
                monthly_savings = avg_daily_cost * 30
                rec = OptimizationRecommendation(
                    id=uuid.uuid4(),
                    provider=data["provider"],
                    resource_id=rid,
                    resource_type=data["service"],
                    recommendation_type="idle_resource",
                    title=f"Idle resource detected: {rid}",
                    description=(
                        f"Resource {rid} has averaged ${avg_daily_cost:.2f}/day "
                        f"with minimal usage ({avg_daily_usage:.4f}) over the "
                        f"last {days} days. Consider shutting it down."
                    ),
                    estimated_monthly_savings=round(monthly_savings, 2),
                    confidence=0.85,
                    status="open",
                )
                recommendations.append(rec)

        logger.info(
            "Idle resource analysis: %d recommendations", len(recommendations)
        )
        return recommendations

    # ------------------------------------------------------------------
    # Right-sizing
    # ------------------------------------------------------------------

    def analyze_rightsizing(
        self,
        resources: list[dict[str, Any]],
    ) -> list[OptimizationRecommendation]:
        """Identify over-provisioned resources.

        Parameters
        ----------
        resources:
            Resource dicts with ``resource_id``, ``resource_type``,
            ``provider``, and optionally ``cpu_avg``, ``memory_avg``,
            ``instance_type``.

        Returns
        -------
        list[OptimizationRecommendation]
        """
        recommendations: list[OptimizationRecommendation] = []

        for resource in resources:
            cpu_avg = resource.get("cpu_avg")
            mem_avg = resource.get("memory_avg")
            if cpu_avg is None and mem_avg is None:
                continue

            over_cpu = cpu_avg is not None and cpu_avg < RIGHTSIZING_CPU_THRESHOLD
            over_mem = mem_avg is not None and mem_avg < RIGHTSIZING_MEM_THRESHOLD

            if over_cpu or over_mem:
                rid = resource.get("resource_id", "unknown")
                instance_type = resource.get("instance_type", "unknown")

                # Estimate savings as a percentage of typical instance cost
                utilization = max(cpu_avg or 0, mem_avg or 0)
                savings_factor = 1.0 - (utilization / 0.5)  # rough heuristic
                savings_factor = max(min(savings_factor, 0.6), 0.1)
                estimated_savings = savings_factor * 100  # placeholder $/month

                metrics_note = []
                if over_cpu:
                    metrics_note.append(f"CPU avg {cpu_avg:.1%}")
                if over_mem:
                    metrics_note.append(f"memory avg {mem_avg:.1%}")

                rec = OptimizationRecommendation(
                    id=uuid.uuid4(),
                    provider=resource.get("provider", "unknown"),
                    resource_id=rid,
                    resource_type=resource.get("resource_type", "unknown"),
                    recommendation_type="rightsizing",
                    title=f"Rightsize {instance_type} ({rid})",
                    description=(
                        f"Resource {rid} ({instance_type}) is over-provisioned: "
                        f"{', '.join(metrics_note)}. Consider downsizing to a "
                        f"smaller instance type."
                    ),
                    estimated_monthly_savings=round(estimated_savings, 2),
                    confidence=0.75,
                    status="open",
                )
                recommendations.append(rec)

        logger.info(
            "Rightsizing analysis: %d recommendations", len(recommendations)
        )
        return recommendations

    # ------------------------------------------------------------------
    # Reserved capacity
    # ------------------------------------------------------------------

    def analyze_reserved_capacity(
        self,
        costs: list[dict[str, Any]],
    ) -> list[OptimizationRecommendation]:
        """Recommend Reserved Instance or Savings Plan purchases.

        Looks at steady-state on-demand usage to identify services where
        committed pricing would yield savings.

        Parameters
        ----------
        costs:
            Cost records.

        Returns
        -------
        list[OptimizationRecommendation]
        """
        cutoff = date.today() - timedelta(days=60)

        # Aggregate by provider + service
        service_costs: dict[str, dict[str, Any]] = defaultdict(
            lambda: {"total": 0.0, "days": set(), "provider": ""}
        )

        for cost in costs:
            cost_date = self._parse_date(cost.get("date"))
            if cost_date and cost_date >= cutoff:
                key = f"{cost.get('provider', '')}:{cost.get('service', '')}"
                bucket = service_costs[key]
                bucket["total"] += cost.get("cost_usd", 0.0)
                bucket["days"].add(cost_date)
                bucket["provider"] = cost.get("provider", "")

        recommendations: list[OptimizationRecommendation] = []

        for key, data in service_costs.items():
            provider, service = key.split(":", 1)
            num_days = len(data["days"])
            if num_days < 30:
                continue  # not enough data

            avg_daily = data["total"] / num_days
            consistency = num_days / 60.0  # fraction of days with spend

            if consistency >= RESERVED_CAPACITY_COVERAGE_THRESHOLD and avg_daily > 5.0:
                # Estimate ~30% savings from RI/SP
                monthly_cost = avg_daily * 30
                estimated_savings = monthly_cost * 0.30

                rec = OptimizationRecommendation(
                    id=uuid.uuid4(),
                    provider=provider,
                    resource_id=f"{provider}:{service}",
                    resource_type=service,
                    recommendation_type="reserved_capacity",
                    title=f"Consider reserved pricing for {service}",
                    description=(
                        f"Service {service} on {provider} has consistent daily "
                        f"spend of ~${avg_daily:.2f} over {num_days} days. "
                        f"Reserved Instances or a Savings Plan could save "
                        f"~${estimated_savings:.2f}/month."
                    ),
                    estimated_monthly_savings=round(estimated_savings, 2),
                    confidence=0.80,
                    status="open",
                )
                recommendations.append(rec)

        logger.info(
            "Reserved capacity analysis: %d recommendations",
            len(recommendations),
        )
        return recommendations

    # ------------------------------------------------------------------
    # Spot / preemptible opportunities
    # ------------------------------------------------------------------

    def analyze_spot_opportunities(
        self,
        resources: list[dict[str, Any]],
    ) -> list[OptimizationRecommendation]:
        """Identify workloads eligible for spot or preemptible instances.

        Parameters
        ----------
        resources:
            Resource dicts with ``resource_type``, ``resource_id``,
            ``provider``, and optionally ``interruptible`` flag.

        Returns
        -------
        list[OptimizationRecommendation]
        """
        recommendations: list[OptimizationRecommendation] = []

        for resource in resources:
            rtype = resource.get("resource_type", "")
            if rtype not in SPOT_ELIGIBLE_TYPES:
                continue

            # Skip resources already running as spot
            if resource.get("lifecycle") == "spot":
                continue
            if resource.get("preemptible") is True:
                continue

            rid = resource.get("resource_id", "unknown")
            provider = resource.get("provider", "unknown")

            # Estimate ~60% savings from spot
            estimated_savings = 50.0  # placeholder per-instance $/month

            rec = OptimizationRecommendation(
                id=uuid.uuid4(),
                provider=provider,
                resource_id=rid,
                resource_type=rtype,
                recommendation_type="spot_instance",
                title=f"Use spot/preemptible for {rid}",
                description=(
                    f"Resource {rid} ({rtype}) on {provider} may be eligible "
                    f"for spot/preemptible pricing, potentially saving up to "
                    f"60-90% on compute costs."
                ),
                estimated_monthly_savings=round(estimated_savings, 2),
                confidence=0.60,
                status="open",
            )
            recommendations.append(rec)

        logger.info(
            "Spot opportunity analysis: %d recommendations",
            len(recommendations),
        )
        return recommendations

    # ------------------------------------------------------------------
    # Orchestrator
    # ------------------------------------------------------------------

    def generate_recommendations(
        self,
        costs: list[dict[str, Any]],
        resources: list[dict[str, Any]],
    ) -> list[OptimizationRecommendation]:
        """Run all analyses and return a unified list of recommendations.

        Parameters
        ----------
        costs:
            Cost line items.
        resources:
            Currently active cloud resources.

        Returns
        -------
        list[OptimizationRecommendation]
        """
        all_recs: list[OptimizationRecommendation] = []

        all_recs.extend(self.analyze_idle_resources(costs))
        all_recs.extend(self.analyze_rightsizing(resources))
        all_recs.extend(self.analyze_reserved_capacity(costs))
        all_recs.extend(self.analyze_spot_opportunities(resources))

        # De-duplicate by (provider, resource_id, recommendation_type)
        seen: set[tuple[str, str, str]] = set()
        unique: list[OptimizationRecommendation] = []
        for rec in all_recs:
            key = (rec.provider, rec.resource_id, rec.recommendation_type)
            if key not in seen:
                seen.add(key)
                unique.append(rec)

        logger.info(
            "Generated %d unique recommendations (from %d total)",
            len(unique),
            len(all_recs),
        )
        return unique

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _parse_date(value: Any) -> date | None:
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
