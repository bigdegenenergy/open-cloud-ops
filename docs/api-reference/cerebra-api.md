# Cerebra API Reference

**Base URL:** `http://localhost:8080`
**Version:** v1
**Protocol:** HTTP/REST
**Content-Type:** `application/json`

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Health](#health)
4. [Proxy Endpoints](#proxy-endpoints)
5. [Budget Management](#budget-management)
6. [Analytics](#analytics)
7. [Error Handling](#error-handling)
8. [Data Models](#data-models)

---

## Overview

Cerebra is the LLM Gateway module of Open Cloud Ops. It provides a reverse proxy for LLM API providers with built-in cost tracking, budget enforcement, smart model routing, and analytics.

**Key characteristics:**

- All proxy requests pass through to the upstream LLM provider and return the original response unmodified
- Only metadata (model, tokens, cost, latency) is stored -- prompt and response content is never persisted
- API keys are passed through in-memory and never stored to database, logs, or disk
- Budget checks occur before forwarding to prevent overspend

---

## Authentication

Cerebra uses **passthrough authentication**. You provide your own LLM provider API key in the `Authorization` header, and Cerebra forwards it to the upstream provider.

Additionally, you should include identification headers for cost attribution:

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Your LLM provider API key (e.g., `Bearer sk-...`) |
| `X-Agent-ID` | Recommended | Unique identifier for the AI agent making the request |
| `X-Team-ID` | Recommended | Team identifier for cost allocation |
| `X-Org-ID` | Recommended | Organization identifier for cost allocation |

---

## Health

### GET /health

Returns the health status of the Cerebra service, including connectivity to dependent services.

**Request:**

```bash
curl http://localhost:8080/health
```

**Response (200 OK):**

```json
{
  "status": "healthy",
  "service": "cerebra",
  "version": "0.1.0",
  "uptime_seconds": 3621,
  "dependencies": {
    "postgresql": "connected",
    "redis": "connected"
  }
}
```

**Response (503 Service Unavailable):**

```json
{
  "status": "degraded",
  "service": "cerebra",
  "version": "0.1.0",
  "uptime_seconds": 3621,
  "dependencies": {
    "postgresql": "connected",
    "redis": "disconnected"
  }
}
```

---

## Proxy Endpoints

Cerebra proxies requests to upstream LLM providers by mapping URL paths to provider endpoints. The request body is forwarded as-is, and the response is returned unmodified.

### POST /api/v1/proxy/openai/{path}

Proxies requests to the OpenAI API (`https://api.openai.com/`).

**Supported paths:**

| Path | Upstream URL |
|------|-------------|
| `/api/v1/proxy/openai/v1/chat/completions` | `https://api.openai.com/v1/chat/completions` |
| `/api/v1/proxy/openai/v1/completions` | `https://api.openai.com/v1/completions` |
| `/api/v1/proxy/openai/v1/embeddings` | `https://api.openai.com/v1/embeddings` |

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/proxy/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-api-key" \
  -H "X-Agent-ID: agent-codegen-01" \
  -H "X-Team-ID: team-backend" \
  -H "X-Org-ID: org-acme" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Explain the CAP theorem in one paragraph."}
    ],
    "max_tokens": 256
  }'
```

**Response (200 OK):**

The response is the exact response from OpenAI, returned unmodified:

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1707200000,
  "model": "gpt-4o-2024-08-06",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The CAP theorem states that..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 28,
    "completion_tokens": 120,
    "total_tokens": 148
  }
}
```

**Side effects:** Cerebra logs the following metadata (but NOT the prompt or response content):

```json
{
  "id": "req-uuid-generated",
  "provider": "openai",
  "model": "gpt-4o-2024-08-06",
  "agent_id": "agent-codegen-01",
  "team_id": "team-backend",
  "org_id": "org-acme",
  "input_tokens": 28,
  "output_tokens": 120,
  "total_tokens": 148,
  "cost_usd": 0.00148,
  "latency_ms": 1250,
  "status_code": 200,
  "was_routed": false,
  "timestamp": "2026-02-06T12:00:00Z"
}
```

### POST /api/v1/proxy/anthropic/{path}

Proxies requests to the Anthropic API (`https://api.anthropic.com/`).

**Supported paths:**

| Path | Upstream URL |
|------|-------------|
| `/api/v1/proxy/anthropic/v1/messages` | `https://api.anthropic.com/v1/messages` |

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/proxy/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: sk-ant-your-anthropic-key" \
  -H "anthropic-version: 2023-06-01" \
  -H "X-Agent-ID: agent-codegen-01" \
  -H "X-Team-ID: team-backend" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 256,
    "messages": [
      {"role": "user", "content": "Explain the CAP theorem in one paragraph."}
    ]
  }'
```

**Response (200 OK):**

The response is the exact response from Anthropic, returned unmodified:

```json
{
  "id": "msg_abc123",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "The CAP theorem states that..."
    }
  ],
  "model": "claude-sonnet-4-20250514",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 20,
    "output_tokens": 115
  }
}
```

### POST /api/v1/proxy/gemini/{path}

Proxies requests to the Google Gemini API (`https://generativelanguage.googleapis.com/`).

