// Package middleware provides Gin middleware functions for the Cerebra LLM Gateway.
// It includes CORS handling, request logging, rate limiting, and API key authentication.
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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

// AuthMiddleware returns a Gin middleware handler that validates the X-API-Key header.
// If no key is provided or the key is invalid, the request is rejected with 401.
// In production, this would validate against a database or auth service.
func AuthMiddleware() gin.HandlerFunc {
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

		// Basic validation: key must be at least 16 characters
		if len(apiKey) < 16 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid API key format.",
			})
			c.Abort()
			return
		}

		// Store the validated API key in the context for downstream handlers
		c.Set("api_key", apiKey)

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
