-- =============================================================================
-- Open Cloud Ops - Database Initialization Script
-- =============================================================================
-- This script runs automatically on first PostgreSQL startup via
-- docker-entrypoint-initdb.d. It creates all tables, hypertables,
-- indexes, and seed data for the three application modules.
-- =============================================================================

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- Cerebra (LLM Gateway) Tables
-- =============================================================================

-- Organizations using the LLM gateway
CREATE TABLE IF NOT EXISTS organizations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL UNIQUE,
    slug            VARCHAR(255) NOT NULL UNIQUE,
    billing_email   VARCHAR(255),
    max_monthly_budget_cents BIGINT DEFAULT 0,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Teams within an organization
CREATE TABLE IF NOT EXISTS teams (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(255) NOT NULL,
    max_monthly_budget_cents BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, slug)
);

-- AI agents registered with the gateway
CREATE TABLE IF NOT EXISTS agents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    api_key_hash    VARCHAR(512) NOT NULL,
    model_whitelist TEXT[],
    rate_limit_rpm  INT DEFAULT 60,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Model pricing reference table
CREATE TABLE IF NOT EXISTS model_pricing (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider        VARCHAR(50) NOT NULL,
    model_name      VARCHAR(255) NOT NULL,
    input_cost_per_million_tokens  NUMERIC(12, 6) NOT NULL,
    output_cost_per_million_tokens NUMERIC(12, 6) NOT NULL,
    effective_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, model_name, effective_date)
);

-- Budgets for organizations and teams
CREATE TABLE IF NOT EXISTS budgets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    team_id         UUID REFERENCES teams(id) ON DELETE CASCADE,
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    budget_cents    BIGINT NOT NULL,
    spent_cents     BIGINT DEFAULT 0,
    alert_threshold_pct INT DEFAULT 80,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT budget_has_owner CHECK (organization_id IS NOT NULL OR team_id IS NOT NULL)
);

-- Time-series log of every API request through the gateway
CREATE TABLE IF NOT EXISTS api_requests (
    time            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id              UUID DEFAULT uuid_generate_v4(),
    agent_id        UUID NOT NULL,
    organization_id UUID NOT NULL,
    team_id         UUID NOT NULL,
    provider        VARCHAR(50) NOT NULL,
    model           VARCHAR(255) NOT NULL,
    input_tokens    INT NOT NULL DEFAULT 0,
    output_tokens   INT NOT NULL DEFAULT 0,
    total_tokens    INT NOT NULL DEFAULT 0,
    cost_cents      NUMERIC(12, 4) NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 0,
    status_code     INT NOT NULL DEFAULT 200,
    error_message   TEXT,
    metadata        JSONB DEFAULT '{}'
);

-- =============================================================================
-- Economist (FinOps Core) Tables
-- =============================================================================

-- Time-series cloud cost data ingested from providers
CREATE TABLE IF NOT EXISTS cloud_costs (
    time            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id              UUID DEFAULT uuid_generate_v4(),
    provider        VARCHAR(50) NOT NULL,
    account_id      VARCHAR(255) NOT NULL,
    service         VARCHAR(255) NOT NULL,
    region          VARCHAR(100),
    resource_id     VARCHAR(512),
    resource_name   VARCHAR(512),
    cost_amount     NUMERIC(14, 6) NOT NULL,
    currency        VARCHAR(10) DEFAULT 'USD',
    usage_quantity  NUMERIC(14, 6),
    usage_unit      VARCHAR(100),
    tags            JSONB DEFAULT '{}',
    metadata        JSONB DEFAULT '{}'
);

-- AI-driven optimization recommendations
CREATE TABLE IF NOT EXISTS optimization_recommendations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider        VARCHAR(50) NOT NULL,
    account_id      VARCHAR(255) NOT NULL,
    resource_id     VARCHAR(512),
    resource_name   VARCHAR(512),
    category        VARCHAR(100) NOT NULL,
    title           VARCHAR(512) NOT NULL,
    description     TEXT NOT NULL,
    estimated_monthly_savings_cents BIGINT DEFAULT 0,
    effort_level    VARCHAR(20) DEFAULT 'medium',
    risk_level      VARCHAR(20) DEFAULT 'low',
    status          VARCHAR(50) DEFAULT 'open',
    implemented_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Governance policies for cost management
CREATE TABLE IF NOT EXISTS governance_policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    policy_type     VARCHAR(100) NOT NULL,
    provider        VARCHAR(50),
    conditions      JSONB NOT NULL,
    actions         JSONB NOT NULL,
    severity        VARCHAR(20) DEFAULT 'warning',
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Policy violation records
CREATE TABLE IF NOT EXISTS policy_violations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id       UUID NOT NULL REFERENCES governance_policies(id) ON DELETE CASCADE,
    provider        VARCHAR(50) NOT NULL,
    account_id      VARCHAR(255) NOT NULL,
    resource_id     VARCHAR(512),
    violation_detail TEXT NOT NULL,
    severity        VARCHAR(20) NOT NULL,
    status          VARCHAR(50) DEFAULT 'open',
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Aegis (Resilience Engine) Tables
-- =============================================================================

