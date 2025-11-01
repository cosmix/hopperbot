// Package notion provides a client for interacting with the Notion API.
//
// This package handles:
// - Creating entries in Notion databases with proper type mapping
// - Fetching and caching valid customer organization names
// - Validating field values against business rules and Notion constraints
// - Converting form fields to Notion property types (Title, RichText, Select, MultiSelect)
//
// The client manages two Notion databases:
//  1. Main database: Stores submitted ideas/topics with all form fields
//  2. Customers database: Reference database containing valid customer organization names
//
// Field validation includes:
// - Length limits (2000 chars for title and comments)
// - Multi-select limits (max 10 customer orgs)
// - Value validation against predefined lists or dynamically fetched customers
package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rudderlabs/hopperbot/pkg/constants"
	"github.com/rudderlabs/hopperbot/pkg/metrics"
	"go.uber.org/zap"
)

// Client manages interactions with the Notion API including database operations
// and caching of valid customer organization names and workspace users.
//
// The client maintains two in-memory caches:
// 1. customerMap: Mapping of customer organization names to Notion page IDs (for relations)
// 2. validUsers: Mapping of email addresses to Notion user UUIDs
//
// Both caches are populated during initialization and used for validation
// and mapping in form submissions.
type Client struct {
	apiKey        string
	databaseID    string
	customersDBID string
	httpClient    *http.Client
	customerMap   map[string]string // Cached mapping of customer name -> Notion page ID
	validUsers    map[string]string // Cached mapping of email -> Notion user UUID
	logger        *zap.Logger
	metrics       *metrics.Metrics
}

// NewClient creates a new Notion API client configured with authentication and database IDs.
//
// Parameters:
// - apiKey: Notion integration secret (starts with "secret_")
// - databaseID: ID of the main database where ideas/topics are stored
// - customersDBID: ID of the Customers database containing valid customer organization names
// - logger: Zap logger for structured logging
//
// The client must call InitializeCustomers() and InitializeUsers() before accepting
// form submissions to populate the caches.
func NewClient(apiKey, databaseID, customersDBID string, logger *zap.Logger) *Client {
	return &Client{
		apiKey:        apiKey,
		databaseID:    databaseID,
		customersDBID: customersDBID,
		httpClient: &http.Client{
			Timeout: constants.DefaultHTTPTimeout,
		},
		customerMap: make(map[string]string),
		validUsers:  make(map[string]string),
		logger:      logger,
	}
}

// InitializeCustomers fetches the list of valid customer names and their page IDs from the Customers database.
//
// This method should be called during application startup before accepting requests.
// It queries the Customers database and extracts all customer organization names and their
// corresponding Notion page IDs to populate the in-memory cache used for validation and relations.
//
// The method handles pagination automatically to fetch all customers regardless of database size.
// Updates the client_cache_size metric upon successful initialization.
//
// Returns an error if the Notion API call fails or the response cannot be parsed.
func (c *Client) InitializeCustomers() error {
	start := time.Now()

	customerMap, err := c.fetchCustomersFromDatabase()
	c.recordNotionRequest("initialize_customers", start, err)

	if err != nil {
		return fmt.Errorf("failed to fetch customers: %w", err)
	}

	c.customerMap = customerMap

	// Update customer cache size metric
	if c.metrics != nil {
		c.metrics.ClientCacheSize.Set(float64(len(c.customerMap)))
	}

	return nil
}

// GetValidCustomers returns the list of valid customer names for dropdown options
func (c *Client) GetValidCustomers() []string {
	customerNames := make([]string, 0, len(c.customerMap))
	for name := range c.customerMap {
		customerNames = append(customerNames, name)
	}
	return customerNames
}

// InitializeUsers fetches all workspace users from Notion and builds the email-to-UUID mapping.
//
// This method should be called during application startup before accepting requests.
// It queries the Notion Users API to fetch all workspace members and extracts their
// email addresses to build an in-memory cache for Slack-to-Notion user mapping.
//
// The method handles pagination automatically to fetch all users regardless of workspace size.
// Updates the user_cache_size metric upon successful initialization.
//
// Returns an error if the Notion API call fails or the response cannot be parsed.
func (c *Client) InitializeUsers() error {
	start := time.Now()

	userMap, err := c.fetchUsersFromWorkspace()
	c.recordNotionRequest("initialize_users", start, err)

	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	c.validUsers = userMap

	// Update user cache size metric
	if c.metrics != nil {
		c.metrics.UserCacheSize.Set(float64(len(c.validUsers)))
	}

	// Log the loaded users (emails only, not UUIDs for brevity)
	emails := make([]string, 0, len(c.validUsers))
	for email := range c.validUsers {
		emails = append(emails, email)
	}

	c.logger.Info("initialized Notion users cache",
		zap.Int("count", len(c.validUsers)),
		zap.Strings("cached_emails", emails),
	)

	return nil
}

