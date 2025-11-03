// Package constants defines all constant values used throughout the application.
//
// This package centralizes:
// - Notion database field names and aliases
// - Valid values for select fields (themes, product areas)
// - Business rule limits (max selections, character limits)
// - Time-based constraints and timeouts
// - API configuration (Notion API version, endpoints, page size)
//
// Centralizing constants here ensures consistency across the application
// and makes it easy to adjust business rules without touching core logic.
package constants

import "time"

// Field names used in Notion database.
// These must match the exact column names in the Notion database schema.
const (
	FieldIdeaTopic     = "Idea/Topic"
	FieldThemeCategory = "Theme/Category"
	FieldProductArea   = "Product Area"
	FieldComments      = "Comments"
	FieldCustomerOrg   = "Customer Org"
	FieldSubmittedBy   = "Submitted By"
)

// Field aliases for title field.
// Allows flexible input from different sources (form fields, API calls).
const (
	AliasTitle = "title"
	AliasIdea  = "idea"
	AliasTopic = "topic"
)

// Field aliases for theme/category field
const (
	AliasTheme    = "theme"
	AliasCategory = "category"
)

// Field aliases for product area field
const (
	AliasProductArea = "product_area"
	AliasArea        = "area"
)

// Field aliases for comments field
const (
	AliasComments = "comments"
	AliasComment  = "comment"
)

// Field aliases for customer org field
const (
	AliasCustomerOrg = "customer_org"
	AliasCustomer    = "customer"
	AliasOrg         = "org"
)

// Field aliases for submitted by field
const (
	AliasSubmittedBy = "submitted_by"
)

// ValidThemeCategories defines the allowed values for the Theme/Category field.
//
// These categories help classify ideas into distinct types:
// - "New Feature Idea": Completely new functionality
// - "Feature Improvement": Enhancement to existing features
// - "Market/Competition Intelligence": Competitive insights or market trends
// - "Customer Pain Point": Issues or frustrations reported by customers
//
// Users must select exactly one theme per submission.
var ValidThemeCategories = []string{
	"New Feature Idea",
	"Feature Improvement",
	"Market/Competition Intelligence",
	"Customer Pain Point",
}

// ValidProductAreas defines the allowed values for the Product Area field.
//
// Represents the different product areas within the organization.
// Users must select exactly one product area per submission.
//
// Areas cover the full product portfolio from AI/ML to warehouse ingestion.
var ValidProductAreas = []string{
	"AI/ML",
	"Integrations/SDKs",
	"Data Governance",
	"Systems",
	"UX",
	"Activation Kits",
	"Activation Core",
	"rETL",
	"Transformations",
	"EventStream",
	"WH Ingestion",
}

// Selection limits enforce business rules on multi-select fields.
const (
	// MaxCustomerOrgSelections limits customer org selections to 10.
	// Rationale: Reasonable upper bound for multi-tenant features while
	// preventing abuse. Most ideas relate to fewer than 10 customers.
	MaxCustomerOrgSelections = 10

	// MaxOptionsResults limits the number of options returned in external select menus.
	// Rationale: Slack recommends limiting to 100 options for good UX and performance.
	// Users can narrow results by typing more specific search queries.
	MaxOptionsResults = 100
)

// Input length limits are based on Notion API constraints.
const (
	// MaxTitleLength is the maximum character limit for title fields.
	// Notion enforces a 2000 character limit on title properties.
	MaxTitleLength = 2000

	// MaxCommentLength is the maximum character limit for rich text fields.
	// Notion enforces a 2000 character limit on rich text properties.
	MaxCommentLength = 2000
)

// Time-based security limits.
const (
	// MaxSlackRequestAge is the maximum age of a Slack request signature.
	// Requests older than this are rejected to prevent replay attacks.
	// Slack recommends 5 minutes as a reasonable window.
	MaxSlackRequestAge = 300 // seconds (5 minutes)
)

// Timeouts for various operations.
const (
	// DefaultHTTPTimeout is the default timeout for HTTP clients.
	// Used for Notion API calls.
	DefaultHTTPTimeout = 30 * time.Second

	// ServerReadTimeout is the maximum duration for reading the entire request.
	// Prevents slow client attacks.
	ServerReadTimeout = 10 * time.Second

	// ServerWriteTimeout is the maximum duration before timing out writes.
	// Allows time for Notion API calls and response generation.
	ServerWriteTimeout = 30 * time.Second

	// ServerIdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled.
	ServerIdleTimeout = 120 * time.Second

	// GracefulShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Allows in-flight requests to complete before forcing shutdown.
	GracefulShutdownTimeout = 30 * time.Second
)

// Notion API configuration constants.
const (
	// NotionPageSize is the number of items to fetch per page.
	// Notion's maximum is 100 items per page.
	NotionPageSize = 100

	// NotionAPIVersion specifies which version of the Notion API to use.
	// Using a fixed version ensures consistent behavior.
	NotionAPIVersion = "2022-06-28"

	// NotionAPIBaseURL is the base URL for all Notion API requests.
	NotionAPIBaseURL = "https://api.notion.com/v1"
)

// Default configuration values.
const (
	// DefaultPort is the default HTTP server port.
	DefaultPort = "8080"
)
