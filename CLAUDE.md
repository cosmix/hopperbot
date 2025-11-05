# Hopperbot Development Guide

## Project Overview

Go-based Slack bot providing an interactive modal interface for submitting structured data to Notion. Users invoke `/hopperbot` to open a searchable form that validates and submits to a Notion database.

## Notion API Version

**Current Version**: `2025-09-03`

The bot uses Notion API v2025-09-03, which introduces **multi-source database support**. Key changes:

- **Data Sources**: Databases are now containers with one or more data sources
- **Discovery**: Bot discovers data source IDs on startup via `InitializeDataSources()`
- **Operations**: All operations (queries, page creation) use data source IDs instead of database IDs
- **Multi-source Handling**: When multiple data sources exist, the bot uses the first one and logs a warning

### Migration Notes

Upgraded from v2022-06-28 to v2025-09-03 (2025-11-05):

- Parent type in page creation changed from `database_id` to `data_source_id`
- Query endpoints changed from `/v1/databases/:id/query` to `/v1/data_sources/:id/query`
- Added automatic data source discovery during initialization
- Updated `internal/notion/client.go`, `internal/slack/handler.go`, `pkg/constants/constants.go`, and tests

## Database Schema

The bot handles exactly 6 fields:

**Required:**

1. Idea/Topic (title) - aliases: title, idea, topic
2. Theme/Category (multi-select, max 1) - Notion expects multi_select, Slack form allows single selection only. Valid values: "new feature idea", "feature improvement", "market/competition intelligence", "customer pain point"
3. Product Area (select) - valid values: AI/ML, Integrations/SDKs, Data Governance, Systems, UX, Activation Kits, Activation, rETL, Transformations, EventStream, WH Ingestion
4. Submitted by (people) - Automatically populated with the Notion user mapped from the Slack user's email

**Optional:**

1. Comments (text) - aliases: comments, comment
2. Customer Organization (multi-select, max 10) - aliases: customer_org, customer, org

The bot validates all field values against the allowed lists and enforces max selection constraints.

## Architecture

### Components

- **Main Server** (`cmd/hopperbot/main.go`) - HTTP server with graceful shutdown, panic recovery, and explicit timeouts
- **Slack Handler** (`internal/slack/handler.go`) - Slash commands, interactive events, signature verification
- **Notion Client** (`internal/notion/client.go`) - API interface for database operations
- **Modal Builder** (`internal/slack/modals.go`) - Interactive modal construction with searchable dropdowns
- **Configuration** (`pkg/config/config.go`) - Environment variable management
- **Observability** - Metrics (`pkg/metrics`), health checks (`pkg/health`), middleware (`pkg/middleware`)

## Key Features

### Core Functionality

- Slack modal interface with signature verification and real-time validation
- **Rotating modal titles**: Each modal invocation displays a randomly selected witty title relevant to the submission type (feature ideas, improvements, customer intelligence)
- Notion API integration with support for Title, Rich Text, Select, Multi-select, and People properties
- Field validation: length limits (2000 chars), allowed values, max selections (10 for Customer Organization)
- Slack-to-Notion user mapping via email (requires `users:read.email` OAuth scope)
- External select menus for unlimited customers with in-memory search

### Production Readiness

- Graceful shutdown (30s timeout), panic recovery, HTTP timeouts (read: 10s, write: 30s, idle: 120s)
- Prometheus metrics (20+ metrics covering HTTP, Slack, Notion API, cache operations)
- Health checks (`/health` liveness, `/ready` readiness with dependency checks)
- Structured logging with zap
- Version endpoint (`/version`) with build metadata

### Test Coverage

- 140+ unit tests with 70%+ coverage across core packages
- Table-driven tests for handlers, modals, Notion client, health checks, metrics
- Mock-based testing for cache manager with comprehensive scenario coverage

### Cache Refresh Mechanism (Added 2025-11-03)

Automatic and manual cache refresh for customer and user data:

- **Automatic Refresh**: Periodic refresh via `CACHE_REFRESH_INTERVAL` env var (default: 60 minutes)
- **Manual Refresh**: Silent `/hopperbot refresh-cache` command (non-blocking)
- **Retry Strategy**: Exponential backoff (3s→192s) with 5-minute max retry window
- **Failure Handling**: Retains old cache on failure, logs errors, increments failure metric only when retries exhausted
- **Metrics**: `CacheRefreshTotal`, `CacheRefreshDuration`, `CacheLastRefreshTimestamp`, `CacheRefreshRetriesTotal`
- **Alert on**: `rate(hopperbot_cache_refresh_total{status="failure"}[5m]) > 0` (permanent failures only)