// GetNotionUserIDByEmail looks up a Notion user UUID by email address.
//
// Returns the Notion user UUID and true if found, or empty string and false if not found.
// The lookup is case-insensitive to handle email variations.
func (c *Client) GetNotionUserIDByEmail(email string) (string, bool) {
	// Normalize email to lowercase for case-insensitive lookup
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	userID, found := c.validUsers[normalizedEmail]
	return userID, found
}

// GetUserCacheSize returns the number of users in the cache.
func (c *Client) GetUserCacheSize() int {
	return len(c.validUsers)
}

// GetCachedUserEmails returns a list of all cached email addresses (for debugging).
// Returns emails in their normalized (lowercase) form as stored in the cache.
func (c *Client) GetCachedUserEmails() []string {
	emails := make([]string, 0, len(c.validUsers))
	for email := range c.validUsers {
		emails = append(emails, email)
	}
	return emails
}

// Property represents a Notion database property with its value.
//
// Notion supports different property types, each with its own structure:
// - Title: Text that serves as the page title (only one per database)
// - RichText: Formatted text content
// - Select: Single selection from predefined options
// - MultiSelect: Multiple selections from predefined options
// - People: References to Notion users (workspace members)
// - Relation: References to pages in another database
//
// Only one field should be populated based on the property type.
type Property struct {
	Title       []RichText     `json:"title,omitempty"`
	RichText    []RichText     `json:"rich_text,omitempty"`
	Select      *Select        `json:"select,omitempty"`
	MultiSelect []Select       `json:"multi_select,omitempty"`
	People      []NotionUser   `json:"people,omitempty"`
	Relation    []RelationPage `json:"relation,omitempty"`
}

// RichText represents formatted text content in Notion.
// Can contain styling, links, and other formatting options.
type RichText struct {
	Text Text `json:"text"`
}

// Text represents the plain text content within a RichText object.
type Text struct {
	Content string `json:"content"`
}

// Select represents a single selection option in Notion.
// Used for both Select and MultiSelect property types.
// The Name field must match a valid option defined in the database schema.
type Select struct {
	Name string `json:"name"`
}

// NotionUser represents a Notion user reference for People properties.
// Used to assign pages to workspace members or track who created content.
type NotionUser struct {
	Object string `json:"object"` // Always "user"
	ID     string `json:"id"`     // Notion user UUID
}

// RelationPage represents a reference to a page in another Notion database.
// Used for Relation properties to link between databases.
type RelationPage struct {
	ID string `json:"id"` // Notion page UUID
}

// CreatePageRequest represents a request to create a page in Notion.
//
// A page in Notion is created within a parent (database or page).
// Properties are mapped by their database column names to Property values.
type CreatePageRequest struct {
	Parent     Parent              `json:"parent"`
	Properties map[string]Property `json:"properties"`
}

// Parent identifies the parent container for a new Notion page.
// For database entries, DatabaseID specifies which database to create the page in.
type Parent struct {
	DatabaseID string `json:"database_id"`
}

// multiSelectConfig defines validation rules for multi-select fields.
//
// Used to enforce business rules on multi-select fields:
// - maxItems: Maximum number of selections allowed (e.g., max 10 customer orgs)
// - validValues: List of allowed values (empty means skip validation)
// - fieldName: Display name for error messages
//
// Why these limits exist:
// - Customer org limit (10): Reasonable upper bound for multi-tenant features
type multiSelectConfig struct {
	maxItems    int
	validValues []string
	fieldName   string
}

