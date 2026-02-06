"""
Governance policy engine.

Defines, evaluates, and enforces governance policies such as budget
limits, tag compliance, region restrictions, and service allow-lists.
"""

from __future__ import annotations

import logging
import uuid
from datetime import date, datetime
from typing import Any

from sqlalchemy.orm import Session

from pkg.database import GovernancePolicy, PolicyViolation

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Supported policy types
# ---------------------------------------------------------------------------
POLICY_TYPES = {
    "budget_limit",
    "tag_compliance",
    "region_restriction",
    "service_allowlist",
}


class PolicyEngine:
    """Create, store, and evaluate governance policies."""

    # ------------------------------------------------------------------
    # Policy CRUD
    # ------------------------------------------------------------------

    def create_policy(
        self,
        name: str,
        description: str,
        policy_type: str,
        rules: dict[str, Any],
        severity: str = "warning",
        db: Session | None = None,
    ) -> GovernancePolicy:
        """Create and persist a new governance policy.

        Parameters
        ----------
        name:
            Human-readable policy name (must be unique).
        description:
            Longer explanation of the policy intent.
        policy_type:
            One of ``budget_limit``, ``tag_compliance``,
            ``region_restriction``, ``service_allowlist``.
        rules:
            JSON-serializable dict defining the policy rules.
        severity:
            ``"critical"`` | ``"warning"`` | ``"info"``.
        db:
            An active SQLAlchemy session.  If ``None`` the policy
            object is returned without being persisted.

        Raises
        ------
        ValueError
            If *policy_type* is not recognised.
        """
        if policy_type not in POLICY_TYPES:
            raise ValueError(
                f"Unknown policy_type '{policy_type}'. "
                f"Must be one of: {sorted(POLICY_TYPES)}"
            )

        policy = GovernancePolicy(
            id=uuid.uuid4(),
            name=name,
            description=description,
            policy_type=policy_type,
            rules=rules,
            severity=severity,
            enabled=True,
        )

        if db is not None:
            db.add(policy)
            db.commit()
            db.refresh(policy)
            logger.info("Created policy '%s' (id=%s)", name, policy.id)

        return policy

    def update_policy(
        self,
        policy_id: str,
        updates: dict[str, Any],
        db: Session,
    ) -> GovernancePolicy | None:
        """Update an existing policy.

        Parameters
        ----------
        policy_id:
            UUID of the policy to update.
        updates:
            Dict of fields to update (``name``, ``description``,
            ``rules``, ``severity``, ``enabled``).
        db:
            Active SQLAlchemy session.

        Returns
        -------
        GovernancePolicy | None
            The updated policy, or ``None`` if not found.
        """
        policy = db.query(GovernancePolicy).filter(
            GovernancePolicy.id == policy_id
        ).first()
        if policy is None:
            return None

        allowed = {"name", "description", "rules", "severity", "enabled"}
        for key, value in updates.items():
            if key in allowed:
                setattr(policy, key, value)

        db.commit()
        db.refresh(policy)
        logger.info("Updated policy %s", policy_id)
        return policy

    def list_policies(
        self,
        db: Session,
        enabled_only: bool = True,
    ) -> list[GovernancePolicy]:
        """Return all (or only enabled) policies."""
        query = db.query(GovernancePolicy)
        if enabled_only:
            query = query.filter(GovernancePolicy.enabled.is_(True))
        return query.order_by(GovernancePolicy.created_at.desc()).all()

    # ------------------------------------------------------------------
    # Evaluation
    # ------------------------------------------------------------------

    def evaluate_policies(
        self,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
        db: Session,
    ) -> list[PolicyViolation]:
        """Evaluate all enabled policies against current resources/costs.

        Parameters
        ----------
        resources:
            Current cloud resources.
        costs:
            Current cost records.
        db:
            Active SQLAlchemy session (used to read policies and store
            violations).

        Returns
        -------
        list[PolicyViolation]
            Newly detected violations.
        """
        policies = self.list_policies(db, enabled_only=True)
        all_violations: list[PolicyViolation] = []

        for policy in policies:
            violations = self._evaluate_single(policy, resources, costs)
            all_violations.extend(violations)

        # Persist violations
        if all_violations:
            db.bulk_save_objects(all_violations)
            db.commit()
            logger.info(
                "Detected and stored %d policy violations",
                len(all_violations),
            )

        return all_violations

    def detect_violations(
        self,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
        policies: list[GovernancePolicy],
    ) -> list[PolicyViolation]:
        """Evaluate a list of policies without database persistence.

        Useful for dry-run / preview scenarios and testing.
        """
        all_violations: list[PolicyViolation] = []
        for policy in policies:
            all_violations.extend(
                self._evaluate_single(policy, resources, costs)
            )
        return all_violations

    # ------------------------------------------------------------------
    # Individual policy evaluators
    # ------------------------------------------------------------------

    def _evaluate_single(
        self,
        policy: GovernancePolicy,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
    ) -> list[PolicyViolation]:
        """Dispatch to the correct evaluator based on *policy_type*."""
        evaluators = {
            "budget_limit": self._eval_budget_limit,
            "tag_compliance": self._eval_tag_compliance,
            "region_restriction": self._eval_region_restriction,
            "service_allowlist": self._eval_service_allowlist,
        }
        evaluator = evaluators.get(policy.policy_type)
        if evaluator is None:
            logger.warning(
                "No evaluator for policy_type=%s", policy.policy_type
            )
            return []
        return evaluator(policy, resources, costs)

    # -- budget_limit --------------------------------------------------

    @staticmethod
    def _eval_budget_limit(
        policy: GovernancePolicy,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
    ) -> list[PolicyViolation]:
        """Check whether costs exceed a configured budget.

        Expected rules schema::

            {
                "monthly_budget_usd": 10000,
                "scope": "all" | "<provider>" | "<account_id>",
            }
        """
        rules = policy.rules or {}
        budget = rules.get("monthly_budget_usd", float("inf"))
        scope = rules.get("scope", "all")

        # Aggregate costs for the current month
        today = date.today()
        month_start = today.replace(day=1)
        total = 0.0

        for cost in costs:
            cost_date = _parse_date(cost.get("date"))
            if cost_date is None or cost_date < month_start:
                continue

            if scope != "all":
                if (
                    cost.get("provider") != scope
                    and cost.get("account_id") != scope
                ):
                    continue

            total += cost.get("cost_usd", 0.0)

        violations: list[PolicyViolation] = []

        if total > budget:
            violations.append(
                PolicyViolation(
                    id=uuid.uuid4(),
                    policy_id=policy.id,
                    resource_id=f"budget:{scope}",
                    provider=scope if scope != "all" else "multi-cloud",
                    description=(
                        f"Monthly cost ${total:,.2f} exceeds budget "
                        f"${budget:,.2f} for scope '{scope}'."
                    ),
                    severity=policy.severity,
                )
            )

        return violations

    # -- tag_compliance ------------------------------------------------

    @staticmethod
    def _eval_tag_compliance(
        policy: GovernancePolicy,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
    ) -> list[PolicyViolation]:
        """Check that resources carry required tags.

        Expected rules schema::

            {
                "required_tags": ["environment", "owner", "cost-center"],
            }
        """
        rules = policy.rules or {}
        required_tags = set(rules.get("required_tags", []))
        if not required_tags:
            return []

        violations: list[PolicyViolation] = []

        for resource in resources:
            tags = resource.get("tags") or {}
            missing = required_tags - set(tags.keys())
            if missing:
                violations.append(
                    PolicyViolation(
                        id=uuid.uuid4(),
                        policy_id=policy.id,
                        resource_id=resource.get("resource_id", "unknown"),
                        provider=resource.get("provider", "unknown"),
                        description=(
                            f"Resource {resource.get('resource_id')} is missing "
                            f"required tags: {sorted(missing)}."
                        ),
                        severity=policy.severity,
                    )
                )

        return violations

    # -- region_restriction --------------------------------------------

    @staticmethod
    def _eval_region_restriction(
        policy: GovernancePolicy,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
    ) -> list[PolicyViolation]:
        """Ensure resources are only in allowed regions.

        Expected rules schema::

            {
                "allowed_regions": ["us-east-1", "eu-west-1"],
            }
        """
        rules = policy.rules or {}
        allowed = set(rules.get("allowed_regions", []))
        if not allowed:
            return []

        violations: list[PolicyViolation] = []

        for resource in resources:
            region = resource.get("region", "")
            if region and region not in allowed:
                violations.append(
                    PolicyViolation(
                        id=uuid.uuid4(),
                        policy_id=policy.id,
                        resource_id=resource.get("resource_id", "unknown"),
                        provider=resource.get("provider", "unknown"),
                        description=(
                            f"Resource {resource.get('resource_id')} is in "
                            f"disallowed region '{region}'. Allowed: {sorted(allowed)}."
                        ),
                        severity=policy.severity,
                    )
                )

        return violations

    # -- service_allowlist ---------------------------------------------

    @staticmethod
    def _eval_service_allowlist(
        policy: GovernancePolicy,
        resources: list[dict[str, Any]],
        costs: list[dict[str, Any]],
    ) -> list[PolicyViolation]:
        """Ensure only approved cloud services are used.

        Expected rules schema::

            {
                "allowed_services": ["Amazon Elastic Compute Cloud", "Amazon S3"],
            }
        """
        rules = policy.rules or {}
        allowed = set(rules.get("allowed_services", []))
        if not allowed:
            return []

        # Check cost records for disallowed services
        seen_services: dict[str, str] = {}  # service -> provider

        for cost in costs:
            service = cost.get("service", "")
            if service and service not in allowed:
                if service not in seen_services:
                    seen_services[service] = cost.get("provider", "unknown")

        violations: list[PolicyViolation] = []
        for service, provider in seen_services.items():
            violations.append(
                PolicyViolation(
                    id=uuid.uuid4(),
                    policy_id=policy.id,
                    resource_id=f"service:{service}",
                    provider=provider,
                    description=(
                        f"Service '{service}' is not in the approved "
                        f"allow-list: {sorted(allowed)}."
                    ),
                    severity=policy.severity,
                )
            )

        return violations


# ---------------------------------------------------------------------------
# Module-level helpers
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
