"""
Azure cloud provider integration for the Economist module.

Uses the Azure SDK to pull cost data from Azure Cost Management,
list active resources, and generate optimization recommendations.
"""

from __future__ import annotations

import logging
from datetime import date, timedelta
from typing import Any

from pkg.cloud.base import CloudProvider
from pkg.config import get_settings

logger = logging.getLogger(__name__)


class AzureProvider(CloudProvider):
    """Azure cost and resource provider."""

    def __init__(self, settings=None):
        self._settings = settings or get_settings()
        self._credential = None
        self._cost_client = None
        self._resource_client = None
        self._initialized = False

    # ------------------------------------------------------------------
    # Lazy initialization
    # ------------------------------------------------------------------

    def _ensure_clients(self) -> None:
        """Lazily create Azure SDK clients so import-time errors are avoided."""
        if self._initialized:
            return

        try:
            from azure.identity import ClientSecretCredential
            from azure.mgmt.costmanagement import CostManagementClient
            from azure.mgmt.resource import ResourceManagementClient

            self._credential = ClientSecretCredential(
                tenant_id=self._settings.azure_tenant_id,
                client_id=self._settings.azure_client_id,
                client_secret=self._settings.azure_client_secret,
            )
            self._cost_client = CostManagementClient(
                credential=self._credential,
                subscription_id=self._settings.azure_subscription_id,
            )
            self._resource_client = ResourceManagementClient(
                credential=self._credential,
                subscription_id=self._settings.azure_subscription_id,
            )
            self._initialized = True
        except Exception as exc:
            logger.error("Failed to initialize Azure clients: %s", exc)
            raise

    # ------------------------------------------------------------------
    # CloudProvider interface
    # ------------------------------------------------------------------

    @property
    def name(self) -> str:
        return "azure"

    async def get_costs(
        self,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Retrieve cost data from Azure Cost Management."""
        try:
            self._ensure_clients()

            scope = (
                f"/subscriptions/{self._settings.azure_subscription_id}"
            )

            # Build the query for cost data
            from azure.mgmt.costmanagement.models import (
                QueryDefinition,
                QueryDataset,
                QueryAggregation,
                QueryGrouping,
                QueryTimePeriod,
                TimeframeType,
                ExportType,
            )

            query = QueryDefinition(
                type=ExportType.ACTUAL_COST,
                timeframe=TimeframeType.CUSTOM,
                time_period=QueryTimePeriod(
                    from_property=start_date,
                    to=end_date,
                ),
                dataset=QueryDataset(
                    granularity="Daily",
                    aggregation={
                        "totalCost": QueryAggregation(
                            name="Cost", function="Sum"
                        ),
                    },
                    grouping=[
                        QueryGrouping(
                            type="Dimension", name="ServiceName"
                        ),
                        QueryGrouping(
                            type="Dimension", name="ResourceLocation"
                        ),
                    ],
                ),
            )

            result = self._cost_client.query.usage(scope=scope, parameters=query)

            costs: list[dict[str, Any]] = []
            columns = [col.name for col in result.columns] if result.columns else []

            for row in result.rows or []:
                row_dict = dict(zip(columns, row))
                cost_amount = float(row_dict.get("Cost", 0))
                service = row_dict.get("ServiceName", "Unknown")
                region = row_dict.get("ResourceLocation", "unknown")
                cost_date = row_dict.get("UsageDate", start_date)

                # Azure returns dates as integers YYYYMMDD
                if isinstance(cost_date, (int, float)):
                    cost_date_str = str(int(cost_date))
                    cost_date = date(
                        int(cost_date_str[:4]),
                        int(cost_date_str[4:6]),
                        int(cost_date_str[6:8]),
                    )

                costs.append(
                    {
                        "provider": "azure",
                        "service": service,
                        "resource_id": f"azure:{service}:{region}",
                        "resource_name": service,
                        "cost_usd": cost_amount,
                        "currency": row_dict.get("Currency", "USD"),
                        "usage_quantity": None,
                        "usage_unit": None,
                        "region": region,
                        "account_id": self._settings.azure_subscription_id,
                        "tags": None,
                        "date": (
                            cost_date.isoformat()
                            if isinstance(cost_date, date)
                            else str(cost_date)
                        ),
                    }
                )

            logger.info("Fetched %d Azure cost records", len(costs))
            return costs

        except Exception as exc:
            logger.error("Failed to fetch Azure costs: %s", exc)
            return []

    async def get_resources(self) -> list[dict[str, Any]]:
        """List active resources in the Azure subscription."""
        try:
            self._ensure_clients()

            resources: list[dict[str, Any]] = []
            for resource in self._resource_client.resources.list():
                resources.append(
                    {
                        "resource_id": resource.id,
                        "resource_type": resource.type,
                        "region": resource.location or "unknown",
                        "provider": "azure",
                        "tags": dict(resource.tags) if resource.tags else {},
                        "name": resource.name,
                        "kind": resource.kind,
                    }
                )

            logger.info("Discovered %d Azure resources", len(resources))
            return resources

        except Exception as exc:
            logger.error("Failed to list Azure resources: %s", exc)
            return []

    async def get_recommendations(self) -> list[dict[str, Any]]:
        """Fetch cost optimization recommendations from Azure Advisor."""
        try:
            self._ensure_clients()

            from azure.mgmt.advisor import AdvisorManagementClient

            advisor = AdvisorManagementClient(
                credential=self._credential,
                subscription_id=self._settings.azure_subscription_id,
            )

            recommendations: list[dict[str, Any]] = []
            for rec in advisor.recommendations.list(filter="Category eq 'Cost'"):
                impact = rec.impact or "Medium"
                estimated_savings = self._extract_azure_savings(rec)
                confidence = (
                    0.9 if impact == "High" else 0.7 if impact == "Medium" else 0.5
                )

                recommendations.append(
                    {
                        "provider": "azure",
                        "resource_id": rec.resource_metadata.resource_id
                        if rec.resource_metadata
                        else "unknown",
                        "resource_type": rec.impacted_field or "unknown",
                        "recommendation_type": self._map_azure_rec_type(rec.category),
                        "title": rec.short_description.problem if rec.short_description else "Azure recommendation",
                        "description": (
                            rec.short_description.solution
                            if rec.short_description
                            else ""
                        ),
                        "estimated_monthly_savings": estimated_savings,
                        "confidence": confidence,
                    }
                )

            logger.info("Fetched %d Azure recommendations", len(recommendations))
            return recommendations

        except Exception as exc:
            logger.error("Failed to fetch Azure recommendations: %s", exc)
            return []

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _extract_azure_savings(rec) -> float:
        """Try to extract estimated savings from an Advisor recommendation."""
        try:
            extended = rec.extended_properties or {}
            savings_str = extended.get("annualSavingsAmount", "0")
            annual = float(savings_str)
            return round(annual / 12.0, 2)
        except (ValueError, TypeError):
            return 0.0

    @staticmethod
    def _map_azure_rec_type(category: str | None) -> str:
        """Map Azure Advisor category to our internal recommendation type."""
        mapping = {
            "Cost": "cost_reduction",
            "HighAvailability": "reliability",
            "Performance": "rightsizing",
            "Security": "security",
            "OperationalExcellence": "operational",
        }
        return mapping.get(category or "", "general")