// validateMultiSelect validates multi-select items against configuration rules.
//
// Performs two types of validation:
// 1. Count validation: Ensures number of selections doesn't exceed maxItems
// 2. Value validation: Ensures each selected value exists in validValues list (if provided)
//
// Returns nil if validation passes, or a descriptive error if validation fails.
func validateMultiSelect(items []Select, config multiSelectConfig) error {
	if len(items) > config.maxItems {
		return fmt.Errorf("%s can have at most %d selections, got %d",
			config.fieldName, config.maxItems, len(items))
	}

	// If no valid values specified, skip value validation
	if len(config.validValues) == 0 {
		return nil
	}

	// Validate each item against the allowed values
	for _, item := range items {
		if !contains(config.validValues, item.Name) {
			return fmt.Errorf("invalid %s value: '%s' (must be one of: %s)",
				config.fieldName, item.Name, strings.Join(config.validValues, ", "))
		}
	}

	return nil
}

// validateAndTrimInput validates and trims input strings with length constraints.
//
// Performs the following validations:
// 1. Trims leading/trailing whitespace
// 2. Checks if the result is empty (for required fields at call site)
// 3. Validates length doesn't exceed maxLength
//
// Returns the trimmed value if valid, or an error with user-friendly message.
// Notion has strict limits: 2000 characters for title and rich text fields.
func validateAndTrimInput(value string, maxLength int, fieldName string) (string, error) {
	// Trim whitespace first
	trimmed := strings.TrimSpace(value)

	// Check if empty (for required field validation at call site)
	if trimmed == "" {
		return "", fmt.Errorf("%s cannot be empty", fieldName)
	}

	// Check length limit
	if len(trimmed) > maxLength {
		return "", fmt.Errorf("%s exceeds maximum length of %d characters (current: %d)",
			fieldName, maxLength, len(trimmed))
	}

	return trimmed, nil
}

// buildTitleProperty creates a title property with validation.
//
// Title properties are special in Notion - each database has exactly one title property
// that serves as the page name. This is mapped to the "Idea/Topic" field in our schema.
//
// Validates that the title is non-empty and within the 2000 character limit.
func buildTitleProperty(value string) (Property, error) {
	validated, err := validateAndTrimInput(value, constants.MaxTitleLength, "Title")
	if err != nil {
		return Property{}, err
	}

	return Property{
		Title: []RichText{{Text: Text{Content: validated}}},
	}, nil
}

// buildRichTextProperty creates a rich text property with validation.
//
// Rich text properties can contain formatted content. In our use case, we use them
// for the Comments field to allow users to provide additional context.
//
// Validates that the text is non-empty and within the 2000 character limit.
func buildRichTextProperty(value string, fieldName string) (Property, error) {
	validated, err := validateAndTrimInput(value, constants.MaxCommentLength, fieldName)
	if err != nil {
		return Property{}, err
	}

	return Property{
		RichText: []RichText{{Text: Text{Content: validated}}},
	}, nil
}

// buildSelectProperty creates and validates a select property.
//
// Select properties allow choosing a single option from a predefined list.
// Used for the Product Area and Theme/Category fields where users select exactly one option.
//
// Validates that:
// - The value is non-empty (after trimming whitespace)
// - The value exists in the validValues list (database schema options)
func buildSelectProperty(value string, validValues []string, fieldName string) (Property, error) {
	// Trim whitespace from the value
	trimmed := strings.TrimSpace(value)

	if trimmed == "" {
		return Property{}, fmt.Errorf("%s cannot be empty", fieldName)
	}

	if !contains(validValues, trimmed) {
		return Property{}, fmt.Errorf("invalid %s value: %s (must be one of: %s)",
			fieldName, trimmed, strings.Join(validValues, ", "))
	}
	return Property{
		Select: &Select{Name: trimmed},
	}, nil
}

// buildMultiSelectProperty creates and validates a multi-select property.
//
// Multi-select properties allow choosing multiple options from a predefined list.
//
// The value parameter should be a comma-separated string of selections.
// Validates both the number of selections and each individual value.
func buildMultiSelectProperty(value string, config multiSelectConfig) (Property, error) {
	items := parseMultiSelect(value)

	if err := validateMultiSelect(items, config); err != nil {
		return Property{}, err
	}

	return Property{
		MultiSelect: items,
	}, nil
}

