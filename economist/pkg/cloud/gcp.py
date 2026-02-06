"""
GCP cloud provider integration for the Economist module.

Uses the Google Cloud Billing and Resource Manager SDKs to retrieve
cost data, list active resources, and produce optimization
recommendations.
"""

from __future__ import annotations

import logging
import re
from datetime import date
from typing import Any

from pkg.cloud.base import CloudProvider
from pkg.config import get_settings

# GCP project IDs: 6-30 chars, lowercase letters, digits, hyphens
_PROJECT_ID_RE = re.compile(r"^[a-z][a-z0-9\-]{4,28}[a-z0-9]$")

logger = logging.getLogger(__name__)


class GCPProvider(CloudProvider):
    """Google Cloud Platform cost and resource provider."""

    def __init__(self, settings=None):
        self._settings = settings or get_settings()
        # Validate project ID to prevent query injection in BigQuery
        pid = self._settings.gcp_project_id
        if pid and not _PROJECT_ID_RE.match(pid):
            raise ValueError(
                f"Invalid GCP project ID format: {pid!r}. "
                "Expected 6-30 lowercase alphanumeric/hyphen characters."
            )
        self._initialized = False
        self._billing_client = None
        self._resource_client = None
        self._recommender_client = None

    # ------------------------------------------------------------------
    # Lazy initialization
    # ------------------------------------------------------------------

    def _ensure_clients(self) -> None:
        """Lazily initialize GCP SDK clients."""
        if self._initialized:
            return

        try:
            from google.cloud import billing_v1
            from google.cloud import resourcemanager_v3
            from google.cloud import recommender_v1

            self._billing_client = billing_v1.CloudBillingClient()
            self._resource_client = resourcemanager_v3.ProjectsClient()
            self._recommender_client = recommender_v1.RecommenderClient()
            self._initialized = True
        except Exception as exc:
            logger.error("Failed to initialize GCP clients: %s", exc)
            raise

    # ------------------------------------------------------------------
    # CloudProvider interface
    # ------------------------------------------------------------------

    @property
    def name(self) -> str:
        return "gcp"

    async def get_costs(
        self,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Retrieve cost data from GCP BigQuery billing export.

        In production this queries the BigQuery billing export table.
        Here we use the Cloud Billing API to enumerate billing accounts
        and services, then query the BigQuery export for detailed costs.
        """
        try:
            self._ensure_clients()

            from google.cloud import bigquery

            bq_client = bigquery.Client(project=self._settings.gcp_project_id)

            # Standard billing export table naming convention
            query = f"""
                SELECT
                    service.description AS service,
                    sku.description AS sku,
                    location.region AS region,
                    usage_start_time,
                    cost,
                    currency,
                    usage.amount AS usage_amount,
                    usage.unit AS usage_unit,
                    project.id AS project_id,
                    labels
                FROM `{self._settings.gcp_project_id}.billing_export.gcp_billing_export_v1_*`
                WHERE DATE(usage_start_time) >= @start_date
                  AND DATE(usage_start_time) < @end_date
            """

            job_config = bigquery.QueryJobConfig(
                query_parameters=[
                    bigquery.ScalarQueryParameter(
                        "start_date", "DATE", start_date.isoformat()
                    ),
                    bigquery.ScalarQueryParameter(
                        "end_date", "DATE", end_date.isoformat()
                    ),
                ]
            )

            results = bq_client.query(query, job_config=job_config).result()

            costs: list[dict[str, Any]] = []
            for row in results:
                tags = {}
                if row.labels:
                    for label in row.labels:
                        tags[label["key"]] = label["value"]

                usage_time = row.usage_start_time
                cost_date = (
                    usage_time.date() if hasattr(usage_time, "date") else start_date
                )

                costs.append(
                    {
                        "provider": "gcp",
                        "service": row.service or "Unknown",
                        "resource_id": f"gcp:{row.service}:{row.sku}",
                        "resource_name": row.sku or row.service,
                        "cost_usd": float(row.cost or 0),
                        "currency": row.currency or "USD",
                        "usage_quantity": float(row.usage_amount or 0),
                        "usage_unit": row.usage_unit or "",
                        "region": row.region or "global",
                        "account_id": row.project_id or self._settings.gcp_project_id,
                        "tags": tags,
                        "date": cost_date.isoformat(),
                    }
                )

            logger.info("Fetched %d GCP cost records", len(costs))
            return costs

        except Exception as exc:
            logger.error("Failed to fetch GCP costs: %s", exc)
            return []

    async def get_resources(self) -> list[dict[str, Any]]:
        """List active resources via the GCP Cloud Asset Inventory."""
        try:
            self._ensure_clients()

            from google.cloud import asset_v1

            asset_client = asset_v1.AssetServiceClient()
            project_path = f"projects/{self._settings.gcp_project_id}"

            resources: list[dict[str, Any]] = []

            # List compute instances
            request = asset_v1.ListAssetsRequest(
                parent=project_path,
                asset_types=[
                    "compute.googleapis.com/Instance",
                    "sqladmin.googleapis.com/Instance",
                    "storage.googleapis.com/Bucket",
                ],
                content_type=asset_v1.ContentType.RESOURCE,
            )

            for asset in asset_client.list_assets(request=request):
                resource_data = asset.resource
                labels = {}
                if resource_data and resource_data.data:
                    labels = dict(resource_data.data.get("labels", {}))

                resources.append(
                    {
                        "resource_id": asset.name,
                        "resource_type": asset.asset_type,
                        "region": (
                            resource_data.location if resource_data else "unknown"
                        ),
                        "provider": "gcp",
                        "tags": labels,
                        "name": asset.name.split("/")[-1] if asset.name else "",
                        "state": (
                            resource_data.data.get("status", "")
                            if resource_data and resource_data.data
                            else ""
                        ),
                    }
                )

            logger.info("Discovered %d GCP resources", len(resources))
            return resources

        except Exception as exc:
            logger.error("Failed to list GCP resources: %s", exc)
            return []

    async def get_recommendations(self) -> list[dict[str, Any]]:
        """Fetch cost recommendations from the GCP Recommender API."""
        try:
            self._ensure_clients()

            recommendations: list[dict[str, Any]] = []

            # Recommender IDs relevant to cost optimization
            recommender_ids = [
                "google.compute.instance.MachineTypeRecommender",
                "google.compute.instance.IdleResourceRecommender",
                "google.compute.disk.IdleResourceRecommender",
                "google.compute.commitment.UsageCommitmentRecommender",
            ]

            # We need to iterate over zones; use a representative set
            zones = self._get_zones()

            for zone in zones:
                for recommender_id in recommender_ids:
                    try:
                        parent = (
                            f"projects/{self._settings.gcp_project_id}"
                            f"/locations/{zone}"
                            f"/recommenders/{recommender_id}"
                        )
                        recs = self._recommender_client.list_recommendations(
                            parent=parent
                        )

                        for rec in recs:
                            savings = self._extract_gcp_savings(rec)
                            recommendations.append(
                                {
                                    "provider": "gcp",
                                    "resource_id": (
                                        rec.content.operation_groups[0]
                                        .operations[0]
                                        .resource
                                        if rec.content and rec.content.operation_groups
                                        else "unknown"
                                    ),
                                    "resource_type": self._map_recommender_to_type(
                                        recommender_id
                                    ),
                                    "recommendation_type": self._map_recommender_to_rec_type(
                                        recommender_id
                                    ),
                                    "title": rec.description or recommender_id,
                                    "description": (rec.description or ""),
                                    "estimated_monthly_savings": savings,
                                    "confidence": self._priority_to_confidence(
                                        rec.priority
                                    ),
                                }
                            )
                    except Exception as inner_exc:
                        logger.debug(
                            "No recommendations for %s/%s: %s",
                            zone,
                            recommender_id,
                            inner_exc,
                        )

            logger.info("Fetched %d GCP recommendations", len(recommendations))
            return recommendations

        except Exception as exc:
            logger.error("Failed to fetch GCP recommendations: %s", exc)
            return []

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _get_zones(self) -> list[str]:
        """Return a representative set of GCP zones to query."""
        try:
            from google.cloud import compute_v1

            client = compute_v1.ZonesClient()
            zones = [z.name for z in client.list(project=self._settings.gcp_project_id)]
            return zones or ["us-central1-a"]
        except Exception:
            return ["us-central1-a", "us-east1-b", "europe-west1-b"]

    @staticmethod
    def _extract_gcp_savings(rec) -> float:
        """Extract estimated monthly savings from a GCP recommendation."""
        try:
            impact = rec.primary_impact
            if impact and impact.cost_projection:
                cost = impact.cost_projection.cost
                # cost is a google.type.Money message
                return abs(float(cost.units or 0) + float(cost.nanos or 0) / 1e9)
        except Exception:
            pass
        return 0.0

    @staticmethod
    def _map_recommender_to_type(recommender_id: str) -> str:
        mapping = {
            "google.compute.instance.MachineTypeRecommender": "compute:instance",
            "google.compute.instance.IdleResourceRecommender": "compute:instance",
            "google.compute.disk.IdleResourceRecommender": "compute:disk",
            "google.compute.commitment.UsageCommitmentRecommender": "compute:commitment",
        }
        return mapping.get(recommender_id, "unknown")

    @staticmethod
    def _map_recommender_to_rec_type(recommender_id: str) -> str:
        mapping = {
            "google.compute.instance.MachineTypeRecommender": "rightsizing",
            "google.compute.instance.IdleResourceRecommender": "idle_resource",
            "google.compute.disk.IdleResourceRecommender": "idle_resource",
            "google.compute.commitment.UsageCommitmentRecommender": "reserved_capacity",
        }
        return mapping.get(recommender_id, "general")

    @staticmethod
    def _priority_to_confidence(priority) -> float:
        """Map GCP priority enum to a confidence score."""
        priority_map = {
            "P1": 0.95,
            "P2": 0.85,
            "P3": 0.7,
            "P4": 0.5,
        }
        return priority_map.get(str(priority), 0.6)
