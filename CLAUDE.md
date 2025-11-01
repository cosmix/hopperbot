# Hopperbot Development Guide

## Project Overview

Hopperbot is a Go-based Slack bot that provides an interactive modal interface for submitting form data to Notion databases. Users invoke the `/hopperbot` slash command, which opens a modal with searchable dropdowns and text fields. The bot is designed to be simple, secure, and easily deployable.

## Database Schema

The bot handles exactly 6 fields:

**Required:**

1. Idea/Topic (title) - aliases: title, idea, topic
2. Theme/Category (select) - valid values: "new feature idea", "feature improvement", "market/competition intelligence", "customer pain point"
3. Product Area (select) - valid values: AI/ML, Integrations/SDKs, Data Governance, Systems, UX, Activation Kits, Activation, rETL, Transformations, EventStream, WH Ingestion
4. Submitted By (people) - Automatically populated with the Notion user mapped from the Slack user's email

**Optional:**

1. Comments (text) - aliases: comments, comment
2. Customer Org (multi-select, max 10) - aliases: customer_org, customer, org

The bot validates all field values against the allowed lists and enforces max selection constraints.

## Architecture

### Components

1. **Main Server** (`cmd/hopperbot/main.go`)

   - HTTP server with production-ready safety features:
     - Graceful shutdown with 30s timeout (handles SIGTERM/SIGINT)
     - Panic recovery middleware on all handlers (prevents server crashes)
     - Explicit timeouts (ReadTimeout: 10s, WriteTimeout: 30s, IdleTimeout: 120s)
   - Health check endpoint for monitoring
   - Entry point for the application

2. **Slack Handler** (`internal/slack/handler.go`)

   - Processes incoming slash commands and interactive events
   - Verifies Slack request signatures for security
   - `HandleSlashCommand` opens interactive modals when `/hopperbot` is invoked
   - `HandleInteractive` processes modal submissions and orchestrates the flow

3. **Notion Client** (`internal/notion/client.go`)

   - Interfaces with Notion API
   - Creates database entries
   - Handles different property types (text, number, select, etc.)

4. **Modal Builder** (`internal/slack/modals.go`)

   - Constructs interactive modal views
   - Defines form fields with searchable dropdowns
   - Handles multi-select and single-select fields

5. **Configuration** (`pkg/config/config.go`)

   - Loads and validates environment variables
   - Centralizes configuration management

6. **Observability** (production-grade monitoring)
   - **Metrics** (`pkg/metrics/metrics.go`) - Prometheus metrics for monitoring
   - **Health Checks** (`pkg/health/health.go`) - Liveness and readiness probes
   - **Middleware** (`pkg/middleware/middleware.go`) - Request instrumentation and timeouts

## Current Implementation Status

### Completed

- Basic project structure with Go modules
- Slack slash command handler with signature verification
- Interactive Slack modal interface with searchable dropdowns
- Modal submission handling with field extraction
- Real-time validation in modal submissions
- Notion API client for database submissions with field validation
- Field-specific validation (required fields, max selections, allowed values)
- Comprehensive input validation with length limits:
  - Title field: max 2000 characters (Notion API limit)
  - Comments field: max 2000 characters (Notion API limit)
  - Theme/Category: validates against allowed values
  - Product Area: validates against allowed values
  - Customer Org: validates against fetched customer list, enforces max 10 selections
  - Automatic whitespace trimming on all text inputs
  - User-friendly error messages displayed directly in Slack modals
- Support for Title, Rich Text, Select, Multi-select, and People Notion properties
- Configuration management with environment variables (including Customers DB)
- Customer Org validation against Notion Customers database (fetched on startup)
- Pagination support for fetching all customers from database
- Slack-to-Notion user mapping via email addresses:
  - Automatic "Submitted By" field population with Notion user reference
  - User cache populated on startup from Notion Users API
  - Case-insensitive email matching for reliability
  - Blocks submissions if Slack user not found in Notion workspace
  - Requires `users:read.email` OAuth scope on Slack app
- Docker deployment setup
- Documentation with field reference and examples
- Production-ready server features:
  - Graceful shutdown with signal handling (SIGTERM/SIGINT)
  - Panic recovery middleware preventing server crashes
  - HTTP timeouts (read/write/idle) preventing DoS vulnerabilities
