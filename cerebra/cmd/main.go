package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/analytics"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/api"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/budget"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/config"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/database"
	"github.com/bigdegenenergy/open-cloud-ops/cerebra/internal/proxy"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// apiKeyAuth returns a Gin middleware that validates the X-Admin-Key header.
func apiKeyAuth(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Admin-Key")
		if key == "" {
			key = c.GetHeader("Authorization")
			if len(key) > 7 && key[:7] == "Bearer " {
				key = key[7:]
			}
		}
		if key != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid or missing admin API key"})
			return
		}
		c.Next()
	}
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("  Cerebra - Open Cloud Ops LLM Gateway")
	fmt.Println("==============================================")

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	fmt.Printf("Starting server on port %s...\n", cfg.Port)

	// Initialize database connection.
	db, err := database.New(cfg.DSN())
	if err != nil {
		log.Printf("WARNING: Database unavailable (%v). Running in proxy-only mode.", err)
		db = nil
	} else {
		defer db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := db.Migrate(ctx); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		if err := db.SeedPricing(ctx); err != nil {
			log.Printf("WARNING: Failed to seed pricing data: %v", err)
		}
		log.Println("Database connected and migrations applied.")
	}

	// Initialize Redis connection.
	var rdb *redis.Client
	rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: Redis unavailable (%v). Budget enforcement will be permissive.", err)
		rdb = nil
	} else {
		defer rdb.Close()
		log.Println("Redis connected.")
	}

	// Initialize components.
	enforcer := budget.NewEnforcer(rdb, cfg.BudgetFailOpen)
	proxyHandler := proxy.NewProxyHandler(cfg, db, enforcer)
	apiHandlers := api.NewHandlers(db, enforcer)

	var insightsEngine *analytics.InsightsEngine
	if db != nil {
		insightsEngine = analytics.NewInsightsEngine(db.Pool)
	}

	// Set up Gin router.
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS for dashboard.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Admin-Key", "X-Agent-ID", "X-Team-ID", "X-Org-ID", "X-API-Key", "X-Goog-Api-Key"},
		ExposeHeaders:    []string{"X-Request-ID", "X-Cost-USD", "X-Latency-Ms"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check.
	r.GET("/health", apiHandlers.HealthCheck)

	// API v1 routes (protected by admin API key).
	// Fail-secure: if no key is configured, block all management requests.
	v1 := r.Group("/api/v1")
	if cfg.AdminAPIKey != "" {
		v1.Use(apiKeyAuth(cfg.AdminAPIKey))
		log.Println("Management API authentication enabled.")
	} else {
		log.Println("WARNING: CEREBRA_ADMIN_API_KEY not set. Management API is disabled (fail-secure).")
		v1.Use(func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "management API disabled: CEREBRA_ADMIN_API_KEY not configured"})
		})
	}
	{
		// Cost data.
		v1.GET("/costs/summary", apiHandlers.GetCostSummary)
		v1.GET("/costs/requests", apiHandlers.GetRecentRequests)

		// Budget management.
		v1.GET("/budgets", apiHandlers.ListBudgets)
		v1.POST("/budgets", apiHandlers.CreateBudget)
		v1.GET("/budgets/:scope/:entity_id", apiHandlers.GetBudget)

		// Analytics / insights.
		v1.GET("/insights", func(c *gin.Context) {
			if insightsEngine == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics unavailable"})
				return
			}
			spikes, _ := insightsEngine.DetectSpikes(c.Request.Context())
			switches, _ := insightsEngine.RecommendModelSwitches(c.Request.Context())
			var all []analytics.Insight
			all = append(all, spikes...)
			all = append(all, switches...)
			c.JSON(http.StatusOK, gin.H{"count": len(all), "data": all})
		})

		v1.GET("/report", func(c *gin.Context) {
			if insightsEngine == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics unavailable"})
				return
			}
			from := time.Now().AddDate(0, -1, 0)
			to := time.Now()
			report, err := insightsEngine.GenerateReport(c.Request.Context(), from, to)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, report)
		})
	}

	// LLM Proxy routes — the core gateway.
	proxyGroup := r.Group("/v1/proxy")
	if cfg.ProxyAPIKey != "" {
		proxyGroup.Use(apiKeyAuth(cfg.ProxyAPIKey))
		log.Println("Proxy endpoint authentication enabled.")
	} else {
		log.Println("WARNING: CEREBRA_PROXY_API_KEY not set. Proxy endpoints are UNAUTHENTICATED.")
		log.Println("WARNING: Ensure this service is on a private network or set CEREBRA_PROXY_API_KEY.")
	}
	{
		proxyGroup.Any("/openai/*path", proxyHandler.HandleOpenAI)
		proxyGroup.Any("/anthropic/*path", proxyHandler.HandleAnthropic)
		proxyGroup.Any("/gemini/*path", proxyHandler.HandleGemini)
	}

	// Start HTTP server with graceful shutdown.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Cerebra LLM Gateway is ready on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited.")
}