### TODO

- Integration tests with mocked Slack/Notion APIs
- Rate limiting (per-user and global)
- Grafana dashboards

## Development Workflow

**Local**: Set `.env`, run `go run cmd/hopperbot/main.go`, use ngrok for Slack webhooks

**Testing**: Type `/hopperbot` in Slack (requires properly configured app with interactive components)

**Deploy**: Docker, HTTPS required, configure health checks (`/health`, `/ready`)

**Build with version info**:

```bash
go build -ldflags "-X main.version=$(git describe --tags) -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o hopperbot cmd/hopperbot/main.go
```

## Customer Validation & External Select Menus

Uses Slack's **external select menu** pattern for unlimited customers with dynamic search.

**Architecture**:

- Customers fetched from Notion on startup and cached in memory
- User types → Slack calls `/slack/options` → Bot returns filtered results (3-tier matching: exact, prefix, contains)
- Submission validates against cached list
- Performance: <200ms for 1000+ customers, no DB calls during search

**CRITICAL CONFIG**: Set **Options Load URL** to `https://your-domain.com/slack/options` in Slack app → Interactivity & Shortcuts → Select Menus
(Without this, modal fails with `invalid_arguments`)

## Slack-to-Notion User Mapping

Automatically populates "Submitted by" field by mapping Slack users to Notion users via email.

**Flow**:
Startup: Notion Users API → Cache (`email → UUID`) → Submission: Slack `users.info` → Email lookup → Notion People property

**Requirements**:

- Slack OAuth scope: `users:read.email` (required)
- Notion: User information capability enabled

**Error Handling**: Blocks submission if user not found in Notion, case-insensitive email matching

**Performance**: 1 Slack API call per submission, 0 Notion calls (cached)

## Modal Architecture & Endpoints

**Flow**: `/hopperbot` → Modal opens → Submit → Notion

**Endpoints**: `/slack/command`, `/slack/interactive`, `/slack/options`, `/metrics`, `/health`, `/ready`, `/version`

**Field Extraction**: `view.State.Values[blockID][actionID].{SelectedOptions|Value}`

**Library**: `slack-go/slack`

## Extending

- **New Commands**: Register in Slack app, update routing in `main.go`
- **Modal Fields**: Modify `internal/slack/modals.go`
- **Notion Schema**: Modify `internal/notion/client.go`

## Security

- Slack signature verification, env vars for secrets, HTTPS required
- Panic recovery, HTTP timeouts (10s/30s/120s), graceful shutdown (30s)

## Observability & Monitoring

### Metrics (`/metrics`)

Prometheus endpoint with 20+ metrics:

- **HTTP**: requests_total, duration, in_flight, response_size
- **Slack**: commands, interactions, modal_submissions
- **Notion API**: requests, duration, errors
- **Application**: validation_errors, cache sizes (customers/users), panic_recoveries
- **Cache Refresh**: refresh_total, duration, last_timestamp, retries (by cache_type: customers/users)

### Health Checks

- **`/health`**: Liveness (200 if running)
- **`/ready`**: Readiness (checks Notion API, cache populated, returns 503 if unavailable, JSON with detailed check results)

### Middleware

Recovery (panic handling), metrics recording, 30s timeouts, structured logging

### Key Monitoring Queries

- **Error rate**: `rate(hopperbot_http_requests_total{status=~"5.."}[5m])`
- **p95 latency**: `histogram_quantile(0.95, rate(hopperbot_http_request_duration_seconds_bucket[5m]))`
- **Cache health**: `hopperbot_customer_cache_size > 0`
- **Refresh failures**: `rate(hopperbot_cache_refresh_total{status="failure"}[5m])`

### Alert On

High error rate (>5%), high latency (p95 >2s), Notion API down, empty cache, cache refresh failures, panic recoveries

## Code Quality Standards

Go best practices and production-quality code:

- **Documentation**: Package-level godoc, exported types/functions documented with examples, business context in comments
- **Naming**: Descriptive variable names (no generics), consistent patterns, clear intent
- **Organization**: Single responsibility packages, minimal coupling (interfaces), type safety
- **Quality**: `go fmt`, `go vet` (zero issues), `go mod tidy`, comprehensive table-driven tests
- **Build**: Version injection via ldflags, clean builds, production-ready error handling

## Next Steps

Future improvements:

1. Integration tests with mocked Slack/Notion APIs
2. Rate limiting (per-user and global)
3. Admin commands (view stats, manual operations)
4. Grafana dashboards and Prometheus alerts
5. CI/CD pipeline with automated testing