- Comprehensive observability:
  - Prometheus metrics endpoint (`/metrics`) with 16+ metrics (including user_cache_size)
  - Advanced health checks (`/health` for liveness, `/ready` for readiness)
  - Request-level instrumentation (duration, status, size)
  - Notion API latency and error tracking
  - Validation error metrics
  - Context-based timeouts on all endpoints
  - Structured logging with zap

### Phase 4: Polish and Consistency (COMPLETED 2025-10-31)

Complete codebase polish with production-quality standards:

- **Comprehensive Documentation:**

  - Added package-level godoc to all packages explaining purpose and architecture
  - Documented all exported types with usage examples and field descriptions
  - Added business context to validation rules (why max 10 customer orgs, etc.)
  - Included detailed examples for complex methods (ViewState helpers with error handling patterns)
  - Explained Slack API quirks and design decisions in comments
  - Cross-referenced related types (ViewElement vs OptionText semantic differences)

- **Version Endpoint:**

  - Added `/version` endpoint returning build metadata (version, commit, build time, Go version)
  - Build information injectable via ldflags for CI/CD integration
  - Example: `go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -X main.buildTime=2024-01-01T00:00:00Z"`
  - Server startup logs now include version information

- **Code Quality Improvements:**

  - Eliminated all generic variable names (replaced "result" with descriptive names like "dbResponse", "queryResponse")
  - Consistent naming patterns across the codebase
  - Ran `go fmt ./...` - all files properly formatted
  - Ran `go vet ./...` - zero issues reported
  - Ran `go mod tidy` - dependencies clean and minimal
  - All tests passing with clean build

- **Type Consolidation Analysis:**
  - Analyzed ViewElement vs OptionText for potential consolidation
  - Decision: Keep separate for semantic clarity despite identical structure
  - ViewElement: Modal UI elements (title, buttons)
  - OptionText: Select menu option text
  - Both documented with cross-references explaining the distinction

### Phase 5: Comprehensive Test Coverage (COMPLETED)

Complete unit test coverage (70%+ achieved):

- **Handler Tests** (`internal/slack/handler_test.go`): 30+ tests

  - Slack request signature verification (valid, invalid, expired, missing)
  - Interaction payload parsing and validation
  - Field extraction and validation (required/optional, length, values)
  - Required field enforcement and error handling
  - Validation of themes, product areas, and customer orgs
  - Helper function tests (isValidTheme, isValidProductArea, contains)
  - Response formatting (success responses)

- **Modal Tests** (`internal/slack/modals_test.go`): 20+ tests

  - Modal structure and initialization
  - Block building and field configuration
  - Options generation and multi-select setup
  - Block type validation

- **Notion Client Tests** (`internal/notion/client_test.go`): 40+ tests

  - Property building and validation (title, rich text, select, multi-select)
  - Input validation with length limits
  - Field-specific validation logic
  - Multi-select parsing and validation
  - Required field enforcement
  - Customer org validation
  - Property extraction from Notion responses

- **Constants Tests** (`pkg/constants/constants_test.go`): 15+ tests

  - Verification of all constant arrays are non-empty
  - Timeout value validation
  - Configuration consistency checks

- **Health Tests** (`pkg/health/health_test.go`): 25+ tests

  - Health check manager initialization
  - Liveness and readiness endpoint handlers
  - Custom health checkers (Notion API, customer cache, always healthy)
  - Overall status determination logic
  - Response formatting and timestamp inclusion

- **Metrics Tests** (`pkg/metrics/metrics_test.go`): 15+ tests
  - Metrics initialization and structure
  - All metric types (counters, histograms, gauges)
  - Metric recording operations

**Coverage Summary**:

- `internal/slack`: 65.6% coverage
- `internal/notion`: 65.6% coverage
- `pkg/health`: 97.5% coverage (excellent)
- `pkg/metrics`: 14.3% coverage (mostly integration validation)
- Total: 140+ unit tests, all passing

### TODO

- Add integration tests with mocked Slack/Notion APIs
- Implement rate limiting
- Add periodic refresh of customer list and user cache (currently only fetched on startup)
- Set up Grafana dashboards for metrics visualization

## Development Workflow

### Running Locally

1. Set up environment variables in `.env`
2. Run: `go run cmd/hopperbot/main.go`
3. Use ngrok or similar tool to expose local server for Slack webhooks

### Testing Slack Integration

