// Package database provides PostgreSQL connection management and schema initialization
// for the Cerebra LLM Gateway. It uses pgx for high-performance database access.
package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/config"
)

// DB wraps a pgx connection pool with Cerebra-specific functionality.
type DB struct {
	Pool *pgxpool.Pool
}

// NewPool creates a new PostgreSQL connection pool from the given configuration.
// It configures pool sizing, timeouts, and verifies connectivity with a ping.
func NewPool(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: failed to parse connection URL: %w", err)
	}

	// Configure connection pool settings for production use
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: failed to ping database: %w", err)
	}

	db := &DB{Pool: pool}

	// Initialize schema
	if err := db.initSchema(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: failed to initialize schema: %w", err)
	}

	log.Println("database: connected and schema initialized")
	return db, nil
}

// Close gracefully shuts down the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Println("database: connection pool closed")
	}
}

// initSchema creates the required database tables if they do not already exist.
// It uses TimescaleDB hypertable for the api_requests table to enable efficient
// time-series queries on request data.
func (db *DB) initSchema(ctx context.Context) error {
	schema := `
	-- Enable TimescaleDB extension if available (fails gracefully if not installed)
	CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

	-- Organizations table
	CREATE TABLE IF NOT EXISTS organizations (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Teams table
	CREATE TABLE IF NOT EXISTS teams (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Agents table
	CREATE TABLE IF NOT EXISTS agents (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		team_id     TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		tags        TEXT[] DEFAULT '{}',
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Model pricing table
	CREATE TABLE IF NOT EXISTS model_pricing (
		provider            TEXT NOT NULL,
		model               TEXT NOT NULL,
		input_per_m_token   DOUBLE PRECISION NOT NULL DEFAULT 0,
		output_per_m_token  DOUBLE PRECISION NOT NULL DEFAULT 0,
		updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (provider, model)
	);

	-- Budgets table
	CREATE TABLE IF NOT EXISTS budgets (
		id          TEXT PRIMARY KEY,
		scope       TEXT NOT NULL,
		entity_id   TEXT NOT NULL,
		limit_usd   DOUBLE PRECISION NOT NULL DEFAULT 0,
		spent_usd   DOUBLE PRECISION NOT NULL DEFAULT 0,
		period      INTERVAL NOT NULL DEFAULT INTERVAL '30 days',
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(scope, entity_id)
	);

	-- API requests table (time-series data for cost tracking)
	CREATE TABLE IF NOT EXISTS api_requests (
		id              TEXT NOT NULL,
		provider        TEXT NOT NULL,
		model           TEXT NOT NULL,
		agent_id        TEXT NOT NULL DEFAULT '',
		team_id         TEXT NOT NULL DEFAULT '',
		org_id          TEXT NOT NULL DEFAULT '',
		input_tokens    BIGINT NOT NULL DEFAULT 0,
		output_tokens   BIGINT NOT NULL DEFAULT 0,
		total_tokens    BIGINT NOT NULL DEFAULT 0,
		cost_usd        DOUBLE PRECISION NOT NULL DEFAULT 0,
		latency_ms      BIGINT NOT NULL DEFAULT 0,
		status_code     INTEGER NOT NULL DEFAULT 0,
		was_routed      BOOLEAN NOT NULL DEFAULT FALSE,
		original_model  TEXT NOT NULL DEFAULT '',
		routed_model    TEXT NOT NULL DEFAULT '',
		savings_usd     DOUBLE PRECISION NOT NULL DEFAULT 0,
		timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Create indexes for common query patterns
	CREATE INDEX IF NOT EXISTS idx_api_requests_timestamp ON api_requests (timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_api_requests_agent_id ON api_requests (agent_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_api_requests_team_id ON api_requests (team_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_api_requests_org_id ON api_requests (org_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_api_requests_model ON api_requests (model, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_api_requests_provider ON api_requests (provider, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_budgets_scope_entity ON budgets (scope, entity_id);
	`

	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Convert api_requests to a TimescaleDB hypertable if not already done.
	// This call is idempotent when migrate_data => true and if_not_exists is used.
	hypertableSQL := `
	SELECT create_hypertable('api_requests', 'timestamp',
		migrate_data => true,
		if_not_exists => true
	);
	`
	_, err = db.Pool.Exec(ctx, hypertableSQL)
	if err != nil {
		// Log but do not fail; TimescaleDB may not be installed in all environments.
		log.Printf("database: warning: could not create hypertable (TimescaleDB may not be installed): %v", err)
	}

	return nil
}
