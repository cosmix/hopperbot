package notion

import (
	"context"
	"time"

	"github.com/rudderlabs/hopperbot/pkg/metrics"
)

// SetMetrics sets the metrics instance for the client
func (c *Client) SetMetrics(m *metrics.Metrics) {
	c.metrics = m
	// Update customer cache size metric
	if m != nil {
		m.ClientCacheSize.Set(float64(len(c.customerMap)))
	}
}

// recordNotionRequest records metrics for Notion API requests
func (c *Client) recordNotionRequest(operation string, startTime time.Time, err error) {
	if c.metrics == nil {
		return
	}

	duration := time.Since(startTime).Seconds()
	c.metrics.NotionAPIRequestDuration.WithLabelValues(operation).Observe(duration)

	status := "success"
	if err != nil {
		status = "error"
		errorType := "unknown"
		if err == context.DeadlineExceeded {
			errorType = "timeout"
		} else if err == context.Canceled {
			errorType = "canceled"
		} else {
			errorType = "api_error"
		}
		c.metrics.NotionAPIErrors.WithLabelValues(operation, errorType).Inc()
	}

	c.metrics.NotionAPIRequestsTotal.WithLabelValues(operation, status).Inc()
}

// HealthCheck performs a lightweight health check to verify Notion API connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	start := time.Now()
	_, err := c.GetDatabaseSchema()
	c.recordNotionRequest("health_check", start, err)
	return err
}
