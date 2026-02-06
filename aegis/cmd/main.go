// Package main is the entry point for the Aegis Resilience Engine.
//
// It wires together all components: configuration, storage, Kubernetes client,
// backup manager, recovery manager, policy engine, health checker, and the
// HTTP API server. It supports graceful shutdown on SIGINT/SIGTERM.
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

	"github.com/bigdegenenergy/open-cloud-ops/aegis/api"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/health"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/policy"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/recovery"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/config"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
	"github.com/gin-gonic/gin"
	// Conceptual imports for production Kubernetes integration.
	// These would be uncommented when building with actual k8s dependencies:
	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/rest"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("  Aegis - Open Cloud Ops Resilience Engine")
	fmt.Println("==============================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded: port=%s, log_level=%s, storage=%s, retention=%d days",
		cfg.Port, cfg.LogLevel, cfg.BackupStoragePath, cfg.DefaultRetentionDays)

	// Initialize database connection pool
	var dbPool *pgxpool.Pool
	pool, poolErr := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if poolErr != nil {
		log.Printf("WARNING: Failed to connect to database: %v (running without persistence)", poolErr)
	} else {
		dbPool = pool
		defer dbPool.Close()
		log.Printf("Database connected: %s", maskDSN(cfg.DatabaseURL))
	}

	// Initialize Redis connection
	// In production:
	//   rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	//   if err := rdb.Ping(context.Background()).Err(); err != nil {
	//       log.Fatalf("Failed to connect to Redis: %v", err)
	//   }
	//   defer rdb.Close()
	log.Printf("Redis configured: %s", cfg.RedisURL)

	// Initialize Kubernetes client
	// In production with client-go:
	//   var kubeConfig *rest.Config
	//   if cfg.KubeConfigPath != "" {
	//       kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
	//   } else {
	//       kubeConfig, err = rest.InClusterConfig()
	//   }
	//   clientset, err := kubernetes.NewForConfig(kubeConfig)
	kubeClient := NewSimulatedKubeClient()
	log.Printf("Kubernetes client initialized (simulated mode)")

	// Initialize storage backend
	storage, err := backup.NewLocalStorage(cfg.BackupStoragePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage backend initialized: %s", cfg.BackupStoragePath)

	// Initialize managers
	backupManager := backup.NewBackupManager(kubeClient, storage, cfg.BackupStoragePath, cfg.DefaultRetentionDays)

	// Attach persistent store if database is available
	if dbPool != nil {
		store := backup.NewPgStore(dbPool)
		backupManager.SetStore(store)
		if err := backupManager.LoadFromStore(context.Background()); err != nil {
			log.Printf("WARNING: Failed to load backup state from database: %v", err)
		}
	}

	recoveryManager := recovery.NewRecoveryManager(kubeClient, backupManager, storage)
	policyEngine := policy.NewPolicyEngine(backupManager)
	healthChecker := health.NewHealthChecker(kubeClient)

	log.Printf("All managers initialized")

	// Setup Gin router
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Register API routes
	handler := api.NewHandler(backupManager, recoveryManager, policyEngine, healthChecker)
	handler.RegisterRoutes(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start backup scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runBackupScheduler(ctx, backupManager)

	// Start server in a goroutine
	go func() {
		log.Printf("Aegis Resilience Engine is ready on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Aegis Resilience Engine...")

	// Cancel background tasks
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Aegis Resilience Engine stopped")
}

// runBackupScheduler periodically checks for due backup jobs and executes them.
// It runs until the context is cancelled.
func runBackupScheduler(ctx context.Context, mgr *backup.BackupManager) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Backup scheduler started (checking every 1 minute)")

	for {
		select {
		case <-ctx.Done():
			log.Println("Backup scheduler stopped")
			return
		case <-ticker.C:
			dueJobs, err := mgr.ScheduleBackups(ctx)
			if err != nil {
				log.Printf("Scheduler: error checking for due jobs: %v", err)
				continue
			}

			for _, job := range dueJobs {
				log.Printf("Scheduler: executing due backup job %s (%s)", job.ID, job.Name)
				if _, err := mgr.ExecuteBackup(ctx, job.ID); err != nil {
					log.Printf("Scheduler: failed to execute job %s: %v", job.ID, err)
				}
			}
		}
	}
}

// maskDSN masks the password in a database connection string for safe logging.
func maskDSN(dsn string) string {
	// Simple masking: replace password portion
	// Input: postgres://user:password@host:port/db
	masked := dsn
	atIdx := -1
	colonCount := 0
	for i, c := range dsn {
		if c == ':' {
			colonCount++
		}
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx > 0 && colonCount >= 2 {
		// Find the second colon (after postgres://user:)
		firstColon := -1
		secondColon := -1
		for i, c := range dsn {
			if c == ':' {
				if firstColon == -1 {
					firstColon = i
				} else if secondColon == -1 {
					secondColon = i
					break
				}
			}
		}
		if secondColon > 0 && secondColon < atIdx {
			masked = dsn[:secondColon+1] + "****" + dsn[atIdx:]
		}
	}
	return masked
}

// SimulatedKubeClient provides a mock Kubernetes client for development
// and testing. It simulates a cluster with sample resources across namespaces.
type SimulatedKubeClient struct {
	resources map[string][]models.KubernetesResource // key: "namespace/type"
}

// NewSimulatedKubeClient creates a SimulatedKubeClient with sample data.
func NewSimulatedKubeClient() *SimulatedKubeClient {
	client := &SimulatedKubeClient{
		resources: make(map[string][]models.KubernetesResource),
	}

	// Populate with sample resources for the "default" namespace
	sampleNamespaces := []string{"default", "production", "staging"}

	for _, ns := range sampleNamespaces {
		client.resources[ns+"/Deployment"] = []models.KubernetesResource{
			{APIVersion: "apps/v1", Kind: "Deployment", Name: "web-server", Namespace: ns, Labels: map[string]string{"app": "web"}, Manifest: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web-server"}}`)},
			{APIVersion: "apps/v1", Kind: "Deployment", Name: "api-gateway", Namespace: ns, Labels: map[string]string{"app": "api"}, Manifest: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"api-gateway"}}`)},
		}
		client.resources[ns+"/Service"] = []models.KubernetesResource{
			{APIVersion: "v1", Kind: "Service", Name: "web-service", Namespace: ns, Labels: map[string]string{"app": "web"}, Manifest: []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"web-service"}}`)},
			{APIVersion: "v1", Kind: "Service", Name: "api-service", Namespace: ns, Labels: map[string]string{"app": "api"}, Manifest: []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"api-service"}}`)},
		}
		client.resources[ns+"/ConfigMap"] = []models.KubernetesResource{
			{APIVersion: "v1", Kind: "ConfigMap", Name: "app-config", Namespace: ns, Labels: map[string]string{"app": "config"}, Manifest: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"app-config"}}`)},
		}
	}

	return client
}

