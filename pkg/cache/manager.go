// Package cache provides cache management with automatic refresh capabilities.
//
// The Manager handles periodic and manual refresh of two critical caches:
// 1. Customer cache - Valid customer organization names from Notion Customers database
// 2. User cache - Notion workspace users for Slack-to-Notion user mapping
//
// Features:
// - Automatic periodic refresh in background goroutine
// - Manual refresh on-demand (non-blocking)
// - Exponential backoff retry with configurable window
// - Graceful shutdown with context cancellation
// - Comprehensive metrics and structured logging
// - Thread-safe with proper coordination via sync.WaitGroup
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rudderlabs/hopperbot/pkg/metrics"
	"go.uber.org/zap"
)

const (
	// CacheTypeCustomers identifies the customer cache type in metrics and logs
	CacheTypeCustomers = "customers"
	// CacheTypeUsers identifies the user cache type in metrics and logs
	CacheTypeUsers = "users"

	// Retry configuration
	initialBackoff  = 3 * time.Second // Start with 3 second delay
	backoffMultiple = 2               // Double the backoff each retry
	maxRetryWindow  = 5 * time.Minute // Stop retrying after 5 minutes
)

// CacheRefresher defines the interface for cache initialization operations.
//
// Implementations should handle fetching data from external sources
// and updating internal caches with proper thread safety.
type CacheRefresher interface {
	// InitializeCustomers fetches and updates the customer cache
	InitializeCustomers() error

	// InitializeUsers fetches and updates the user cache
	InitializeUsers() error
}

// Manager orchestrates automatic and manual cache refresh operations.
//
// The manager runs a background goroutine that periodically refreshes both
// caches (customers and users) by calling the CacheRefresher's Initialize methods.
// On failure, it implements exponential backoff retry up to a configurable window.
//
// Thread safety:
// - Background goroutine is the only one calling refresh methods
// - CacheRefresher implementations handle their own locking internally
// - Context cancellation stops the background goroutine gracefully
// - WaitGroup ensures proper shutdown coordination
type Manager struct {
	refresher       CacheRefresher   // Interface for cache operations
	metrics         *metrics.Metrics // For recording cache refresh metrics
	logger          *zap.Logger      // Structured logging
	refreshInterval time.Duration    // How often to refresh (from config)
	ticker          *time.Ticker     // For periodic refresh
	ctx             context.Context  // For cancellation
	cancel          context.CancelFunc
	wg              sync.WaitGroup // To wait for goroutine completion
}

// NewManager creates a new cache manager.
//
// Parameters:
// - refresher: Implementation with InitializeCustomers() and InitializeUsers() methods
// - metrics: Metrics instance for recording refresh operations
// - logger: Zap logger for structured logging
// - refreshInterval: How often to refresh caches (e.g., 1 hour)
//
// The manager is created in a stopped state. Call Start() to begin automatic refresh.
func NewManager(
	refresher CacheRefresher,
	metrics *metrics.Metrics,
	logger *zap.Logger,
	refreshInterval time.Duration,
) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		refresher:       refresher,
		metrics:         metrics,
		logger:          logger,
		refreshInterval: refreshInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start begins the background cache refresh goroutine.
//
// The goroutine runs until Stop() is called or the context is cancelled.
// It refreshes both caches on each tick, implementing retry logic on failures.
//
// This method returns immediately - the refresh happens in the background.
// Call Stop() to gracefully shut down the background goroutine.
func (m *Manager) Start() {
	m.ticker = time.NewTicker(m.refreshInterval)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer m.ticker.Stop()

		m.logger.Info("cache manager started",
			zap.Duration("refresh_interval", m.refreshInterval),
		)

		for {
			select {
			case <-m.ticker.C:
				m.logger.Debug("periodic cache refresh triggered")
				m.refreshAll()
			case <-m.ctx.Done():
				m.logger.Info("cache manager stopping due to context cancellation")
				return
			}
		}
	}()
}

// Stop gracefully shuts down the cache manager.
//
// Cancels the context to stop the background goroutine, stops the ticker,
// and waits for the goroutine to complete before returning.
//
// This ensures no refresh operations are in progress when Stop() returns.
func (m *Manager) Stop() {
	m.logger.Info("cache manager shutdown initiated")
	m.cancel() // Signal the goroutine to stop
	m.wg.Wait()
	m.logger.Info("cache manager shutdown complete")
}

// ManualRefresh triggers an immediate cache refresh in a separate goroutine.
//
// This method returns immediately without blocking the caller.
// Useful for triggering refresh via admin commands or API endpoints.
//
// The refresh follows the same retry logic as automatic refreshes.
func (m *Manager) ManualRefresh() {
	// Check if the manager has been stopped before spawning goroutine
	select {
	case <-m.ctx.Done():
		m.logger.Info("manual cache refresh skipped - manager stopped")
		return
	default:
		m.logger.Info("manual cache refresh triggered")
		// Run in separate goroutine so we don't block the caller
		go m.refreshAll()
	}
}

