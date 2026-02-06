// Package cache provides a Redis client wrapper for caching and budget tracking
// in the Cerebra LLM Gateway. It supports fast budget spend lookups and atomic
// increment operations for real-time cost tracking.
package cache

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client with Cerebra-specific caching operations.
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Redis cache client connected to the given address.
// The redisURL should be in "host:port" format.
func NewCache(ctx context.Context, redisURL string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	// Verify connectivity
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache: failed to connect to Redis at %s: %w", redisURL, err)
	}

	log.Printf("cache: connected to Redis at %s", redisURL)
	return &Cache{client: client}, nil
}

// Close gracefully shuts down the Redis client connection.
func (c *Cache) Close() error {
	if c.client != nil {
		log.Println("cache: closing Redis connection")
		return c.client.Close()
	}
	return nil
}

// Get retrieves a value from the cache by key.
// Returns an empty string and no error if the key does not exist.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cache: get %q: %w", key, err)
	}
	return val, nil
}

// Set stores a key-value pair in the cache with the given TTL.
// A zero TTL means the key will not expire.
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

// Delete removes one or more keys from the cache.
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: delete: %w", err)
	}
	return nil
}

// budgetKey constructs the Redis key for budget spend tracking.
// Format: "budget:spend:{scope}:{entityID}" with a monthly TTL.
func budgetKey(scope, entityID string) string {
	return fmt.Sprintf("budget:spend:%s:%s", scope, entityID)
}

// GetBudgetSpend retrieves the current accumulated spend for a given budget scope
// and entity from Redis. Returns 0 if no spend has been recorded yet.
func (c *Cache) GetBudgetSpend(ctx context.Context, scope, entityID string) (float64, error) {
	key := budgetKey(scope, entityID)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cache: get budget spend %q: %w", key, err)
	}

	spend, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("cache: parse budget spend %q=%q: %w", key, val, err)
	}
	return spend, nil
}

// IncrBudgetSpend atomically increments the budget spend for a given scope and entity.
// It uses INCRBYFLOAT for atomic updates and sets a 31-day TTL on first creation
// to automatically clean up stale budget data after a billing cycle.
func (c *Cache) IncrBudgetSpend(ctx context.Context, scope, entityID string, amount float64) (float64, error) {
	key := budgetKey(scope, entityID)

	// Atomically increment the spend
	newVal, err := c.client.IncrByFloat(ctx, key, amount).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: incr budget spend %q: %w", key, err)
	}

	// Set TTL if this is a new key (TTL returns -1 for keys without expiry, -2 for missing)
	ttl, err := c.client.TTL(ctx, key).Result()
	if err == nil && (ttl == -1*time.Second || ttl < 0) {
		// Set a 31-day expiry to cover a full billing month plus buffer
		c.client.Expire(ctx, key, 31*24*time.Hour)
	}

	return newVal, nil
}

// SetBudgetSpend directly sets the budget spend for a given scope and entity.
// This is used when initializing from database values.
func (c *Cache) SetBudgetSpend(ctx context.Context, scope, entityID string, amount float64) error {
	key := budgetKey(scope, entityID)
	amountStr := strconv.FormatFloat(amount, 'f', 10, 64)
	if err := c.client.Set(ctx, key, amountStr, 31*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("cache: set budget spend %q: %w", key, err)
	}
	return nil
}

// RateLimitCheck performs a sliding window rate limit check for a given key.
// It returns true if the request is allowed (under limit), false if rate-limited.
// The window is defined by the TTL and maxRequests is the limit within that window.
func (c *Cache) RateLimitCheck(ctx context.Context, key string, maxRequests int64, window time.Duration) (bool, error) {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)

	// Use a pipeline for atomic increment + expire
	pipe := c.client.Pipeline()
	incrCmd := pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("cache: rate limit check: %w", err)
	}

	count := incrCmd.Val()
	return count <= maxRequests, nil
}

// Client returns the underlying Redis client for advanced operations.
func (c *Cache) Client() *redis.Client {
	return c.client
}