**Supported paths:**

| Path | Upstream URL |
|------|-------------|
| `/api/v1/proxy/gemini/v1beta/models/{model}:generateContent` | `https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent` |

**Request:**

```bash
curl -X POST "http://localhost:8080/api/v1/proxy/gemini/v1beta/models/gemini-pro:generateContent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-gemini-api-key" \
  -H "X-Agent-ID: agent-codegen-01" \
  -H "X-Team-ID: team-backend" \
  -d '{
    "contents": [
      {
        "parts": [
          {"text": "Explain the CAP theorem in one paragraph."}
        ]
      }
    ]
  }'
```

**Response (200 OK):**

The response is the exact response from Gemini, returned unmodified.

---

## Budget Management

### GET /api/v1/budgets

List all configured budgets.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `scope` | string | (all) | Filter by scope: `agent`, `team`, `user`, `org` |
| `entity_id` | string | (all) | Filter by specific entity ID |

**Request:**

```bash
curl "http://localhost:8080/api/v1/budgets?scope=team"
```

**Response (200 OK):**

```json
{
  "budgets": [
    {
      "id": "budget-001",
      "scope": "team",
      "entity_id": "team-backend",
      "limit_usd": 500.00,
      "spent_usd": 127.45,
      "remaining_usd": 372.55,
      "utilization_pct": 25.49,
      "period": "monthly",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-02-06T12:00:00Z"
    },
    {
      "id": "budget-002",
      "scope": "team",
      "entity_id": "team-frontend",
      "limit_usd": 200.00,
      "spent_usd": 89.20,
      "remaining_usd": 110.80,
      "utilization_pct": 44.60,
      "period": "monthly",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-02-06T11:30:00Z"
    }
  ],
  "total": 2
}
```

### POST /api/v1/budgets

Create a new budget for an entity.

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/budgets \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "agent",
    "entity_id": "agent-codegen-01",
    "limit_usd": 100.00,
    "period": "monthly"
  }'
```

**Response (201 Created):**

```json
{
  "id": "budget-003",
  "scope": "agent",
  "entity_id": "agent-codegen-01",
  "limit_usd": 100.00,
  "spent_usd": 0.00,
  "remaining_usd": 100.00,
  "utilization_pct": 0.00,
  "period": "monthly",
  "created_at": "2026-02-06T12:05:00Z",
  "updated_at": "2026-02-06T12:05:00Z"
}
```

### GET /api/v1/budgets/{budget_id}

Get a specific budget by ID.

**Request:**

```bash
curl http://localhost:8080/api/v1/budgets/budget-001
```

**Response (200 OK):**

```json
{
  "id": "budget-001",
  "scope": "team",
  "entity_id": "team-backend",
  "limit_usd": 500.00,
  "spent_usd": 127.45,
  "remaining_usd": 372.55,
  "utilization_pct": 25.49,
  "period": "monthly",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-02-06T12:00:00Z"
}
```

### PUT /api/v1/budgets/{budget_id}

Update an existing budget.

**Request:**

```bash
curl -X PUT http://localhost:8080/api/v1/budgets/budget-001 \
  -H "Content-Type: application/json" \
  -d '{
    "limit_usd": 750.00
  }'