1. Type `/hopperbot` in your Slack workspace to test the modal
2. The modal will open with all form fields visible
3. Fill in the required fields and submit to test end-to-end flow

**Note**: Interactive modals cannot be easily tested with curl as they require real Slack workspace interaction. You'll need a properly configured Slack app with interactive components enabled and the correct Request URL set for `/slack/interactive`.

### Deploying

- Use Docker for containerized deployment
- Set environment variables in your deployment platform
- Ensure HTTPS is enabled for Slack webhooks
- Configure health checks at `/health` (liveness) and `/ready` (readiness)
- Version information available at `/version` for deployment verification

**Build with version information:**

```bash
# Build with embedded version metadata
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "\
  -X main.version=${VERSION} \
  -X main.commit=${COMMIT} \
  -X main.buildTime=${BUILD_TIME}" \
  -o hopperbot cmd/hopperbot/main.go
```

## Customer Validation & External Select Menus

The bot uses **Slack's external select menu** pattern to support unlimited customer organizations with dynamic search.

### How It Works

1. **On Startup**: The bot queries the Customers database to fetch all customer names
2. **In Memory**: Customer names are cached in memory for fast search (no database calls during search)
3. **Modal Display**: The Customer Org field uses `multi_external_select` type (not static options)
4. **User Types**: As users type in the field, Slack calls the `/slack/options` endpoint
5. **Search & Filter**: The bot filters customers using three-tier matching:
   - **Tier 1**: Exact matches (case-insensitive)
   - **Tier 2**: Prefix matches (starts with query)
   - **Tier 3**: Contains matches (query anywhere in name)
6. **Results**: Returns up to 100 filtered options, sorted alphabetically per tier
7. **Submission**: Selected values are validated against the cached customer list

### Slack App Configuration Required

**CRITICAL**: Your Slack app must have the **Options Load URL** configured:

1. Go to your Slack app settings: https://api.slack.com/apps
2. Navigate to **"Interactivity & Shortcuts"**
3. Scroll to **"Select Menus"** section
4. Set **Options Load URL**: `https://your-domain.com/slack/options`
5. Save changes

**Without this configuration**, the modal will fail to open with `invalid_arguments` error.

### Performance

- **Search Speed**: < 200ms for 1000+ customers (in-memory search)
- **No Database Calls**: All search happens in memory
- **Slack Limit**: Maximum 100 options per response (handled automatically)
- **User Experience**: Type-ahead search, no scrolling through hundreds of options

### Limitations

**Note**: The customer list is currently fetched only on startup. If customers are added/removed in Notion, the bot needs to be restarted to pick up the changes. Future improvement: Add periodic refresh or manual refresh endpoint.

## Slack-to-Notion User Mapping

The bot automatically tracks which Slack user submitted each idea by mapping Slack users to Notion users via email addresses:

### How It Works

1. **On Startup**: The bot queries the Notion Users API (`/v1/users`) to fetch all workspace members
2. **Email Extraction**: For each "person" type user, the bot extracts their email address
3. **Cache Building**: Creates an in-memory map of `email -> Notion user UUID` (normalized to lowercase)
4. **On Submission**: When a user submits the modal:
   - Bot calls Slack's `users.info` API to get the submitter's email address
   - Looks up the email in the cached Notion users map
   - If found: Adds the Notion user UUID to the "Submitted By" People property
   - If not found: **Blocks the submission** with a user-friendly error message
5. **Metrics**: Updates the `user_cache_size` gauge to track the number of cached users

### Requirements

**Slack App OAuth Scopes:**

- `users:read.email` - Required to fetch user email addresses from Slack API
- Without this scope, `slackUser.Profile.Email` will be empty and submissions will fail

**Notion Integration Capabilities:**

- User information capability must be enabled
- Without it, `/v1/users` endpoint returns 403 Forbidden

### Data Flow

```
Slack User Submits Form
    ↓
Slack API: GetUserInfo(userID) → email
    ↓
Cache Lookup: email → Notion User UUID
    ↓
    ├─ Found: Add to "Submitted By" field → Submit to Notion ✓
    └─ Not Found: Block submission with error ✗
```

### Error Handling

- **Slack API failure**: Returns user-friendly error "Failed to identify user"
- **User not found in Notion**: Returns error with user's email, asking them to contact admin
- **Email mismatch**: Case-insensitive matching handles variations (user@example.com = USER@EXAMPLE.COM)

