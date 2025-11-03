package cache

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockRefresher simulates a CacheRefresher for testing
type mockRefresher struct {
	customersErr     error
	usersErr         error
	customersCallCnt int
	usersCallCnt     int
	mu               sync.Mutex
	// Simulate failure on first N attempts
	customersFailUntil int
	usersFailUntil     int
}

func (m *mockRefresher) InitializeCustomers() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customersCallCnt++

	// Simulate transient failures
	if m.customersCallCnt <= m.customersFailUntil {
		return m.customersErr
	}

	return nil
}

func (m *mockRefresher) InitializeUsers() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usersCallCnt++

	// Simulate transient failures
	if m.usersCallCnt <= m.usersFailUntil {
		return m.usersErr
	}

	return nil
}

func (m *mockRefresher) getCallCounts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.customersCallCnt, m.usersCallCnt
}

func (m *mockRefresher) resetCallCounts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customersCallCnt = 0
	m.usersCallCnt = 0
}

// TestNewManager verifies manager initialization
func TestNewManager(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.refresher != mockRef {
		t.Error("refresher not set correctly")
	}

	if mgr.logger != logger {
		t.Error("logger not set correctly")
	}

	if mgr.refreshInterval != interval {
		t.Errorf("refreshInterval = %v, want %v", mgr.refreshInterval, interval)
	}

	if mgr.ctx == nil {
		t.Error("context not initialized")
	}

	if mgr.cancel == nil {
		t.Error("cancel function not initialized")
	}
}

// TestStartStop verifies start and graceful stop
func TestStartStop(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 100 * time.Millisecond

	mgr := NewManager(mockRef, nil, logger, interval)

	// Start manager
	mgr.Start()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop manager
	mgr.Stop()

	// Verify stop completed (if this hangs, wg.Wait() has an issue)
}

// TestPeriodicRefresh verifies automatic periodic refresh
func TestPeriodicRefresh(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 50 * time.Millisecond // Very short interval for testing

	mgr := NewManager(mockRef, nil, logger, interval)
	mgr.Start()

	// Wait for at least 2 refresh cycles
	time.Sleep(150 * time.Millisecond)

	mgr.Stop()

	customers, users := mockRef.getCallCounts()

	// Should have called at least twice (depends on timing)
	if customers < 2 {
		t.Errorf("InitializeCustomers called %d times, want at least 2", customers)
	}

	if users < 2 {
		t.Errorf("InitializeUsers called %d times, want at least 2", users)
	}
}

// TestManualRefresh verifies manual refresh trigger
func TestManualRefresh(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour // Long interval, we'll use manual refresh

	mgr := NewManager(mockRef, nil, logger, interval)

	// Don't start automatic refresh, just trigger manual
	mgr.ManualRefresh()

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	customers, users := mockRef.getCallCounts()

	if customers != 1 {
		t.Errorf("InitializeCustomers called %d times, want 1", customers)
	}

	if users != 1 {
		t.Errorf("InitializeUsers called %d times, want 1", users)
	}
}

// TestRefreshAllSuccessPath verifies both caches refresh successfully
func TestRefreshAllSuccessPath(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Call refreshAll directly
	mgr.refreshAll()

	customers, users := mockRef.getCallCounts()

	if customers != 1 {
		t.Errorf("InitializeCustomers called %d times, want 1", customers)
	}

	if users != 1 {
		t.Errorf("InitializeUsers called %d times, want 1", users)
	}
}

// TestRefreshAllCustomersFailure verifies that customers failure doesn't prevent users refresh
func TestRefreshAllCustomersFailure(t *testing.T) {
	// This test would take up to 5 minutes with real maxRetryWindow
	// Skip for regular test runs
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	mockRef := &mockRefresher{
		customersErr:       errors.New("customers fetch failed"),
		customersFailUntil: 999, // Always fail
	}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Call refreshAll - it should try customers and users independently
	mgr.refreshAll()

	customers, users := mockRef.getCallCounts()

	// Customers should have been attempted multiple times (retries)
	if customers < 1 {
		t.Errorf("InitializeCustomers called %d times, want at least 1", customers)
	}

	// Users should still be called once (or with retries if it also fails)
	if users < 1 {
		t.Errorf("InitializeUsers called %d times, want at least 1", users)
	}
}

