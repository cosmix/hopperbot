package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// Note: Due to the global prometheus registry, we can only create metrics once.
// These tests verify the structure and functionality using a singleton approach.

var metricsOnce sync.Once
var testMetrics *Metrics

func getTestMetrics() *Metrics {
	metricsOnce.Do(func() {
		testMetrics = NewMetrics()
	})
	return testMetrics
}

// TestNewMetrics tests metrics initialization
func TestNewMetrics_AllMetricsPresent(t *testing.T) {
	metrics := getTestMetrics()

	if metrics == nil {
		t.Fatal("getTestMetrics should not return nil")
	}

	// Test HTTP metrics
	if metrics.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal should not be nil")
	}

	if metrics.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration should not be nil")
	}

	if metrics.HTTPRequestsInFlight == nil {
		t.Error("HTTPRequestsInFlight should not be nil")
	}

	if metrics.HTTPResponseSize == nil {
		t.Error("HTTPResponseSize should not be nil")
	}

	// Test Slack metrics
	if metrics.SlackCommandsTotal == nil {
		t.Error("SlackCommandsTotal should not be nil")
	}

	if metrics.SlackInteractionsTotal == nil {
		t.Error("SlackInteractionsTotal should not be nil")
	}

	if metrics.SlackModalSubmissions == nil {
		t.Error("SlackModalSubmissions should not be nil")
	}

	// Test Notion metrics
	if metrics.NotionAPIRequestsTotal == nil {
		t.Error("NotionAPIRequestsTotal should not be nil")
	}

	if metrics.NotionAPIRequestDuration == nil {
		t.Error("NotionAPIRequestDuration should not be nil")
	}

	if metrics.NotionAPIErrors == nil {
		t.Error("NotionAPIErrors should not be nil")
	}

	// Test application metrics
	if metrics.ValidationErrorsTotal == nil {
		t.Error("ValidationErrorsTotal should not be nil")
	}

	if metrics.ClientCacheSize == nil {
		t.Error("ClientCacheSize should not be nil")
	}

	if metrics.PanicRecoveriesTotal == nil {
		t.Error("PanicRecoveriesTotal should not be nil")
	}
}

// TestHTTPRequestsTotal tests counter metric operations
func TestHTTPRequestsTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	// Should be able to record metrics (won't panic)
	metrics.HTTPRequestsTotal.WithLabelValues("/health", "GET", "200").Inc()
	metrics.HTTPRequestsTotal.WithLabelValues("/slack/command", "POST", "200").Inc()
}

// TestHTTPRequestDuration tests histogram metric operations
func TestHTTPRequestDuration_Operations(t *testing.T) {
	metrics := getTestMetrics()

	// Should be able to observe durations
	metrics.HTTPRequestDuration.WithLabelValues("/health", "GET").Observe(0.123)
	metrics.HTTPRequestDuration.WithLabelValues("/slack/command", "POST").Observe(0.456)
}

// TestHTTPRequestsInFlight tests gauge metric operations
func TestHTTPRequestsInFlight_Operations(t *testing.T) {
	metrics := getTestMetrics()

	// Should be able to set gauge value
	metrics.HTTPRequestsInFlight.Set(1)
	metrics.HTTPRequestsInFlight.Inc()
	metrics.HTTPRequestsInFlight.Dec()
}

// TestHTTPResponseSize tests histogram metric with multiple labels
func TestHTTPResponseSize_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.HTTPResponseSize.WithLabelValues("/health", "GET").Observe(256)
	metrics.HTTPResponseSize.WithLabelValues("/slack/command", "POST").Observe(1024)
}

// TestSlackCommandsTotal tests Slack-specific counter
func TestSlackCommandsTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.SlackCommandsTotal.WithLabelValues("/hopperbot", "success").Inc()
	metrics.SlackCommandsTotal.WithLabelValues("/hopperbot", "error").Inc()
}

// TestSlackInteractionsTotal tests Slack interactions counter
func TestSlackInteractionsTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.SlackInteractionsTotal.WithLabelValues("view_submission", "submit_form_modal", "success").Inc()
	metrics.SlackInteractionsTotal.WithLabelValues("view_submission", "submit_form_modal", "error").Inc()
}