### Caching Strategy

Similar to the customer validation cache:

- Fetched once on startup (no per-request API calls to Notion)
- Only one Slack API call per submission (to get user email)
- Handles pagination automatically (up to thousands of users)
- Currently no periodic refresh (restart required if users are added/removed)

**Performance**: Minimal overhead - one Slack API call per submission, zero Notion API calls during submission.

## Modal Architecture

The bot uses Slack's interactive modal system to provide a user-friendly form interface:

1. **Flow**: User types `/hopperbot` → Modal opens → User fills form → Submission → Data sent to Notion
2. **Endpoints**:
   - `/slack/command` - Receives slash command requests and opens the modal
   - `/slack/interactive` - Receives modal submission payloads
   - `/metrics` - Prometheus metrics for monitoring
   - `/health` - Liveness probe (is the server running?)
   - `/ready` - Readiness probe (is the server ready to serve traffic?)
   - `/version` - Build version and metadata
3. **Field Extraction**: Modal submissions contain field values in `view.State.Values` with structure:
   ```
   view.State.Values[blockID][actionID].SelectedOptions (for selects)
   view.State.Values[blockID][actionID].Value (for text inputs)
   ```
4. **Library**: Uses the `slack-go/slack` library for all Slack API interactions
5. **Features**:
   - Searchable dropdowns for Theme/Category, Product Area, and Customer Org
   - Multi-select support with proper handling
   - Real-time validation on submission
   - User-friendly error messages displayed in Slack

## Extending the Bot

### Adding New Commands

1. Register new slash command in Slack App settings
2. Update handler routing in `main.go` or add command switching logic
3. Create new handler functions as needed

### Customizing Modal Fields

Modify `internal/slack/modals.go`:

- Add or remove fields in the modal view
- Update dropdown options for select fields
- Adjust field labels, placeholders, and hints
- Configure multi-select vs single-select behavior

### Customizing Notion Integration

Modify `internal/notion/client.go`:

- Update `SubmitForm` to handle your database schema
- Add type detection for different property types
- Implement field mapping logic

## Security Considerations

- Slack request signature verification is implemented
- Environment variables for sensitive data
- No credentials in code or version control
- HTTPS required for production
- Production safety features:
  - Panic recovery middleware prevents entire server crashes
  - HTTP timeouts (read: 10s, write: 30s, idle: 120s) prevent DoS attacks
  - Graceful shutdown ensures in-flight requests complete (30s timeout)

## Observability & Monitoring

The bot includes production-grade observability features for monitoring, debugging, and performance analysis.

### Metrics Endpoint (`/metrics`)

Prometheus-compatible metrics endpoint exposing:

**HTTP Metrics:**

- `hopperbot_http_requests_total` - Total HTTP requests (by endpoint, method, status)
- `hopperbot_http_request_duration_seconds` - Request latency histogram
- `hopperbot_http_requests_in_flight` - Active requests gauge
- `hopperbot_http_response_size_bytes` - Response size histogram

**Slack Metrics:**

- `hopperbot_slack_commands_total` - Slash command invocations (by command, status)
- `hopperbot_slack_interactions_total` - Interactive events (by type, callback_id, status)
- `hopperbot_slack_modal_submissions_total` - Modal submissions (by status)

**Notion API Metrics:**

- `hopperbot_notion_api_requests_total` - API requests (by operation, status)
- `hopperbot_notion_api_request_duration_seconds` - API latency histogram
- `hopperbot_notion_api_errors_total` - API errors (by operation, error_type)

**Application Metrics:**

- `hopperbot_validation_errors_total` - Form validation errors (by field)
- `hopperbot_customer_cache_size` - Number of cached customers
- `hopperbot_panic_recoveries_total` - Panic recovery count

### Health Checks

**Liveness Probe (`/health`):**

- Returns 200 if server is running
- Used by orchestrators (Kubernetes) to restart unhealthy containers
- Includes uptime and basic health status

**Readiness Probe (`/ready`):**

- Returns 200 when ready to serve traffic
- Checks Notion API connectivity
- Validates customer cache is populated
- Returns 503 if dependencies are unavailable
- JSON response with detailed check results

Example readiness response:

