"""
Abstract base class for multi-cloud provider integrations.

Each concrete provider (AWS, Azure, GCP) must implement this interface
so the ingestion collector and optimizer can work provider-agnostically.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from datetime import date
from typing import Any


class CloudProvider(ABC):
    """Interface that every cloud-cost provider adapter must satisfy."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Return the canonical provider name (e.g. ``'aws'``)."""
        ...

    @abstractmethod
    async def get_costs(
        self,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Retrieve cost line items for the given date range.

        Each returned dict should match the ``CloudCost`` model fields:
        ``provider``, ``service``, ``resource_id``, ``resource_name``,
        ``cost_usd``, ``currency``, ``usage_quantity``, ``usage_unit``,
        ``region``, ``account_id``, ``tags``, ``date``.
        """
        ...

    @abstractmethod
    async def get_resources(self) -> list[dict[str, Any]]:
        """List currently active resources visible to this provider.

        Returns a list of dicts with at least:
        ``resource_id``, ``resource_type``, ``region``, ``tags``,
        ``provider``.
        """
        ...

    @abstractmethod
    async def get_recommendations(self) -> list[dict[str, Any]]:
        """Retrieve cost-optimization recommendations from the provider.

        Each dict should contain at least:
        ``provider``, ``resource_id``, ``resource_type``,
        ``recommendation_type``, ``title``, ``description``,
        ``estimated_monthly_savings``, ``confidence``.
        """
        ...
