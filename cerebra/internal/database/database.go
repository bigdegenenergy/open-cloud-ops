// Package database manages PostgreSQL connections and provides the data access layer.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the PostgreSQL connection pool and provides query methods.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool.
func New(dsn string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// Migrate runs database schema migrations.
// An advisory lock prevents concurrent replicas from racing on DDL statements.
func (db *DB) Migrate(ctx context.Context) error {
	// Acquire a dedicated connection for the advisory lock.
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection for migration: %w", err)
	}
	defer conn.Release()

	// Application-specific lock ID to avoid collisions with other apps on the
	// same PostgreSQL instance. Derived from CRC32('cerebra_migrations').
	const migrationLockID int64 = 0x4F43_4F01 // "OCO" prefix + 01
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquiring migration lock: %w", err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockID)

	schema := `
	CREATE TABLE IF NOT EXISTS organizations (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS teams (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		org_id      TEXT NOT NULL REFERENCES organizations(id),
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS agents (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		team_id     TEXT NOT NULL REFERENCES teams(id),
		tags        TEXT[] DEFAULT '{}',
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS model_pricing (
		provider           TEXT NOT NULL,
		model              TEXT NOT NULL,
		input_per_m_token  DOUBLE PRECISION NOT NULL,
		output_per_m_token DOUBLE PRECISION NOT NULL,
		updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (provider, model)
	);

	CREATE TABLE IF NOT EXISTS api_requests (
		id              TEXT PRIMARY KEY,
		provider        TEXT NOT NULL,
		model           TEXT NOT NULL,
		agent_id        TEXT NOT NULL,
		team_id         TEXT NOT NULL,
		org_id          TEXT NOT NULL,
		input_tokens    BIGINT NOT NULL DEFAULT 0,
		output_tokens   BIGINT NOT NULL DEFAULT 0,
		total_tokens    BIGINT NOT NULL DEFAULT 0,
		cost_usd        DOUBLE PRECISION NOT NULL DEFAULT 0,
		latency_ms      BIGINT NOT NULL DEFAULT 0,
		status_code     INTEGER NOT NULL DEFAULT 0,
		was_routed      BOOLEAN NOT NULL DEFAULT FALSE,
		original_model  TEXT,
		routed_model    TEXT,
		savings_usd     DOUBLE PRECISION NOT NULL DEFAULT 0,
		timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS budgets (
		id          TEXT PRIMARY KEY,
		scope       TEXT NOT NULL,
		entity_id   TEXT NOT NULL,
		limit_usd   DOUBLE PRECISION NOT NULL,
		spent_usd   DOUBLE PRECISION NOT NULL DEFAULT 0,
		period_days INTEGER NOT NULL DEFAULT 30,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE (scope, entity_id)
	);

	CREATE INDEX IF NOT EXISTS idx_api_requests_agent_id ON api_requests(agent_id);
	CREATE INDEX IF NOT EXISTS idx_api_requests_team_id ON api_requests(team_id);
	CREATE INDEX IF NOT EXISTS idx_api_requests_org_id ON api_requests(org_id);
	CREATE INDEX IF NOT EXISTS idx_api_requests_timestamp ON api_requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_api_requests_provider ON api_requests(provider);
	CREATE INDEX IF NOT EXISTS idx_api_requests_model ON api_requests(model);
	`

	_, err = conn.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

// SeedPricing inserts default model pricing data.
func (db *DB) SeedPricing(ctx context.Context) error {
	pricing := []struct {
		Provider string
		Model    string
		Input    float64
		Output   float64
	}{
		// OpenAI
		{"openai", "gpt-4o", 2.50, 10.00},
		{"openai", "gpt-4o-mini", 0.15, 0.60},
		{"openai", "gpt-4-turbo", 10.00, 30.00},
		{"openai", "o1", 15.00, 60.00},
		{"openai", "o1-mini", 3.00, 12.00},
		// Anthropic
		{"anthropic", "claude-sonnet-4-20250514", 3.00, 15.00},
		{"anthropic", "claude-haiku-3-20250414", 0.25, 1.25},
		{"anthropic", "claude-opus-4-20250514", 15.00, 75.00},
		// Google Gemini
		{"gemini", "gemini-2.0-flash", 0.10, 0.40},
		{"gemini", "gemini-2.0-pro", 1.25, 10.00},
		{"gemini", "gemini-1.5-flash", 0.075, 0.30},
	}

	for _, p := range pricing {
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO model_pricing (provider, model, input_per_m_token, output_per_m_token)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (provider, model) DO UPDATE
			SET input_per_m_token = EXCLUDED.input_per_m_token,
			    output_per_m_token = EXCLUDED.output_per_m_token,
			    updated_at = NOW()
		`, p.Provider, p.Model, p.Input, p.Output)
		if err != nil {
			return fmt.Errorf("seeding pricing for %s/%s: %w", p.Provider, p.Model, err)
		}
	}

	return nil
}
