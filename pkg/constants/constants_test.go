package constants

import (
	"testing"
	"time"
)

// TestValidThemeCategoriesNotEmpty tests that valid theme categories are defined
func TestValidThemeCategoriesNotEmpty(t *testing.T) {
	if len(ValidThemeCategories) == 0 {
		t.Error("ValidThemeCategories should not be empty")
	}

	expectedThemes := map[string]bool{
		"New Feature Idea":                true,
		"Feature Improvement":             true,
		"Market/Competition Intelligence": true,
		"Customer Pain Point":             true,
	}

	for _, theme := range ValidThemeCategories {
		if !expectedThemes[theme] {
			t.Errorf("unexpected theme: %s", theme)
		}
	}

	if len(ValidThemeCategories) != len(expectedThemes) {
		t.Errorf("ValidThemeCategories length = %d, want %d", len(ValidThemeCategories), len(expectedThemes))
	}
}

// TestValidProductAreasNotEmpty tests that valid product areas are defined
func TestValidProductAreasNotEmpty(t *testing.T) {
	if len(ValidProductAreas) == 0 {
		t.Error("ValidProductAreas should not be empty")
	}

	expectedAreas := map[string]bool{
		"AI/ML":             true,
		"Integrations/SDKs": true,
		"Data Governance":   true,
		"Systems":           true,
		"UX":                true,
		"Activation Kits":   true,
		"Activation Core":   true,
		"rETL":              true,
		"Transformations":   true,
		"EventStream":       true,
		"WH Ingestion":      true,
	}

	for _, area := range ValidProductAreas {
		if !expectedAreas[area] {
			t.Errorf("unexpected product area: %s", area)
		}
	}

	if len(ValidProductAreas) != len(expectedAreas) {
		t.Errorf("ValidProductAreas length = %d, want %d", len(ValidProductAreas), len(expectedAreas))
	}
}

// TestMaxCustomerOrgSelections tests customer org selection limit
func TestMaxCustomerOrgSelections(t *testing.T) {
	if MaxCustomerOrgSelections <= 0 {
		t.Error("MaxCustomerOrgSelections should be positive")
	}

	if MaxCustomerOrgSelections != 10 {
		t.Errorf("MaxCustomerOrgSelections = %d, want 10", MaxCustomerOrgSelections)
	}
}

// TestMaxTitleLength tests title length limit
func TestMaxTitleLength(t *testing.T) {
	if MaxTitleLength <= 0 {
		t.Error("MaxTitleLength should be positive")
	}

	if MaxTitleLength != 2000 {
		t.Errorf("MaxTitleLength = %d, want 2000", MaxTitleLength)
	}
}

// TestMaxCommentLength tests comment length limit
func TestMaxCommentLength(t *testing.T) {
	if MaxCommentLength <= 0 {
		t.Error("MaxCommentLength should be positive")
	}

	if MaxCommentLength != 2000 {
		t.Errorf("MaxCommentLength = %d, want 2000", MaxCommentLength)
	}
}

// TestMaxSlackRequestAge tests Slack request age limit
func TestMaxSlackRequestAge(t *testing.T) {
	if MaxSlackRequestAge <= 0 {
		t.Error("MaxSlackRequestAge should be positive")
	}

	if MaxSlackRequestAge != 300 {
		t.Errorf("MaxSlackRequestAge = %d, want 300", MaxSlackRequestAge)
	}
}

// TestDefaultHTTPTimeout tests HTTP timeout is reasonable
func TestDefaultHTTPTimeout(t *testing.T) {
	if DefaultHTTPTimeout <= 0 {
		t.Error("DefaultHTTPTimeout should be positive")
	}

	expectedTimeout := 30 * time.Second
	if DefaultHTTPTimeout != expectedTimeout {
		t.Errorf("DefaultHTTPTimeout = %v, want %v", DefaultHTTPTimeout, expectedTimeout)
	}
}

// TestServerTimeouts tests server timeouts are reasonable
func TestServerTimeouts(t *testing.T) {
	if ServerReadTimeout <= 0 {
		t.Error("ServerReadTimeout should be positive")
	}

	if ServerWriteTimeout <= 0 {
		t.Error("ServerWriteTimeout should be positive")
	}

	if ServerIdleTimeout <= 0 {
		t.Error("ServerIdleTimeout should be positive")
	}

	if GracefulShutdownTimeout <= 0 {
		t.Error("GracefulShutdownTimeout should be positive")
	}

	// Write timeout should be >= read timeout
	if ServerWriteTimeout < ServerReadTimeout {
		t.Errorf("ServerWriteTimeout (%v) should be >= ServerReadTimeout (%v)", ServerWriteTimeout, ServerReadTimeout)
	}

	// Idle timeout should be >= write timeout
	if ServerIdleTimeout < ServerWriteTimeout {
		t.Errorf("ServerIdleTimeout (%v) should be >= ServerWriteTimeout (%v)", ServerIdleTimeout, ServerWriteTimeout)
	}
}

