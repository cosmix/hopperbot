package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
	HTTPResponseSize     *prometheus.HistogramVec

	// Slack-specific metrics
	SlackCommandsTotal     *prometheus.CounterVec
	SlackInteractionsTotal *prometheus.CounterVec
	SlackModalSubmissions  *prometheus.CounterVec

	// Notion API metrics
	NotionAPIRequestsTotal   *prometheus.CounterVec
	NotionAPIRequestDuration *prometheus.HistogramVec
	NotionAPIErrors          *prometheus.CounterVec

	// Application metrics
	ValidationErrorsTotal *prometheus.CounterVec
	ClientCacheSize       prometheus.Gauge
	UserCacheSize         prometheus.Gauge
	PanicRecoveriesTotal  prometheus.Counter
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP request counter by endpoint and status code
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_http_requests_total",
				Help: "Total number of HTTP requests by endpoint and status code",
			},
			[]string{"endpoint", "method", "status"},
		),

		// HTTP request duration histogram by endpoint
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hopperbot_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
			},
			[]string{"endpoint", "method"},
		),

		// HTTP requests currently in flight
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "hopperbot_http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),

		// HTTP response size histogram
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hopperbot_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to 100MB
			},
			[]string{"endpoint", "method"},
		),

		// Slack slash command invocations
		SlackCommandsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_slack_commands_total",
				Help: "Total number of Slack slash commands received",
			},
			[]string{"command", "status"},
		),

		// Slack interactive component submissions
		SlackInteractionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_slack_interactions_total",
				Help: "Total number of Slack interactive component events received",
			},
			[]string{"type", "callback_id", "status"},
		),

		// Modal submissions specifically
		SlackModalSubmissions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_slack_modal_submissions_total",
				Help: "Total number of Slack modal submissions",
			},
			[]string{"status"},
		),

		// Notion API request counter
		NotionAPIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_notion_api_requests_total",
				Help: "Total number of Notion API requests",
			},
			[]string{"operation", "status"},
		),

		// Notion API request duration
		NotionAPIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hopperbot_notion_api_request_duration_seconds",
				Help:    "Notion API request duration in seconds",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}, // Up to 30s timeout
			},
			[]string{"operation"},
		),

		// Notion API errors
		NotionAPIErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_notion_api_errors_total",
				Help: "Total number of Notion API errors",
			},
			[]string{"operation", "error_type"},
		),

		// Form validation errors
		ValidationErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hopperbot_validation_errors_total",
				Help: "Total number of form validation errors",
			},
			[]string{"field"},
		),

		// Client cache size (number of valid clients loaded)
		ClientCacheSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "hopperbot_client_cache_size",
				Help: "Number of valid clients currently cached",
			},
		),

		// User cache size (number of Notion users loaded for email mapping)
		UserCacheSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "hopperbot_user_cache_size",
				Help: "Number of Notion users currently cached for Slack-to-Notion mapping",
			},
		),

		// Panic recoveries
		PanicRecoveriesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "hopperbot_panic_recoveries_total",
				Help: "Total number of panic recoveries in HTTP handlers",
			},
		),
	}
}

// GetMetrics returns the singleton metrics instance
var defaultMetrics *Metrics

// Init initializes the default metrics instance
func Init() *Metrics {
	if defaultMetrics == nil {
		defaultMetrics = NewMetrics()
	}
	return defaultMetrics
}

// Get returns the default metrics instance
func Get() *Metrics {
	if defaultMetrics == nil {
		return Init()
	}
	return defaultMetrics
}