// buildRelationProperty creates and validates a relation property.
//
// Relation properties link to pages in another database.
// Used for Customer Org field to link to customer pages.
//
// The value parameter should be a comma-separated string of customer names.
// The customerMap is used to look up page IDs for the selected names.
//
// Validates:
// - Maximum number of relations (e.g., max 10 customers)
// - Each customer name exists in the customerMap
func buildRelationProperty(value string, customerMap map[string]string, maxItems int, fieldName string) (Property, error) {
	// Parse comma-separated customer names
	customerNames := strings.Split(value, ",")
	relations := make([]RelationPage, 0, len(customerNames))

	for _, name := range customerNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue // Skip empty values
		}

		// Look up the page ID for this customer name
		pageID, found := customerMap[trimmed]
		if !found {
			return Property{}, fmt.Errorf("invalid %s value: '%s' (not found in customer database)", fieldName, trimmed)
		}

		relations = append(relations, RelationPage{ID: pageID})
	}

	// Validate max items constraint
	if len(relations) > maxItems {
		return Property{}, fmt.Errorf("%s can have at most %d selections, got %d",
			fieldName, maxItems, len(relations))
	}

	return Property{
		Relation: relations,
	}, nil
}

// buildPeopleProperty creates a People property with a Notion user reference.
//
// People properties assign pages to workspace members. In our use case, this tracks
// which Slack user (mapped to a Notion user) submitted the idea.
//
// The value parameter should be a Notion user UUID (not an email address).
// Validates that the UUID is non-empty.
func buildPeopleProperty(notionUserID string) (Property, error) {
	trimmed := strings.TrimSpace(notionUserID)

	if trimmed == "" {
		return Property{}, fmt.Errorf("notion user ID cannot be empty")
	}

	return Property{
		People: []NotionUser{{
			Object: "user",
			ID:     trimmed,
		}},
	}, nil
}

// buildProperties converts form fields into Notion properties with comprehensive validation.
//
// Maps form field names (including aliases) to Notion database property names and validates
// each field according to its type and business rules:
//
// - Title (Idea/Topic): Required, max 2000 chars
// - Theme/Category: Required, single-select, predefined values
// - Product Area: Required, single-select, predefined values
// - Submitted By: Required, People property with Notion user UUID
// - Comments: Optional, rich text, max 2000 chars
// - Customer Org: Optional, multi-select, max 10 selections, validated against Customers database
//
// Empty values (after trimming) are skipped. Field aliases are supported for flexibility.
// Returns a map of Notion property names to Property objects, or an error if validation fails.
func (c *Client) buildProperties(fields map[string]string) (map[string]Property, error) {
	properties := make(map[string]Property)

	for key, value := range fields {
		// Trim whitespace from value before checking if empty
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue // Skip empty values
		}

		var prop Property
		var err error

		switch key {
		case constants.FieldIdeaTopic, constants.AliasTitle, constants.AliasIdea, constants.AliasTopic:
			// Validate and build title property with length limit
			prop, err = buildTitleProperty(trimmedValue)
			if err != nil {
				return nil, fmt.Errorf("title validation failed: %w", err)
			}
			properties[constants.FieldIdeaTopic] = prop

		case constants.FieldThemeCategory, constants.AliasTheme, constants.AliasCategory:
			// Validate theme selection against valid values
			prop, err = buildSelectProperty(trimmedValue, constants.ValidThemeCategories, constants.FieldThemeCategory)
			if err != nil {
				return nil, err
			}
			properties[constants.FieldThemeCategory] = prop

		case constants.FieldProductArea, constants.AliasProductArea, constants.AliasArea:
			// Validate product area against valid values
			prop, err = buildSelectProperty(trimmedValue, constants.ValidProductAreas, constants.FieldProductArea)
			if err != nil {
				return nil, err
			}
			properties[constants.FieldProductArea] = prop

		case constants.FieldComments, constants.AliasComments, constants.AliasComment:
			// Validate comments with length limit
			prop, err = buildRichTextProperty(trimmedValue, constants.FieldComments)
			if err != nil {
				return nil, fmt.Errorf("comments validation failed: %w", err)
			}
			properties[constants.FieldComments] = prop

		case constants.FieldCustomerOrg, constants.AliasCustomerOrg, constants.AliasCustomer, constants.AliasOrg:
			// Validate customer org selections against fetched customer list and max count
			// Use relation property to link to customer database pages
			prop, err = buildRelationProperty(
				trimmedValue,
				c.customerMap,
				constants.MaxCustomerOrgSelections,
				constants.FieldCustomerOrg,
			)
			if err != nil {
				return nil, err
			}
			properties[constants.FieldCustomerOrg] = prop

		case constants.FieldSubmittedBy, constants.AliasSubmittedBy:
			// Build People property with Notion user UUID
			// The value should already be a Notion user UUID (mapped from Slack user email)
			prop, err = buildPeopleProperty(trimmedValue)
			if err != nil {
				return nil, fmt.Errorf("submitted by validation failed: %w", err)
			}
			properties[constants.FieldSubmittedBy] = prop

		default:
			return nil, fmt.Errorf("unknown field: %s", key)
		}
	}

	return properties, nil
}