// TestSlackModalSubmissions tests modal submissions counter
func TestSlackModalSubmissions_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.SlackModalSubmissions.WithLabelValues("success").Inc()
	metrics.SlackModalSubmissions.WithLabelValues("error").Inc()
}

// TestNotionAPIRequestsTotal tests Notion API requests counter
func TestNotionAPIRequestsTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.NotionAPIRequestsTotal.WithLabelValues("submit_form", "success").Inc()
	metrics.NotionAPIRequestsTotal.WithLabelValues("fetch_clients", "error").Inc()
}

// TestNotionAPIRequestDuration tests Notion API duration histogram
func TestNotionAPIRequestDuration_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.NotionAPIRequestDuration.WithLabelValues("submit_form").Observe(0.5)
	metrics.NotionAPIRequestDuration.WithLabelValues("fetch_clients").Observe(1.2)
}

// TestNotionAPIErrors tests Notion API errors counter
func TestNotionAPIErrors_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.NotionAPIErrors.WithLabelValues("submit_form", "api_error").Inc()
	metrics.NotionAPIErrors.WithLabelValues("fetch_clients", "timeout").Inc()
	metrics.NotionAPIErrors.WithLabelValues("health_check", "connection_error").Inc()
}

// TestValidationErrorsTotal tests validation errors counter
func TestValidationErrorsTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.ValidationErrorsTotal.WithLabelValues("title").Inc()
	metrics.ValidationErrorsTotal.WithLabelValues("theme").Inc()
	metrics.ValidationErrorsTotal.WithLabelValues("product_area").Inc()
}

// TestClientCacheSize tests gauge metric for cache size
func TestClientCacheSize_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.ClientCacheSize.Set(10)
	metrics.ClientCacheSize.Set(25)
	metrics.ClientCacheSize.Set(0) // Empty cache
}

// TestPanicRecoveriesTotal tests panic recovery counter
func TestPanicRecoveriesTotal_Operations(t *testing.T) {
	metrics := getTestMetrics()

	metrics.PanicRecoveriesTotal.Inc()
	metrics.PanicRecoveriesTotal.Inc()
}

// TestMetricsStructure tests that all metrics are properly initialized
func TestMetricsStructure(t *testing.T) {
	metrics := getTestMetrics()

	// Count non-nil metrics
	nonNilMetrics := 0
	if metrics.HTTPRequestsTotal != nil {
		nonNilMetrics++
	}
	if metrics.HTTPRequestDuration != nil {
		nonNilMetrics++
	}
	if metrics.HTTPRequestsInFlight != nil {
		nonNilMetrics++
	}
	if metrics.HTTPResponseSize != nil {
		nonNilMetrics++
	}
	if metrics.SlackCommandsTotal != nil {
		nonNilMetrics++
	}
	if metrics.SlackInteractionsTotal != nil {
		nonNilMetrics++
	}
	if metrics.SlackModalSubmissions != nil {
		nonNilMetrics++
	}
	if metrics.NotionAPIRequestsTotal != nil {
		nonNilMetrics++
	}
	if metrics.NotionAPIRequestDuration != nil {
		nonNilMetrics++
	}
	if metrics.NotionAPIErrors != nil {
		nonNilMetrics++
	}
	if metrics.ValidationErrorsTotal != nil {
		nonNilMetrics++
	}
	if metrics.ClientCacheSize != nil {
		nonNilMetrics++
	}
	if metrics.PanicRecoveriesTotal != nil {
		nonNilMetrics++
	}

	expectedMetrics := 13
	if nonNilMetrics != expectedMetrics {
		t.Errorf("expected %d non-nil metrics, got %d", expectedMetrics, nonNilMetrics)
	}
}

// TestMetricsTypesAssertable tests that metrics are of expected types
func TestMetricsTypesAssertable(t *testing.T) {
	metrics := getTestMetrics()

	// Test that we can type assert to expected Prometheus types
	var _ prometheus.Collector = metrics.HTTPRequestsTotal
	var _ prometheus.Collector = metrics.HTTPRequestDuration
	var _ prometheus.Metric = metrics.HTTPRequestsInFlight
	var _ prometheus.Collector = metrics.SlackCommandsTotal
	var _ prometheus.Collector = metrics.NotionAPIRequestsTotal
}
