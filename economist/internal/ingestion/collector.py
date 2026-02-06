"""
Cost data collection engine.

Manages a registry of cloud provider adapters and orchestrates parallel
cost ingestion, storing results in the database.
"""

from __future__ import annotations

import asyncio
import logging
from datetime import date, datetime, timedelta
from typing import Any

from sqlalchemy.orm import Session

from pkg.cloud.base import CloudProvider
from pkg.database import CloudCost, get_session

logger = logging.getLogger(__name__)


class CostCollector:
    """Orchestrates cost data collection across all registered providers."""

    def __init__(self) -> None:
        self._providers: dict[str, CloudProvider] = {}

    # ------------------------------------------------------------------
    # Provider registry
    # ------------------------------------------------------------------

    def register_provider(self, name: str, provider: CloudProvider) -> None:
        """Register a cloud provider adapter.

        Parameters
        ----------
        name:
            Canonical identifier for the provider (e.g. ``"aws"``).
        provider:
            An instance implementing :class:`CloudProvider`.
        """
        self._providers[name] = provider
        logger.info("Registered cloud provider: %s", name)

    def get_provider(self, name: str) -> CloudProvider | None:
        """Retrieve a registered provider by name."""
        return self._providers.get(name)

    @property
    def provider_names(self) -> list[str]:
        """Return the names of all registered providers."""
        return list(self._providers.keys())

    # ------------------------------------------------------------------
    # Collection
    # ------------------------------------------------------------------

    async def collect_all(
        self,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Collect cost data from every registered provider in parallel.

        Parameters
        ----------
        start_date:
            Beginning of the date range (inclusive).
        end_date:
            End of the date range (exclusive).

        Returns
        -------
        list[dict]
            Aggregated cost records from all providers.
        """
        if not self._providers:
            logger.warning("No providers registered; nothing to collect.")
            return []

        tasks = [
            self._collect_from_provider(name, provider, start_date, end_date)
            for name, provider in self._providers.items()
        ]

        results = await asyncio.gather(*tasks, return_exceptions=True)

        all_costs: list[dict[str, Any]] = []
        for name, result in zip(self._providers.keys(), results):
            if isinstance(result, Exception):
                logger.error(
                    "Collection from %s failed: %s", name, result
                )
            elif isinstance(result, list):
                all_costs.extend(result)
                logger.info(
                    "Collected %d records from %s", len(result), name
                )

        return all_costs

    async def collect_from(
        self,
        provider_name: str,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Collect cost data from a single named provider.

        Parameters
        ----------
        provider_name:
            The registered name of the provider.
        start_date, end_date:
            Date range for collection.

        Returns
        -------
        list[dict]

        Raises
        ------
        ValueError
            If the provider is not registered.
        """
        provider = self._providers.get(provider_name)
        if provider is None:
            raise ValueError(f"Unknown provider: {provider_name}")
        return await self._collect_from_provider(
            provider_name, provider, start_date, end_date
        )

    # ------------------------------------------------------------------
    # Storage
    # ------------------------------------------------------------------

    def store_costs(
        self,
        costs: list[dict[str, Any]],
        db: Session,
    ) -> int:
        """Batch-insert cost records into the database.

        Parameters
        ----------
        costs:
            List of cost dicts matching :class:`CloudCost` fields.
        db:
            An active SQLAlchemy session.

        Returns
        -------
        int
            Number of records inserted.
        """
        if not costs:
            return 0

        objects: list[CloudCost] = []
        for cost in costs:
            cost_date = cost.get("date")
            if isinstance(cost_date, str):
                cost_date = date.fromisoformat(cost_date[:10])
            elif isinstance(cost_date, datetime):
                cost_date = cost_date.date()

            obj = CloudCost(
                provider=cost.get("provider", "unknown"),
                service=cost.get("service", "unknown"),
                resource_id=cost.get("resource_id", ""),
                resource_name=cost.get("resource_name"),
                cost_usd=float(cost.get("cost_usd", 0)),
                currency=cost.get("currency", "USD"),
                usage_quantity=cost.get("usage_quantity"),
                usage_unit=cost.get("usage_unit"),
                region=cost.get("region"),
                account_id=cost.get("account_id"),
                tags=cost.get("tags"),
                date=cost_date,
            )
            objects.append(obj)

        # Batch insert
        batch_size = 500
        inserted = 0
        for i in range(0, len(objects), batch_size):
            batch = objects[i : i + batch_size]
            db.bulk_save_objects(batch)
            inserted += len(batch)

        db.commit()
        logger.info("Stored %d cost records in database", inserted)
        return inserted

    # ------------------------------------------------------------------
    # Scheduling helpers
    # ------------------------------------------------------------------

    def schedule_collection(
        self,
        lookback_days: int = 1,
    ) -> dict[str, Any]:
        """Return the parameters for a periodic collection job.

        This returns a configuration dict that a task scheduler (e.g.
        Celery beat) can use.  It does not start a scheduler itself.

        Parameters
        ----------
        lookback_days:
            How many days back to collect on each run.

        Returns
        -------
        dict
            Scheduling metadata.
        """
        end_date = date.today()
        start_date = end_date - timedelta(days=lookback_days)
        return {
            "task": "collect_costs",
            "start_date": start_date.isoformat(),
            "end_date": end_date.isoformat(),
            "providers": self.provider_names,
            "lookback_days": lookback_days,
            "schedule": "daily",
        }

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    @staticmethod
    async def _collect_from_provider(
        name: str,
        provider: CloudProvider,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Collect costs from a single provider with error isolation."""
        logger.info(
            "Collecting costs from %s (%s to %s)",
            name,
            start_date,
            end_date,
        )
        try:
            return await provider.get_costs(start_date, end_date)
        except Exception as exc:
            logger.error("Error collecting from %s: %s", name, exc)
            raise