-- Backup job definitions (columns aligned with Go models.BackupJob)
CREATE TABLE IF NOT EXISTS backup_jobs (
    id                VARCHAR(64) PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    namespace         VARCHAR(255) NOT NULL,
    resource_types    TEXT[] NOT NULL DEFAULT '{}',
    schedule          VARCHAR(100) NOT NULL,
    retention_days    INT DEFAULT 30,
    storage_location  VARCHAR(512) DEFAULT '',
    status            VARCHAR(50) DEFAULT 'active',
    last_run          TIMESTAMPTZ,
    next_run          TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Individual backup execution records (columns aligned with Go models.BackupRecord)
CREATE TABLE IF NOT EXISTS backup_records (
    id              VARCHAR(64) PRIMARY KEY,
    job_id          VARCHAR(64) NOT NULL REFERENCES backup_jobs(id) ON DELETE CASCADE,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    size_bytes      BIGINT DEFAULT 0,
    duration_ms     BIGINT DEFAULT 0,
    resource_count  INT DEFAULT 0,
    storage_path    VARCHAR(512) DEFAULT '',
    error_message   TEXT DEFAULT '',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

-- Disaster recovery plans
CREATE TABLE IF NOT EXISTS recovery_plans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    target_rto_minutes INT NOT NULL,
    target_rpo_minutes INT NOT NULL,
    steps           JSONB NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    last_tested_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recovery execution history
CREATE TABLE IF NOT EXISTS recovery_executions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id         UUID NOT NULL REFERENCES recovery_plans(id) ON DELETE CASCADE,
    trigger_type    VARCHAR(50) NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    actual_rto_minutes INT,
    steps_completed JSONB DEFAULT '[]',
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Disaster recovery policies
CREATE TABLE IF NOT EXISTS dr_policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    policy_type     VARCHAR(100) NOT NULL,
    conditions      JSONB NOT NULL,
    actions         JSONB NOT NULL,
    priority        INT DEFAULT 0,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Infrastructure health check records
CREATE TABLE IF NOT EXISTS health_checks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_name     VARCHAR(255) NOT NULL,
    target_type     VARCHAR(100) NOT NULL,
    target_endpoint VARCHAR(512),
    status          VARCHAR(50) NOT NULL,
    response_time_ms INT,
    details         JSONB DEFAULT '{}',
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API keys for authentication (shared across modules)
CREATE TABLE IF NOT EXISTS api_keys (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_prefix      VARCHAR(8) NOT NULL,
    key_hash        VARCHAR(64) NOT NULL,
    entity_id       UUID NOT NULL,
    entity_type     VARCHAR(50) NOT NULL DEFAULT 'organization',
    description     TEXT DEFAULT '',
    revoked         BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys (key_prefix) WHERE revoked = false;

-- =============================================================================
-- TimescaleDB Hypertables (for time-series data)
-- =============================================================================

SELECT create_hypertable('api_requests', 'time', if_not_exists => TRUE);
SELECT create_hypertable('cloud_costs', 'time', if_not_exists => TRUE);

-- =============================================================================
-- Indexes for Common Query Patterns
-- =============================================================================

-- Cerebra indexes
CREATE INDEX IF NOT EXISTS idx_api_requests_agent_id ON api_requests (agent_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_api_requests_org_id ON api_requests (organization_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_api_requests_team_id ON api_requests (team_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_api_requests_provider_model ON api_requests (provider, model, time DESC);
CREATE INDEX IF NOT EXISTS idx_agents_team_id ON agents (team_id);
CREATE INDEX IF NOT EXISTS idx_teams_org_id ON teams (organization_id);
CREATE INDEX IF NOT EXISTS idx_budgets_org_id ON budgets (organization_id);
CREATE INDEX IF NOT EXISTS idx_budgets_team_id ON budgets (team_id);
CREATE INDEX IF NOT EXISTS idx_model_pricing_provider ON model_pricing (provider, model_name, effective_date DESC);

-- Economist indexes
CREATE INDEX IF NOT EXISTS idx_cloud_costs_provider ON cloud_costs (provider, time DESC);
CREATE INDEX IF NOT EXISTS idx_cloud_costs_account ON cloud_costs (account_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_cloud_costs_service ON cloud_costs (service, time DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_status ON optimization_recommendations (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_provider ON optimization_recommendations (provider, account_id);
CREATE INDEX IF NOT EXISTS idx_policy_violations_status ON policy_violations (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_violations_policy ON policy_violations (policy_id);

-- Aegis indexes
CREATE INDEX IF NOT EXISTS idx_backup_records_job ON backup_records (job_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_records_status ON backup_records (status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_recovery_executions_plan ON recovery_executions (plan_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_target ON health_checks (target_name, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_status ON health_checks (status, checked_at DESC);

-- =============================================================================
-- Seed Data: Default Model Pricing
-- =============================================================================

INSERT INTO model_pricing (provider, model_name, input_cost_per_million_tokens, output_cost_per_million_tokens)
VALUES
    -- OpenAI models
    ('openai', 'gpt-4o',              2.500,  10.000),
    ('openai', 'gpt-4o-mini',         0.150,   0.600),
    ('openai', 'gpt-4-turbo',        10.000,  30.000),
    ('openai', 'gpt-4',              30.000,  60.000),
    ('openai', 'gpt-3.5-turbo',      0.500,   1.500),
    ('openai', 'o1',                 15.000,  60.000),
    ('openai', 'o1-mini',             3.000,  12.000),

    -- Anthropic models
    ('anthropic', 'claude-3-5-sonnet-20241022', 3.000,  15.000),
    ('anthropic', 'claude-3-5-haiku-20241022',  0.800,   4.000),
    ('anthropic', 'claude-3-opus-20240229',    15.000,  75.000),
    ('anthropic', 'claude-3-sonnet-20240229',   3.000,  15.000),
    ('anthropic', 'claude-3-haiku-20240307',    0.250,   1.250),

    -- Google models
    ('google', 'gemini-1.5-pro',      3.500,  10.500),
    ('google', 'gemini-1.5-flash',    0.075,   0.300),
    ('google', 'gemini-2.0-flash',    0.100,   0.400),
    ('google', 'gemini-1.0-pro',      0.500,   1.500)
ON CONFLICT (provider, model_name, effective_date) DO NOTHING;
