# Hopperbot Observability Guide

## Table of Contents

- [Overview](#overview)
- [What Was Implemented](#what-was-implemented)
- [Quick Start](#quick-start)
- [Production Setup](#production-setup)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Next Steps](#next-steps)

## Overview

Hopperbot includes production-grade observability features following modern monitoring, alerting, and debugging best practices. The implementation provides comprehensive visibility into application health, performance, and operational metrics.

**Key Capabilities:**
- Prometheus metrics for performance and business metrics
- Health checks for liveness and readiness probes
- Request-level instrumentation with middleware
- Distributed system monitoring support
- Production-ready alerting templates

## What Was Implemented

### 1. Prometheus Metrics (`pkg/metrics/metrics.go`)

A complete metrics package with 15+ Prometheus metrics covering all aspects of the application:

#### HTTP Metrics
- `hopperbot_http_requests_total` - Counter for all HTTP requests (labels: endpoint, method, status)
- `hopperbot_http_request_duration_seconds` - Histogram for request latency
- `hopperbot_http_requests_in_flight` - Gauge for concurrent requests
- `hopperbot_http_response_size_bytes` - Histogram for response sizes

#### Slack Metrics
- `hopperbot_slack_commands_total` - Counter for slash command invocations
- `hopperbot_slack_interactions_total` - Counter for interactive events
- `hopperbot_slack_modal_submissions_total` - Counter for modal submissions

#### Notion API Metrics
- `hopperbot_notion_api_requests_total` - Counter for API requests (by operation)
- `hopperbot_notion_api_request_duration_seconds` - Histogram for API latency
- `hopperbot_notion_api_errors_total` - Counter for API errors (with error types)

#### Application Metrics
- `hopperbot_validation_errors_total` - Counter for form validation errors
- `hopperbot_client_cache_size` - Gauge for cached client count
- `hopperbot_user_cache_size` - Gauge for cached user count
- `hopperbot_panic_recoveries_total` - Counter for panic recoveries

### 2. Health Checks (`pkg/health/health.go`)

Production-ready health check system with liveness and readiness probes:

**Features:**
- Separate liveness and readiness endpoints
- Parallel check execution with timeout protection
- Detailed JSON responses with check status, duration, and metadata
- Status hierarchy: healthy, degraded, unhealthy
- Built-in checkers for common scenarios

**Endpoints:**
- `/health` - Liveness probe (is the server running?)
- `/ready` - Readiness probe (can we serve traffic?)

**Included Checks:**
- Notion API connectivity check
- Client cache validation (ensures cache is populated)
- User cache validation (ensures user mapping is ready)
- Server health check (always healthy baseline)

### 3. Middleware (`pkg/middleware/middleware.go`)

Comprehensive middleware stack for request instrumentation:

**Middleware Functions:**
- `WithMetrics` - Records request metrics (duration, status, size)
- `WithTimeout` - Context-based timeout protection (prevents hanging requests)
- `WithRecovery` - Panic recovery with stack trace logging
- `WithLogging` - Structured request/response logging
- `Chain` - Middleware composition helper

**Request Wrapping:**
- Custom `responseWriter` captures status codes and response sizes
- Non-blocking timeout handling
- Panic recovery prevents server crashes

### 4. Instrumented Handlers

**Slack Handler Instrumentation:**
- Metrics recording for all Slack operations
- Command invocation tracking
- Modal submission metrics
- Validation error tracking
- Health check integration

**Notion Client Instrumentation:**
- API call latency tracking
- Error classification (timeout, canceled, API error)
- Operation-specific metrics (submit_form, initialize_clients, health_check)
- Client and user cache size updates

### 5. Main Server Integration (`cmd/hopperbot/main.go`)

Complete integration of observability features:

**Endpoints:**
- `/metrics` - Prometheus metrics (via `promhttp.Handler()`)
- `/health` - Liveness probe
- `/ready` - Readiness probe with dependency checks
- `/version` - Build version and metadata
- `/slack/command` - With full middleware stack
- `/slack/interactive` - With full middleware stack

**Middleware Stack (applied to all Slack endpoints):**
1. Recovery middleware (outermost - catches panics)
2. Metrics middleware (records request stats)
3. Timeout middleware (30s context timeout)
4. Logging middleware (structured logging)

**Startup Sequence:**
1. Initialize logger
2. Load configuration
3. Initialize metrics
4. Create and configure handlers
5. Initialize dependencies (fetch clients and users from Notion)
6. Register health checks
7. Configure endpoints with middleware
8. Start server with graceful shutdown

### File Structure

```
pkg/
├── metrics/
│   └── metrics.go              # Prometheus metrics definitions
├── health/
│   └── health.go               # Health check system
└── middleware/
    └── middleware.go           # Request middleware

internal/
├── slack/
│   ├── handler.go              # Updated with metrics field
│   └── instrumented_handler.go # Metrics recording methods
└── notion/
    ├── client.go               # Updated with metrics field
    └── instrumented_client.go  # Metrics recording methods

cmd/hopperbot/
└── main.go                     # Integrated observability
```

## Quick Start

### Running Locally

```bash
# Start the bot (requires environment variables set)
go run cmd/hopperbot/main.go

# The server will log:
# - Metrics endpoint: /metrics
# - Health endpoint: /health
# - Readiness endpoint: /ready
# - Version endpoint: /version
```

### Checking Observability Endpoints

#### Health Checks

```bash
# Liveness (is server running?)
curl http://localhost:8080/health

# Readiness (can we serve traffic?)
curl http://localhost:8080/ready | jq

# Version information
curl http://localhost:8080/version
```

**Example Healthy Response:**
```json
{
  "status": "healthy",
  "uptime": "5m30s",
  "timestamp": "2025-10-31T10:30:00Z",
  "checks": [
    {
      "name": "notion_api",
      "status": "healthy",
      "message": "Notion API is reachable",
      "duration": "145ms"
    },
    {
      "name": "client_cache",
      "status": "healthy",
      "message": "Client cache is populated",
      "duration": "500µs",
      "metadata": {
        "count": 42
      }
    },
    {
      "name": "user_cache",
      "status": "healthy",
      "message": "User cache is populated",
      "duration": "300µs",
      "metadata": {
        "count": 156
      }
    }
  ]
}
```

#### Metrics

```bash
# View all Prometheus metrics
curl http://localhost:8080/metrics

# Check specific metrics
curl http://localhost:8080/metrics | grep hopperbot_client_cache_size
curl http://localhost:8080/metrics | grep hopperbot_http_requests_total
```

## Production Setup

### Docker Compose Setup

Create a `docker-compose.yml` for local testing or development:

```yaml
version: '3'
services:
  hopperbot:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}
      - SLACK_SIGNING_SECRET=${SLACK_SIGNING_SECRET}
      - NOTION_API_KEY=${NOTION_API_KEY}
      - NOTION_DATABASE_ID=${NOTION_DATABASE_ID}
      - NOTION_CLIENTS_DB_ID=${NOTION_CLIENTS_DB_ID}

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./prometheus-rules.yml:/etc/prometheus/prometheus-rules.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
```

### Prometheus Configuration

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# Load alerting rules
rule_files:
  - 'prometheus-rules.yml'

scrape_configs:
  - job_name: 'hopperbot'
    static_configs:
      - targets: ['hopperbot:8080']
    metrics_path: '/metrics'
```

### Kubernetes Deployment

#### Pod with Health Probes

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hopperbot
  labels:
    app: hopperbot
spec:
  containers:
  - name: hopperbot
    image: hopperbot:latest
    ports:
    - containerPort: 8080
      name: http
    env:
    - name: SLACK_BOT_TOKEN
      valueFrom:
        secretKeyRef:
          name: hopperbot-secrets
          key: slack-bot-token
    - name: SLACK_SIGNING_SECRET
      valueFrom:
        secretKeyRef:
          name: hopperbot-secrets
          key: slack-signing-secret
    - name: NOTION_API_KEY
      valueFrom:
        secretKeyRef:
          name: hopperbot-secrets
          key: notion-api-key
    - name: NOTION_DATABASE_ID
      valueFrom:
        configMapKeyRef:
          name: hopperbot-config
          key: notion-database-id
    - name: NOTION_CLIENTS_DB_ID
      valueFrom:
        configMapKeyRef:
          name: hopperbot-config
          key: notion-clients-db-id
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 30
      timeoutSeconds: 5
      failureThreshold: 3
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
      timeoutSeconds: 5
      failureThreshold: 2
```

#### ServiceMonitor (for Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hopperbot
  labels:
    app: hopperbot
spec:
  selector:
    matchLabels:
      app: hopperbot
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
```

### Grafana Setup

1. Access Grafana at http://localhost:3000
2. Login with `admin` / `admin` (change on first login)
3. Add Prometheus as a data source:
   - **Configuration** → **Data Sources** → **Add data source**
   - Select **Prometheus**
   - URL: `http://prometheus:9090`
   - Click **Save & Test**

#### Dashboard Panels

**Request Rate:**
```promql
sum(rate(hopperbot_http_requests_total[5m])) by (endpoint)
```

**Error Rate:**
```promql
sum(rate(hopperbot_http_requests_total{status=~"5.."}[5m])) by (endpoint)
/
sum(rate(hopperbot_http_requests_total[5m])) by (endpoint)
```

**Latency Percentiles (p50, p95, p99):**
```promql
histogram_quantile(0.50, sum(rate(hopperbot_http_request_duration_seconds_bucket[5m])) by (le, endpoint))
histogram_quantile(0.95, sum(rate(hopperbot_http_request_duration_seconds_bucket[5m])) by (le, endpoint))
histogram_quantile(0.99, sum(rate(hopperbot_http_request_duration_seconds_bucket[5m])) by (le, endpoint))
```

**Cache Health:**
```promql
hopperbot_client_cache_size
hopperbot_user_cache_size
```

**Notion API Performance:**
```promql
# Request rate by operation
rate(hopperbot_notion_api_requests_total[5m])

# Error rate
rate(hopperbot_notion_api_errors_total[5m])

# Latency (p95)
histogram_quantile(0.95, rate(hopperbot_notion_api_request_duration_seconds_bucket[5m]))
```

## Monitoring and Alerting

### Key Metrics to Watch

#### HTTP Performance
```promql
# Request rate
rate(hopperbot_http_requests_total[5m])

# Error rate
rate(hopperbot_http_requests_total{status=~"5.."}[5m])

# Latency (p95)
histogram_quantile(0.95, rate(hopperbot_http_request_duration_seconds_bucket[5m]))
```

#### Notion API Health
```promql
# API request rate
rate(hopperbot_notion_api_requests_total[5m])

# API error rate
rate(hopperbot_notion_api_errors_total[5m])

# API latency (p95)
histogram_quantile(0.95, rate(hopperbot_notion_api_request_duration_seconds_bucket[5m]))
```

#### Application Health
```promql
# Client cache size (should be > 0)
hopperbot_client_cache_size

# User cache size (should be > 0)
hopperbot_user_cache_size

# Validation errors
rate(hopperbot_validation_errors_total[5m])

# Panic recoveries (should be 0)
increase(hopperbot_panic_recoveries_total[1h])
```

### Alert Rules

Create `prometheus-rules.yml`:

```yaml
groups:
  - name: hopperbot_alerts
    interval: 30s
    rules:
      # Critical Alerts
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(hopperbot_http_requests_total{status=~"5.."}[5m]))
            /
            sum(rate(hopperbot_http_requests_total[5m]))
          ) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(hopperbot_http_request_duration_seconds_bucket[5m])) by (le, endpoint)
          ) > 2
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High latency on {{ $labels.endpoint }}"
          description: "p95 latency is {{ $value }}s"

      - alert: NotionAPIDown
        expr: |
          rate(hopperbot_notion_api_errors_total[5m]) > 0.1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Notion API is experiencing errors"
          description: "Error rate: {{ $value | humanize }} errors/sec"

      - alert: EmptyClientCache
        expr: hopperbot_client_cache_size == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Client cache is empty"
          description: "No clients loaded from Notion database"

      - alert: EmptyUserCache
        expr: hopperbot_user_cache_size == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "User cache is empty"
          description: "No users loaded from Notion workspace"

      - alert: PanicDetected
        expr: increase(hopperbot_panic_recoveries_total[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Application panic detected"
          description: "{{ $value }} panics recovered in the last 5 minutes"

      # Warning Alerts
      - alert: ElevatedErrorRate
        expr: |
          (
            sum(rate(hopperbot_http_requests_total{status=~"5.."}[5m]))
            /
            sum(rate(hopperbot_http_requests_total[5m]))
          ) > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Elevated error rate"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: IncreasedLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(hopperbot_http_request_duration_seconds_bucket[5m])) by (le, endpoint)
          ) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Increased latency on {{ $labels.endpoint }}"
          description: "p95 latency is {{ $value }}s"

      - alert: ValidationFailureSpike
        expr: |
          rate(hopperbot_validation_errors_total[5m]) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Spike in validation errors"
          description: "Validation error rate: {{ $value | humanize }} errors/sec"

      - alert: NotionAPISlowness
        expr: |
          histogram_quantile(0.95,
            rate(hopperbot_notion_api_request_duration_seconds_bucket[5m])
          ) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Notion API responding slowly"
          description: "p95 latency is {{ $value }}s"
```

### Recommended Dashboards

Create these dashboards in Grafana:

**1. Overview Dashboard**
- Request rate, error rate, latency
- Notion API health
- Cache status (client and user caches)
- Active connections

**2. Performance Dashboard**
- Latency percentiles (p50, p95, p99)
- Request distribution by endpoint
- In-flight requests
- Response size distribution

**3. Error Dashboard**
- Error rate by endpoint
- Validation errors by field
- Notion API errors by type
- Panic recovery count

**4. Business Metrics Dashboard**
- Slack command invocations
- Modal submissions (success vs failure)
- Client selection frequency
- User activity patterns

## Troubleshooting

### Metrics Not Appearing

```bash
# Check if metrics endpoint is accessible
curl http://localhost:8080/metrics

# Verify Prometheus can reach the target
# In Prometheus UI: Status → Targets
# Look for the 'hopperbot' job and check if it's UP
```

**Common Issues:**
- Firewall blocking port 8080
- Prometheus configuration pointing to wrong target
- Metrics endpoint not enabled in the application

### Health Check Failing

```bash
# Check readiness probe details
curl http://localhost:8080/ready | jq

# Look for failed checks in the response
```

**Common Issues:**
- **Notion API check failing**: Invalid API key or network connectivity issues
- **Client cache empty**: Clients database not shared with integration or wrong database ID
- **User cache empty**: Notion users API permission not granted to integration

### High Latency

```bash
# Check Notion API latency specifically
curl http://localhost:8080/metrics | grep notion_api_request_duration

# Review specific slow operations
curl http://localhost:8080/metrics | grep hopperbot_http_request_duration_seconds_bucket
```

**Common Causes:**
- Notion API responding slowly (check Notion status page)
- Network latency between your server and Notion
- Large result sets from database queries (need pagination)
- Client or user cache not initialized (triggering API calls per request)

### Empty Cache Issues

```bash
# Check cache sizes
curl http://localhost:8080/metrics | grep cache_size

# Check readiness endpoint for cache details
curl http://localhost:8080/ready | jq '.checks[] | select(.name | contains("cache"))'
```

**Solutions:**
- Verify databases are shared with the Notion integration
- Check database IDs are correct in environment variables
- Review server logs for errors during startup initialization
- Restart the bot to re-fetch data from Notion

### Request Signature Validation Errors

**Symptoms:** Metrics show high `hopperbot_http_requests_total{status="401"}`

**Causes:**
- Wrong signing secret in configuration
- Request body modified by proxy/load balancer
- Clock skew between Slack and your server

**Solutions:**
- Verify `SLACK_SIGNING_SECRET` matches the value in Slack App settings
- Ensure proxy/load balancer passes raw request body
- Synchronize server clock using NTP

## Best Practices

### 1. Metric Labels
- Use appropriate cardinality with meaningful labels (endpoint, status, operation)
- Avoid high-cardinality labels (like user IDs, request IDs)
- Keep label combinations reasonable (< 1000 unique combinations per metric)

### 2. Histogram Buckets
- Sensible defaults for latency: 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10 seconds
- Adjust buckets based on actual latency patterns
- Response size buckets: 100B, 1KB, 10KB, 100KB, 1MB

### 3. Error Classification
- Detailed error types for better debugging
- Separate transient errors (timeout, network) from permanent errors (auth, not found)
- Track error context (which operation, which endpoint)

### 4. Health Check Separation
- Liveness: Is the application alive? (simple, fast check)
- Readiness: Can it serve traffic? (check dependencies)
- Never fail liveness for dependency issues
- Keep health checks < 1 second response time

### 5. Timeout Handling
- Context-based timeouts prevent resource exhaustion
- Set reasonable timeouts (30s for web requests, 10s for API calls)
- Propagate timeouts through call chains
- Monitor timeout rates (high rate = capacity or performance issue)

### 6. Panic Recovery
- Graceful degradation instead of server crashes
- Log stack traces for debugging
- Alert on ANY panic (should be rare in production)
- Don't hide bugs by recovering silently

### 7. Structured Logging
- Consistent log format with request context
- Include request ID for tracing
- Log levels: DEBUG < INFO < WARN < ERROR
- Aggregate logs centrally (e.g., Loki, ELK)

### 8. Middleware Composition
- Clean separation of concerns
- Apply middleware in correct order (recovery → metrics → timeout → logging)
- Keep middleware focused and testable

## Next Steps

### Recommended Enhancements

1. **Grafana Dashboards**
   - Create pre-built JSON dashboards for common views
   - Include dashboard versioning in the repository
   - Document dashboard installation process

2. **Alert Manager Integration**
   - Configure Alertmanager for alert routing
   - Set up notification channels (Slack, PagerDuty, email)
   - Define alert grouping and suppression rules

3. **Distributed Tracing** (Optional)
   - Add OpenTelemetry for distributed tracing
   - Trace requests through Slack → Hopperbot → Notion
   - Correlate traces with metrics and logs
   - Use Jaeger or Tempo for trace storage

4. **Log Aggregation**
   - Ship logs to centralized logging (Loki, ELK, CloudWatch)
   - Correlate logs with traces and metrics
   - Set up log-based alerts for specific error patterns

5. **SLIs/SLOs**
   - Define Service Level Indicators (availability, latency, error rate)
   - Set Service Level Objectives (99.9% uptime, p95 < 1s)
   - Implement error budget tracking
   - Create SLO-based alerts

6. **Runbooks**
   - Create operational runbooks for common issues
   - Link alerts to runbooks (use alert annotations)
   - Document escalation procedures
   - Include troubleshooting flowcharts

7. **Performance Profiling**
   - Add pprof endpoints for CPU and memory profiling
   - Create performance regression tests
   - Monitor memory leaks and goroutine leaks
   - Set up continuous profiling

8. **Cost Optimization**
   - Monitor metrics storage costs
   - Implement metric retention policies
   - Use recording rules for expensive queries
   - Optimize high-cardinality metrics

### Dependencies

The observability features use:
- `github.com/prometheus/client_golang v1.23.2` - Prometheus client library

All other features use the standard library or existing dependencies.

---

**For more information:**
- [Prometheus Querying Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/best-practices-for-creating-dashboards/)
- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
