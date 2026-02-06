// Package config handles application configuration loading from environment variables.
//
// Configuration follows the same patterns as other Open Cloud Ops modules,
// using AEGIS_* prefixed environment variables with sensible defaults for
// local development. Database and Redis configuration uses the shared
// POSTGRES_* and REDIS_* prefixes.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration values for the Aegis Resilience Engine.
type Config struct {
	// Port is the HTTP port the API server listens on.
	Port string

	// LogLevel controls the verbosity of log output (debug, info, warn, error).
	LogLevel string

	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string

	// RedisURL is the Redis connection address.
	RedisURL string

	// KubeConfigPath is the path to the kubeconfig file.
	// If empty, in-cluster configuration is used.
	KubeConfigPath string

	// BackupStoragePath is the root directory for storing backup archives.
	BackupStoragePath string

	// DefaultRetentionDays is the default number of days to retain backups
	// when no explicit retention is configured on a job.
	DefaultRetentionDays int

	// AllowedOrigins defines the CORS allowed origins for the API.
	AllowedOrigins []string
}

// Load reads configuration from environment variables and returns a Config.
// It follows the .env.example pattern using POSTGRES_*, REDIS_*, and AEGIS_* prefixes.
func Load() (*Config, error) {
	cfg := &Config{}

	// Aegis-specific settings
	cfg.Port = getEnvOrDefault("AEGIS_PORT", "8082")
	cfg.LogLevel = getEnvOrDefault("AEGIS_LOG_LEVEL", "info")
	cfg.KubeConfigPath = os.Getenv("AEGIS_KUBECONFIG")
	cfg.BackupStoragePath = getEnvOrDefault("AEGIS_BACKUP_STORAGE_PATH", "/var/aegis/backups")

	retentionStr := getEnvOrDefault("AEGIS_DEFAULT_RETENTION_DAYS", "30")
	retentionDays, err := strconv.Atoi(retentionStr)
	if err != nil {
		return nil, fmt.Errorf("config: invalid AEGIS_DEFAULT_RETENTION_DAYS value %q: %w", retentionStr, err)
	}
	cfg.DefaultRetentionDays = retentionDays

	// Build PostgreSQL connection URL from individual components
	pgHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
	pgPort := getEnvOrDefault("POSTGRES_PORT", "5432")
	pgDB := getEnvOrDefault("POSTGRES_DB", "aegis")
	pgUser := getEnvOrDefault("POSTGRES_USER", "aegis")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgSSLMode := getEnvOrDefault("POSTGRES_SSLMODE", "require")

	// Use url.UserPassword to properly percent-encode credentials that may
	// contain reserved URI characters (@, :, /, etc.).
	dsn := &url.URL{
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%s", pgHost, pgPort),
		Path:     pgDB,
		RawQuery: fmt.Sprintf("sslmode=%s", pgSSLMode),
	}
	if pgPassword == "" {
		dsn.User = url.User(pgUser)
	} else {
		dsn.User = url.UserPassword(pgUser, pgPassword)
	}
	cfg.DatabaseURL = dsn.String()

	// Allow overriding with a full DATABASE_URL if provided
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	// Build Redis URL
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefault("REDIS_PORT", "6379")
	cfg.RedisURL = fmt.Sprintf("%s:%s", redisHost, redisPort)

	// Allow overriding with a full REDIS_URL if provided
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisURL = redisURL
	}

	// CORS allowed origins
	originsStr := getEnvOrDefault("AEGIS_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.AllowedOrigins = strings.Split(originsStr, ",")
	for i, origin := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(origin)
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are set and valid.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("config: AEGIS_PORT is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: database URL could not be constructed")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("config: Redis URL could not be constructed")
	}
	if c.BackupStoragePath == "" {
		return fmt.Errorf("config: AEGIS_BACKUP_STORAGE_PATH is required")
	}
	if c.DefaultRetentionDays <= 0 {
		return fmt.Errorf("config: AEGIS_DEFAULT_RETENTION_DAYS must be positive")
	}
	return nil
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or the defaultValue if the variable is not set or empty.
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