// refreshAll refreshes both caches sequentially with retry logic.
//
// Order of operations:
// 1. Refresh customers cache (with retries)
// 2. Refresh users cache (with retries)
//
// Each cache refresh is independent - failure of one doesn't prevent the other.
// On failure, the old cache is retained (handled by CacheRefresher.Initialize methods).
func (m *Manager) refreshAll() {
	m.logger.Info("refreshing all caches")

	// Refresh customers cache first
	if err := m.refreshCacheWithRetry(CacheTypeCustomers, m.refresher.InitializeCustomers); err != nil {
		m.logger.Error("customers cache refresh failed after retries",
			zap.Error(err),
		)
	}

	// Refresh users cache
	if err := m.refreshCacheWithRetry(CacheTypeUsers, m.refresher.InitializeUsers); err != nil {
		m.logger.Error("users cache refresh failed after retries",
			zap.Error(err),
		)
	}

	m.logger.Info("cache refresh cycle complete")
}

// refreshCacheWithRetry refreshes a single cache with exponential backoff retry.
//
// Retry strategy:
// - Initial backoff: 3 seconds
// - Backoff multiplier: 2x each retry
// - Backoff sequence: 3s, 6s, 12s, 24s, 48s, 96s, 192s (~381s total)
// - Max retry window: 5 minutes (300 seconds)
// - Context cancellation: Stops retrying immediately
//
// On success:
// - Records success metrics (counter, duration, timestamp)
// - Logs success
// - Returns nil
//
// On failure after retries:
// - Records failure metric (counter)
// - Logs error
// - Returns error
//
// Thread safety: Only called from background goroutine or ManualRefresh goroutine.
func (m *Manager) refreshCacheWithRetry(cacheType string, refreshFunc func() error) error {
	startTime := time.Now()
	attempt := 1
	backoffDuration := initialBackoff

	for {
		// Attempt refresh
		attemptStart := time.Now()
		err := refreshFunc()
		duration := time.Since(attemptStart)

		if err == nil {
			// Success! Record metrics and return
			m.recordSuccess(cacheType, duration)
			m.logger.Info("cache refresh succeeded",
				zap.String("cache_type", cacheType),
				zap.Int("attempt", attempt),
				zap.Duration("duration", duration),
			)
			return nil
		}

		// Check if we've exceeded the retry window
		if time.Since(startTime) >= maxRetryWindow {
			// Record final failure after all retries exhausted
			m.recordFailure(cacheType)
			m.logger.Error("cache refresh failed after max retry window",
				zap.String("cache_type", cacheType),
				zap.Duration("total_time", time.Since(startTime)),
				zap.Int("attempts", attempt),
				zap.Error(err),
			)
			return fmt.Errorf("cache refresh failed after %d attempts: %w", attempt, err)
		}

		// Record retry metric
		m.recordRetry(cacheType)

		// Log warning about retry
		m.logger.Warn("cache refresh failed, retrying with backoff",
			zap.String("cache_type", cacheType),
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoffDuration),
			zap.Error(err),
		)

		// Exponential backoff with context cancellation check
		select {
		case <-time.After(backoffDuration):
			// Continue with retry
		case <-m.ctx.Done():
			// Context cancelled, stop retrying
			m.logger.Info("cache refresh cancelled during backoff",
				zap.String("cache_type", cacheType),
				zap.Int("attempt", attempt),
			)
			return m.ctx.Err()
		}

		// Exponential backoff
		attempt++
		backoffDuration *= backoffMultiple
	}
}

// recordSuccess records success metrics for a cache refresh.
//
// Metrics recorded:
// - CacheRefreshTotal{cache_type, "success"} - Counter incremented
// - CacheRefreshDuration{cache_type} - Histogram of refresh duration
// - CacheLastRefreshTimestamp{cache_type} - Unix timestamp of this refresh
func (m *Manager) recordSuccess(cacheType string, duration time.Duration) {
	if m.metrics == nil {
		return
	}

	m.metrics.CacheRefreshTotal.WithLabelValues(cacheType, "success").Inc()
	m.metrics.CacheRefreshDuration.WithLabelValues(cacheType).Observe(duration.Seconds())
	m.metrics.CacheLastRefreshTimestamp.WithLabelValues(cacheType).Set(float64(time.Now().Unix()))
}

// recordFailure records failure metrics when cache refresh retries are exhausted.
//
// This should only be called after the max retry window has been exceeded,
// not on individual retry attempts. This provides a clean metric for alerting
// on permanent failures in Grafana.
//
// Metrics recorded:
// - CacheRefreshTotal{cache_type, "failure"} - Counter incremented
func (m *Manager) recordFailure(cacheType string) {
	if m.metrics == nil {
		return
	}

	m.metrics.CacheRefreshTotal.WithLabelValues(cacheType, "failure").Inc()
}

// recordRetry records retry metrics for a cache refresh.
//
// Metrics recorded:
// - CacheRefreshRetriesTotal{cache_type} - Counter incremented on each retry
func (m *Manager) recordRetry(cacheType string) {
	if m.metrics == nil {
		return
	}

	m.metrics.CacheRefreshRetriesTotal.WithLabelValues(cacheType).Inc()
}
