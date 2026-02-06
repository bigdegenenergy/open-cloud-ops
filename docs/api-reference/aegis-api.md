# Aegis API Reference

**Base URL:** `http://localhost:8082`
**Version:** v1
**Protocol:** HTTP/REST
**Content-Type:** `application/json`

---

## Table of Contents

1. [Overview](#overview)
2. [Health](#health)
3. [Backup Jobs](#backup-jobs)
4. [Backup Records](#backup-records)
5. [Recovery Plans](#recovery-plans)
6. [Recovery Execution](#recovery-execution)
7. [DR Policies](#dr-policies)
8. [DR Compliance](#dr-compliance)
9. [Health Summary](#health-summary)
10. [Error Handling](#error-handling)
11. [Data Models](#data-models)

---

## Overview

Aegis is the Resilience Engine module of Open Cloud Ops. It provides Kubernetes-native backup, disaster recovery, and health monitoring capabilities, orchestrating operations through Velero and tracking state in PostgreSQL.

**Key characteristics:**

- Integrates with Velero for Kubernetes backup and restore operations
- Supports multiple storage backends (AWS S3, GCS, Azure Blob Storage)
- Manages DR policies with configurable schedules, retention rules, and RPO/RTO targets
- Tracks backup job history and recovery plan execution
- Provides a unified health summary across all managed clusters

---

## Health

### GET /health

Returns the health status of the Aegis service, including Velero and Kubernetes connectivity.

**Request:**

```bash
curl http://localhost:8082/health
```

**Response (200 OK):**

```json
{
  "status": "healthy",
  "service": "aegis",
  "version": "0.1.0",
  "uptime_seconds": 86421,
  "dependencies": {
    "postgresql": "connected",
    "redis": "connected",
    "velero": "connected",
    "kubernetes": "connected"
  }
}
```

**Response (503 Service Unavailable):**

```json
{
  "status": "degraded",
  "service": "aegis",
  "version": "0.1.0",
  "uptime_seconds": 86421,
  "dependencies": {
    "postgresql": "connected",
    "redis": "connected",
    "velero": "disconnected",
    "kubernetes": "connected"
  }
}
```

---

## Backup Jobs

### POST /api/v1/backups/jobs

Create and trigger a new backup job.

**Request:**

```bash
curl -X POST http://localhost:8082/api/v1/backups/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "manual-backup-prod-2026-02-06",
    "cluster": "prod-us-east-1",
    "namespaces": ["default", "app-backend", "app-frontend"],
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "ttl_hours": 720,
    "labels": {
      "trigger": "manual",
      "environment": "production"
    }
  }'
```

**Response (201 Created):**

```json
{
  "job": {
    "id": "job-uuid-001",
    "name": "manual-backup-prod-2026-02-06",
    "cluster": "prod-us-east-1",
    "namespaces": ["default", "app-backend", "app-frontend"],
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "ttl_hours": 720,
    "labels": {
      "trigger": "manual",
      "environment": "production"
    },
    "status": "in_progress",
    "velero_backup_name": "oco-manual-backup-prod-2026-02-06-1707235200",
    "started_at": "2026-02-06T16:00:00Z",
    "completed_at": null,
    "size_bytes": null,
    "resource_count": null,
    "error_message": null
  }
}
```

### GET /api/v1/backups/jobs

List backup jobs with optional filtering.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cluster` | string | (all) | Filter by cluster name |
| `status` | string | (all) | Filter by status: `pending`, `in_progress`, `completed`, `failed`, `partially_failed` |
| `from` | ISO 8601 datetime | 30 days ago | Start of time range |
| `to` | ISO 8601 datetime | now | End of time range |
| `limit` | integer | 50 | Maximum number of results |
| `offset` | integer | 0 | Pagination offset |

**Request:**

```bash
curl "http://localhost:8082/api/v1/backups/jobs?cluster=prod-us-east-1&status=completed&limit=5"
```

**Response (200 OK):**

```json
{
  "jobs": [
    {
      "id": "job-uuid-001",
      "name": "manual-backup-prod-2026-02-06",
      "cluster": "prod-us-east-1",
      "namespaces": ["default", "app-backend", "app-frontend"],
      "status": "completed",
      "started_at": "2026-02-06T16:00:00Z",
      "completed_at": "2026-02-06T16:08:32Z",
      "size_bytes": 2147483648,
      "resource_count": 245,
      "error_message": null
    },
    {
      "id": "job-uuid-002",
      "name": "scheduled-daily-prod-2026-02-05",
      "cluster": "prod-us-east-1",
      "namespaces": ["*"],
      "status": "completed",
      "started_at": "2026-02-05T02:00:00Z",
      "completed_at": "2026-02-05T02:12:45Z",
      "size_bytes": 3221225472,
      "resource_count": 312,
      "error_message": null
    }
  ],
  "total": 2,
  "limit": 5,
  "offset": 0
}
```

### GET /api/v1/backups/jobs/{job_id}

Get details of a specific backup job.

**Request:**

```bash
curl http://localhost:8082/api/v1/backups/jobs/job-uuid-001
```

**Response (200 OK):**

```json
{
  "job": {
    "id": "job-uuid-001",
    "name": "manual-backup-prod-2026-02-06",
    "cluster": "prod-us-east-1",
    "namespaces": ["default", "app-backend", "app-frontend"],
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "ttl_hours": 720,
    "labels": {
      "trigger": "manual",
      "environment": "production"
    },
    "status": "completed",
    "velero_backup_name": "oco-manual-backup-prod-2026-02-06-1707235200",
    "started_at": "2026-02-06T16:00:00Z",
    "completed_at": "2026-02-06T16:08:32Z",
    "size_bytes": 2147483648,
    "resource_count": 245,
    "error_message": null,
    "resources_backed_up": {
      "deployments": 12,
      "services": 18,
      "configmaps": 34,
      "secrets": 22,
      "persistent_volume_claims": 8,
      "other": 151
    }
  }
}
```

### DELETE /api/v1/backups/jobs/{job_id}

Cancel a pending or in-progress backup job, or delete a completed job record.

**Request:**

```bash
curl -X DELETE http://localhost:8082/api/v1/backups/jobs/job-uuid-001
```

**Response (204 No Content):**

No response body.

---

## Backup Records

### GET /api/v1/backups/records

List available backup records (completed backups that can be used for recovery).

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cluster` | string | (all) | Filter by cluster |
| `from` | ISO 8601 datetime | 90 days ago | Start of time range |
| `to` | ISO 8601 datetime | now | End of time range |
| `storage_backend` | string | (all) | Filter by storage: `s3`, `gcs`, `azure_blob` |
| `limit` | integer | 50 | Maximum number of results |

**Request:**

```bash
curl "http://localhost:8082/api/v1/backups/records?cluster=prod-us-east-1&limit=3"
```

**Response (200 OK):**

```json
{
  "records": [
    {
      "id": "rec-uuid-001",
      "backup_job_id": "job-uuid-001",
      "name": "manual-backup-prod-2026-02-06",
      "cluster": "prod-us-east-1",
      "namespaces": ["default", "app-backend", "app-frontend"],
      "storage_backend": "s3",
      "storage_location": "s3://oco-backups/prod-us-east-1/oco-manual-backup-prod-2026-02-06-1707235200/",
      "size_bytes": 2147483648,
      "resource_count": 245,
      "created_at": "2026-02-06T16:08:32Z",
      "expires_at": "2026-03-08T16:08:32Z",
      "is_restorable": true
    },
    {
      "id": "rec-uuid-002",
      "backup_job_id": "job-uuid-002",
      "name": "scheduled-daily-prod-2026-02-05",
      "cluster": "prod-us-east-1",
      "namespaces": ["*"],
      "storage_backend": "s3",
      "storage_location": "s3://oco-backups/prod-us-east-1/oco-scheduled-daily-prod-2026-02-05/",
      "size_bytes": 3221225472,
      "resource_count": 312,
      "created_at": "2026-02-05T02:12:45Z",
      "expires_at": "2026-03-07T02:12:45Z",
      "is_restorable": true
    }
  ],
  "total": 2,
  "total_size_bytes": 5368709120
}
```

### GET /api/v1/backups/records/{record_id}

Get details of a specific backup record.

**Request:**

```bash
curl http://localhost:8082/api/v1/backups/records/rec-uuid-001
```

**Response (200 OK):**

```json
{
  "record": {
    "id": "rec-uuid-001",
    "backup_job_id": "job-uuid-001",
    "name": "manual-backup-prod-2026-02-06",
    "cluster": "prod-us-east-1",
    "namespaces": ["default", "app-backend", "app-frontend"],
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/oco-manual-backup-prod-2026-02-06-1707235200/",
    "size_bytes": 2147483648,
    "resource_count": 245,
    "created_at": "2026-02-06T16:08:32Z",
    "expires_at": "2026-03-08T16:08:32Z",
    "is_restorable": true,
    "velero_backup_name": "oco-manual-backup-prod-2026-02-06-1707235200",
    "included_resources": {
      "deployments": 12,
      "services": 18,
      "configmaps": 34,
      "secrets": 22,
      "persistent_volume_claims": 8,
      "other": 151
    }
  }
}
```

---

## Recovery Plans

### POST /api/v1/recovery/plans

Create a new recovery plan based on an existing backup record.

**Request:**

```bash
curl -X POST http://localhost:8082/api/v1/recovery/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "recover-prod-backend-2026-02-06",
    "backup_record_id": "rec-uuid-001",
    "target_cluster": "prod-us-east-1",
    "target_namespaces": ["app-backend"],
    "restore_volumes": true,
    "namespace_mapping": {},
    "preserve_node_ports": false,
    "description": "Recover app-backend namespace after accidental deployment deletion."
  }'
```

**Response (201 Created):**

```json
{
  "plan": {
    "id": "plan-uuid-001",
    "name": "recover-prod-backend-2026-02-06",
    "backup_record_id": "rec-uuid-001",
    "target_cluster": "prod-us-east-1",
    "target_namespaces": ["app-backend"],
    "restore_volumes": true,
    "namespace_mapping": {},
    "preserve_node_ports": false,
    "description": "Recover app-backend namespace after accidental deployment deletion.",
    "status": "pending",
    "created_at": "2026-02-06T17:00:00Z",
    "created_by": "admin"
  }
}
```

### GET /api/v1/recovery/plans

List recovery plans.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `status` | string | (all) | Filter by status: `pending`, `approved`, `executing`, `completed`, `failed`, `cancelled` |
| `cluster` | string | (all) | Filter by target cluster |
| `limit` | integer | 50 | Maximum number of results |

**Request:**

```bash
curl "http://localhost:8082/api/v1/recovery/plans?status=pending"
```

**Response (200 OK):**

```json
{
  "plans": [
    {
      "id": "plan-uuid-001",
      "name": "recover-prod-backend-2026-02-06",
      "backup_record_id": "rec-uuid-001",
      "target_cluster": "prod-us-east-1",
      "target_namespaces": ["app-backend"],
      "status": "pending",
      "created_at": "2026-02-06T17:00:00Z",
      "description": "Recover app-backend namespace after accidental deployment deletion."
    }
  ],
  "total": 1
}
```

### GET /api/v1/recovery/plans/{plan_id}

Get details of a specific recovery plan.

**Request:**

```bash
curl http://localhost:8082/api/v1/recovery/plans/plan-uuid-001
```

**Response (200 OK):**

```json
{
  "plan": {
    "id": "plan-uuid-001",
    "name": "recover-prod-backend-2026-02-06",
    "backup_record_id": "rec-uuid-001",
    "target_cluster": "prod-us-east-1",
    "target_namespaces": ["app-backend"],
    "restore_volumes": true,
    "namespace_mapping": {},
    "preserve_node_ports": false,
    "description": "Recover app-backend namespace after accidental deployment deletion.",
    "status": "pending",
    "created_at": "2026-02-06T17:00:00Z",
    "created_by": "admin",
    "execution_history": []
  }
}
```

### DELETE /api/v1/recovery/plans/{plan_id}

Delete a pending recovery plan (cannot delete plans that are executing or completed).

**Request:**

```bash
curl -X DELETE http://localhost:8082/api/v1/recovery/plans/plan-uuid-001
```

**Response (204 No Content):**

No response body.

---

## Recovery Execution

### POST /api/v1/recovery/execute/{plan_id}

Execute a recovery plan. This triggers the actual Velero restore operation.

**Request:**

```bash
curl -X POST http://localhost:8082/api/v1/recovery/execute/plan-uuid-001 \
  -H "Content-Type: application/json" \
  -d '{
    "confirmed": true,
    "dry_run": false
  }'
```

**Response (202 Accepted):**

```json
{
  "execution": {
    "id": "exec-uuid-001",
    "plan_id": "plan-uuid-001",
    "status": "executing",
    "velero_restore_name": "oco-recover-prod-backend-2026-02-06-1707242400",
    "started_at": "2026-02-06T18:00:00Z",
    "completed_at": null,
    "resources_restored": null,
    "warnings": [],
    "errors": []
  }
}
```

### GET /api/v1/recovery/execute/{plan_id}/status

Get the current status of a recovery execution.

**Request:**

```bash
curl http://localhost:8082/api/v1/recovery/execute/plan-uuid-001/status
```

**Response (200 OK) -- In Progress:**

```json
{
  "execution": {
    "id": "exec-uuid-001",
    "plan_id": "plan-uuid-001",
    "status": "executing",
    "progress_pct": 65,
    "velero_restore_name": "oco-recover-prod-backend-2026-02-06-1707242400",
    "started_at": "2026-02-06T18:00:00Z",
    "completed_at": null,
    "resources_restored": {
      "deployments": 5,
      "services": 8,
      "configmaps": 12,
      "secrets": 0,
      "persistent_volume_claims": 0
    },
    "warnings": [],
    "errors": []
  }
}
```

**Response (200 OK) -- Completed:**

```json
{
  "execution": {
    "id": "exec-uuid-001",
    "plan_id": "plan-uuid-001",
    "status": "completed",
    "progress_pct": 100,
    "velero_restore_name": "oco-recover-prod-backend-2026-02-06-1707242400",
    "started_at": "2026-02-06T18:00:00Z",
    "completed_at": "2026-02-06T18:05:42Z",
    "duration_seconds": 342,
    "resources_restored": {
      "deployments": 8,
      "services": 12,
      "configmaps": 20,
      "secrets": 15,
      "persistent_volume_claims": 4
    },
    "warnings": [
      "Service 'api-gateway' nodePort 30080 was already in use, assigned new port 30081"
    ],
    "errors": []
  }
}
```

---

## DR Policies

### POST /api/v1/dr/policies

Create a new disaster recovery policy.

**Request:**

```bash
curl -X POST http://localhost:8082/api/v1/dr/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-daily-backup-policy",
    "description": "Daily backups for all production namespaces with 30-day retention.",
    "cluster": "prod-us-east-1",
    "namespaces": ["*"],
    "exclude_namespaces": ["kube-system", "kube-public", "velero"],
    "schedule": "0 2 * * *",
    "retention_days": 30,
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "rpo_hours": 24,
    "rto_hours": 4,
    "enabled": true
  }'
```

**Response (201 Created):**

```json
{
  "policy": {
    "id": "drpol-uuid-001",
    "name": "prod-daily-backup-policy",
    "description": "Daily backups for all production namespaces with 30-day retention.",
    "cluster": "prod-us-east-1",
    "namespaces": ["*"],
    "exclude_namespaces": ["kube-system", "kube-public", "velero"],
    "schedule": "0 2 * * *",
    "schedule_human": "Daily at 02:00 UTC",
    "retention_days": 30,
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "rpo_hours": 24,
    "rto_hours": 4,
    "enabled": true,
    "created_at": "2026-02-06T10:00:00Z",
    "updated_at": "2026-02-06T10:00:00Z",
    "next_run_at": "2026-02-07T02:00:00Z",
    "last_run_at": null,
    "last_run_status": null
  }
}
```

### GET /api/v1/dr/policies

List all DR policies.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cluster` | string | (all) | Filter by cluster |
| `enabled` | boolean | (all) | Filter by enabled status |

**Request:**

```bash
curl "http://localhost:8082/api/v1/dr/policies?enabled=true"
```

**Response (200 OK):**

```json
{
  "policies": [
    {
      "id": "drpol-uuid-001",
      "name": "prod-daily-backup-policy",
      "cluster": "prod-us-east-1",
      "schedule": "0 2 * * *",
      "schedule_human": "Daily at 02:00 UTC",
      "retention_days": 30,
      "rpo_hours": 24,
      "rto_hours": 4,
      "enabled": true,
      "next_run_at": "2026-02-07T02:00:00Z",
      "last_run_at": "2026-02-06T02:00:00Z",
      "last_run_status": "completed"
    },
    {
      "id": "drpol-uuid-002",
      "name": "prod-hourly-critical-namespaces",
      "cluster": "prod-us-east-1",
      "schedule": "0 * * * *",
      "schedule_human": "Every hour",
      "retention_days": 7,
      "rpo_hours": 1,
      "rto_hours": 1,
      "enabled": true,
      "next_run_at": "2026-02-06T19:00:00Z",
      "last_run_at": "2026-02-06T18:00:00Z",
      "last_run_status": "completed"
    }
  ],
  "total": 2
}
```

### GET /api/v1/dr/policies/{policy_id}

Get details of a specific DR policy.

**Request:**

```bash
curl http://localhost:8082/api/v1/dr/policies/drpol-uuid-001
```

**Response (200 OK):**

```json
{
  "policy": {
    "id": "drpol-uuid-001",
    "name": "prod-daily-backup-policy",
    "description": "Daily backups for all production namespaces with 30-day retention.",
    "cluster": "prod-us-east-1",
    "namespaces": ["*"],
    "exclude_namespaces": ["kube-system", "kube-public", "velero"],
    "schedule": "0 2 * * *",
    "schedule_human": "Daily at 02:00 UTC",
    "retention_days": 30,
    "include_volumes": true,
    "storage_backend": "s3",
    "storage_location": "s3://oco-backups/prod-us-east-1/",
    "rpo_hours": 24,
    "rto_hours": 4,
    "enabled": true,
    "created_at": "2026-02-06T10:00:00Z",
    "updated_at": "2026-02-06T10:00:00Z",
    "next_run_at": "2026-02-07T02:00:00Z",
    "last_run_at": "2026-02-06T02:00:00Z",
    "last_run_status": "completed",
    "run_history": [
      {"run_at": "2026-02-06T02:00:00Z", "status": "completed", "duration_seconds": 765},
      {"run_at": "2026-02-05T02:00:00Z", "status": "completed", "duration_seconds": 723},
      {"run_at": "2026-02-04T02:00:00Z", "status": "completed", "duration_seconds": 698}
    ]
  }
}
```

### PUT /api/v1/dr/policies/{policy_id}

Update an existing DR policy.

**Request:**

```bash
curl -X PUT http://localhost:8082/api/v1/dr/policies/drpol-uuid-001 \
  -H "Content-Type: application/json" \
  -d '{
    "retention_days": 60,
    "rpo_hours": 12
  }'
```

**Response (200 OK):**

```json
{
  "policy": {
    "id": "drpol-uuid-001",
    "name": "prod-daily-backup-policy",
    "retention_days": 60,
    "rpo_hours": 12,
    "rto_hours": 4,
    "updated_at": "2026-02-06T19:00:00Z"
  }
}
```

### DELETE /api/v1/dr/policies/{policy_id}

Delete a DR policy. Existing backup records created by this policy are not deleted.

**Request:**

```bash
curl -X DELETE http://localhost:8082/api/v1/dr/policies/drpol-uuid-002
```

**Response (204 No Content):**

No response body.

---

## DR Compliance

### GET /api/v1/dr/compliance

Get DR compliance status across all policies and clusters. Reports whether each policy is meeting its RPO/RTO targets.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cluster` | string | (all) | Filter by cluster |

**Request:**

```bash
curl "http://localhost:8082/api/v1/dr/compliance"
```

**Response (200 OK):**

```json
{
  "compliance": {
    "overall_status": "compliant",
    "total_policies": 2,
    "compliant_policies": 2,
    "non_compliant_policies": 0,
    "policies": [
      {
        "policy_id": "drpol-uuid-001",
        "policy_name": "prod-daily-backup-policy",
        "cluster": "prod-us-east-1",
        "rpo_target_hours": 24,
        "rpo_actual_hours": 16.0,
        "rpo_compliant": true,
        "rto_target_hours": 4,
        "rto_estimated_hours": 1.5,
        "rto_compliant": true,
        "last_successful_backup": "2026-02-06T02:12:45Z",
        "backup_count_30d": 30,
        "failed_backups_30d": 0
      },
      {
        "policy_id": "drpol-uuid-002",
        "policy_name": "prod-hourly-critical-namespaces",
        "cluster": "prod-us-east-1",
        "rpo_target_hours": 1,
        "rpo_actual_hours": 0.5,
        "rpo_compliant": true,
        "rto_target_hours": 1,
        "rto_estimated_hours": 0.3,
        "rto_compliant": true,
        "last_successful_backup": "2026-02-06T18:00:00Z",
        "backup_count_30d": 720,
        "failed_backups_30d": 2
      }
    ]
  }
}
```

**Response (200 OK) -- Non-Compliant Example:**

```json
{
  "compliance": {
    "overall_status": "non_compliant",
    "total_policies": 2,
    "compliant_policies": 1,
    "non_compliant_policies": 1,
    "policies": [
      {
        "policy_id": "drpol-uuid-001",
        "policy_name": "prod-daily-backup-policy",
        "cluster": "prod-us-east-1",
        "rpo_target_hours": 24,
        "rpo_actual_hours": 48.5,
        "rpo_compliant": false,
        "rto_target_hours": 4,
        "rto_estimated_hours": 1.5,
        "rto_compliant": true,
        "last_successful_backup": "2026-02-04T02:12:45Z",
        "backup_count_30d": 28,
        "failed_backups_30d": 2,
        "issue": "Last 2 scheduled backups failed. RPO target of 24 hours is not being met (actual: 48.5 hours since last successful backup)."
      }
    ]
  }
}
```

---

## Health Summary

### GET /api/v1/health/summary

Get a comprehensive health summary across all managed clusters, aggregating backup status, DR compliance, and recovery readiness.

**Request:**

```bash
curl http://localhost:8082/api/v1/health/summary
```

**Response (200 OK):**

```json
{
  "health_summary": {
    "overall_status": "healthy",
    "assessed_at": "2026-02-06T18:30:00Z",
    "clusters": [
      {
        "cluster": "prod-us-east-1",
        "status": "healthy",
        "backup_status": {
          "total_policies": 2,
          "active_policies": 2,
          "last_backup_at": "2026-02-06T18:00:00Z",
          "last_backup_status": "completed",
          "backups_last_24h": 25,
          "failed_last_24h": 0,
          "total_backup_size_bytes": 48318382080
        },
        "dr_compliance": {
          "compliant": true,
          "rpo_met": true,
          "rto_met": true
        },
        "recovery_readiness": {
          "restorable_backups": 42,
          "oldest_restorable": "2026-01-07T02:12:45Z",
          "newest_restorable": "2026-02-06T18:00:00Z",
          "estimated_full_restore_minutes": 15
        }
      },
      {
        "cluster": "staging-us-west-2",
        "status": "warning",
        "backup_status": {
          "total_policies": 1,
          "active_policies": 1,
          "last_backup_at": "2026-02-06T02:00:00Z",
          "last_backup_status": "completed",
          "backups_last_24h": 1,
          "failed_last_24h": 0,
          "total_backup_size_bytes": 8589934592
        },
        "dr_compliance": {
          "compliant": true,
          "rpo_met": true,
          "rto_met": true
        },
        "recovery_readiness": {
          "restorable_backups": 7,
          "oldest_restorable": "2026-01-30T02:00:00Z",
          "newest_restorable": "2026-02-06T02:00:00Z",
          "estimated_full_restore_minutes": 8
        }
      }
    ],
    "totals": {
      "total_clusters": 2,
      "healthy_clusters": 1,
      "warning_clusters": 1,
      "critical_clusters": 0,
      "total_backup_policies": 3,
      "total_backup_size_bytes": 56908316672,
      "total_restorable_backups": 49
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
    "message": "Human-readable description",
    "details": {}
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|-----------|-------------|
| 400 | `bad_request` | Malformed request body or invalid parameters |
| 404 | `not_found` | Resource not found (backup, plan, policy) |
| 409 | `conflict` | Conflicting operation (e.g., executing an already-executing plan) |
| 422 | `validation_error` | Request body fails validation (e.g., invalid cron schedule) |
| 424 | `dependency_failed` | Velero or Kubernetes is unavailable |
| 500 | `internal_error` | Unexpected server error |
| 503 | `service_unavailable` | Service is starting up or shutting down |

### Velero-Specific Errors

```json
{
  "error": {
    "code": "velero_error",
    "message": "Velero backup failed: unable to access storage location",
    "details": {
      "velero_backup_name": "oco-manual-backup-prod-2026-02-06-1707235200",
      "velero_error": "rpc error: code = Unknown desc = AccessDenied: Access Denied",
      "suggestion": "Verify that the Velero service account has write access to the S3 bucket 'oco-backups'."
    }
  }
}
```

---

## Data Models

### BackupJob

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique job identifier (UUID) |
| `name` | string | Human-readable job name |
| `cluster` | string | Kubernetes cluster name |
| `namespaces` | string[] | Namespaces to back up (`["*"]` for all) |
| `include_volumes` | boolean | Whether to include persistent volumes |
| `storage_backend` | string | Storage type: `s3`, `gcs`, `azure_blob` |
| `storage_location` | string | Storage URI (e.g., `s3://bucket/path/`) |
| `ttl_hours` | integer | Time-to-live for the backup in hours |
| `labels` | object | Key-value labels for organization |
| `status` | string | Job status: `pending`, `in_progress`, `completed`, `failed`, `partially_failed` |
| `velero_backup_name` | string | Corresponding Velero Backup resource name |
| `started_at` | datetime | Job start time |
| `completed_at` | datetime | Job completion time |
| `size_bytes` | integer | Total backup size in bytes |
| `resource_count` | integer | Number of Kubernetes resources backed up |
| `error_message` | string | Error details if the job failed |

### BackupRecord

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique record identifier |
| `backup_job_id` | string | Associated backup job ID |
| `name` | string | Backup name |
| `cluster` | string | Source cluster |
| `namespaces` | string[] | Backed-up namespaces |
| `storage_backend` | string | Storage type |
| `storage_location` | string | Full storage path to backup data |
| `size_bytes` | integer | Backup size |
| `resource_count` | integer | Number of resources in backup |
| `created_at` | datetime | When the backup was created |
| `expires_at` | datetime | When the backup expires (based on TTL) |
| `is_restorable` | boolean | Whether the backup can be restored |

### RecoveryPlan

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique plan identifier |
| `name` | string | Plan name |
| `backup_record_id` | string | Source backup to restore from |
| `target_cluster` | string | Target cluster for recovery |
| `target_namespaces` | string[] | Namespaces to restore |
| `restore_volumes` | boolean | Whether to restore persistent volumes |
| `namespace_mapping` | object | Remap namespaces (e.g., `{"prod": "staging"}`) |
| `preserve_node_ports` | boolean | Whether to preserve original NodePort assignments |
| `description` | string | Plan description |
| `status` | string | Status: `pending`, `approved`, `executing`, `completed`, `failed`, `cancelled` |
| `created_at` | datetime | Plan creation time |
| `created_by` | string | User who created the plan |

### DRPolicy

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique policy identifier |
| `name` | string | Policy name |
| `description` | string | Policy description |
| `cluster` | string | Target cluster |
| `namespaces` | string[] | Namespaces covered |
| `exclude_namespaces` | string[] | Namespaces to exclude |
| `schedule` | string | Cron expression for backup schedule |
| `retention_days` | integer | How long to retain backups |
| `include_volumes` | boolean | Whether to back up persistent volumes |
| `storage_backend` | string | Storage type |
| `storage_location` | string | Storage URI |
| `rpo_hours` | integer | Recovery Point Objective in hours |
| `rto_hours` | integer | Recovery Time Objective in hours |
| `enabled` | boolean | Whether the policy is active |
| `created_at` | datetime | Policy creation time |
| `updated_at` | datetime | Last update time |
| `next_run_at` | datetime | Next scheduled backup time |
| `last_run_at` | datetime | Last backup execution time |
| `last_run_status` | string | Status of the last backup run |