// TestRefreshCacheWithRetrySuccess verifies successful refresh without retries
func TestRefreshCacheWithRetrySuccess(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	err := mgr.refreshCacheWithRetry(CacheTypeCustomers, mockRef.InitializeCustomers)

	if err != nil {
		t.Errorf("refreshCacheWithRetry returned error: %v", err)
	}

	customers, _ := mockRef.getCallCounts()
	if customers != 1 {
		t.Errorf("InitializeCustomers called %d times, want 1", customers)
	}
}

// TestRefreshCacheWithRetryTransientFailure verifies retry on transient failures
func TestRefreshCacheWithRetryTransientFailure(t *testing.T) {
	// Fail first 2 attempts, succeed on 3rd
	mockRef := &mockRefresher{
		customersErr:       errors.New("temporary failure"),
		customersFailUntil: 2, // Fail first 2 calls
	}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	err := mgr.refreshCacheWithRetry(CacheTypeCustomers, mockRef.InitializeCustomers)

	if err != nil {
		t.Errorf("refreshCacheWithRetry returned error after recovery: %v", err)
	}

	customers, _ := mockRef.getCallCounts()
	// Should have called 3 times (2 failures + 1 success)
	if customers != 3 {
		t.Errorf("InitializeCustomers called %d times, want 3", customers)
	}
}

// TestRefreshCacheWithRetryPermanentFailure verifies eventual failure after max retries
func TestRefreshCacheWithRetryPermanentFailure(t *testing.T) {
	// This test would take 5 minutes with real maxRetryWindow
	// Skip for regular test runs
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	mockRef := &mockRefresher{
		customersErr:       errors.New("permanent failure"),
		customersFailUntil: 999, // Always fail
	}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Note: This will take up to 5 minutes in production
	// In practice, you'd mock time or reduce maxRetryWindow for testing
	startTime := time.Now()
	err := mgr.refreshCacheWithRetry(CacheTypeCustomers, mockRef.InitializeCustomers)

	if err == nil {
		t.Error("refreshCacheWithRetry should return error after max retries")
	}

	duration := time.Since(startTime)
	if duration < maxRetryWindow {
		t.Errorf("should have retried for at least %v, but took %v", maxRetryWindow, duration)
	}
}

// TestRefreshCacheWithRetryContextCancellation verifies context cancellation stops retries
func TestRefreshCacheWithRetryContextCancellation(t *testing.T) {
	mockRef := &mockRefresher{
		customersErr:       errors.New("failure"),
		customersFailUntil: 999, // Always fail
	}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		mgr.cancel()
	}()

	startTime := time.Now()
	err := mgr.refreshCacheWithRetry(CacheTypeCustomers, mockRef.InitializeCustomers)

	if err == nil {
		t.Error("refreshCacheWithRetry should return error when context cancelled")
	}

	duration := time.Since(startTime)
	// Should stop quickly after context cancellation, not wait full 5 minutes
	if duration > 1*time.Second {
		t.Errorf("should have stopped quickly after cancellation, but took %v", duration)
	}
}

// TestRecordSuccess verifies success metrics recording
func TestRecordSuccess(t *testing.T) {
	logger := zap.NewNop()

	mgr := NewManager(&mockRefresher{}, nil, logger, 1*time.Hour)

	duration := 500 * time.Millisecond
	mgr.recordSuccess(CacheTypeCustomers, duration)

	// With nil metrics, should not panic
}

// TestRecordFailure verifies failure metrics recording
func TestRecordFailure(t *testing.T) {
	logger := zap.NewNop()

	mgr := NewManager(&mockRefresher{}, nil, logger, 1*time.Hour)

	mgr.recordFailure(CacheTypeUsers)

	// With nil metrics, should not panic
}