// validateRequiredFields ensures all required fields are present and valid.
//
// Required fields per business rules:
// 1. Title (Idea/Topic): Every submission must have a descriptive title
// 2. Theme/Category: Must categorize the idea (single selection)
// 3. Product Area: Must specify which product area the idea relates to
// 4. Submitted By: Must track which user submitted the idea
//
// Optional fields (not checked here):
// - Comments: Additional context is optional
// - Customer Org: Customer association is optional
//
// Returns an error if any required field is missing from the properties map.
func (c *Client) validateRequiredFields(properties map[string]Property) error {
	// Check for title field
	if _, hasTitle := properties[constants.FieldIdeaTopic]; !hasTitle {
		return fmt.Errorf("required field 'title' is missing")
	}

	// Check for theme/category field
	if _, hasTheme := properties[constants.FieldThemeCategory]; !hasTheme {
		return fmt.Errorf("required field 'theme' is missing")
	}

	// Check for product area field
	if _, hasProductArea := properties[constants.FieldProductArea]; !hasProductArea {
		return fmt.Errorf("required field 'product_area' is missing")
	}

	// Check for submitted by field
	if _, hasSubmittedBy := properties[constants.FieldSubmittedBy]; !hasSubmittedBy {
		return fmt.Errorf("required field 'submitted_by' is missing")
	}

	return nil
}

