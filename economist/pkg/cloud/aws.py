"""
AWS cloud provider integration for the Economist module.

Uses boto3 to pull cost data from AWS Cost Explorer, list active resources
via EC2/RDS/S3, and fetch optimization recommendations from Cost Explorer
and Compute Optimizer.
"""

from __future__ import annotations

import logging
from datetime import date, datetime
from typing import Any

import boto3
from botocore.exceptions import BotoCoreError, ClientError

from pkg.cloud.base import CloudProvider
from pkg.config import get_settings

logger = logging.getLogger(__name__)


class AWSProvider(CloudProvider):
    """AWS cost and resource provider."""

    def __init__(self, settings=None):
        self._settings = settings or get_settings()
        self._session = self._build_session()

    # ------------------------------------------------------------------
    # CloudProvider interface
    # ------------------------------------------------------------------

    @property
    def name(self) -> str:
        return "aws"

    async def get_costs(
        self,
        start_date: date,
        end_date: date,
    ) -> list[dict[str, Any]]:
        """Pull cost data from AWS Cost Explorer."""
        try:
            ce = self._session.client("ce", region_name=self._settings.aws_region)

            results: list[dict[str, Any]] = []
            next_token: str | None = None

            while True:
                kwargs: dict[str, Any] = {
                    "TimePeriod": {
                        "Start": start_date.isoformat(),
                        "End": end_date.isoformat(),
                    },
                    "Granularity": "DAILY",
                    "Metrics": ["UnblendedCost", "UsageQuantity"],
                    "GroupBy": [
                        {"Type": "DIMENSION", "Key": "SERVICE"},
                        {"Type": "DIMENSION", "Key": "REGION"},
                    ],
                }
                if next_token:
                    kwargs["NextPageToken"] = next_token

                response = ce.get_cost_and_usage(**kwargs)

                for result_by_time in response.get("ResultsByTime", []):
                    period_start = result_by_time["TimePeriod"]["Start"]
                    for group in result_by_time.get("Groups", []):
                        keys = group.get("Keys", [])
                        service = keys[0] if len(keys) > 0 else "Unknown"
                        region = keys[1] if len(keys) > 1 else "global"
                        metrics = group.get("Metrics", {})
                        cost_amount = float(
                            metrics.get("UnblendedCost", {}).get("Amount", 0)
                        )
                        cost_unit = metrics.get("UnblendedCost", {}).get("Unit", "USD")
                        usage_qty = float(
                            metrics.get("UsageQuantity", {}).get("Amount", 0)
                        )
                        usage_unit = metrics.get("UsageQuantity", {}).get("Unit", "")

                        results.append(
                            {
                                "provider": "aws",
                                "service": service,
                                "resource_id": f"aws:{service}:{region}",
                                "resource_name": service,
                                "cost_usd": cost_amount,
                                "currency": cost_unit,
                                "usage_quantity": usage_qty,
                                "usage_unit": usage_unit,
                                "region": region,
                                "account_id": self._get_account_id(),
                                "tags": None,
                                "date": period_start,
                            }
                        )

                next_token = response.get("NextPageToken")
                if not next_token:
                    break

            logger.info("Fetched %d AWS cost records", len(results))
            return results

        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to fetch AWS costs: %s", exc)
            return []

    async def get_resources(self) -> list[dict[str, Any]]:
        """List active EC2 instances, RDS instances, and S3 buckets."""
        resources: list[dict[str, Any]] = []

        resources.extend(self._list_ec2_instances())
        resources.extend(self._list_rds_instances())
        resources.extend(self._list_s3_buckets())

        logger.info("Discovered %d AWS resources", len(resources))
        return resources

    async def get_recommendations(self) -> list[dict[str, Any]]:
        """Fetch AWS optimization recommendations."""
        recommendations: list[dict[str, Any]] = []

        recommendations.extend(self._get_ce_recommendations())
        recommendations.extend(self._get_compute_optimizer_recommendations())

        logger.info("Fetched %d AWS recommendations", len(recommendations))
        return recommendations

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _build_session(self) -> boto3.Session:
        kwargs: dict[str, Any] = {
            "region_name": self._settings.aws_region,
        }
        if self._settings.aws_access_key_id:
            kwargs["aws_access_key_id"] = self._settings.aws_access_key_id
        if self._settings.aws_secret_access_key.get_secret_value():
            kwargs["aws_secret_access_key"] = (
                self._settings.aws_secret_access_key.get_secret_value()
            )
        return boto3.Session(**kwargs)

    def _get_account_id(self) -> str:
        try:
            sts = self._session.client("sts")
            return sts.get_caller_identity()["Account"]
        except Exception:
            return "unknown"

    # -- EC2 -----------------------------------------------------------

    def _list_ec2_instances(self) -> list[dict[str, Any]]:
        try:
            ec2 = self._session.client("ec2", region_name=self._settings.aws_region)
            paginator = ec2.get_paginator("describe_instances")
            resources: list[dict[str, Any]] = []

            for page in paginator.paginate():
                for reservation in page.get("Reservations", []):
                    for instance in reservation.get("Instances", []):
                        tags = {t["Key"]: t["Value"] for t in instance.get("Tags", [])}
                        resources.append(
                            {
                                "resource_id": instance["InstanceId"],
                                "resource_type": "ec2:instance",
                                "region": self._settings.aws_region,
                                "provider": "aws",
                                "tags": tags,
                                "instance_type": instance.get("InstanceType"),
                                "state": instance.get("State", {}).get("Name"),
                                "launch_time": (
                                    instance["LaunchTime"].isoformat()
                                    if isinstance(instance.get("LaunchTime"), datetime)
                                    else str(instance.get("LaunchTime", ""))
                                ),
                            }
                        )
            return resources
        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to list EC2 instances: %s", exc)
            return []

    # -- RDS -----------------------------------------------------------

    def _list_rds_instances(self) -> list[dict[str, Any]]:
        try:
            rds = self._session.client("rds", region_name=self._settings.aws_region)
            paginator = rds.get_paginator("describe_db_instances")
            resources: list[dict[str, Any]] = []

            for page in paginator.paginate():
                for db in page.get("DBInstances", []):
                    resources.append(
                        {
                            "resource_id": db["DBInstanceIdentifier"],
                            "resource_type": "rds:instance",
                            "region": self._settings.aws_region,
                            "provider": "aws",
                            "tags": {
                                t["Key"]: t["Value"] for t in db.get("TagList", [])
                            },
                            "instance_type": db.get("DBInstanceClass"),
                            "engine": db.get("Engine"),
                            "state": db.get("DBInstanceStatus"),
                        }
                    )
            return resources
        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to list RDS instances: %s", exc)
            return []

    # -- S3 ------------------------------------------------------------

    def _list_s3_buckets(self) -> list[dict[str, Any]]:
        try:
            s3 = self._session.client("s3")
            response = s3.list_buckets()
            resources: list[dict[str, Any]] = []

            for bucket in response.get("Buckets", []):
                resources.append(
                    {
                        "resource_id": bucket["Name"],
                        "resource_type": "s3:bucket",
                        "region": "global",
                        "provider": "aws",
                        "tags": {},
                        "creation_date": (
                            bucket["CreationDate"].isoformat()
                            if isinstance(bucket.get("CreationDate"), datetime)
                            else str(bucket.get("CreationDate", ""))
                        ),
                    }
                )
            return resources
        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to list S3 buckets: %s", exc)
            return []

    # -- Recommendations -----------------------------------------------

    def _get_ce_recommendations(self) -> list[dict[str, Any]]:
        """Reservation purchase recommendations from Cost Explorer."""
        try:
            ce = self._session.client("ce", region_name=self._settings.aws_region)
            response = ce.get_reservation_purchase_recommendation(
                Service="Amazon Elastic Compute Cloud - Compute",
                LookbackPeriodInDays="SIXTY_DAYS",
                TermInYears="ONE_YEAR",
                PaymentOption="NO_UPFRONT",
            )

            recs: list[dict[str, Any]] = []
            for detail in (
                response.get("Recommendations", [{}])[0].get(
                    "RecommendationDetails", []
                )
                if response.get("Recommendations")
                else []
            ):
                estimated_savings = float(
                    detail.get("EstimatedMonthlySavingsAmount", 0)
                )
                recs.append(
                    {
                        "provider": "aws",
                        "resource_id": detail.get("InstanceDetails", {})
                        .get("EC2InstanceDetails", {})
                        .get("InstanceType", "unknown"),
                        "resource_type": "ec2:instance",
                        "recommendation_type": "reserved_capacity",
                        "title": "Purchase Reserved Instance",
                        "description": (
                            f"Consider purchasing a Reserved Instance for "
                            f"{detail.get('InstanceDetails', {}).get('EC2InstanceDetails', {}).get('InstanceType', 'N/A')} "
                            f"to save ~${estimated_savings:.2f}/month."
                        ),
                        "estimated_monthly_savings": estimated_savings,
                        "confidence": 0.8,
                    }
                )
            return recs
        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to get CE recommendations: %s", exc)
            return []

    def _get_compute_optimizer_recommendations(self) -> list[dict[str, Any]]:
        """Right-sizing recommendations from AWS Compute Optimizer."""
        try:
            co = self._session.client(
                "compute-optimizer", region_name=self._settings.aws_region
            )
            response = co.get_ec2_instance_recommendations()

            recs: list[dict[str, Any]] = []
            for rec in response.get("instanceRecommendations", []):
                finding = rec.get("finding", "")
                if finding in ("OVER_PROVISIONED", "UNDER_PROVISIONED"):
                    current_type = rec.get("currentInstanceType", "unknown")
                    options = rec.get("recommendationOptions", [])
                    suggested_type = (
                        options[0].get("instanceType", "unknown")
                        if options
                        else "unknown"
                    )
                    estimated_savings = float(
                        options[0]
                        .get("projectedUtilizationMetrics", [{}])[0]
                        .get("value", 0)
                        if options
                        else 0
                    )

                    recs.append(
                        {
                            "provider": "aws",
                            "resource_id": rec.get("instanceArn", "unknown"),
                            "resource_type": "ec2:instance",
                            "recommendation_type": "rightsizing",
                            "title": f"Rightsize {current_type} to {suggested_type}",
                            "description": (
                                f"Instance {rec.get('instanceArn', 'N/A')} is "
                                f"{finding.lower().replace('_', '-')}. "
                                f"Consider changing from {current_type} to "
                                f"{suggested_type}."
                            ),
                            "estimated_monthly_savings": estimated_savings,
                            "confidence": 0.75,
                        }
                    )
            return recs
        except (BotoCoreError, ClientError) as exc:
            logger.error("Failed to get Compute Optimizer recommendations: %s", exc)
            return []
