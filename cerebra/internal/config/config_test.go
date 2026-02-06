package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure env vars are clean.
	os.Unsetenv("CEREBRA_PORT")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("REDIS_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected default DB host localhost, got %s", cfg.DBHost)
	}
	if cfg.DBPort != 5432 {
		t.Errorf("expected default DB port 5432, got %d", cfg.DBPort)
	}
	if cfg.RedisPort != 6379 {
		t.Errorf("expected default Redis port 6379, got %d", cfg.RedisPort)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("CEREBRA_PORT", "9090")
	os.Setenv("POSTGRES_HOST", "db.example.com")
	os.Setenv("POSTGRES_PORT", "5433")
	defer func() {
		os.Unsetenv("CEREBRA_PORT")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("expected DB host db.example.com, got %s", cfg.DBHost)
	}
	if cfg.DBPort != 5433 {
		t.Errorf("expected DB port 5433, got %d", cfg.DBPort)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("POSTGRES_PORT", "not_a_number")
	defer os.Unsetenv("POSTGRES_PORT")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid POSTGRES_PORT, got nil")
	}
}

func TestDSN(t *testing.T) {
	cfg := &Config{
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if cfg.DSN() != expected {
		t.Errorf("DSN() = %s, want %s", cfg.DSN(), expected)
	}
}

func TestRedisAddr(t *testing.T) {
	cfg := &Config{
		RedisHost: "redis.example.com",
		RedisPort: 6380,
	}

	expected := "redis.example.com:6380"
	if cfg.RedisAddr() != expected {
		t.Errorf("RedisAddr() = %s, want %s", cfg.RedisAddr(), expected)
	}
}
