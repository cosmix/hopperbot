package slack

import (
	"github.com/rudderlabs/hopperbot/internal/notion"
	"github.com/rudderlabs/hopperbot/pkg/metrics"
)

// SetMetrics sets the metrics instance for the handler and its dependencies
func (h *Handler) SetMetrics(m *metrics.Metrics) {
	h.metrics = m
	// Also set metrics on the Notion client
	if h.notionClient != nil {
		h.notionClient.SetMetrics(m)
	}
}

// recordSlackCommand records metrics for slash command invocations
func (h *Handler) recordSlackCommand(command, status string) {
	if h.metrics != nil {
		h.metrics.SlackCommandsTotal.WithLabelValues(command, status).Inc()
	}
}

// recordSlackInteraction records metrics for interactive component events
func (h *Handler) recordSlackInteraction(interactionType, callbackID, status string) {
	if h.metrics != nil {
		h.metrics.SlackInteractionsTotal.WithLabelValues(interactionType, callbackID, status).Inc()
	}
}

// recordModalSubmission records metrics for modal submissions
func (h *Handler) recordModalSubmission(status string) {
	if h.metrics != nil {
		h.metrics.SlackModalSubmissions.WithLabelValues(status).Inc()
	}
}

// recordValidationError records metrics for field validation errors
func (h *Handler) recordValidationError(field string) {
	if h.metrics != nil {
		h.metrics.ValidationErrorsTotal.WithLabelValues(field).Inc()
	}
}

// GetClientCount returns the count of cached clients for health checks
func (h *Handler) GetClientCount() int {
	if h.notionClient != nil {
		return len(h.notionClient.GetValidCustomers())
	}
	return 0
}

// NotionClient returns the Notion client for health checks
func (h *Handler) NotionClient() *notion.Client {
	return h.notionClient
}