```

**Response (200 OK):**

```json
{
  "id": "budget-001",
  "scope": "team",
  "entity_id": "team-backend",
  "limit_usd": 750.00,
  "spent_usd": 127.45,
  "remaining_usd": 622.55,
  "utilization_pct": 16.99,
  "period": "monthly",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-02-06T12:10:00Z"
}
```

### DELETE /api/v1/budgets/{budget_id}

Delete a budget. Requests for the associated entity will no longer be budget-limited.

**Request:**

```bash
curl -X DELETE http://localhost:8080/api/v1/budgets/budget-003
```

**Response (204 No Content):**

No response body.

---

## Analytics

### GET /api/v1/analytics/summary

Get an aggregated cost summary for a time period.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | ISO 8601 datetime | 30 days ago | Start of the time range |
| `to` | ISO 8601 datetime | now | End of the time range |
| `group_by` | string | `provider` | Grouping dimension: `provider`, `model`, `agent`, `team`, `org` |

**Request:**

```bash
curl "http://localhost:8080/api/v1/analytics/summary?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z&group_by=model"
```

**Response (200 OK):**

```json
{
  "from": "2026-01-01T00:00:00Z",
  "to": "2026-02-01T00:00:00Z",
  "group_by": "model",
  "data": [
    {
      "dimension": "model",
      "dimension_id": "gpt-4o",
      "dimension_name": "GPT-4o",
      "total_cost_usd": 342.18,
      "total_requests": 12450,
      "total_tokens": 8923000,
      "avg_latency_ms": 1250,
      "total_savings_usd": 0.00
    },
    {
      "dimension": "model",
      "dimension_id": "claude-sonnet-4-20250514",
      "dimension_name": "Claude Sonnet",
      "total_cost_usd": 218.45,
      "total_requests": 8200,
      "total_tokens": 6100000,
      "avg_latency_ms": 980,
      "total_savings_usd": 45.20
    },
    {
      "dimension": "model",
      "dimension_id": "gpt-4o-mini",
      "dimension_name": "GPT-4o Mini",
      "total_cost_usd": 12.30,
      "total_requests": 45000,
      "total_tokens": 15200000,
      "avg_latency_ms": 450,
      "total_savings_usd": 180.50
    }
  ],
  "totals": {
    "total_cost_usd": 572.93,
    "total_requests": 65650,
    "total_tokens": 30223000,
    "total_savings_usd": 225.70
  }
}
```

### GET /api/v1/analytics/trends

Get cost trends over time (time-series data for charts).

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | ISO 8601 datetime | 30 days ago | Start of the time range |
| `to` | ISO 8601 datetime | now | End of the time range |
| `interval` | string | `day` | Aggregation interval: `hour`, `day`, `week`, `month` |
| `group_by` | string | (none) | Optional grouping: `provider`, `model`, `agent`, `team` |

**Request:**

```bash
curl "http://localhost:8080/api/v1/analytics/trends?from=2026-01-01T00:00:00Z&to=2026-01-08T00:00:00Z&interval=day"
```

**Response (200 OK):**

```json
{
  "from": "2026-01-01T00:00:00Z",
  "to": "2026-01-08T00:00:00Z",
  "interval": "day",
  "data": [
    {
      "timestamp": "2026-01-01T00:00:00Z",
      "total_cost_usd": 18.42,
      "total_requests": 2100,
      "total_tokens": 980000
    },
    {
      "timestamp": "2026-01-02T00:00:00Z",
      "total_cost_usd": 22.15,
      "total_requests": 2450,
      "total_tokens": 1120000
    },
    {
      "timestamp": "2026-01-03T00:00:00Z",
      "total_cost_usd": 19.80,
      "total_requests": 2200,
      "total_tokens": 1010000
    }
  ]
}
```

### GET /api/v1/analytics/insights

Get AI-powered cost insights and recommendations.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `severity` | string | (all) | Filter by severity: `info`, `warning`, `critical` |
| `type` | string | (all) | Filter by type: `cost_spike`, `model_switch`, `budget_warning`, `anomaly_detected`, `savings_found` |
| `dismissed` | boolean | `false` | Include dismissed insights |

**Request:**

```bash
curl "http://localhost:8080/api/v1/analytics/insights?severity=warning"
```

**Response (200 OK):**

```json
{
  "insights": [
    {
      "id": "insight-001",
      "type": "model_switch",
      "severity": "warning",
      "title": "Agent 'codegen-01' could save $45/month by using GPT-4o Mini",
      "description": "Analysis of the last 30 days shows that 68% of requests from agent 'codegen-01' are simple code completions that could be handled by GPT-4o Mini instead of GPT-4o, maintaining 95%+ quality while reducing costs.",
      "estimated_saving": 45.00,
      "affected_entity": "agent-codegen-01",
      "created_at": "2026-02-06T08:00:00Z",
      "dismissed": false
    },
    {
      "id": "insight-002",
      "type": "cost_spike",
      "severity": "warning",
      "title": "Team 'data-science' cost spike: 3.2x above baseline",
      "description": "Spending for team 'data-science' increased by 220% compared to the 7-day rolling average. This appears to be driven by increased usage of Claude Opus for batch analysis jobs.",
      "estimated_saving": 0.00,
      "affected_entity": "team-data-science",
      "created_at": "2026-02-06T06:00:00Z",
      "dismissed": false
    }
  ],
  "total": 2
}
```

### POST /api/v1/analytics/insights/{insight_id}/dismiss

Dismiss an insight.

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/analytics/insights/insight-001/dismiss
```

**Response (200 OK):**

```json
{
  "id": "insight-001",
  "dismissed": true
}
```

### GET /api/v1/analytics/reports

Generate a cost report for a time period.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | ISO 8601 datetime | Start of current month | Start of the report period |
| `to` | ISO 8601 datetime | now | End of the report period |

**Request:**

```bash
curl "http://localhost:8080/api/v1/analytics/reports?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z"
```

**Response (200 OK):**

