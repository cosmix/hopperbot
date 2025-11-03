# Hopperbot

A Slack bot that provides an interactive modal form for submitting data to Notion databases.

## Features

- Interactive modal interface for easy form submission
- Searchable dropdowns for selecting customers, themes, and product areas
- Real-time validation of form inputs
- Seamless integration with Notion databases
- Secure request verification for Slack interactions

## Setup

### Prerequisites

- Go 1.21 or higher
- A Slack workspace with admin access
- A Notion workspace with API access

### 1. Slack App Configuration

#### Step 1: Create a New Slack App

1. Navigate to [api.slack.com/apps](https://api.slack.com/apps) in your browser
2. Click the **"Create New App"** button
3. Choose **"From scratch"**
4. Enter an **App Name** (e.g., "Hopperbot")
5. Select your **Slack Workspace** from the dropdown
6. Click **"Create App"**

#### Step 2: Configure Slash Commands

1. In your app's settings page, click on **"Slash Commands"** in the left sidebar (under "Features")
2. Click **"Create New Command"**
3. Fill in the command details:
   - **Command**: `/hopperbot`
   - **Request URL**: `https://your-domain.com/slack/command`
     - ⚠️ This must be a publicly accessible HTTPS URL
     - For local development, use [ngrok](https://ngrok.com/) or similar tunneling service
   - **Short Description**: "Submit ideas and feedback to Notion"
   - **Usage Hint**: (leave empty)
4. Check the box **"Escape channels, users, and links sent to your app"**
5. Click **"Save"**

#### Step 3: Enable Interactivity for Modals

1. Click on **"Interactivity & Shortcuts"** in the left sidebar (under "Features")
2. Toggle **"Interactivity"** to **ON**
3. Set the **Request URL**: `https://your-domain.com/slack/interactive`
   - ⚠️ This must be the same base domain as your slash command URL
   - This endpoint handles modal submissions and interactive components
4. Scroll down to the **"Select Menus"** section
5. Set the **Options Load URL**: `https://your-domain.com/slack/options`
   - ⚠️ **REQUIRED** for customer organization search functionality
   - This endpoint provides dynamic options as users type in the customer org field
   - Without this, the modal will fail to open with "invalid_arguments" error
6. Click **"Save Changes"** at the bottom of the page

#### Step 4: Configure OAuth Scopes

1. Click on **"OAuth & Permissions"** in the left sidebar (under "Features")
2. Scroll down to the **"Scopes"** section
3. Under **"Bot Token Scopes"**, click **"Add an OAuth Scope"** and add:
   - `commands` - Allows your app to add slash commands
   - `users:read.email` - **Required** to map Slack users to Notion users by email
     - ⚠️ Without this scope, submissions will fail with "user not found" errors
4. Scroll up and click **"Install to Workspace"** (or "Reinstall to Workspace" if already installed)
5. Review the permissions and click **"Allow"**

#### Step 5: Retrieve Your Bot Token

1. After installation, you'll be redirected to the **"OAuth & Permissions"** page
2. At the top, find the **"Bot User OAuth Token"**
3. Click **"Copy"** to copy the token (it starts with `xoxb-`)
4. **Save this token securely** - you'll add it to your `.env` file as `SLACK_BOT_TOKEN`

#### Step 6: Retrieve Your Signing Secret

1. Click on **"Basic Information"** in the left sidebar
2. Scroll down to the **"App Credentials"** section
3. Find **"Signing Secret"** and click **"Show"**
4. Click **"Copy"** to copy the secret
5. **Save this secret securely** - you'll add it to your `.env` file as `SLACK_SIGNING_SECRET`

**Security Note**: Never commit your Bot Token or Signing Secret to version control. Always use environment variables or a secrets manager.

### 2. Notion Integration

#### Step 1: Create a Notion Integration

1. Navigate to [notion.so/my-integrations](https://www.notion.so/my-integrations) in your browser
   - ⚠️ You must be a **workspace owner** to create integrations
2. Click **"+ New integration"**
3. Fill in the integration details:
   - **Name**: "Hopperbot" (or your preferred name)
   - **Associated workspace**: Select your workspace
   - **Type**: Choose **"Internal integration"** (for use within your workspace only)
4. Click **"Submit"** to create the integration
5. You'll be redirected to the integration's settings page

#### Step 2: Retrieve Your Integration Token

1. On the integration settings page, go to the **"Secrets"** tab (or "Configuration" tab)
2. Find the **"Internal Integration Token"** (also called "Integration Secret")
3. Click **"Show"** then **"Copy"** to copy the token (starts with `secret_`)
4. **Save this token securely** - you'll add it to your `.env` file as `NOTION_API_KEY`

**Security Warning**: Never store this token in your source code or commit it to version control. Treat it like a password.

#### Step 3: Set Up Your Notion Databases

You need to create or identify two databases in Notion:

**A. Main Submissions Database** (where form submissions will be stored)

This database should have the following properties:

- **Idea/Topic** (Title property) - The main title field
- **Theme/Category** (Select) - With options: "New feature idea", "Feature improvement", "Market/competition intelligence", "Customer pain point"
- **Product Area** (Select) - With options: "AI/ML", "Integrations/SDKs", "Data Governance", "Systems", "UX", "Activation Kits", "Activation", "rETL", "Transformations", "EventStream", "WH Ingestion"
- **Comments** (Text or Rich text)
- **Customer Org** (Multi-select or Relation)
- **Submitted By** (Person property) - Will be automatically populated

**B. Customers Database** (list of customer organizations)

This database should have:

- **Name** or **Customer** (Title property) - The customer/organization name
- Add all your customer organizations as individual pages in this database

#### Step 4: Share Databases with Your Integration

**Important**: Integrations cannot access your Notion content by default. You must explicitly share each database.

**For the Main Submissions Database:**

1. Open the database in Notion (as a full page, not inline)
2. Click the **"..."** menu in the top-right corner
3. Scroll down and click **"Add connections"** (or "Connections")
4. Search for your integration name (e.g., "Hopperbot")
5. Click on your integration to grant it access
6. You should see a confirmation that the integration has been added

**For the Customers Database:**

1. Repeat the same steps above for your Customers database
2. Ensure your integration appears in the connections list

**Troubleshooting**: If you don't see the "Add connections" option, make sure you've opened the database as a full page (not an inline view).

#### Step 5: Retrieve Database IDs from URLs

**Understanding the URL Format:**

Notion database URLs look like this:

```
https://www.notion.so/29e1443a6500808ebb9bf38da8219096?v=29e1443a650080b4baa0000c541906da
                      └────────────────┬──────────────────┘
                                 Database ID (32 characters)
```

**To Find Your Database IDs:**

1. Open your **Main Submissions Database** as a full page in Notion
2. Look at the URL in your browser's address bar
3. The database ID is the **32-character hexadecimal string** immediately after `notion.so/` and before the `?v=` parameter
   - Example from URL above: `29e1443a6500808ebb9bf38da8219096`
   - It's always 32 characters without hyphens (or sometimes 36 characters with hyphens in UUID format)
   - Either format works with the Notion API
4. **Copy this ID** - you'll add it to your `.env` file as `NOTION_DATABASE_ID`

5. Repeat the same process for your **Customers Database**
6. **Copy that ID** - you'll add it to your `.env` file as `NOTION_CUSTOMERS_DB_ID`

**Tips:**

- If you have an inline database, click the "Open as full page" icon in the top-right of the database first
- The database ID remains the same even if you change the database name or view
- You can verify you have the correct ID by using it in a Notion API call (the bot will validate this on startup)

### 3. Environment Configuration

#### Create Your Configuration File

1. In the project root directory, copy the example environment file:

   ```bash
   cp .env.example .env
   ```

2. Open `.env` in your text editor and fill in your credentials:

   ```bash
   # Slack Configuration
   # Get these from https://api.slack.com/apps → Your App → Basic Information
   SLACK_SIGNING_SECRET=your_slack_signing_secret_here
   SLACK_BOT_TOKEN=xoxb-your-bot-token-here

   # Notion Configuration
   # Get these from https://www.notion.so/my-integrations
   NOTION_API_KEY=secret_your_notion_integration_key_here
   NOTION_DATABASE_ID=your_main_database_id_here
   NOTION_CUSTOMERS_DB_ID=your_customers_database_id_here

   # Server Configuration
   PORT=8080
   ```

3. Replace each placeholder with the actual values you copied from the previous steps:

   - `SLACK_SIGNING_SECRET`: From Slack App → Basic Information → App Credentials
   - `SLACK_BOT_TOKEN`: From Slack App → OAuth & Permissions (starts with `xoxb-`)
   - `NOTION_API_KEY`: From Notion Integration settings (starts with `secret_`)
   - `NOTION_DATABASE_ID`: The 32-char ID from your main submissions database URL
   - `NOTION_CUSTOMERS_DB_ID`: The 32-char ID from your customers database URL

4. **Verify your `.env` file is in `.gitignore`** to prevent accidentally committing secrets

#### Configuration Checklist

Before proceeding, verify you have:

- ✅ Slack Signing Secret (looks like a long random string)
- ✅ Slack Bot Token (starts with `xoxb-`)
- ✅ Notion API Key (starts with `secret_`)
- ✅ Main Database ID (32 alphanumeric characters)
- ✅ Customers Database ID (32 alphanumeric characters)
- ✅ Both databases shared with your Notion integration
- ✅ Slack app has `commands` and `users:read.email` OAuth scopes

### 4. Build and Run

```bash
# Install dependencies
go mod tidy

# Build the application
go build -o hopperbot cmd/hopperbot/main.go

# Run the application
./hopperbot
```

Or run directly:

```bash
go run cmd/hopperbot/main.go
```

## Usage

Once the bot is running and configured in Slack, simply type `/hopperbot` in any Slack channel to open an interactive form.

### How It Works

1. Type `/hopperbot` in any Slack channel
2. An interactive modal form will appear with the following fields
3. Fill out the form and click Submit
4. The bot validates your input in real-time and submits to Notion

### Form Fields

The modal provides the following fields:

**Required Fields:**

1. **Idea/Topic** (Text input)

   - The title or main idea you want to submit
   - Example: "Add dark mode support"

2. **Theme/Category** (Single-select dropdown with search)

   - Available options:
     - New feature idea
     - Feature improvement
     - Market/competition intelligence
     - Customer pain point
   - Type to search and select one theme

3. **Product Area** (Single-select dropdown with search)
   - Available options:
     - AI/ML
     - Integrations/SDKs
     - Data Governance
     - Systems
     - UX
     - Activation Kits
     - Activation
     - rETL
     - Transformations
     - EventStream
     - WH Ingestion
   - Type to search and select one product area

**Optional Fields:**

1. **Comments** (Text input)

   - Additional context or notes about your submission
   - Example: "Requested by multiple customers in Q4"

2. **Customer Org** (Multi-select dropdown with search, max 10 selections)
   - Searchable list of customers from your Notion Customers database
   - Type to search and select up to 10 customer organizations
   - The list is automatically synced from Notion on bot startup

### Usage Examples

**Basic workflow:**

1. Type `/hopperbot` in a Slack channel
2. A form modal appears
3. Enter "Improve API response times" in the Idea/Topic field
4. Select "Feature improvement" from Theme dropdown
5. Select "Integrations/SDKs" from Product Area dropdown
6. Click Submit

**With optional fields:**

1. Type `/hopperbot` in a Slack channel
2. Fill in required fields (Idea, Theme, Product Area)
3. Add comments: "Critical for enterprise customers"
4. Search and select customer orgs: "Acme Corp", "TechStart Inc"
5. Click Submit

**The modal provides easy selection:**

- Searchable dropdowns make it easy to find options
- Multi-select fields show selected items with removal buttons
- All required fields are clearly marked
- Submit button activates only when required fields are filled

### Validation Rules

The modal validates your input in real-time:

- **Idea/Topic**: Required, cannot be empty
- **Theme/Category**: Required, must select exactly 1 option
- **Product Area**: Required, must select exactly 1 option
- **Comments**: Optional, free text
- **Customer Org**: Optional, can select up to 10 organizations

The bot will:

1. Fetch valid customer names from the Customers database on startup
2. Display an interactive modal when `/hopperbot` is invoked
3. Validate form inputs in real-time as you type and select
4. Submit validated data to your Notion database with proper field types
5. Show a success or error message after submission

## Project Structure

```
hopperbot/
├── cmd/
│   └── hopperbot/
│       └── main.go           # Application entry point
├── internal/
│   ├── notion/
│   │   └── client.go         # Notion API client
│   ├── parser/
│   │   └── parser.go         # Command text parser
│   └── slack/
│       └── handler.go        # Slack slash command handler
├── pkg/
│   └── config/
│       └── config.go         # Configuration management
├── .env.example              # Example environment variables
├── .gitignore
├── go.mod
└── README.md
```

## Customization

### Adding New Field Mappings

The bot is pre-configured with field mappings for the standard database schema. To add new fields or aliases:

1. Edit `internal/notion/client.go`
2. Add a new case in the `SubmitForm` switch statement
3. Map to the appropriate Notion property type

Example for adding a new text field:

```go
case "new_field", "alias":
    properties["New Field Name"] = Property{
        RichText: []RichText{
            {Text: Text{Content: value}},
        },
    }
```

### Supported Property Types

- **Title**: Main title of the database entry
- **Rich Text**: Multi-line text fields
- **Select**: Single choice from dropdown
- **Multi-Select**: Multiple choices (comma-separated in command)
- **Date**: ISO 8601 date format (YYYY-MM-DD)
- **Person**: Notion user ID
- **Relation**: Related page IDs (comma-separated)

## Deployment

For production deployment, consider:

1. Use a process manager or container orchestration
2. Set up HTTPS with a valid certificate
3. Use environment variables for sensitive data
4. Set up logging and monitoring
5. Configure health checks at `/health`

Example Docker deployment:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o hopperbot cmd/hopperbot/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/hopperbot .
EXPOSE 8080
CMD ["./hopperbot"]
```

## Observability and Monitoring

Hopperbot includes production-grade observability features following modern monitoring, alerting, and debugging best practices. The implementation provides comprehensive visibility into application health, performance, and operational metrics.

**Key Capabilities:**
- Prometheus metrics for performance and business metrics
- Health checks for liveness and readiness probes
- Request-level instrumentation with middleware
- Distributed system monitoring support
- Production-ready alerting templates

### Metrics Overview

A complete metrics package with 16 Prometheus metrics covering all aspects of the application:

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

### Observability Endpoints

**Endpoints:**
- `/metrics` - Prometheus metrics endpoint
- `/health` - Liveness probe (is the server running?)
- `/ready` - Readiness probe (can we serve traffic?)
- `/version` - Build version and metadata

**Health Check Example:**

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
    }
  ]
}
```

### Prometheus Setup

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

### Kubernetes Health Probes

```yaml
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

### Key Metrics to Monitor

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

      - alert: PanicDetected
        expr: increase(hopperbot_panic_recoveries_total[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Application panic detected"
          description: "{{ $value }} panics recovered in the last 5 minutes"
```

### Grafana Dashboard Panels

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

## Development

Run tests:

```bash
go test ./...
```

Format code:

```bash
go fmt ./...
```

Lint code:

```bash
golangci-lint run
```

## Troubleshooting

### Common Setup Issues

#### 1. "Invalid Slack signature" Error

**Symptoms**: The bot returns a 401 Unauthorized error when you invoke `/hopperbot`

**Causes & Solutions**:

- ❌ Wrong signing secret in `.env` file
  - ✅ Go to Slack App → Basic Information → App Credentials and copy the correct Signing Secret
- ❌ Request is taking too long (> 5 minutes old)
  - ✅ Slack signatures expire after 5 minutes; check your server's clock is synchronized
- ❌ Proxy or load balancer modifying the request
  - ✅ Ensure your infrastructure passes the raw request body unmodified

#### 2. "Notion API Error: object not found" or 403 Forbidden

**Symptoms**: Bot starts but fails when fetching customers or submitting forms

**Causes & Solutions**:

- ❌ Database not shared with the integration
  - ✅ Open the database in Notion → Click "..." → Add connections → Select your integration
- ❌ Wrong database ID in `.env`
  - ✅ Verify the 32-character ID from the database URL (between workspace name and `?v=`)
- ❌ Integration doesn't have the right capabilities
  - ✅ Go to Notion Integration settings → Ensure "Read content", "Update content", and "Insert content" are enabled

#### 3. "User not found in Notion workspace" Error

**Symptoms**: Modal submission fails with user mapping error

**Causes & Solutions**:

- ❌ Missing `users:read.email` OAuth scope on Slack app
  - ✅ Go to Slack App → OAuth & Permissions → Bot Token Scopes → Add `users:read.email`
  - ✅ Reinstall the app to your workspace after adding the scope
- ❌ Slack user's email doesn't match any Notion workspace member
  - ✅ Check the Notion workspace members at Settings & Members → ensure the user is added
  - ✅ Verify their email address matches between Slack and Notion
- ❌ User cache not populated on startup
  - ✅ Check the bot logs for "Initialized user cache with X users"
  - ✅ Restart the bot if the cache is empty

#### 4. Modal Doesn't Appear When Using `/hopperbot`

**Symptoms**: You type `/hopperbot` but no modal opens

**Causes & Solutions**:

- ❌ Slash command not configured correctly
  - ✅ Go to Slack App → Slash Commands → Ensure Request URL is correct and accessible
- ❌ Server not responding within 3 seconds
  - ✅ Check your server logs for errors
  - ✅ Verify the server is running and reachable at the Request URL
- ❌ Invalid response format from server
  - ✅ Check logs for JSON formatting errors in the modal response

#### 5. "Failed to submit to Notion" Error

**Symptoms**: Modal validation passes but submission fails

**Causes & Solutions**:

- ❌ Notion database schema doesn't match expected format
  - ✅ Verify your database has all required properties with correct types:
    - Title field (any name)
    - Theme/Category (Select)
    - Product Area (Select)
    - Submitted By (Person)
- ❌ Property names don't match (case-sensitive)
  - ✅ Check that property names in Notion exactly match what the bot expects
- ❌ Network connectivity issues to Notion API
  - ✅ Check `/ready` endpoint for Notion API connectivity status

#### 6. Empty Customer List in Modal

**Symptoms**: The "Customer Org" dropdown is empty or missing customers

**Causes & Solutions**:

- ❌ Customers database is empty
  - ✅ Add customer organizations as pages in your Customers database (each page = one customer)
- ❌ Customers database not shared with integration
  - ✅ Open Customers database → "..." → Add connections → Select your integration
- ❌ Wrong database ID for `NOTION_CUSTOMERS_DB_ID`
  - ✅ Verify you're using the Customers database ID, not the main database ID
- ❌ Customer list only fetched on startup
  - ✅ Restart the bot after adding new customers to the database

### Verifying Your Setup

Use these commands to test your configuration:

```bash
# Check if the server is running
curl http://localhost:8080/health

# Check if dependencies are ready (includes Notion connectivity)
curl http://localhost:8080/ready | jq

# View Prometheus metrics (includes cache sizes)
curl http://localhost:8080/metrics | grep hopperbot_customer_cache_size

# Check server version
curl http://localhost:8080/version
```

**Healthy Output Example**:

```json
{
  "status": "healthy",
  "uptime": "30s",
  "checks": [
    {
      "name": "notion_api",
      "status": "healthy",
      "message": "Notion API is reachable"
    },
    {
      "name": "customer_cache",
      "status": "healthy",
      "message": "Customer cache is populated",
      "metadata": { "count": 42 }
    }
  ]
}
```

### Getting Help

If you continue experiencing issues:

1. **Check the logs**: Run the bot with verbose logging and look for error messages
2. **Test endpoints individually**: Use `curl` to test `/health`, `/ready`, and `/metrics`
3. **Verify credentials**: Double-check all tokens, secrets, and IDs are correctly copied
4. **Review Slack app settings**: Ensure all configurations match the setup guide
5. **Check Notion integration**: Verify databases are shared and properties match expected schema

For local development issues with HTTPS requirements, use a tunneling service like [ngrok](https://ngrok.com/):

```bash
ngrok http 8080
# Use the HTTPS URL (e.g., https://abc123.ngrok.io) in your Slack app configuration
```

## License

[Add your license here]
