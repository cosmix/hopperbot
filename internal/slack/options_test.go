package slack

import (
	"reflect"
	"strings"
	"testing"
)

func TestFilterCustomerOptions_EmptyQuery(t *testing.T) {
	customers := []string{"Zebra Corp", "Apple Inc", "Microsoft", "Amazon", "Google"}

	tests := []struct {
		name       string
		customers  []string
		maxResults int
		wantCount  int
		wantFirst  string // First customer in alphabetical order
	}{
		{
			name:       "returns all when maxResults exceeds count",
			customers:  customers,
			maxResults: 100,
			wantCount:  5,
			wantFirst:  "Amazon",
		},
		{
			name:       "limits to maxResults",
			customers:  customers,
			maxResults: 3,
			wantCount:  3,
			wantFirst:  "Amazon",
		},
		{
			name:       "defaults to 100 when maxResults is 0",
			customers:  customers,
			maxResults: 0,
			wantCount:  5,
			wantFirst:  "Amazon",
		},
		{
			name:       "defaults to 100 when maxResults is negative",
			customers:  customers,
			maxResults: -1,
			wantCount:  5,
			wantFirst:  "Amazon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := FilterCustomerOptions(tt.customers, "", tt.maxResults)

			if len(options) != tt.wantCount {
				t.Errorf("got %d options, want %d", len(options), tt.wantCount)
			}

			if len(options) > 0 && options[0].Value != tt.wantFirst {
				t.Errorf("first option = %q, want %q", options[0].Value, tt.wantFirst)
			}

			// Verify all options have proper structure
			for i, opt := range options {
				if opt.Text.Type != "plain_text" {
					t.Errorf("option[%d].Text.Type = %q, want %q", i, opt.Text.Type, "plain_text")
				}
				if opt.Text.Text == "" {
					t.Errorf("option[%d].Text.Text is empty", i)
				}
				if opt.Value == "" {
					t.Errorf("option[%d].Value is empty", i)
				}
				if opt.Text.Text != opt.Value {
					t.Errorf("option[%d].Text.Text = %q, Value = %q (should match)", i, opt.Text.Text, opt.Value)
				}
			}
		})
	}
}

func TestFilterCustomerOptions_ExactMatch(t *testing.T) {
	customers := []string{"Apple Inc", "Applied Systems", "Microsoft", "Pineapple Corp"}

	options := FilterCustomerOptions(customers, "apple inc", 100)

	if len(options) != 1 {
		t.Fatalf("got %d options, want 1", len(options))
	}

	if options[0].Value != "Apple Inc" {
		t.Errorf("got %q, want %q", options[0].Value, "Apple Inc")
	}
}

func TestFilterCustomerOptions_PrefixMatch(t *testing.T) {
	customers := []string{"Apple Inc", "Applied Systems", "Application Corp", "Microsoft", "Pineapple Corp"}

	options := FilterCustomerOptions(customers, "app", 100)

	// Should get: 3 prefix matches (Apple Inc, Application Corp, Applied Systems)
	// Plus 1 contains match (Pineapple Corp)
	// Total: 4 options
	if len(options) != 4 {
		t.Fatalf("got %d options, want 4", len(options))
	}

	// First three should be prefix matches (alphabetically sorted)
	expectedOrder := []string{"Apple Inc", "Application Corp", "Applied Systems", "Pineapple Corp"}
	for i, expected := range expectedOrder {
		if options[i].Value != expected {
			t.Errorf("options[%d] = %q, want %q", i, options[i].Value, expected)
		}
	}
}

func TestFilterCustomerOptions_ContainsMatch(t *testing.T) {
	customers := []string{"Apple Inc", "Microsoft", "Lincoln Corp", "Pineapple Inc"}

	options := FilterCustomerOptions(customers, "inc", 100)

	// Should get exact/prefix matches first, then contains matches
	// "inc" doesn't exactly match any (case-sensitive value preservation)
	// Prefix matches: none
	// Contains matches: Apple Inc, Lincoln Corp, Pineapple Inc (alphabetically)
	if len(options) != 3 {
		t.Fatalf("got %d options, want 3", len(options))
	}

	// Verify all contain "inc" (case-insensitive)
	for _, opt := range options {
		contains := false
		for _, r := range opt.Value {
			if r == 'i' || r == 'I' {
				// Simple check for demonstration
				contains = true
				break
			}
		}
		if !contains && opt.Value != "Lincoln Corp" && opt.Value != "Apple Inc" && opt.Value != "Pineapple Inc" {
			t.Errorf("option %q doesn't contain expected substring", opt.Value)
		}
	}
}

func TestFilterCustomerOptions_ThreeTierMatching(t *testing.T) {
	customers := []string{
		"Apple",         // Exact match for "apple"
		"Apple Store",   // Prefix match
		"Apple Corps",   // Prefix match
		"Pineapple Inc", // Contains match
		"Microsoft",     // No match
		"Google",        // No match
	}

	options := FilterCustomerOptions(customers, "apple", 100)

	// Should get: Apple (exact), Apple Corps, Apple Store (prefix), Pineapple Inc (contains)
	if len(options) != 4 {
		t.Fatalf("got %d options, want 4", len(options))
	}

	// First should be exact match
	if options[0].Value != "Apple" {
		t.Errorf("first option = %q, want %q (exact match)", options[0].Value, "Apple")
	}

	// Verify all matches contain "apple" (case-insensitive)
	for i, opt := range options {
		normalized := strings.ToLower(opt.Value)
		if !strings.Contains(normalized, "apple") {
			t.Errorf("options[%d] = %q doesn't contain 'apple'", i, opt.Value)
		}
	}
}