```json
{
  "report": {
    "period": {
      "from": "2026-01-01T00:00:00Z",
      "to": "2026-02-01T00:00:00Z"
    },
    "total_cost_usd": 572.93,
    "total_requests": 65650,
    "total_tokens": 30223000,
    "total_savings_usd": 225.70,
    "cost_by_provider": [
      {"provider": "openai", "cost_usd": 354.48, "requests": 57450},
      {"provider": "anthropic", "cost_usd": 218.45, "requests": 8200}
    ],
    "cost_by_team": [
      {"team_id": "team-backend", "cost_usd": 312.40, "requests": 32000},
      {"team_id": "team-data-science", "cost_usd": 180.33, "requests": 22000},
      {"team_id": "team-frontend", "cost_usd": 80.20, "requests": 11650}
    ],
    "top_agents": [
      {"agent_id": "agent-codegen-01", "cost_usd": 145.20, "requests": 15000},
      {"agent_id": "agent-analysis-02", "cost_usd": 98.50, "requests": 8500}
    ],
    "insights_count": {
      "info": 3,
      "warning": 2,
      "critical": 0
    }
  }
}
```

---

## Error Handling

All errors follow a consistent format:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable description of the error",
    "details": {}
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|-----------|-------------|
| 400 | `bad_request` | Malformed request body or invalid parameters |
| 401 | `unauthorized` | Missing or invalid API key |
| 403 | `forbidden` | Insufficient permissions |
| 404 | `not_found` | Resource not found |
| 429 | `budget_exceeded` | The agent/team budget limit has been reached |
| 429 | `rate_limited` | Too many requests; retry after the indicated interval |
| 502 | `upstream_error` | The upstream LLM provider returned an error |
| 503 | `service_unavailable` | Cerebra is unable to process requests (database or Redis down) |

### Budget Exceeded Response

When a request is blocked due to budget exhaustion:

```bash
curl -X POST http://localhost:8080/api/v1/proxy/openai/v1/chat/completions \
  -H "Authorization: Bearer sk-..." \
  -H "X-Agent-ID: agent-codegen-01" \
  -d '{"model": "gpt-4o", "messages": [...]}'
```

**Response (429 Too Many Requests):**

```json
{
  "error": {
    "code": "budget_exceeded",
    "message": "Budget limit reached for agent 'agent-codegen-01'. Monthly limit: $100.00, spent: $100.02.",
    "details": {
      "scope": "agent",
      "entity_id": "agent-codegen-01",
      "limit_usd": 100.00,
      "spent_usd": 100.02,
      "resets_at": "2026-03-01T00:00:00Z"
    }
  }
}
```

---

## Data Models

### APIRequest

The core data model for tracking LLM API requests. Stored in the `api_requests` TimescaleDB hypertable.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique request identifier (UUID) |
| `provider` | string | LLM provider: `openai`, `anthropic`, `gemini` |
| `model` | string | Model identifier (e.g., `gpt-4o`, `claude-sonnet-4-20250514`) |
| `agent_id` | string | Agent identifier from `X-Agent-ID` header |
| `team_id` | string | Team identifier from `X-Team-ID` header |
| `org_id` | string | Organization identifier from `X-Org-ID` header |
| `input_tokens` | integer | Number of input/prompt tokens |
| `output_tokens` | integer | Number of output/completion tokens |
| `total_tokens` | integer | Total tokens (input + output) |
| `cost_usd` | float | Computed cost in USD |
| `latency_ms` | integer | Request latency in milliseconds |
| `status_code` | integer | HTTP status code from upstream |
| `was_routed` | boolean | Whether smart routing was applied |
| `original_model` | string | Originally requested model (if routed) |
| `routed_model` | string | Model selected by smart router (if routed) |
| `savings_usd` | float | Estimated savings from routing |
| `timestamp` | datetime | Request timestamp (ISO 8601) |

### Budget

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique budget identifier |
| `scope` | string | Budget scope: `agent`, `team`, `user`, `org` |
| `entity_id` | string | The ID of the entity this budget applies to |
| `limit_usd` | float | Monthly spending limit in USD |
| `spent_usd` | float | Current spend in the billing period |
| `period` | string | Budget period (e.g., `monthly`) |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |

### Insight

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique insight identifier |
| `type` | string | Insight type: `cost_spike`, `model_switch`, `budget_warning`, `anomaly_detected`, `savings_found` |
| `severity` | string | Severity level: `info`, `warning`, `critical` |
| `title` | string | Short description of the insight |
| `description` | string | Detailed explanation and recommendation |
| `estimated_saving` | float | Potential monthly savings in USD |
| `affected_entity` | string | Agent, team, or model affected |
| `created_at` | datetime | When the insight was generated |
| `dismissed` | boolean | Whether the insight has been dismissed |
