// Package config handles loading and validating configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the Cerebra LLM Gateway.
type Config struct {
	// Server
	Port     string
	LogLevel string

	// Database
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// Database SSL
	DBSSLMode string

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string

	// Provider API Keys (passed through, never stored)
	OpenAIKey    string
	AnthropicKey string
	GeminiKey    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("CEREBRA_PORT", "8080"),
		LogLevel: getEnv("CEREBRA_LOG_LEVEL", "info"),

		DBHost:     getEnv("POSTGRES_HOST", "localhost"),
		DBName:     getEnv("POSTGRES_DB", "opencloudops"),
		DBUser:     getEnv("POSTGRES_USER", "oco_user"),
		DBPassword: getEnv("POSTGRES_PASSWORD", ""),
		DBSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		GeminiKey:    os.Getenv("GOOGLE_API_KEY"),
	}

	dbPort, err := strconv.Atoi(getEnv("POSTGRES_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}
	cfg.DBPort = dbPort

	redisPort, err := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}
	cfg.RedisPort = redisPort

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

// RedisAddr returns the Redis address in host:port format.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