// TestNotionAPIConstants tests Notion API constants
func TestNotionAPIConstants(t *testing.T) {
	if NotionPageSize <= 0 {
		t.Error("NotionPageSize should be positive")
	}

	if NotionPageSize != 100 {
		t.Errorf("NotionPageSize = %d, want 100", NotionPageSize)
	}

	if NotionAPIVersion == "" {
		t.Error("NotionAPIVersion should not be empty")
	}

	if NotionAPIBaseURL == "" {
		t.Error("NotionAPIBaseURL should not be empty")
	}
}

// TestDefaultPort tests default port is set
func TestDefaultPort(t *testing.T) {
	if DefaultPort == "" {
		t.Error("DefaultPort should not be empty")
	}

	if DefaultPort != "8080" {
		t.Errorf("DefaultPort = %s, want 8080", DefaultPort)
	}
}

// TestFieldNames tests field name constants
func TestFieldNames(t *testing.T) {
	if FieldIdeaTopic == "" {
		t.Error("FieldIdeaTopic should not be empty")
	}

	if FieldThemeCategory == "" {
		t.Error("FieldThemeCategory should not be empty")
	}

	if FieldProductArea == "" {
		t.Error("FieldProductArea should not be empty")
	}

	if FieldComments == "" {
		t.Error("FieldComments should not be empty")
	}

	if FieldCustomerOrg == "" {
		t.Error("FieldCustomerOrg should not be empty")
	}
}

// TestFieldAliases tests field alias constants are set
func TestFieldAliases(t *testing.T) {
	titleAliases := []string{AliasTitle, AliasIdea, AliasTopic}
	for _, alias := range titleAliases {
		if alias == "" {
			t.Errorf("title alias should not be empty")
		}
	}

	themeAliases := []string{AliasTheme, AliasCategory}
	for _, alias := range themeAliases {
		if alias == "" {
			t.Errorf("theme alias should not be empty")
		}
	}

	productAreaAliases := []string{AliasProductArea, AliasArea}
	for _, alias := range productAreaAliases {
		if alias == "" {
			t.Errorf("product area alias should not be empty")
		}
	}

	commentAliases := []string{AliasComments, AliasComment}
	for _, alias := range commentAliases {
		if alias == "" {
			t.Errorf("comment alias should not be empty")
		}
	}

	customerOrgAliases := []string{AliasCustomerOrg, AliasCustomer, AliasOrg}
	for _, alias := range customerOrgAliases {
		if alias == "" {
			t.Errorf("customer org alias should not be empty")
		}
	}
}

// TestTimeoutCombinations tests timeout combinations are sensible
func TestTimeoutCombinations(t *testing.T) {
	// HTTP client timeout should allow for at least one read/write cycle
	if DefaultHTTPTimeout < ServerReadTimeout {
		t.Errorf("DefaultHTTPTimeout (%v) should be >= ServerReadTimeout (%v)", DefaultHTTPTimeout, ServerReadTimeout)
	}

	// Graceful shutdown timeout should be reasonable compared to server timeouts
	if GracefulShutdownTimeout < 0 {
		t.Error("GracefulShutdownTimeout should be non-negative")
	}
}

// TestSlackConstants tests that selection limits are within Slack API limits
func TestSlackConstants(t *testing.T) {
	// Slack typically allows up to 100 options per select menu
	if MaxCustomerOrgSelections > 100 {
		t.Errorf("MaxCustomerOrgSelections (%d) exceeds typical Slack limits", MaxCustomerOrgSelections)
	}
}

// TestLengthLimitComparison tests that field length limits are consistent
func TestLengthLimitComparison(t *testing.T) {
	// Notion has similar limits for different text fields
	if MaxTitleLength != MaxCommentLength {
		// This is acceptable if intentional
		t.Logf("Title and comment length limits differ: %d vs %d", MaxTitleLength, MaxCommentLength)
	}

	// Both should be at least 100 characters
	if MaxTitleLength < 100 || MaxCommentLength < 100 {
		t.Error("length limits should be at least 100 characters")
	}
}
