// Package middleware provides Gin middleware functions for the Cerebra LLM Gateway.
// It includes CORS handling, request logging, rate limiting, and API key authentication.
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/cache"
)

// CORSMiddleware returns a Gin middleware handler that sets CORS headers
// to allow cross-origin requests from specified origins.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	originsMap := make(map[string]bool)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		originsMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowAll || originsMap[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Agent-ID, X-Team-ID, X-Org-ID")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware returns a Gin middleware handler that logs request and
// response metadata including method, path, status code, latency, and client IP.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process the request
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		bodySize := c.Writer.Size()

		if query != "" {
			path = path + "?" + query
		}

		// Determine log level based on status code
		switch {
		case statusCode >= 500:
			log.Printf("[ERROR] %s %s | %d | %v | %s | %d bytes | errors: %s",
				method, path, statusCode, latency, clientIP, bodySize, c.Errors.ByType(gin.ErrorTypePrivate).String())
		case statusCode >= 400:
			log.Printf("[WARN]  %s %s | %d | %v | %s | %d bytes",
				method, path, statusCode, latency, clientIP, bodySize)
		default:
			log.Printf("[INFO]  %s %s | %d | %v | %s | %d bytes",
				method, path, statusCode, latency, clientIP, bodySize)
		}
	}
}

// RateLimitMiddleware returns a Gin middleware handler that enforces per-API-key
// rate limiting using Redis. It allows maxRequests within the specified window.
func RateLimitMiddleware(c *cache.Cache, maxRequests int64, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Extract the API key to use as the rate limit identifier
		apiKey := ctx.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = ctx.GetHeader("Authorization")
		}

		// If no API key, rate limit by IP address
		if apiKey == "" {
			apiKey = ctx.ClientIP()
		}

		// Clean the key (remove "Bearer " prefix if present)
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")

		// Use only the first 16 chars of the key for privacy in Redis
		rateLimitID := apiKey
		if len(rateLimitID) > 16 {
			rateLimitID = rateLimitID[:16]
		}

		allowed, err := c.RateLimitCheck(ctx.Request.Context(), rateLimitID, maxRequests, window)
		if err != nil {
			// On Redis error, allow the request but log the issue
			log.Printf("middleware: rate limit check error: %v", err)
			ctx.Next()
			return
		}

		if !allowed {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
			})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// hashAPIKey returns the hex-encoded SHA-256 hash of the given API key.
func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// AuthMiddleware returns a Gin middleware handler that validates the X-API-Key header
// against the api_keys table in the database. Keys are validated by looking up
// the key_prefix (first 8 chars) and comparing the full SHA-256 hash.
// Results are cached in Redis using the hash (not the raw key) as the cache key.
func AuthMiddleware(pool *pgxpool.Pool, redisCache *cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		// Also check Authorization header with Bearer scheme
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing API key. Provide X-API-Key header or Authorization: Bearer <key>.",
			})
			c.Abort()
			return
		}

		// Format validation: keys must be at least 16 chars to have a prefix + secret
		if len(apiKey) < 16 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid API key format.",
			})
			c.Abort()
			return
		}

		keyHash := hashAPIKey(apiKey)
		// Use hash-based cache key (never store raw key in cache)
		cacheKey := "auth:" + keyHash[:16]

		// Check Redis cache first for previously validated keys
		if redisCache != nil {
			entityID, err := redisCache.Get(c.Request.Context(), cacheKey)
			if err == nil && entityID != "" {
				c.Set("api_key", apiKey)
				c.Set("entity_id", entityID)
				c.Next()
				return
			}
		}

		// Validate against the api_keys table
		if pool != nil {
			var entityID, storedHash string
			err := pool.QueryRow(
				c.Request.Context(),
				`SELECT entity_id, key_hash FROM api_keys
				 WHERE key_prefix = $1 AND revoked = false
				 LIMIT 1`,
				apiKey[:8],
			).Scan(&entityID, &storedHash)

			if err != nil || storedHash != keyHash {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "unauthorized",
					"message": "Invalid API key.",
				})
				c.Abort()
				return
			}

			// Cache the validated key in Redis for 5 minutes
			if redisCache != nil {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				_ = redisCache.Set(ctx, cacheKey, entityID, 5*time.Minute)
				cancel()
			}

			c.Set("api_key", apiKey)
			c.Set("entity_id", entityID)
		}

		c.Next()
	}
}

// RecoveryMiddleware returns a Gin middleware that recovers from panics
// and returns a 500 error instead of crashing the server.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] recovered from panic: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_server_error",
					"message": "An unexpected error occurred.",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