// createNotionPage makes the API call to create a page in the Notion database.
//
// Constructs a CreatePageRequest with the validated properties and sends it to
// the Notion API. The page is created in the database specified by c.databaseID.
//
// Returns nil on success, or an error if the API call fails.
// API errors include details from the Notion response for debugging.
func (c *Client) createNotionPage(properties map[string]Property) error {
	request := CreatePageRequest{
		Parent: Parent{
			DatabaseID: c.databaseID,
		},
		Properties: properties,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/pages", constants.NotionAPIBaseURL)
	resp, err := c.makeNotionRequest("POST", endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SubmitForm creates a new entry in the Notion database with the provided fields.
//
// This is the main entry point for form submissions. It orchestrates the entire flow:
// 1. Converts and validates form fields to Notion properties
// 2. Ensures all required fields are present
// 3. Creates the page in the Notion database
// 4. Records metrics for monitoring
//
// Parameters:
// - fields: Map of field names (or aliases) to their string values
//
// Returns nil on success, or an error describing what went wrong (validation or API error).
// All errors are recorded in metrics for observability.
func (c *Client) SubmitForm(fields map[string]string) error {
	start := time.Now()

	properties, err := c.buildProperties(fields)
	if err != nil {
		c.recordNotionRequest("submit_form", start, err)
		return err
	}

	if err := c.validateRequiredFields(properties); err != nil {
		c.recordNotionRequest("submit_form", start, err)
		return err
	}

	err = c.createNotionPage(properties)
	c.recordNotionRequest("submit_form", start, err)
	return err
}

// makeNotionRequest creates and executes an HTTP request to the Notion API.
//
// Handles authentication, versioning, and error handling for all Notion API calls.
// Sets required headers:
// - Authorization: Bearer token for API authentication
// - Notion-Version: API version for request compatibility
// - Content-Type: application/json for request body
//
// Returns the HTTP response on success (status 200), or an error with details.
// Non-200 responses include the full response body in the error message for debugging.
func (c *Client) makeNotionRequest(method, endpoint string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Notion-Version", constants.NotionAPIVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("notion API error (status %d): failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("notion API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// contains checks if a string is in a slice.
// Used for validating selections against allowed values.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// parseMultiSelect splits a comma-separated string into Select items.
//
// Handles comma-separated values from multi-select form fields.
// Trims whitespace from each value and filters out empty strings.
//
// Example: "new feature idea, customer pain point" -> [{"new feature idea"}, {"customer pain point"}]
func parseMultiSelect(value string) []Select {
	parts := strings.Split(value, ",")
	selections := make([]Select, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			selections = append(selections, Select{Name: trimmed})
		}
	}
	return selections
}

// GetDatabaseSchema retrieves the schema of the Notion database.
//
// Queries the database metadata to get property names and their types.
// Useful for debugging and understanding the database structure.
//
// Returns a map of property names to property types (e.g., "title", "rich_text", "select").
func (c *Client) GetDatabaseSchema() (map[string]string, error) {
	endpoint := fmt.Sprintf("%s/databases/%s", constants.NotionAPIBaseURL, c.databaseID)
	resp, err := c.makeNotionRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dbResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&dbResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract property names and types
	schema := make(map[string]string)
	if properties, ok := dbResponse["properties"].(map[string]interface{}); ok {
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propType, ok := propMap["type"].(string); ok {
					schema[name] = propType
				}
			}
		}
	}

	return schema, nil
}

// fetchCustomersPage fetches a single page of customers from the Customers database.
//
// Notion paginates results with a maximum of 100 items per page.
// This method handles fetching one page and returns pagination metadata.
//
// Parameters:
// - cursor: Pagination cursor from previous page (empty string for first page)
//
// Returns:
// - customers: Map of customer name -> Notion page ID from this page
// - nextCursor: Cursor for fetching the next page
// - hasMore: Whether more pages are available
// - err: Any error that occurred during the fetch
func (c *Client) fetchCustomersPage(cursor string) (customers map[string]string, nextCursor string, hasMore bool, err error) {
	requestBody := map[string]interface{}{
		"page_size": constants.NotionPageSize,
	}
	if cursor != "" {
		requestBody["start_cursor"] = cursor
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/databases/%s/query", constants.NotionAPIBaseURL, c.customersDBID)
	resp, err := c.makeNotionRequest("POST", endpoint, body)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	var queryResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&queryResponse); err != nil {
		return nil, "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract customer names and page IDs from the results
	customers = make(map[string]string)
	if results, ok := queryResponse["results"].([]interface{}); ok {
		for _, pageInterface := range results {
			if page, ok := pageInterface.(map[string]interface{}); ok {
				// Extract page ID
				pageID, _ := page["id"].(string)

				// Extract customer name from properties
				if properties, ok := page["properties"].(map[string]interface{}); ok {
					customerName := extractTitleFromProperties(properties)
					if customerName != "" && pageID != "" {
						customers[customerName] = pageID
					}
				}
			}
		}
	}

	// Extract pagination info
	hasMore, _ = queryResponse["has_more"].(bool)
	nextCursor, _ = queryResponse["next_cursor"].(string)

	return customers, nextCursor, hasMore, nil
}

// fetchCustomersFromDatabase queries the Customers database and extracts all customer names and page IDs.
//
// Automatically handles pagination to fetch all customers regardless of total count.
// Continues fetching pages until hasMore is false.
//
// Returns a complete map of customer organization names to their Notion page IDs.
// These are used to populate dropdown options, validate selections, and build relation properties.
func (c *Client) fetchCustomersFromDatabase() (map[string]string, error) {
	allCustomers := make(map[string]string)
	cursor := ""
	hasMore := true

	for hasMore {
		customers, nextCursor, more, err := c.fetchCustomersPage(cursor)
		if err != nil {
			return allCustomers, fmt.Errorf("failed to fetch customers page: %w", err)
		}

		// Merge customers from this page into the map
		for name, pageID := range customers {
			allCustomers[name] = pageID
		}

		cursor = nextCursor
		hasMore = more
	}

	return allCustomers, nil
}

// extractTitleFromProperties extracts the title field from page properties.
//
// Searches through page properties to find the title property and extract its text content.
// Notion's title property has a specific nested structure that this function navigates.
//
// Returns the title text if found, or an empty string if no title property exists.
// In the Customers database, the title contains the customer organization name.
func extractTitleFromProperties(properties map[string]interface{}) string {
	for _, propInterface := range properties {
		prop, ok := propInterface.(map[string]interface{})
		if !ok {
			continue
		}

		propType, ok := prop["type"].(string)
		if !ok || propType != "title" {
			continue
		}

		titleArray, ok := prop["title"].([]interface{})
		if !ok || len(titleArray) == 0 {
			continue
		}

		titleObj, ok := titleArray[0].(map[string]interface{})
		if !ok {
			continue
		}

		textObj, ok := titleObj["text"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := textObj["content"].(string)
		if ok {
			return content
		}
	}
	return ""
}

// fetchUsersFromWorkspace queries the Notion Users API and extracts all user email-to-UUID mappings.
//
// Automatically handles pagination to fetch all workspace users.
// Only includes "person" type users with valid email addresses.
// Normalizes email addresses to lowercase for case-insensitive lookups.
//
// Returns a map of normalized email addresses to Notion user UUIDs.
func (c *Client) fetchUsersFromWorkspace() (map[string]string, error) {
	userMap := make(map[string]string)
	cursor := ""
	hasMore := true

	for hasMore {
		users, nextCursor, more, err := c.fetchUsersPage(cursor)
		if err != nil {
			return userMap, fmt.Errorf("failed to fetch users page: %w", err)
		}

		// Add all users to the map
		for email, userID := range users {
			userMap[email] = userID
		}

		cursor = nextCursor
		hasMore = more
	}

	return userMap, nil
}

// fetchUsersPage fetches a single page of users from the Notion workspace.
//
// Notion paginates results with a maximum of 100 items per page.
// This method handles fetching one page and returns pagination metadata.
//
// Parameters:
// - cursor: Pagination cursor from previous page (empty string for first page)
//
// Returns:
// - users: Map of normalized email -> Notion user UUID from this page
// - nextCursor: Cursor for fetching the next page
// - hasMore: Whether more pages are available
// - err: Any error that occurred during the fetch
func (c *Client) fetchUsersPage(cursor string) (users map[string]string, nextCursor string, hasMore bool, err error) {
	endpoint := fmt.Sprintf("%s/users", constants.NotionAPIBaseURL)
	if cursor != "" {
		endpoint = fmt.Sprintf("%s?start_cursor=%s&page_size=%d", endpoint, cursor, constants.NotionPageSize)
	} else {
		endpoint = fmt.Sprintf("%s?page_size=%d", endpoint, constants.NotionPageSize)
	}

	resp, err := c.makeNotionRequest("GET", endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	var usersResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return nil, "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract users from the results
	users = make(map[string]string)
	if results, ok := usersResponse["results"].([]interface{}); ok {
		for _, userInterface := range results {
			if userObj, ok := userInterface.(map[string]interface{}); ok {
				email, userID := extractEmailAndIDFromUser(userObj)
				if email != "" && userID != "" {
					// Normalize email to lowercase for case-insensitive lookup
					normalizedEmail := strings.ToLower(strings.TrimSpace(email))
					users[normalizedEmail] = userID
				}
			}
		}
	}

	// Extract pagination info
	hasMore, _ = usersResponse["has_more"].(bool)
	nextCursor, _ = usersResponse["next_cursor"].(string)

	return users, nextCursor, hasMore, nil
}

// extractEmailAndIDFromUser extracts the email and UUID from a Notion user object.
//
// Notion user objects have different types (person, bot). Only "person" type users
// have email addresses associated with them.
//
// User object structure:
//
//	{
//	  "object": "user",
//	  "id": "c2f20311-9e54-4d11-8c79-7398424ae41e",
//	  "type": "person",
//	  "person": {
//	    "email": "user@example.com"
//	  }
//	}
//
// Returns the email and user ID if found, or empty strings if not a person user or email missing.
func extractEmailAndIDFromUser(userObj map[string]interface{}) (email string, userID string) {
	// Extract user ID
	userID, _ = userObj["id"].(string)

	// Check if this is a person (not a bot)
	userType, ok := userObj["type"].(string)
	if !ok || userType != "person" {
		// User is a bot or has no type - skip
		return "", ""
	}

	// Extract email from person object
	person, ok := userObj["person"].(map[string]interface{})
	if !ok {
		// Person object missing - skip
		return "", ""
	}

	email, _ = person["email"].(string)

	// Only return if both email and ID are present
	if email == "" || userID == "" {
		// Email or ID missing - skip
		return "", ""
	}

	return email, userID
}
