package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/analytics"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/budget"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/middleware"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/proxy"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/router"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/cache"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/config"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/database"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/pkg/models"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("  Cerebra - Open Cloud Ops LLM Gateway")
	fmt.Println("==============================================")

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	fmt.Printf("Starting server on port %s...\n", cfg.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	db, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisCache, err := cache.NewCache(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Build default pricing map
	pricing := buildDefaultPricing()

	// Initialize budget enforcement
	enforcer := budget.NewEnforcer(db.Pool, redisCache)

	// Initialize proxy handler
	proxyHandler := proxy.NewProxyHandler(db.Pool, enforcer, pricing)

	// Initialize smart model router
	modelRouter := router.NewRouter(router.StrategyCostOptimized, pricing)

	// Initialize analytics engine
	insightsEngine := analytics.NewInsightsEngine(db.Pool)

	// Suppress debug output in production
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup Gin router with middleware
	engine := gin.New()

	// Global middleware
	engine.Use(middleware.RecoveryMiddleware())
	engine.Use(middleware.LoggingMiddleware())
	engine.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))

	// Health check endpoint (no auth required)
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "cerebra",
			"version": "0.1.0",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Readiness check
	engine.GET("/ready", func(c *gin.Context) {
		// Check DB connectivity
		if err := db.Pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  "database connection failed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API v1 group with auth and rate limiting
	v1 := engine.Group("/v1")
	v1.Use(middleware.AuthMiddleware(db.Pool, redisCache))
	v1.Use(middleware.RateLimitMiddleware(redisCache, 100, 1*time.Minute))

	// Proxy routes - forward requests to LLM providers
	v1.Any("/openai/*path", proxyHandler.HandleRequest)
	v1.Any("/anthropic/*path", proxyHandler.HandleRequest)
	v1.Any("/gemini/*path", proxyHandler.HandleRequest)

	// Budget API routes
	budgetGroup := v1.Group("/budgets")
	{
		budgetGroup.GET("/status", func(c *gin.Context) {
			scope := c.Query("scope")
			entityID := c.Query("entity_id")
			if scope == "" || entityID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "scope and entity_id query params are required"})
				return
			}

			status, err := enforcer.GetBudgetStatus(budget.BudgetScope(scope), entityID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, status)
		})

		budgetGroup.POST("/reset", func(c *gin.Context) {
			if err := enforcer.ResetBudgets(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "all budgets have been reset"})
		})
	}

	// Analytics API routes
	analyticsGroup := v1.Group("/analytics")
	{
		analyticsGroup.GET("/spikes", func(c *gin.Context) {
			hoursStr := c.DefaultQuery("hours", "1")
			hours, err := strconv.Atoi(hoursStr)
			if err != nil {
				hours = 1
			}

			spikes, err := insightsEngine.DetectSpikes(hours)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"spikes": spikes, "count": len(spikes)})
		})

		analyticsGroup.GET("/recommendations", func(c *gin.Context) {
			recs, err := insightsEngine.RecommendModelSwitches()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"recommendations": recs, "count": len(recs)})
		})

		analyticsGroup.GET("/report", func(c *gin.Context) {
			fromStr := c.DefaultQuery("from", time.Now().AddDate(0, 0, -30).Format(time.RFC3339))
			toStr := c.DefaultQuery("to", time.Now().Format(time.RFC3339))

			from, err := time.Parse(time.RFC3339, fromStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date format, use RFC3339"})
				return
			}
			to, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date format, use RFC3339"})
				return
			}

			report, err := insightsEngine.GenerateReport(from, to)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"report": report, "from": from, "to": to})
		})

		analyticsGroup.GET("/top-spenders", func(c *gin.Context) {
			dimension := c.DefaultQuery("dimension", "agent")
			limitStr := c.DefaultQuery("limit", "10")
			sinceStr := c.DefaultQuery("since", time.Now().AddDate(0, 0, -30).Format(time.RFC3339))

			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				limit = 10
			}

			since, err := time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'since' date format, use RFC3339"})
				return
			}

			spenders, err := insightsEngine.GetTopSpenders(c.Request.Context(), dimension, limit, since)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"top_spenders": spenders, "dimension": dimension})
		})
	}

	// Router info endpoint
	v1.GET("/router/models", func(c *gin.Context) {
		registry := modelRouter.GetModelRegistry()
		c.JSON(http.StatusOK, gin.H{"models": registry})
	})

	v1.POST("/router/route", func(c *gin.Context) {
		var req router.RouteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		result := modelRouter.Route(req)
		c.JSON(http.StatusOK, result)
	})

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Cerebra LLM Gateway is ready on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown - wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Cerebra LLM Gateway...")

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Drain any pending log entries before exiting
	proxyHandler.DrainLogs()

	log.Println("Cerebra LLM Gateway stopped gracefully")
}

// buildDefaultPricing creates the default model pricing map based on
// current published pricing from OpenAI, Anthropic, and Google.
func buildDefaultPricing() map[string]models.ModelPricing {
	now := time.Now()
	return map[string]models.ModelPricing{
		// OpenAI
		"openai:gpt-4o-mini": {
			Provider: models.ProviderOpenAI, Model: "gpt-4o-mini",
			InputPerMToken: 0.15, OutputPerMToken: 0.60, UpdatedAt: now,
		},
		"openai:gpt-4o": {
			Provider: models.ProviderOpenAI, Model: "gpt-4o",
			InputPerMToken: 2.50, OutputPerMToken: 10.00, UpdatedAt: now,
		},
		"openai:gpt-4-turbo": {
			Provider: models.ProviderOpenAI, Model: "gpt-4-turbo",
			InputPerMToken: 10.00, OutputPerMToken: 30.00, UpdatedAt: now,
		},
		"openai:o1": {
			Provider: models.ProviderOpenAI, Model: "o1",
			InputPerMToken: 15.00, OutputPerMToken: 60.00, UpdatedAt: now,
		},

		// Anthropic
		"anthropic:claude-3-haiku-20240307": {
			Provider: models.ProviderAnthropic, Model: "claude-3-haiku-20240307",
			InputPerMToken: 0.25, OutputPerMToken: 1.25, UpdatedAt: now,
		},
		"anthropic:claude-3-5-sonnet-20241022": {
			Provider: models.ProviderAnthropic, Model: "claude-3-5-sonnet-20241022",
			InputPerMToken: 3.00, OutputPerMToken: 15.00, UpdatedAt: now,
		},
		"anthropic:claude-3-opus-20240229": {
			Provider: models.ProviderAnthropic, Model: "claude-3-opus-20240229",
			InputPerMToken: 15.00, OutputPerMToken: 75.00, UpdatedAt: now,
		},

		// Gemini
		"gemini:gemini-1.5-flash": {
			Provider: models.ProviderGemini, Model: "gemini-1.5-flash",
			InputPerMToken: 0.075, OutputPerMToken: 0.30, UpdatedAt: now,
		},
		"gemini:gemini-1.5-pro": {
			Provider: models.ProviderGemini, Model: "gemini-1.5-pro",
			InputPerMToken: 1.25, OutputPerMToken: 5.00, UpdatedAt: now,
		},
		"gemini:gemini-ultra": {
			Provider: models.ProviderGemini, Model: "gemini-ultra",
			InputPerMToken: 10.00, OutputPerMToken: 30.00, UpdatedAt: now,
		},
	}
}
