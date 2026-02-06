// Package config handles application configuration loading from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration values for the Cerebra LLM Gateway.
type Config struct {
	Port           string
	LogLevel       string
	DatabaseURL    string
	RedisURL       string
	AllowedOrigins []string
}

// Load reads configuration from environment variables and returns a Config.
// It follows the .env.example pattern using POSTGRES_*, REDIS_*, and CEREBRA_* prefixes.
func Load() (*Config, error) {
	cfg := &Config{}

	// Cerebra-specific settings
	cfg.Port = getEnvOrDefault("CEREBRA_PORT", "8080")
	cfg.LogLevel = getEnvOrDefault("CEREBRA_LOG_LEVEL", "info")

	// Build PostgreSQL connection URL from individual components
	pgHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
	pgPort := getEnvOrDefault("POSTGRES_PORT", "5432")
	pgDB := getEnvOrDefault("POSTGRES_DB", "cerebra")
	pgUser := getEnvOrDefault("POSTGRES_USER", "cerebra")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")

	if pgPassword == "" {
		cfg.DatabaseURL = fmt.Sprintf(
			"postgres://%s@%s:%s/%s?sslmode=disable",
			pgUser, pgHost, pgPort, pgDB,
		)
	} else {
		cfg.DatabaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			pgUser, pgPassword, pgHost, pgPort, pgDB,
		)
	}

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
	originsStr := getEnvOrDefault("CEREBRA_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.AllowedOrigins = strings.Split(originsStr, ",")
	for i, origin := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(origin)
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("config: CEREBRA_PORT is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: database URL could not be constructed")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("config: Redis URL could not be constructed")
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
