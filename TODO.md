# Hopperbot TODO List

This document tracks deferred improvements and technical debt items identified during code review.

## Critical Items (Completed)

- [x] Remove `/debug/users` endpoint (security issue - exposed PII)
- [x] Fix JSON injection vulnerability in `respondToSlack()` function
- [x] Fix goroutine leak in timeout middleware
- [x] Add mutex protection for shared maps (`customerMap`, `validUsers`)

## High Priority (Short-term: 1-2 weeks)

### Testing

- [ ] **Add middleware tests** - `pkg/middleware` currently has 0% coverage

  - Test `WithTimeout` functionality and edge cases
  - Test `WithRecovery` panic handling
  - Test `WithMetrics` instrumentation
  - Test `WithLogging` output
  - Test `Chain` middleware composition

- [ ] **Add config package tests** - `pkg/config` has 0% coverage

  - Test environment variable loading
  - Test validation of required fields
  - Test handling of missing/invalid configuration
  - Test default values

- [ ] **Add integration tests** - End-to-end flow testing
  - Test full Slack slash command → modal → submission → Notion flow
  - Mock Slack and Notion APIs
  - Test error handling in the full pipeline
  - Test timeout scenarios
  - Test user mapping flow

### Code Quality

- [ ] **Standardize error response patterns** - Currently using multiple approaches
  - `respondToSlack()` - JSON encoding (fixed)
  - `respondWithErrors()` - JSON encoding
  - Direct `http.Error()` calls
  - Create a single, consistent error response helper
  - Use proper JSON encoding everywhere

### Operational Improvements

- [ ] **Cache refresh mechanism** - Currently requires restart for data changes

  - Implement periodic refresh (e.g., every hour) for customer list
  - Implement periodic refresh for user cache
  - Add manual refresh endpoint (protected with authentication)
  - Consider event-driven updates if feasible
  - Add metrics for cache refresh operations

- [ ] **Authentication for debug endpoints** - Secure any remaining debug/admin endpoints
  - Add API key or token-based authentication
  - Add IP whitelist option
  - Log all access attempts

## Medium Priority (1 month)

### Architecture & Design

- [ ] **Refactor Handler to separate concerns** - Handler currently does too much

  - Split into `SlackHandler` (HTTP/Slack protocol)
  - Create `SubmissionService` (business logic)
  - Create `NotionRepository` (data persistence)
  - Improves testability and maintainability

- [ ] **Introduce repository pattern** - Abstract external dependencies
  - Create interfaces for Slack client
  - Create interfaces for Notion client
  - Enables easier mocking in tests
  - Reduces coupling between packages

### Security & Resilience

- [ ] **Add rate limiting** - Protect against abuse

  - Per-user rate limits (e.g., 10 submissions per hour)
  - Global rate limits (e.g., 100 submissions per minute)
  - Return 429 status with Retry-After header
  - Add metrics for rate limit hits

- [ ] **Implement circuit breakers** - Resilience for external API calls
  - Circuit breaker for Notion API
  - Circuit breaker for Slack API
  - Configurable thresholds and timeouts
  - Metrics and alerting for open circuits

### Code Quality Improvements

- [ ] **Extract hardcoded magic numbers** - Improve maintainability

  - `healthMgr.RegisterReadinessCheck("client_cache", ..., 10)` - Extract minimum cache count
  - Move to `constants` package or configuration
  - Document the meaning of each constant

- [ ] **Address linter warnings** - Clean up code quality issues
  - Fix SA9003: Empty branch in middleware panic recovery
  - Add package comment to `middleware` package (ST1000)
  - Run `golangci-lint` and address all issues

## Low Priority (Future enhancements)

### Observability

- [ ] **Add OpenTelemetry tracing** - Enhanced distributed tracing

  - Trace requests from Slack through to Notion
  - Visualize latency breakdown
  - Correlate logs, metrics, and traces

- [ ] **Set up Grafana dashboards** - Visualize metrics

  - Request rate, latency, error rate
  - Notion API health and latency
  - Cache hit rates and sizes
  - User activity patterns

- [ ] **Implement structured alerting** - Proactive monitoring
  - High error rate alerts (>5% of requests failing)
  - High latency alerts (p95 > 2s)
  - Notion API down alerts
  - Empty cache alerts
  - Panic recovery alerts

### Features

- [ ] **Admin commands** - Operational management

  - `/hopperbot refresh-cache` - Manual cache refresh
  - `/hopperbot stats` - View bot statistics
  - `/hopperbot health` - Check system health
  - Restrict to admin users only

- [ ] **A/B testing capability** - Feature management
  - Feature flags for gradual rollouts
  - A/B test different modal layouts
  - Collect metrics on user engagement

### Architecture

- [ ] **Event sourcing for audit trail** - Compliance and debugging

  - Store all submission events
  - Enable audit trail and replay
  - Support compliance requirements

- [ ] **Admin UI for configuration** - Operational efficiency
  - Web-based configuration management
  - View and edit customer list
  - View cached users
  - Manage feature flags

## Notes

### Magic Numbers

The code review identified a hardcoded `10` in `cmd/hopperbot/main.go:79`:

```go
healthMgr.RegisterReadinessCheck("client_cache", health.ClientCacheChecker(
    handler.GetClientCount,
    10, // Minimum expected client count
))
```

This should be extracted to a constant if it's a meaningful business rule, or removed if it's just an arbitrary sanity check.

### Cache Refresh Strategy

Currently, the customer list and user cache are only fetched on startup. This means:

- Adding/removing customers in Notion requires bot restart
- Adding/removing users in Notion workspace requires bot restart

A periodic refresh (hourly or daily) would improve operational experience without requiring code changes.

### Test Coverage Summary (Pre-improvement)

- `internal/slack`: 65.6%
- `internal/notion`: 65.6%
- `pkg/health`: 97.5% (excellent)
- `pkg/metrics`: 14.3%
- `pkg/middleware`: **0%** (critical gap)
- `pkg/config`: **0%** (critical gap)
- `cmd/hopperbot`: **0%** (acceptable for main)

Target: 70%+ coverage across all critical packages.

## Completed Items Archive

### 2025-11-02 - Critical Security Fixes

- [x] Removed `/debug/users` endpoint (exposed user email addresses without authentication)
- [x] Fixed JSON injection vulnerability in `respondToSlack()` by using proper JSON encoding
- [x] Fixed goroutine leak in timeout middleware by adding panic recovery and cleanup
- [x] Added mutex protection for `customerMap` and `validUsers` to prevent race conditions