// ListResources returns simulated resources for the given type and namespace.
func (c *SimulatedKubeClient) ListResources(ctx context.Context, resourceType, namespace string) ([]models.KubernetesResource, error) {
	key := namespace + "/" + resourceType
	resources, exists := c.resources[key]
	if !exists {
		return []models.KubernetesResource{}, nil
	}
	return resources, nil
}

// ApplyResource simulates applying a resource to the cluster.
func (c *SimulatedKubeClient) ApplyResource(ctx context.Context, resource models.KubernetesResource) error {
	key := resource.Namespace + "/" + resource.Kind
	c.resources[key] = append(c.resources[key], resource)
	return nil
}

// DeleteResource simulates deleting a resource from the cluster.
func (c *SimulatedKubeClient) DeleteResource(ctx context.Context, resourceType, name, namespace string) error {
	key := namespace + "/" + resourceType
	resources := c.resources[key]
	for i, r := range resources {
		if r.Name == name {
			c.resources[key] = append(resources[:i], resources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("resource %s/%s not found in namespace %s", resourceType, name, namespace)
}

// ResourceExists checks if a simulated resource exists.
func (c *SimulatedKubeClient) ResourceExists(ctx context.Context, resourceType, name, namespace string) (bool, error) {
	key := namespace + "/" + resourceType
	resources := c.resources[key]
	for _, r := range resources {
		if r.Name == name {
			return true, nil
		}
	}
	return false, nil
}
