package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Check represents a single health check
type Check struct {
	Name     string                 `json:"name"`
	Status   Status                 `json:"status"`
	Message  string                 `json:"message,omitempty"`
	Duration string                 `json:"duration,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Response represents the overall health response
type Response struct {
	Status    Status  `json:"status"`
	Uptime    string  `json:"uptime"`
	Timestamp string  `json:"timestamp"`
	Checks    []Check `json:"checks,omitempty"`
}

// Checker defines the interface for health checks
type Checker interface {
	Check(ctx context.Context) Check
}

// CheckerFunc is a function adapter for the Checker interface
type CheckerFunc func(ctx context.Context) Check

func (f CheckerFunc) Check(ctx context.Context) Check {
	return f(ctx)
}

// Manager manages health checks and provides handlers
type Manager struct {
	startTime       time.Time
	livenessChecks  map[string]Checker
	readinessChecks map[string]Checker
	mu              sync.RWMutex
	logger          *zap.Logger
}

// NewManager creates a new health check manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		startTime:       time.Now(),
		livenessChecks:  make(map[string]Checker),
		readinessChecks: make(map[string]Checker),
		logger:          logger,
	}
}

// RegisterLivenessCheck registers a liveness check
// Liveness checks indicate if the application is running and should be restarted if failing
func (m *Manager) RegisterLivenessCheck(name string, checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessChecks[name] = checker
}

// RegisterReadinessCheck registers a readiness check
// Readiness checks indicate if the application is ready to serve traffic
func (m *Manager) RegisterReadinessCheck(name string, checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readinessChecks[name] = checker
}

// runChecks executes all checks in parallel with timeout
func (m *Manager) runChecks(ctx context.Context, checks map[string]Checker) []Check {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]Check, 0, len(checks))
	resultsChan := make(chan Check, len(checks))

	var wg sync.WaitGroup
	for name, checker := range checks {
		wg.Add(1)
		go func(n string, c Checker) {
			defer wg.Done()
			start := time.Now()
			check := c.Check(ctx)
			check.Duration = time.Since(start).String()
			if check.Name == "" {
				check.Name = n
			}
			resultsChan <- check
		}(name, checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for check := range resultsChan {
		results = append(results, check)
	}

	return results
}

// determineOverallStatus determines the overall status based on individual checks
func determineOverallStatus(checks []Check) Status {
	if len(checks) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, check := range checks {
		switch check.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// LivenessHandler returns an HTTP handler for liveness checks
// Liveness endpoint should return 200 if the application is running
func (m *Manager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		checks := m.runChecks(ctx, m.livenessChecks)
		status := determineOverallStatus(checks)

		response := Response{
			Status:    status,
			Uptime:    time.Since(m.startTime).String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		}

		// Liveness should only fail if the application is completely broken
		// Return 503 only for unhealthy status
		statusCode := http.StatusOK
		if status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		m.writeResponse(w, statusCode, response)
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
// Readiness endpoint should return 200 when the application is ready to serve traffic
func (m *Manager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		checks := m.runChecks(ctx, m.readinessChecks)
		status := determineOverallStatus(checks)

		response := Response{
			Status:    status,
			Uptime:    time.Since(m.startTime).String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		}

		// Readiness should fail for both unhealthy and degraded states
		statusCode := http.StatusOK
		if status == StatusUnhealthy || status == StatusDegraded {
			statusCode = http.StatusServiceUnavailable
		}

		m.writeResponse(w, statusCode, response)
	}
}

// writeResponse writes the JSON response
func (m *Manager) writeResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error("failed to encode health response", zap.Error(err))
	}
}

// NotionHealthChecker creates a health checker for Notion API connectivity
func NotionHealthChecker(checkFunc func(ctx context.Context) error) Checker {
	return CheckerFunc(func(ctx context.Context) Check {
		err := checkFunc(ctx)
		if err != nil {
			return Check{
				Name:    "notion_api",
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Failed to connect to Notion API: %v", err),
			}
		}
		return Check{
			Name:    "notion_api",
			Status:  StatusHealthy,
			Message: "Notion API is reachable",
		}
	})
}

// AlwaysHealthyChecker returns a checker that always reports healthy
// Useful for basic liveness checks
func AlwaysHealthyChecker() Checker {
	return CheckerFunc(func(ctx context.Context) Check {
		return Check{
			Name:    "server",
			Status:  StatusHealthy,
			Message: "Server is running",
		}
	})
}

// ClientCacheChecker creates a health checker for the client cache
func ClientCacheChecker(getClientCount func() int, minExpected int) Checker {
	return CheckerFunc(func(ctx context.Context) Check {
		count := getClientCount()

		if count == 0 {
			return Check{
				Name:    "client_cache",
				Status:  StatusUnhealthy,
				Message: "Client cache is empty",
				Metadata: map[string]interface{}{
					"count": count,
				},
			}
		}

		if count < minExpected {
			return Check{
				Name:    "client_cache",
				Status:  StatusDegraded,
				Message: fmt.Sprintf("Client cache has fewer clients than expected (got %d, expected at least %d)", count, minExpected),
				Metadata: map[string]interface{}{
					"count":        count,
					"min_expected": minExpected,
				},
			}
		}

		return Check{
			Name:    "client_cache",
			Status:  StatusHealthy,
			Message: "Client cache is populated",
			Metadata: map[string]interface{}{
				"count": count,
			},
		}
	})
}
