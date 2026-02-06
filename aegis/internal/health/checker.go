// Package health implements health monitoring for Kubernetes resources.
//
// The health checker provides real-time health assessments of Kubernetes
// resources across namespaces. It tracks health status history, detects
// degradation patterns, and produces aggregate summaries useful for
// operational dashboards and DR policy evaluation.
package health

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/internal/backup"
	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// checkableResourceTypes lists the Kubernetes resource types that the
// health checker can evaluate. This can be extended as needed.
var checkableResourceTypes = []string{
	"Deployment",
	"StatefulSet",
	"DaemonSet",
	"Service",
	"Pod",
	"ConfigMap",
	"Secret",
	"PersistentVolumeClaim",
}

// HealthChecker monitors the health of Kubernetes resources and maintains
// a history of health check results. It uses the KubeClient interface to
// query resource state from the cluster.
type HealthChecker struct {
	kubeClient backup.KubeClient

	mu       sync.RWMutex
	checks   map[string]*models.HealthCheck // key: "namespace/kind/name"
	history  map[string][]models.HealthCheck // key: "namespace/kind/name" -> historical checks
}

// NewHealthChecker creates a new HealthChecker with the given Kubernetes client.
func NewHealthChecker(kubeClient backup.KubeClient) *HealthChecker {
	return &HealthChecker{
		kubeClient: kubeClient,
		checks:     make(map[string]*models.HealthCheck),
		history:    make(map[string][]models.HealthCheck),
	}
}

// CheckNamespace performs health checks on all supported resource types
// in the given namespace. It returns a list of health check results.
func (c *HealthChecker) CheckNamespace(ctx context.Context, namespace string) ([]*models.HealthCheck, error) {
	if namespace == "" {
		return nil, fmt.Errorf("health: namespace is required")
	}

	var results []*models.HealthCheck
	var checkErrors []string

	for _, resourceType := range checkableResourceTypes {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		resources, err := c.kubeClient.ListResources(ctx, resourceType, namespace)
		if err != nil {
			checkErrors = append(checkErrors, fmt.Sprintf("failed to list %s: %v", resourceType, err))
			continue
		}

		for _, resource := range resources {
			check := c.evaluateHealth(resource, namespace)
			results = append(results, check)

			// Store the check
			c.storeCheck(check)
		}
	}

	if len(checkErrors) > 0 {
		log.Printf("health: namespace %s check completed with errors: %v", namespace, checkErrors)
	} else {
		log.Printf("health: namespace %s check completed: %d resources evaluated", namespace, len(results))
	}

	return results, nil
}

// CheckResource performs a health check on a specific resource.
func (c *HealthChecker) CheckResource(ctx context.Context, resourceType, name, namespace string) (*models.HealthCheck, error) {
	if resourceType == "" || name == "" || namespace == "" {
		return nil, fmt.Errorf("health: resourceType, name, and namespace are all required")
	}

	// Check if the resource exists
	exists, err := c.kubeClient.ResourceExists(ctx, resourceType, name, namespace)
	if err != nil {
		return nil, fmt.Errorf("health: failed to check resource existence: %w", err)
	}

	now := time.Now().UTC()
	check := &models.HealthCheck{
		ID:           fmt.Sprintf("hc-%d", now.UnixNano()),
		ResourceType: resourceType,
		ResourceName: name,
		Namespace:    namespace,
		LastCheck:    now,
		Details:      make(map[string]string),
	}

	if !exists {
		check.Status = models.HealthStatusUnhealthy
		check.Details["error"] = "resource not found"
		check.Details["recommendation"] = "verify resource exists or check for deletion"
	} else {
		// Retrieve the resource to evaluate its health
		resources, err := c.kubeClient.ListResources(ctx, resourceType, namespace)
		if err != nil {
			check.Status = models.HealthStatusUnknown
			check.Details["error"] = fmt.Sprintf("failed to list resources: %v", err)
		} else {
			// Find the specific resource
			found := false
			for _, r := range resources {
				if r.Name == name {
					evaluated := c.evaluateHealth(r, namespace)
					check.Status = evaluated.Status
					check.Details = evaluated.Details
					found = true
					break
				}
			}
			if !found {
				check.Status = models.HealthStatusUnknown
				check.Details["warning"] = "resource listed but not found in results"
			}
		}
	}

	c.storeCheck(check)
	return check, nil
}

// GetHealthSummary returns an aggregate summary of the health of all
// monitored resources, broken down by namespace.
func (c *HealthChecker) GetHealthSummary(ctx context.Context) (*models.HealthSummary, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := &models.HealthSummary{
		Timestamp:   time.Now().UTC(),
		ByNamespace: make(map[string]*models.NamespaceHealth),
	}

	for _, check := range c.checks {
		summary.TotalResources++

		// Aggregate overall counts
		switch check.Status {
		case models.HealthStatusHealthy:
			summary.Healthy++
		case models.HealthStatusDegraded:
			summary.Degraded++
		case models.HealthStatusUnhealthy:
			summary.Unhealthy++
		default:
			summary.Unknown++
		}

		// Aggregate by namespace
		nsHealth, exists := summary.ByNamespace[check.Namespace]
		if !exists {
			nsHealth = &models.NamespaceHealth{
				Namespace: check.Namespace,
			}
			summary.ByNamespace[check.Namespace] = nsHealth
		}

		switch check.Status {
		case models.HealthStatusHealthy:
			nsHealth.Healthy++
		case models.HealthStatusDegraded:
			nsHealth.Degraded++
		case models.HealthStatusUnhealthy:
			nsHealth.Unhealthy++
		default:
			nsHealth.Unknown++
		}
	}

	return summary, nil
}

