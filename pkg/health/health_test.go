package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestCheckStruct tests Check structure
func TestCheckStruct(t *testing.T) {
	check := Check{
		Name:    "test_check",
		Status:  StatusHealthy,
		Message: "All good",
		Metadata: map[string]interface{}{
			"count": 42,
		},
	}

	if check.Name != "test_check" {
		t.Errorf("check name = %s, want test_check", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("check status = %v, want %v", check.Status, StatusHealthy)
	}

	if check.Message != "All good" {
		t.Errorf("check message = %s, want All good", check.Message)
	}

	if check.Metadata["count"] != 42 {
		t.Errorf("check metadata count = %v, want 42", check.Metadata["count"])
	}
}

// TestResponseStruct tests Response structure and JSON marshaling
func TestResponseStruct(t *testing.T) {
	checks := []Check{
		{
			Name:   "check1",
			Status: StatusHealthy,
		},
	}

	response := Response{
		Status:    StatusHealthy,
		Uptime:    "1h30m",
		Timestamp: "2024-10-31T12:00:00Z",
		Checks:    checks,
	}

	// Should marshal to JSON without error
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Should unmarshal back
	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if unmarshaled.Status != response.Status {
		t.Errorf("status mismatch after marshal/unmarshal")
	}
}

// TestStatusConsts tests Status constants
func TestStatusConsts(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("StatusHealthy = %v, want healthy", StatusHealthy)
	}

	if StatusUnhealthy != "unhealthy" {
		t.Errorf("StatusUnhealthy = %v, want unhealthy", StatusUnhealthy)
	}

	if StatusDegraded != "degraded" {
		t.Errorf("StatusDegraded = %v, want degraded", StatusDegraded)
	}
}

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	if manager == nil {
		t.Fatal("NewManager should not return nil")
	}

	if manager.startTime.IsZero() {
		t.Error("startTime should be set")
	}

	if manager.livenessChecks == nil {
		t.Error("livenessChecks should be initialized")
	}

	if manager.readinessChecks == nil {
		t.Error("readinessChecks should be initialized")
	}
}

// TestRegisterLivenessCheck tests liveness check registration
func TestRegisterLivenessCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	checker := AlwaysHealthyChecker()
	manager.RegisterLivenessCheck("test_check", checker)

	// Verify it was registered by running the checks
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := manager.runChecks(ctx, manager.livenessChecks)
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	}

	// Check name should be set by AlwaysHealthyChecker ("server") or registry key
	if checks[0].Status != StatusHealthy {
		t.Errorf("check status = %v, want %v", checks[0].Status, StatusHealthy)
	}
}

// TestRegisterReadinessCheck tests readiness check registration
func TestRegisterReadinessCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	checker := AlwaysHealthyChecker()
	manager.RegisterReadinessCheck("test_check", checker)

	// Verify it was registered
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := manager.runChecks(ctx, manager.readinessChecks)
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	}
}

