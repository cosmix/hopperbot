// Package slack provides handlers and types for Slack integration.
//
// This file implements external select menu options handling, which allows
// Slack to dynamically load options for select menus as users type. This
// is used for the Customer Organization field which can have thousands of
// customers - too many to send with the initial modal.
//
// When a user interacts with an external select menu, Slack sends a POST
// request to the options endpoint with the user's search query. The server
// responds with a filtered list of options matching that query.
package slack

import (
	"sort"
	"strings"
)

// FilterCustomerOptions filters a list of customers based on a search query
// and returns formatted Option objects for Slack.
//
// Search logic implements three-tier matching for optimal user experience:
// 1. Exact matches (case-insensitive): "apple" matches "Apple"
// 2. Prefix matches: "app" matches "Apple Inc", "Application Systems"
// 3. Contains matches: "inc" matches "Apple Inc", "Lincoln Corp"
//
// Each tier is sorted alphabetically, and tiers are combined in order.
// Results are limited to maxResults (defaults to 100 if <= 0).
//
// When query is empty, returns the first N customers alphabetically.
//
// Example:
//
//	customers := []string{"Apple Inc", "Microsoft", "Amazon", "Applied Systems"}
//	options := FilterCustomerOptions(customers, "app", 100)
//	// Returns: ["Applied Systems", "Apple Inc"] (exact/prefix matches, alphabetically)
func FilterCustomerOptions(customers []string, query string, maxResults int) []Option {
	// Default to 100 results if not specified or invalid
	if maxResults <= 0 {
		maxResults = 100
	}

	// Empty query: return first N alphabetically
	if query == "" {
		return formatFirstNOptions(customers, maxResults)
	}

	// Normalize query for case-insensitive matching
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	// Categorize matches into three tiers
	var exactMatches []string
	var prefixMatches []string
	var containsMatches []string

	for _, customer := range customers {
		normalizedCustomer := strings.ToLower(customer)

		if normalizedCustomer == normalizedQuery {
			// Tier 1: Exact match
			exactMatches = append(exactMatches, customer)
		} else if strings.HasPrefix(normalizedCustomer, normalizedQuery) {
			// Tier 2: Prefix match
			prefixMatches = append(prefixMatches, customer)
		} else if strings.Contains(normalizedCustomer, normalizedQuery) {
			// Tier 3: Contains match
			containsMatches = append(containsMatches, customer)
		}
	}

	// Build final options list from tiers
	return buildOptionsList(exactMatches, prefixMatches, containsMatches, maxResults)
}

// formatFirstNOptions returns the first N customers alphabetically as options.
// Used when the user opens the dropdown without typing a search query.
//
// Example:
//
//	customers := []string{"Zebra Corp", "Apple Inc", "Microsoft"}
//	options := formatFirstNOptions(customers, 2)
//	// Returns: [{"Apple Inc"}, {"Microsoft"}] (alphabetically sorted, first 2)
func formatFirstNOptions(customers []string, n int) []Option {
	// Sort customers alphabetically
	sorted := make([]string, len(customers))
	copy(sorted, customers)
	sort.Strings(sorted)

	// Limit to first N
	if len(sorted) > n {
		sorted = sorted[:n]
	}

	// Convert to Option objects
	options := make([]Option, 0, len(sorted))
	for _, customer := range sorted {
		options = append(options, Option{
			Text:  newOptionText(customer),
			Value: customer,
		})
	}

	return options
}

// buildOptionsList combines three match tiers into a single options list.
// Each tier is sorted alphabetically, and tiers are combined in order:
// exact matches, then prefix matches, then contains matches.
//
// Results are limited to maxResults total across all tiers.
//
// Example:
//
//	exact := []string{"Apple"}
//	prefix := []string{"Applied Systems", "Application Corp"}
//	contains := []string{"Pineapple Inc"}
//	options := buildOptionsList(exact, prefix, contains, 3)
//	// Returns: ["Apple", "Applied Systems", "Application Corp"]
//	// (exact + prefix, limited to 3 results)
func buildOptionsList(exact, prefix, contains []string, maxResults int) []Option {
	// Sort each tier alphabetically
	sort.Strings(exact)
	sort.Strings(prefix)
	sort.Strings(contains)

	// Combine tiers respecting maxResults limit
	var combined []string
	combined = append(combined, exact...)

	if len(combined) < maxResults {
		remaining := maxResults - len(combined)
		if len(prefix) <= remaining {
			combined = append(combined, prefix...)
		} else {
			combined = append(combined, prefix[:remaining]...)
		}
	}

	if len(combined) < maxResults {
		remaining := maxResults - len(combined)
		if len(contains) <= remaining {
			combined = append(combined, contains...)
		} else {
			combined = append(combined, contains[:remaining]...)
		}
	}

	// Convert to Option objects
	options := make([]Option, 0, len(combined))
	for _, customer := range combined {
		options = append(options, Option{
			Text:  newOptionText(customer),
			Value: customer,
		})
	}

	return options
}

// newOptionText creates an OptionText object for a plain text value.
// The Type is always "plain_text" for standard dropdown options.
//
// Example:
//
//	text := newOptionText("Apple Inc")
//	// Returns: OptionText{Type: "plain_text", Text: "Apple Inc", Emoji: false}
func newOptionText(text string) OptionText {
	return OptionText{
		Type:  "plain_text",
		Text:  text,
		Emoji: false,
	}
}