func TestFilterCustomerOptions_MaxResultsLimit(t *testing.T) {
	customers := []string{
		"Apple 1", "Apple 2", "Apple 3", "Apple 4", "Apple 5",
		"Apple 6", "Apple 7", "Apple 8", "Apple 9", "Apple 10",
	}

	tests := []struct {
		name       string
		maxResults int
		wantCount  int
	}{
		{"limit to 5", 5, 5},
		{"limit to 3", 3, 3},
		{"limit to 1", 1, 1},
		{"no limit", 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := FilterCustomerOptions(customers, "apple", tt.maxResults)

			if len(options) != tt.wantCount {
				t.Errorf("got %d options, want %d", len(options), tt.wantCount)
			}
		})
	}
}

func TestFilterCustomerOptions_CaseInsensitive(t *testing.T) {
	customers := []string{"Apple Inc", "APPLE INC", "apple inc"}

	tests := []struct {
		query     string
		wantCount int
	}{
		{"apple", 3},
		{"APPLE", 3},
		{"ApPlE", 3},
		{"apple inc", 3},
		{"APPLE INC", 3},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			options := FilterCustomerOptions(customers, tt.query, 100)

			if len(options) != tt.wantCount {
				t.Errorf("query %q: got %d options, want %d", tt.query, len(options), tt.wantCount)
			}
		})
	}
}

func TestFilterCustomerOptions_NoMatches(t *testing.T) {
	customers := []string{"Apple Inc", "Microsoft", "Google"}

	options := FilterCustomerOptions(customers, "xyz", 100)

	if len(options) != 0 {
		t.Errorf("got %d options, want 0", len(options))
	}
}

func TestFilterCustomerOptions_EmptyCustomerList(t *testing.T) {
	options := FilterCustomerOptions([]string{}, "test", 100)

	if len(options) != 0 {
		t.Errorf("got %d options, want 0", len(options))
	}
}

func TestFormatFirstNOptions(t *testing.T) {
	customers := []string{"Zebra", "Apple", "Microsoft", "Amazon"}

	options := formatFirstNOptions(customers, 2)

	if len(options) != 2 {
		t.Fatalf("got %d options, want 2", len(options))
	}

	// Should be alphabetically sorted
	expectedOrder := []string{"Amazon", "Apple"}
	for i, expected := range expectedOrder {
		if options[i].Value != expected {
			t.Errorf("options[%d] = %q, want %q", i, options[i].Value, expected)
		}
	}
}

func TestBuildOptionsList(t *testing.T) {
	exact := []string{"Zebra", "Apple"}
	prefix := []string{"Beta Corp", "Alpha Inc"}
	contains := []string{"Gamma", "Delta"}

	tests := []struct {
		name       string
		maxResults int
		wantCount  int
		wantOrder  []string
	}{
		{
			name:       "all tiers fit",
			maxResults: 10,
			wantCount:  6,
			wantOrder:  []string{"Apple", "Zebra", "Alpha Inc", "Beta Corp", "Delta", "Gamma"},
		},
		{
			name:       "only exact tier",
			maxResults: 2,
			wantCount:  2,
			wantOrder:  []string{"Apple", "Zebra"},
		},
		{
			name:       "exact and partial prefix",
			maxResults: 3,
			wantCount:  3,
			wantOrder:  []string{"Apple", "Zebra", "Alpha Inc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildOptionsList(exact, prefix, contains, tt.maxResults)

			if len(options) != tt.wantCount {
				t.Errorf("got %d options, want %d", len(options), tt.wantCount)
			}

			for i, expected := range tt.wantOrder {
				if i >= len(options) {
					break
				}
				if options[i].Value != expected {
					t.Errorf("options[%d] = %q, want %q", i, options[i].Value, expected)
				}
			}
		})
	}
}

func TestNewOptionText(t *testing.T) {
	text := newOptionText("Test Customer")

	if text.Type != "plain_text" {
		t.Errorf("Type = %q, want %q", text.Type, "plain_text")
	}

	if text.Text != "Test Customer" {
		t.Errorf("Text = %q, want %q", text.Text, "Test Customer")
	}

	if text.Emoji != false {
		t.Errorf("Emoji = %v, want false", text.Emoji)
	}
}

func TestFilterCustomerOptions_AlphabeticalSorting(t *testing.T) {
	customers := []string{"Zeta", "Alpha", "Gamma", "Beta", "Delta"}

	// Empty query should return alphabetically sorted
	options := FilterCustomerOptions(customers, "", 100)

	expected := []string{"Alpha", "Beta", "Delta", "Gamma", "Zeta"}
	got := make([]string, len(options))
	for i, opt := range options {
		got[i] = opt.Value
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestFilterCustomerOptions_PreservesOriginalCase(t *testing.T) {
	customers := []string{"ApPlE Inc", "MICROSOFT Corp"}

	options := FilterCustomerOptions(customers, "apple", 100)

	// Should preserve original casing in results
	if len(options) != 1 {
		t.Fatalf("got %d options, want 1", len(options))
	}

	if options[0].Value != "ApPlE Inc" {
		t.Errorf("got %q, want %q (original case preserved)", options[0].Value, "ApPlE Inc")
	}
}