// TestAlwaysHealthyChecker tests AlwaysHealthyChecker
func TestAlwaysHealthyChecker(t *testing.T) {
	checker := AlwaysHealthyChecker()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	check := checker.Check(ctx)

	if check.Name != "server" {
		t.Errorf("check name = %s, want server", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("check status = %v, want %v", check.Status, StatusHealthy)
	}
}

// TestNotionHealthChecker tests NotionHealthChecker
func TestNotionHealthChecker(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		checker := NotionHealthChecker(func(ctx context.Context) error {
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := checker.Check(ctx)

		if check.Status != StatusHealthy {
			t.Errorf("check status = %v, want %v", check.Status, StatusHealthy)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		checker := NotionHealthChecker(func(ctx context.Context) error {
			return context.DeadlineExceeded
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := checker.Check(ctx)

		if check.Status != StatusUnhealthy {
			t.Errorf("check status = %v, want %v", check.Status, StatusUnhealthy)
		}
	})
}

// TestClientCacheChecker tests ClientCacheChecker
func TestClientCacheChecker(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		checker := ClientCacheChecker(func() int { return 10 }, 5)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := checker.Check(ctx)

		if check.Status != StatusHealthy {
			t.Errorf("check status = %v, want %v", check.Status, StatusHealthy)
		}
	})

	t.Run("degraded", func(t *testing.T) {
		checker := ClientCacheChecker(func() int { return 2 }, 5)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := checker.Check(ctx)

		if check.Status != StatusDegraded {
			t.Errorf("check status = %v, want %v", check.Status, StatusDegraded)
		}
	})

	t.Run("unhealthy_empty", func(t *testing.T) {
		checker := ClientCacheChecker(func() int { return 0 }, 5)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := checker.Check(ctx)

		if check.Status != StatusUnhealthy {
			t.Errorf("check status = %v, want %v", check.Status, StatusUnhealthy)
		}
	})
}

// TestDetermineOverallStatus tests status determination logic
func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name   string
		checks []Check
		want   Status
	}{
		{
			name:   "all healthy",
			checks: []Check{{Status: StatusHealthy}, {Status: StatusHealthy}},
			want:   StatusHealthy,
		},
		{
			name:   "one unhealthy",
			checks: []Check{{Status: StatusHealthy}, {Status: StatusUnhealthy}},
			want:   StatusUnhealthy,
		},
		{
			name:   "one degraded, rest healthy",
			checks: []Check{{Status: StatusHealthy}, {Status: StatusDegraded}},
			want:   StatusDegraded,
		},
		{
			name:   "unhealthy overrides degraded",
			checks: []Check{{Status: StatusUnhealthy}, {Status: StatusDegraded}},
			want:   StatusUnhealthy,
		},
		{
			name:   "empty checks",
			checks: []Check{},
			want:   StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineOverallStatus(tt.checks)
			if got != tt.want {
				t.Errorf("determineOverallStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLivenessHandler tests liveness endpoint
func TestLivenessHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)
	manager.RegisterLivenessCheck("test", AlwaysHealthyChecker())

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected JSON content type, got %s", w.Header().Get("Content-Type"))
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusHealthy {
		t.Errorf("response status = %v, want %v", response.Status, StatusHealthy)
	}

	if len(response.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(response.Checks))
	}
}

// TestLivenessHandler_Unhealthy tests liveness endpoint with unhealthy check
func TestLivenessHandler_Unhealthy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	unhealthyChecker := CheckerFunc(func(ctx context.Context) Check {
		return Check{
			Name:   "failing_check",
			Status: StatusUnhealthy,
		}
	})

	manager.RegisterLivenessCheck("test", unhealthyChecker)

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusUnhealthy {
		t.Errorf("response status = %v, want %v", response.Status, StatusUnhealthy)
	}
}

// TestReadinessHandler tests readiness endpoint
func TestReadinessHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)
	manager.RegisterReadinessCheck("test", AlwaysHealthyChecker())

	handler := manager.ReadinessHandler()
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusHealthy {
		t.Errorf("response status = %v, want %v", response.Status, StatusHealthy)
	}
}

// TestReadinessHandler_Degraded tests readiness endpoint with degraded status
func TestReadinessHandler_Degraded(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	degradedChecker := CheckerFunc(func(ctx context.Context) Check {
		return Check{
			Name:   "degraded_check",
			Status: StatusDegraded,
		}
	})

	manager.RegisterReadinessCheck("test", degradedChecker)

	handler := manager.ReadinessHandler()
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Readiness should fail on degraded
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// TestUptimeFormatting tests that uptime is included in response
func TestUptimeFormatting(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)
	manager.RegisterLivenessCheck("test", AlwaysHealthyChecker())

	// Wait a bit to ensure non-zero uptime
	time.Sleep(10 * time.Millisecond)

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

// TestTimestampIncluded tests that timestamp is included in response
func TestTimestampIncluded(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)
	manager.RegisterLivenessCheck("test", AlwaysHealthyChecker())

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}

	// Verify it's a valid RFC3339 timestamp
	if _, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
		t.Errorf("invalid timestamp format: %v", err)
	}
}

// TestCheckDuration tests that check duration is recorded
func TestCheckDuration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	slowChecker := CheckerFunc(func(ctx context.Context) Check {
		time.Sleep(10 * time.Millisecond)
		return Check{
			Name:   "slow_check",
			Status: StatusHealthy,
		}
	})

	manager.RegisterLivenessCheck("test", slowChecker)

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Checks) > 0 && response.Checks[0].Duration == "" {
		t.Error("check duration should be recorded")
	}
}

// TestMultipleChecks tests running multiple checks in parallel
func TestMultipleChecks(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(logger)

	for i := 0; i < 5; i++ {
		manager.RegisterLivenessCheck(("check_" + string(rune(i))), AlwaysHealthyChecker())
	}

	handler := manager.LivenessHandler()
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Checks) != 5 {
		t.Errorf("expected 5 checks, got %d", len(response.Checks))
	}
}
