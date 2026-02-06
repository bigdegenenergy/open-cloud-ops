# Economist API Reference

**Base URL:** `http://localhost:8081`
**Version:** v1
**Protocol:** HTTP/REST
**Content-Type:** `application/json`
**Framework:** Python FastAPI (auto-generated OpenAPI docs at `/docs`)

---

## Table of Contents

1. [Overview](#overview)
2. [Health](#health)
3. [Cost Management](#cost-management)
4. [Optimization Recommendations](#optimization-recommendations)
5. [Governance Policies](#governance-policies)
6. [Policy Violations](#policy-violations)
7. [Dashboard](#dashboard)
8. [Error Handling](#error-handling)
9. [Data Models](#data-models)

---

## Overview

Economist is the FinOps Core module of Open Cloud Ops. It provides multi-cloud cost management capabilities including cost data ingestion from AWS, Azure, and GCP, optimization recommendations, and governance policy enforcement.

**Key characteristics:**

- Integrates with AWS Cost Explorer, Azure Cost Management, and GCP Billing APIs
- Stores normalized cost line items with provider, service, resource, region, and tag metadata
- Generates optimization recommendations (idle resources, rightsizing, spot/reserved instances)
- Enforces governance policies with automated violation detection
- Built with FastAPI -- interactive API docs available at `http://localhost:8081/docs`

---

## Health

### GET /health

Returns the health status of the Economist service.

**Request:**

```bash
curl http://localhost:8081/health
```

**Response (200 OK):**

```json
{
  "status": "healthy",
  "service": "economist"
}
```

---

## Cost Management

### GET /api/v1/costs/summary

Get an aggregated summary of cloud costs across all providers.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | date (YYYY-MM-DD) | 30 days ago | Start date |
| `to` | date (YYYY-MM-DD) | today | End date |
| `provider` | string | (all) | Filter by provider: `aws`, `azure`, `gcp` |

**Request:**

```bash
curl "http://localhost:8081/api/v1/costs/summary?from=2026-01-01&to=2026-02-01"
```

**Response (200 OK):**

```json
{
  "summary": {
    "total_cost_usd": 45230.50,
    "currency": "USD",
    "period": {
      "from": "2026-01-01",
      "to": "2026-02-01"
    },
    "by_provider": [
      {
        "provider": "aws",
        "cost_usd": 28450.30,
        "percentage": 62.9,
        "service_count": 24,
        "resource_count": 312
      },
      {
        "provider": "azure",
        "cost_usd": 12380.20,
        "percentage": 27.4,
        "service_count": 15,
        "resource_count": 178
      },
      {
        "provider": "gcp",
        "cost_usd": 4400.00,
        "percentage": 9.7,
        "service_count": 8,
        "resource_count": 65
      }
    ]
  },
  "providers": ["aws", "azure", "gcp"]
}
```

### GET /api/v1/costs/breakdown

Get a detailed cost breakdown by a specified dimension.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | date | 30 days ago | Start date |
| `to` | date | today | End date |
| `provider` | string | (all) | Filter by provider |
| `group_by` | string | `service` | Grouping: `service`, `region`, `account`, `tag` |
| `tag_key` | string | (none) | When `group_by=tag`, the tag key to group by |
| `limit` | integer | 20 | Maximum number of results |

**Request:**

```bash
curl "http://localhost:8081/api/v1/costs/breakdown?from=2026-01-01&to=2026-02-01&provider=aws&group_by=service&limit=5"
```

**Response (200 OK):**

```json
{
  "breakdown": [
    {
      "group": "Amazon EC2",
      "cost_usd": 12340.50,
      "percentage": 43.4,
      "resource_count": 85,
      "change_pct": 5.2,
      "change_direction": "up"
    },
    {
      "group": "Amazon RDS",
      "cost_usd": 5670.20,
      "percentage": 19.9,
      "resource_count": 12,
      "change_pct": -2.1,
      "change_direction": "down"
    },
    {
      "group": "Amazon S3",
      "cost_usd": 3210.00,
      "percentage": 11.3,
      "resource_count": 45,
      "change_pct": 0.8,
      "change_direction": "up"
    },
    {
      "group": "Amazon EKS",
      "cost_usd": 2890.40,
      "percentage": 10.2,
      "resource_count": 6,
      "change_pct": 12.5,
      "change_direction": "up"
    },
    {
      "group": "AWS Lambda",
      "cost_usd": 1230.80,
      "percentage": 4.3,
      "resource_count": 78,
      "change_pct": -8.3,
      "change_direction": "down"
    }
  ],
  "period": {
    "from": "2026-01-01",
    "to": "2026-02-01"
  },
  "provider": "aws",
  "group_by": "service",
  "total_cost_usd": 28450.30
}
```

### GET /api/v1/costs/trend

Get cost trend data over time for charting.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | date | 90 days ago | Start date |
| `to` | date | today | End date |
| `interval` | string | `day` | Aggregation interval: `day`, `week`, `month` |
| `provider` | string | (all) | Filter by provider |

**Request:**

```bash
curl "http://localhost:8081/api/v1/costs/trend?from=2026-01-01&to=2026-01-08&interval=day&provider=aws"
```

**Response (200 OK):**

```json
{
  "trend": [
    {"date": "2026-01-01", "cost_usd": 918.40, "resource_count": 312},
    {"date": "2026-01-02", "cost_usd": 945.20, "resource_count": 314},
    {"date": "2026-01-03", "cost_usd": 902.10, "resource_count": 310},
    {"date": "2026-01-04", "cost_usd": 678.30, "resource_count": 305},
    {"date": "2026-01-05", "cost_usd": 652.80, "resource_count": 305},
    {"date": "2026-01-06", "cost_usd": 935.60, "resource_count": 315},
    {"date": "2026-01-07", "cost_usd": 960.10, "resource_count": 318}
  ],
  "interval": "day",
  "provider": "aws"
}
```

### GET /api/v1/costs/forecast

Get a cost forecast based on historical trends.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `horizon_days` | integer | 30 | Number of days to forecast |
| `provider` | string | (all) | Filter by provider |

**Request:**

```bash
curl "http://localhost:8081/api/v1/costs/forecast?horizon_days=30&provider=aws"
```

**Response (200 OK):**

```json
{
  "forecast": {
    "provider": "aws",
    "horizon_days": 30,
    "predicted_cost_usd": 29120.00,
    "confidence_interval": {
      "lower_usd": 27200.00,
      "upper_usd": 31040.00
    },
    "trend": "increasing",
    "trend_pct": 2.4,
    "daily_forecast": [
      {"date": "2026-02-07", "predicted_cost_usd": 970.50},
      {"date": "2026-02-08", "predicted_cost_usd": 972.30},
      {"date": "2026-02-09", "predicted_cost_usd": 965.10}
    ]
  },
  "based_on": {
    "from": "2025-11-08",
    "to": "2026-02-06",
    "data_points": 91
  }
}
```

---

## Optimization Recommendations

### GET /api/v1/costs/recommendations

Get cost optimization recommendations across all providers.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `provider` | string | (all) | Filter by provider: `aws`, `azure`, `gcp` |
| `type` | string | (all) | Filter by type: `idle_resources`, `rightsizing`, `spot_instances`, `reserved_capacity` |
| `status` | string | `open` | Filter by status: `open`, `accepted`, `dismissed` |
| `min_savings` | float | 0 | Minimum estimated monthly savings in USD |
| `limit` | integer | 50 | Maximum number of results |

**Request:**

```bash
curl "http://localhost:8081/api/v1/costs/recommendations?provider=aws&status=open&min_savings=10"
```

**Response (200 OK):**

```json
{
  "recommendations": [
    {
      "id": "rec-uuid-001",
      "provider": "aws",
      "resource_id": "i-0abc123def456789",
      "resource_type": "EC2 Instance",
      "recommendation_type": "rightsizing",
      "title": "Downsize m5.2xlarge to m5.xlarge in us-east-1",
      "description": "Instance i-0abc123def456789 has averaged 12% CPU utilization over the past 30 days. Downsizing from m5.2xlarge ($0.384/hr) to m5.xlarge ($0.192/hr) would save approximately $138.24/month with minimal performance impact.",
      "estimated_monthly_savings": 138.24,
      "confidence": 0.92,
      "status": "open",
      "created_at": "2026-02-05T08:00:00Z",
      "resolved_at": null
    },
    {
      "id": "rec-uuid-002",
      "provider": "aws",
      "resource_id": "i-0def456abc789012",
      "resource_type": "EC2 Instance",
      "recommendation_type": "idle_resources",
      "title": "Terminate idle instance in eu-west-1",
      "description": "Instance i-0def456abc789012 (t3.medium) has had 0% CPU utilization and no network traffic for 21 days. It appears to be unused and can be safely terminated, saving $30.37/month.",
      "estimated_monthly_savings": 30.37,
      "confidence": 0.98,
      "status": "open",
      "created_at": "2026-02-04T12:00:00Z",
      "resolved_at": null
    },
    {
      "id": "rec-uuid-003",
      "provider": "aws",
      "resource_id": "account-level",
      "resource_type": "EC2 Reservation",
      "recommendation_type": "reserved_capacity",
      "title": "Purchase Reserved Instances for 8 stable m5.xlarge instances",
      "description": "8 m5.xlarge instances in us-east-1 have been running continuously for 90+ days. Purchasing 1-year No Upfront Reserved Instances would save approximately $4,608/year ($384/month).",
      "estimated_monthly_savings": 384.00,
      "confidence": 0.88,
      "status": "open",
      "created_at": "2026-02-03T06:00:00Z",
      "resolved_at": null
    }
  ],
  "summary": {
    "total_recommendations": 3,
    "total_estimated_savings_usd": 552.61,
    "by_type": {
      "idle_resources": 1,
      "rightsizing": 1,
      "reserved_capacity": 1
    }
  },
  "categories": [
    "idle_resources",
    "rightsizing",
    "spot_instances",
    "reserved_capacity"
  ]
}
```

### PUT /api/v1/costs/recommendations/{recommendation_id}

Update the status of a recommendation (accept or dismiss).

**Request:**

```bash
curl -X PUT http://localhost:8081/api/v1/costs/recommendations/rec-uuid-001 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "accepted"
  }'
```

**Response (200 OK):**

```json
{
  "id": "rec-uuid-001",
  "status": "accepted",
  "resolved_at": "2026-02-06T14:30:00Z"
}
```

---

## Governance Policies

### GET /api/v1/governance/policies

List all governance policies.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `policy_type` | string | (all) | Filter by type: `budget_limit`, `tag_enforcement`, `region_restriction`, `resource_approval` |
| `severity` | string | (all) | Filter by severity: `info`, `warning`, `critical` |
| `enabled` | boolean | (all) | Filter by enabled status |

**Request:**

```bash
curl "http://localhost:8081/api/v1/governance/policies?enabled=true"
```

**Response (200 OK):**

```json
{
  "policies": [
    {
      "id": "pol-uuid-001",
      "name": "Mandatory Cost Center Tagging",
      "description": "All resources must have a 'cost-center' tag for cost allocation purposes.",
      "policy_type": "tag_enforcement",
      "rules": {
        "required_tags": ["cost-center", "environment", "owner"],
        "applies_to": ["aws", "azure", "gcp"],
        "resource_types": ["*"]
      },
      "severity": "warning",
      "enabled": true,
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-15T10:00:00Z"
    },
    {
      "id": "pol-uuid-002",
      "name": "Production Region Restriction",
      "description": "Production workloads must only run in approved regions for data residency compliance.",
      "policy_type": "region_restriction",
      "rules": {
        "allowed_regions": ["us-east-1", "us-west-2", "eu-west-1"],
        "applies_to": ["aws"],
        "environment_tag": "production"
      },
      "severity": "critical",
      "enabled": true,
      "created_at": "2026-01-10T08:00:00Z",
      "updated_at": "2026-01-20T14:00:00Z"
    },
    {
      "id": "pol-uuid-003",
      "name": "Monthly Budget Limit per Account",
      "description": "Each AWS account must not exceed $10,000/month in total spend.",
      "policy_type": "budget_limit",
      "rules": {
        "limit_usd": 10000.00,
        "period": "monthly",
        "applies_to": ["aws"],
        "scope": "account"
      },
      "severity": "critical",
      "enabled": true,
      "created_at": "2026-01-05T09:00:00Z",
      "updated_at": "2026-01-05T09:00:00Z"
    }
  ],
  "total": 3
}
```

### POST /api/v1/governance/policies

Create a new governance policy.

**Request:**

```bash
curl -X POST http://localhost:8081/api/v1/governance/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "No Unencrypted Storage",
    "description": "All storage resources must have encryption enabled.",
    "policy_type": "resource_approval",
    "rules": {
      "resource_types": ["s3_bucket", "ebs_volume", "rds_instance"],
      "required_properties": {"encryption": true},
      "applies_to": ["aws"]
    },
    "severity": "critical",
    "enabled": true
  }'
```

**Response (201 Created):**

```json
{
  "id": "pol-uuid-004",
  "name": "No Unencrypted Storage",
  "description": "All storage resources must have encryption enabled.",
  "policy_type": "resource_approval",
  "rules": {
    "resource_types": ["s3_bucket", "ebs_volume", "rds_instance"],
    "required_properties": {"encryption": true},
    "applies_to": ["aws"]
  },
  "severity": "critical",
  "enabled": true,
  "created_at": "2026-02-06T15:00:00Z",
  "updated_at": "2026-02-06T15:00:00Z"
}
```

### PUT /api/v1/governance/policies/{policy_id}

Update an existing governance policy.

**Request:**

```bash
curl -X PUT http://localhost:8081/api/v1/governance/policies/pol-uuid-001 \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "critical",
    "rules": {
      "required_tags": ["cost-center", "environment", "owner", "project"],
      "applies_to": ["aws", "azure", "gcp"],
      "resource_types": ["*"]
    }
  }'
```

**Response (200 OK):**

```json
{
  "id": "pol-uuid-001",
  "name": "Mandatory Cost Center Tagging",
  "severity": "critical",
  "rules": {
    "required_tags": ["cost-center", "environment", "owner", "project"],
    "applies_to": ["aws", "azure", "gcp"],
    "resource_types": ["*"]
  },
  "updated_at": "2026-02-06T15:10:00Z"
}
```

### DELETE /api/v1/governance/policies/{policy_id}

Delete a governance policy.

**Request:**

```bash
curl -X DELETE http://localhost:8081/api/v1/governance/policies/pol-uuid-003
```

**Response (204 No Content):**

No response body.

---

## Policy Violations

### GET /api/v1/governance/violations

List detected policy violations.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `policy_id` | string | (all) | Filter by policy ID |
| `provider` | string | (all) | Filter by provider |
| `severity` | string | (all) | Filter by severity: `info`, `warning`, `critical` |
| `resolved` | boolean | `false` | Include resolved violations |
| `from` | date | 30 days ago | Start date |
| `to` | date | today | End date |
| `limit` | integer | 50 | Maximum number of results |

**Request:**

```bash
curl "http://localhost:8081/api/v1/governance/violations?severity=critical&resolved=false"
```

**Response (200 OK):**

```json
{
  "violations": [
    {
      "id": "viol-uuid-001",
      "policy_id": "pol-uuid-002",
      "resource_id": "i-0xyz789abc123456",
      "provider": "aws",
      "description": "EC2 instance i-0xyz789abc123456 is running in ap-southeast-1, which is not in the approved production region list (us-east-1, us-west-2, eu-west-1).",
      "severity": "critical",
      "detected_at": "2026-02-06T08:00:00Z",
      "resolved_at": null
    },
    {
      "id": "viol-uuid-002",
      "policy_id": "pol-uuid-003",
      "resource_id": "123456789012",
      "provider": "aws",
      "description": "AWS account 123456789012 has exceeded the monthly budget limit of $10,000. Current spend: $10,245.80.",
      "severity": "critical",
      "detected_at": "2026-02-05T12:00:00Z",
      "resolved_at": null
    }
  ],
  "total": 2,
  "summary": {
    "critical": 2,
    "warning": 0,
    "info": 0
  }
}
```

### PUT /api/v1/governance/violations/{violation_id}/resolve

Mark a violation as resolved.

**Request:**

```bash
curl -X PUT http://localhost:8081/api/v1/governance/violations/viol-uuid-001/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "resolution_note": "Instance migrated to us-east-1."
  }'
```

**Response (200 OK):**

```json
{
  "id": "viol-uuid-001",
  "resolved_at": "2026-02-06T16:00:00Z"
}
```

---

## Dashboard

### GET /api/v1/dashboard/overview

Get a consolidated dashboard overview combining cost summaries, top recommendations, and active violations.

**Request:**

```bash
curl http://localhost:8081/api/v1/dashboard/overview
```

**Response (200 OK):**

```json
{
  "overview": {
    "total_monthly_cost_usd": 45230.50,
    "cost_change_pct": 3.2,
    "cost_change_direction": "up",
    "total_potential_savings_usd": 4820.61,
    "active_recommendations": 12,
    "active_violations": 3,
    "providers": {
      "aws": {
        "cost_usd": 28450.30,
        "status": "connected",
        "last_sync": "2026-02-06T12:00:00Z"
      },
      "azure": {
        "cost_usd": 12380.20,
        "status": "connected",
        "last_sync": "2026-02-06T12:00:00Z"
      },
      "gcp": {
        "cost_usd": 4400.00,
        "status": "connected",
        "last_sync": "2026-02-06T11:30:00Z"
      }
    },
    "top_recommendations": [
      {
        "id": "rec-uuid-003",
        "title": "Purchase Reserved Instances for 8 stable m5.xlarge instances",
        "estimated_monthly_savings": 384.00,
        "recommendation_type": "reserved_capacity"
      },
      {
        "id": "rec-uuid-001",
        "title": "Downsize m5.2xlarge to m5.xlarge in us-east-1",
        "estimated_monthly_savings": 138.24,
        "recommendation_type": "rightsizing"
      }
    ],
    "critical_violations": [
      {
        "id": "viol-uuid-001",
        "description": "EC2 instance running in unapproved region",
        "severity": "critical",
        "detected_at": "2026-02-06T08:00:00Z"
      }
    ]
  }
}
```

---

## Error Handling

All errors follow a consistent format:

```json
{
  "detail": "Human-readable error description"
}
```

For validation errors (FastAPI automatic validation):

```json
{
  "detail": [
    {
      "loc": ["query", "from"],
      "msg": "invalid date format",
      "type": "value_error"
    }
  ]
}
```

### HTTP Status Codes

| Status | Description |
|--------|-------------|
| 200 | Success |
| 201 | Resource created |
| 204 | Resource deleted (no content) |
| 400 | Bad request (invalid parameters) |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate policy name) |
| 422 | Validation error (FastAPI automatic) |
| 500 | Internal server error |
| 503 | Service unavailable (database connection issue) |

---

## Data Models

### CloudCost

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique cost line item identifier |
| `provider` | string | Cloud provider: `aws`, `azure`, `gcp` |
| `service` | string | Cloud service name (e.g., `Amazon EC2`, `Azure VMs`) |
| `resource_id` | string | Provider-specific resource identifier |
| `resource_name` | string | Human-readable resource name |
| `cost_usd` | float | Cost in USD |
| `currency` | string | Original currency code (default: `USD`) |
| `usage_quantity` | float | Usage amount |
| `usage_unit` | string | Usage unit (e.g., `hours`, `GB`, `requests`) |
| `region` | string | Cloud region |
| `account_id` | string | Cloud account identifier |
| `tags` | object | Resource tags (JSONB) |
| `date` | date | Cost date |
| `created_at` | datetime | Record creation timestamp |

### OptimizationRecommendation

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique recommendation identifier |
| `provider` | string | Cloud provider |
| `resource_id` | string | Affected resource identifier |
| `resource_type` | string | Resource type (e.g., `EC2 Instance`) |
| `recommendation_type` | string | Type: `idle_resources`, `rightsizing`, `spot_instances`, `reserved_capacity` |
| `title` | string | Short recommendation title |
| `description` | string | Detailed explanation |
| `estimated_monthly_savings` | float | Estimated savings in USD/month |
| `confidence` | float | Confidence score (0.0 to 1.0) |
| `status` | string | Status: `open`, `accepted`, `dismissed` |
| `created_at` | datetime | When the recommendation was generated |
| `resolved_at` | datetime | When the recommendation was acted upon |

### GovernancePolicy

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique policy identifier |
| `name` | string | Policy name (unique) |
| `description` | string | Policy description |
| `policy_type` | string | Type: `budget_limit`, `tag_enforcement`, `region_restriction`, `resource_approval` |
| `rules` | object | Policy rules (JSONB, flexible structure) |
| `severity` | string | Severity: `info`, `warning`, `critical` |
| `enabled` | boolean | Whether the policy is active |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |

### PolicyViolation

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique violation identifier |
| `policy_id` | UUID | Associated policy ID |
| `resource_id` | string | Violating resource identifier |
| `provider` | string | Cloud provider |
| `description` | string | Violation description |
| `severity` | string | Severity: `info`, `warning`, `critical` |
| `detected_at` | datetime | When the violation was detected |
| `resolved_at` | datetime | When the violation was resolved (null if active) |