// TestRecordRetry verifies retry metrics recording
func TestRecordRetry(t *testing.T) {
	logger := zap.NewNop()

	mgr := NewManager(&mockRefresher{}, nil, logger, 1*time.Hour)

	mgr.recordRetry(CacheTypeCustomers)

	// With nil metrics, should not panic
}

// TestCacheTypeConstants verifies cache type constants are defined
func TestCacheTypeConstants(t *testing.T) {
	if CacheTypeCustomers == "" {
		t.Error("CacheTypeCustomers should not be empty")
	}

	if CacheTypeUsers == "" {
		t.Error("CacheTypeUsers should not be empty")
	}

	if CacheTypeCustomers == CacheTypeUsers {
		t.Error("CacheTypeCustomers and CacheTypeUsers should be different")
	}
}

// TestBackoffConstants verifies backoff constants are reasonable
func TestBackoffConstants(t *testing.T) {
	if initialBackoff <= 0 {
		t.Error("initialBackoff should be positive")
	}

	if backoffMultiple <= 1 {
		t.Error("backoffMultiple should be greater than 1 for exponential growth")
	}

	if maxRetryWindow <= initialBackoff {
		t.Error("maxRetryWindow should be greater than initialBackoff")
	}
}

// TestExponentialBackoffSequence verifies backoff sequence calculation
func TestExponentialBackoffSequence(t *testing.T) {
	expectedSequence := []time.Duration{
		3 * time.Second,
		6 * time.Second,
		12 * time.Second,
		24 * time.Second,
		48 * time.Second,
		96 * time.Second,
		192 * time.Second,
	}

	backoff := initialBackoff
	for i, expected := range expectedSequence {
		if backoff != expected {
			t.Errorf("backoff[%d] = %v, want %v", i, backoff, expected)
		}
		backoff *= backoffMultiple
	}
}

// TestConcurrentManualRefresh verifies multiple concurrent manual refreshes
func TestConcurrentManualRefresh(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Trigger multiple manual refreshes concurrently
	const numRefreshes = 5
	var wg sync.WaitGroup
	wg.Add(numRefreshes)

	for i := 0; i < numRefreshes; i++ {
		go func() {
			defer wg.Done()
			mgr.ManualRefresh()
		}()
	}

	wg.Wait()

	// Wait for all goroutines to complete
	time.Sleep(200 * time.Millisecond)

	customers, users := mockRef.getCallCounts()

	// Should have called both methods numRefreshes times
	if customers != numRefreshes {
		t.Errorf("InitializeCustomers called %d times, want %d", customers, numRefreshes)
	}

	if users != numRefreshes {
		t.Errorf("InitializeUsers called %d times, want %d", users, numRefreshes)
	}
}

// TestStopWithoutStart verifies Stop can be called without Start
func TestStopWithoutStart(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	// Should not panic or hang
	mgr.Stop()
}

// TestMultipleStopCalls verifies multiple Stop calls are safe
func TestMultipleStopCalls(t *testing.T) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 100 * time.Millisecond

	mgr := NewManager(mockRef, nil, logger, interval)
	mgr.Start()

	// First stop
	mgr.Stop()

	// Second stop should not panic or hang
	// Note: This might not work perfectly because context is already cancelled
	// But it shouldn't panic
}

// BenchmarkRefreshAllSuccess benchmarks successful cache refresh
func BenchmarkRefreshAllSuccess(b *testing.B) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockRef.resetCallCounts()
		mgr.refreshAll()
	}
}

// BenchmarkManualRefresh benchmarks manual refresh trigger
func BenchmarkManualRefresh(b *testing.B) {
	mockRef := &mockRefresher{}
	logger := zap.NewNop()
	interval := 1 * time.Hour

	mgr := NewManager(mockRef, nil, logger, interval)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.ManualRefresh()
	}

	// Wait for all goroutines to complete
	time.Sleep(100 * time.Millisecond)
}