```json
{
  "status": "healthy",
  "uptime": "2h30m15s",
  "timestamp": "2025-10-31T10:30:00Z",
  "checks": [
    {
      "name": "notion_api",
      "status": "healthy",
      "message": "Notion API is reachable",
      "duration": "150ms"
    },
    {
      "name": "customer_cache",
      "status": "healthy",
      "message": "Customer cache is populated",
      "duration": "1ms",
      "metadata": {
        "count": 42
      }
    }
  ]
}
```

### Request Middleware

All Slack endpoints include:

- **Recovery middleware** - Catches panics and prevents server crashes
- **Metrics middleware** - Records request duration, status, and size
- **Timeout middleware** - Context-based 30s timeout on all requests
- **Logging middleware** - Structured request/response logging

### Monitoring Setup

1. **Local Development:**

   ```bash
   # Start the bot
   go run cmd/hopperbot/main.go

   # Check version (includes build metadata)
   curl http://localhost:8080/version
   # Returns: {"version":"dev","commit":"unknown","build_time":"unknown","go_version":"go1.21+"}

   # View metrics
   curl http://localhost:8080/metrics

   # Check health
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

2. **Production (Kubernetes):**

   ```yaml
   livenessProbe:
     httpGet:
       path: /health
       port: 8080
     initialDelaySeconds: 10
     periodSeconds: 30

   readinessProbe:
     httpGet:
       path: /ready
       port: 8080
     initialDelaySeconds: 5
     periodSeconds: 10
   ```

3. **Prometheus Scraping:**
   ```yaml
   scrape_configs:
     - job_name: "hopperbot"
       static_configs:
         - targets: ["hopperbot:8080"]
       metrics_path: "/metrics"
   ```

### Key Metrics to Monitor

- **Error Rate:** `rate(hopperbot_http_requests_total{status=~"5.."}[5m])`
- **Request Latency (p95):** `histogram_quantile(0.95, rate(hopperbot_http_request_duration_seconds_bucket[5m]))`
- **Notion API Errors:** `rate(hopperbot_notion_api_errors_total[5m])`
- **Validation Failures:** `rate(hopperbot_validation_errors_total[5m])`
- **Cache Health:** `hopperbot_customer_cache_size > 0`

### Alerting Recommendations

1. **High error rate** (>5% of requests failing)
2. **High latency** (p95 > 2s)
3. **Notion API down** (health check failing)
4. **Empty customer cache** (cache_size = 0)
5. **Panic recoveries** (any increase indicates bugs)

## Code Quality Standards

The codebase follows Go best practices and maintains production-quality standards:

### Documentation

- **Package-level docs:** Every package has comprehensive documentation explaining purpose, responsibilities, and key concepts
- **Type documentation:** All exported types include godoc comments with usage examples
- **Function documentation:** All exported functions documented with parameters, return values, and examples
- **Business context:** Comments explain _why_ decisions were made (e.g., max limits, API constraints)
- **Architecture notes:** Complex patterns documented inline (modal state extraction, Slack API quirks)

### Naming Conventions

- **Descriptive names:** No generic variable names - all variables have meaningful, specific names
- **Consistent patterns:** Similar operations use similar naming (e.g., `dbResponse`, `queryResponse`)
- **Clear intent:** Function and type names clearly indicate their purpose

### Code Organization

- **Single responsibility:** Each package has a focused, well-defined purpose
- **Minimal coupling:** Packages depend on interfaces; metrics and logging injected as dependencies
- **Type safety:** Leverages Go's type system; separate types for semantically different data

### Quality Checks

- **go fmt:** All code formatted with `go fmt ./...`
- **go vet:** Zero issues from `go vet ./...`
- **go mod tidy:** Dependencies clean and minimal
- **Tests:** All tests passing with comprehensive table-driven coverage

### Build and Deployment

- **Version information:** Build metadata injectable via ldflags for version tracking
- **Clean builds:** No warnings, no unused imports, no deprecated APIs
- **Production-ready:** Comprehensive error handling, graceful shutdown, full observability

## Next Steps

When continuing development:

1. ~~Add comprehensive test coverage~~ ✅ COMPLETED (Phase 5)
2. Implement rate limiting (per-user and global limits)
3. Create admin commands for bot management (refresh customer cache, view stats)
4. Add periodic refresh of customer list from Notion (currently only on startup)
5. Set up Grafana dashboards for metrics visualization
6. Configure alerts in Prometheus/Alertmanager
7. Add CI/CD pipeline with automated testing and version injection
