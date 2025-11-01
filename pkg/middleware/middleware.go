package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/rudderlabs/hopperbot/pkg/metrics"
	"go.uber.org/zap"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// WithMetrics wraps an HTTP handler with Prometheus metrics collection
func WithMetrics(endpoint string, m *metrics.Metrics, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Increment in-flight requests
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Wrap response writer to capture status and size
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     0,
			size:           0,
		}

		// Call the handler
		handler(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := rw.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		m.HTTPRequestsTotal.WithLabelValues(endpoint, r.Method, strconv.Itoa(status)).Inc()
		m.HTTPRequestDuration.WithLabelValues(endpoint, r.Method).Observe(duration)
		m.HTTPResponseSize.WithLabelValues(endpoint, r.Method).Observe(float64(rw.size))
	}
}

// WithTimeout wraps an HTTP handler with context-based timeout
func WithTimeout(timeout time.Duration, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// Create channel to signal handler completion
		done := make(chan struct{})

		// Run handler in goroutine
		go func() {
			handler(w, r.WithContext(ctx))
			close(done)
		}()

		// Wait for handler to complete or timeout
		select {
		case <-done:
			// Handler completed successfully
			return
		case <-ctx.Done():
			// Timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
		}
	}
}

// WithRecovery wraps HTTP handlers with panic recovery to prevent server crashes
func WithRecovery(logger *zap.Logger, m *metrics.Metrics, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.PanicRecoveriesTotal.Inc()
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		handler(w, r)
	}
}

// WithLogging wraps HTTP handlers with request/response logging
func WithLogging(logger *zap.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     0,
			size:           0,
		}

		// Call the handler
		handler(rw, r)

		// Log the request
		status := rw.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", status),
			zap.Duration("duration", time.Since(start)),
			zap.Int("size", rw.size),
			zap.String("user_agent", r.UserAgent()),
		)
	}
}

// Chain combines multiple middleware functions into one
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	// Apply middleware in reverse order so they execute in the order specified
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