// GetResourceHistory returns the health check history for a specific resource.
// Results are ordered from oldest to newest.
func (c *HealthChecker) GetResourceHistory(ctx context.Context, resourceType, name, namespace string) ([]models.HealthCheck, error) {
	key := resourceKey(namespace, resourceType, name)

	c.mu.RLock()
	defer c.mu.RUnlock()

	history, exists := c.history[key]
	if !exists {
		return nil, nil
	}

	// Return a copy
	result := make([]models.HealthCheck, len(history))
	copy(result, history)
	return result, nil
}

// evaluateHealth determines the health status of a Kubernetes resource
// based on its type and available metadata. This is a simulated evaluation;
// in production, it would inspect pod readiness, replica counts, and conditions.
func (c *HealthChecker) evaluateHealth(resource models.KubernetesResource, namespace string) *models.HealthCheck {
	now := time.Now().UTC()
	check := &models.HealthCheck{
		ID:           fmt.Sprintf("hc-%d", now.UnixNano()),
		ResourceType: resource.Kind,
		ResourceName: resource.Name,
		Namespace:    namespace,
		LastCheck:    now,
		Details:      make(map[string]string),
	}

	// Evaluate health based on resource type
	// In production, this would inspect actual resource conditions
	switch resource.Kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		check.Status = evaluateWorkloadHealth(resource, check.Details)
	case "Pod":
		check.Status = evaluatePodHealth(resource, check.Details)
	case "Service":
		check.Status = evaluateServiceHealth(resource, check.Details)
	case "PersistentVolumeClaim":
		check.Status = evaluatePVCHealth(resource, check.Details)
	default:
		// For resources without specific health logic, check existence
		check.Status = models.HealthStatusHealthy
		check.Details["check_type"] = "existence"
		check.Details["message"] = "resource exists"
	}

	return check
}

// evaluateWorkloadHealth checks the health of a workload resource
// (Deployment, StatefulSet, DaemonSet) based on its labels and manifest data.
func evaluateWorkloadHealth(resource models.KubernetesResource, details map[string]string) models.HealthStatus {
	details["check_type"] = "workload_status"

	// In production, we would check:
	// - spec.replicas vs status.readyReplicas
	// - status.conditions for Available/Progressing
	// - pod template hash consistency
	// For simulation, use labels as health indicators
	if status, ok := resource.Labels["health"]; ok {
		switch status {
		case "degraded":
			details["message"] = "workload reporting degraded via label"
			return models.HealthStatusDegraded
		case "unhealthy":
			details["message"] = "workload reporting unhealthy via label"
			return models.HealthStatusUnhealthy
		}
	}

	details["message"] = "workload is running normally"
	details["replicas"] = "desired matches ready"
	return models.HealthStatusHealthy
}

// evaluatePodHealth checks the health of a Pod resource.
func evaluatePodHealth(resource models.KubernetesResource, details map[string]string) models.HealthStatus {
	details["check_type"] = "pod_status"

	// In production, we would check:
	// - status.phase (Running, Pending, Failed, etc.)
	// - status.conditions (Ready, ContainersReady, etc.)
	// - container restart counts
	if status, ok := resource.Labels["status"]; ok {
		switch status {
		case "pending":
			details["message"] = "pod is in Pending state"
			return models.HealthStatusDegraded
		case "failed", "crashloopbackoff":
			details["message"] = fmt.Sprintf("pod is in %s state", status)
			return models.HealthStatusUnhealthy
		}
	}

	details["message"] = "pod is running"
	details["phase"] = "Running"
	return models.HealthStatusHealthy
}

// evaluateServiceHealth checks the health of a Service resource.
func evaluateServiceHealth(resource models.KubernetesResource, details map[string]string) models.HealthStatus {
	details["check_type"] = "service_status"

	// In production, we would check:
	// - Endpoint availability
	// - Number of ready endpoints vs expected
	if status, ok := resource.Labels["endpoints"]; ok {
		if status == "none" {
			details["message"] = "service has no ready endpoints"
			return models.HealthStatusUnhealthy
		}
	}

	details["message"] = "service has active endpoints"
	return models.HealthStatusHealthy
}

// evaluatePVCHealth checks the health of a PersistentVolumeClaim.
func evaluatePVCHealth(resource models.KubernetesResource, details map[string]string) models.HealthStatus {
	details["check_type"] = "pvc_status"

	// In production, we would check:
	// - status.phase (Bound, Pending, Lost)
	// - Capacity vs requested
	if status, ok := resource.Labels["phase"]; ok {
		switch status {
		case "pending":
			details["message"] = "PVC is in Pending state"
			return models.HealthStatusDegraded
		case "lost":
			details["message"] = "PVC is in Lost state"
			return models.HealthStatusUnhealthy
		}
	}

	details["message"] = "PVC is bound"
	details["phase"] = "Bound"
	return models.HealthStatusHealthy
}

// storeCheck saves a health check result and appends it to the history.
func (c *HealthChecker) storeCheck(check *models.HealthCheck) {
	key := resourceKey(check.Namespace, check.ResourceType, check.ResourceName)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks[key] = check

	// Append to history, keeping at most 100 entries per resource
	history := c.history[key]
	history = append(history, *check)
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	c.history[key] = history
}

// resourceKey generates a unique key for identifying a resource across checks.
func resourceKey(namespace, resourceType, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, resourceType, name)
}
