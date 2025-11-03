package config

import (
	"os"
	"testing"
	"time"
)

// Helper function to set environment variables for testing
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set environment variable %s: %v", key, err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset environment variable %s: %v", key, err)
		}
	})
}

// Helper function to unset environment variables for testing
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset environment variable %s: %v", key, err)
	}
	t.Cleanup(func() {
		// No cleanup needed since we're testing the unset case
	})
}

// TestLoad_SuccessWithAllEnvVars tests successful load with all environment variables set
func TestLoad_SuccessWithAllEnvVars(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
	setEnv(t, "PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.SlackSigningSecret != "test-slack-secret" {
		t.Errorf("SlackSigningSecret = %q, want %q", cfg.SlackSigningSecret, "test-slack-secret")
	}

	if cfg.SlackBotToken != "test-slack-token" {
		t.Errorf("SlackBotToken = %q, want %q", cfg.SlackBotToken, "test-slack-token")
	}

	if cfg.NotionAPIKey != "test-notion-key" {
		t.Errorf("NotionAPIKey = %q, want %q", cfg.NotionAPIKey, "test-notion-key")
	}

	if cfg.NotionDatabaseID != "test-db-id" {
		t.Errorf("NotionDatabaseID = %q, want %q", cfg.NotionDatabaseID, "test-db-id")
	}

	if cfg.NotionClientsDBID != "test-clients-db-id" {
		t.Errorf("NotionClientsDBID = %q, want %q", cfg.NotionClientsDBID, "test-clients-db-id")
	}

	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9000")
	}
}

// TestLoad_SuccessWithDefaultPort tests successful load with default port when PORT env var not set
func TestLoad_SuccessWithDefaultPort(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
	unsetEnv(t, "PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q (default)", cfg.Port, "8080")
	}
}

// TestLoad_SuccessWithCustomPort tests successful load with custom port
func TestLoad_SuccessWithCustomPort(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
	setEnv(t, "PORT", "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "3000")
	}
}

// TestLoad_MissingSlackSigningSecret tests failed load with missing SLACK_SIGNING_SECRET
func TestLoad_MissingSlackSigningSecret(t *testing.T) {
	unsetEnv(t, "SLACK_SIGNING_SECRET")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for missing SLACK_SIGNING_SECRET")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}

	if err.Error() != "SLACK_SIGNING_SECRET is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "SLACK_SIGNING_SECRET is required")
	}
}

// TestLoad_MissingSlackBotToken tests failed load with missing SLACK_BOT_TOKEN
func TestLoad_MissingSlackBotToken(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	unsetEnv(t, "SLACK_BOT_TOKEN")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for missing SLACK_BOT_TOKEN")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}

	if err.Error() != "SLACK_BOT_TOKEN is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "SLACK_BOT_TOKEN is required")
	}
}

// TestLoad_MissingNotionAPIKey tests failed load with missing NOTION_API_KEY
func TestLoad_MissingNotionAPIKey(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	unsetEnv(t, "NOTION_API_KEY")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for missing NOTION_API_KEY")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}

	if err.Error() != "NOTION_API_KEY is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_API_KEY is required")
	}
}

// TestLoad_MissingNotionDatabaseID tests failed load with missing NOTION_DATABASE_ID
func TestLoad_MissingNotionDatabaseID(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	unsetEnv(t, "NOTION_DATABASE_ID")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for missing NOTION_DATABASE_ID")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}

	if err.Error() != "NOTION_DATABASE_ID is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_DATABASE_ID is required")
	}
}

// TestLoad_MissingNotionClientsDBID tests failed load with missing NOTION_CLIENTS_DB_ID
func TestLoad_MissingNotionClientsDBID(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	unsetEnv(t, "NOTION_CLIENTS_DB_ID")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for missing NOTION_CLIENTS_DB_ID")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}

	if err.Error() != "NOTION_CLIENTS_DB_ID is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_CLIENTS_DB_ID is required")
	}
}

// TestValidate_ValidConfig tests Validate() with all required fields present
func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "test-secret",
		SlackBotToken:        "test-token",
		NotionAPIKey:         "test-api-key",
		NotionDatabaseID:     "test-db-id",
		NotionClientsDBID:    "test-clients-db-id",
		Port:                 "8080",
		CacheRefreshInterval: 1 * time.Hour,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

// TestValidate_MissingSlackSigningSecret tests Validate() with missing SLACK_SIGNING_SECRET
func TestValidate_MissingSlackSigningSecret(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "test-api-key",
		NotionDatabaseID:   "test-db-id",
		NotionClientsDBID:  "test-clients-db-id",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing SlackSigningSecret")
	}

	if err.Error() != "SLACK_SIGNING_SECRET is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "SLACK_SIGNING_SECRET is required")
	}
}

// TestValidate_MissingSlackBotToken tests Validate() with missing SLACK_BOT_TOKEN
func TestValidate_MissingSlackBotToken(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "",
		NotionAPIKey:       "test-api-key",
		NotionDatabaseID:   "test-db-id",
		NotionClientsDBID:  "test-clients-db-id",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing SlackBotToken")
	}

	if err.Error() != "SLACK_BOT_TOKEN is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "SLACK_BOT_TOKEN is required")
	}
}

// TestValidate_MissingNotionAPIKey tests Validate() with missing NOTION_API_KEY
func TestValidate_MissingNotionAPIKey(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "",
		NotionDatabaseID:   "test-db-id",
		NotionClientsDBID:  "test-clients-db-id",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing NotionAPIKey")
	}

	if err.Error() != "NOTION_API_KEY is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_API_KEY is required")
	}
}

// TestValidate_MissingNotionDatabaseID tests Validate() with missing NOTION_DATABASE_ID
func TestValidate_MissingNotionDatabaseID(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "test-api-key",
		NotionDatabaseID:   "",
		NotionClientsDBID:  "test-clients-db-id",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing NotionDatabaseID")
	}

	if err.Error() != "NOTION_DATABASE_ID is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_DATABASE_ID is required")
	}
}

// TestValidate_MissingNotionClientsDBID tests Validate() with missing NOTION_CLIENTS_DB_ID
func TestValidate_MissingNotionClientsDBID(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "test-api-key",
		NotionDatabaseID:   "test-db-id",
		NotionClientsDBID:  "",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing NotionClientsDBID")
	}

	if err.Error() != "NOTION_CLIENTS_DB_ID is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "NOTION_CLIENTS_DB_ID is required")
	}
}

// TestValidate_MultipleFieldsMissing tests Validate() with multiple required fields missing
// (should report the first missing field)
func TestValidate_MultipleFieldsMissing(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret: "",
		SlackBotToken:      "",
		NotionAPIKey:       "",
		NotionDatabaseID:   "",
		NotionClientsDBID:  "",
		Port:               "8080",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for missing fields")
	}

	// Should report the first missing field
	if err.Error() != "SLACK_SIGNING_SECRET is required" {
		t.Errorf("error message = %q, want %q (first missing field)", err.Error(), "SLACK_SIGNING_SECRET is required")
	}
}

// TestValidate_PortIsOptional tests Validate() with empty Port field (optional)
func TestValidate_PortIsOptional(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "test-secret",
		SlackBotToken:        "test-token",
		NotionAPIKey:         "test-api-key",
		NotionDatabaseID:     "test-db-id",
		NotionClientsDBID:    "test-clients-db-id",
		Port:                 "",
		CacheRefreshInterval: 1 * time.Hour,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() should not fail for empty Port field: %v", err)
	}
}

// TestLoad_EmptyStringValues tests Load with empty string values for required fields
func TestLoad_EmptyStringValues(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "")
	setEnv(t, "SLACK_BOT_TOKEN", "test-token")
	setEnv(t, "NOTION_API_KEY", "test-api-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() should have returned an error for empty SLACK_SIGNING_SECRET")
	}

	if cfg != nil {
		t.Error("Load() should have returned nil config when validation fails")
	}
}

// TestLoad_PortEdgeCases tests various port values and defaults
func TestLoad_PortEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		portValue string
		wantPort  string
	}{
		{
			name:      "empty port uses default",
			portValue: "",
			wantPort:  "8080",
		},
		{
			name:      "custom port 3000",
			portValue: "3000",
			wantPort:  "3000",
		},
		{
			name:      "custom port 5000",
			portValue: "5000",
			wantPort:  "5000",
		},
		{
			name:      "port as string",
			portValue: "9090",
			wantPort:  "9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, "SLACK_SIGNING_SECRET", "test-secret")
			setEnv(t, "SLACK_BOT_TOKEN", "test-token")
			setEnv(t, "NOTION_API_KEY", "test-api-key")
			setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
			setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

			if tt.portValue == "" {
				unsetEnv(t, "PORT")
			} else {
				setEnv(t, "PORT", tt.portValue)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned unexpected error: %v", err)
			}

			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %q, want %q", cfg.Port, tt.wantPort)
			}
		})
	}
}

// TestConfigStruct tests that Config struct fields are correctly populated
func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "slack-secret-value",
		SlackBotToken:        "slack-token-value",
		NotionAPIKey:         "notion-api-value",
		NotionDatabaseID:     "notion-db-value",
		NotionClientsDBID:    "notion-clients-value",
		Port:                 "8080",
		CacheRefreshInterval: 1 * time.Hour,
	}

	if cfg.SlackSigningSecret != "slack-secret-value" {
		t.Errorf("SlackSigningSecret = %q", cfg.SlackSigningSecret)
	}

	if cfg.SlackBotToken != "slack-token-value" {
		t.Errorf("SlackBotToken = %q", cfg.SlackBotToken)
	}

	if cfg.NotionAPIKey != "notion-api-value" {
		t.Errorf("NotionAPIKey = %q", cfg.NotionAPIKey)
	}

	if cfg.NotionDatabaseID != "notion-db-value" {
		t.Errorf("NotionDatabaseID = %q", cfg.NotionDatabaseID)
	}

	if cfg.NotionClientsDBID != "notion-clients-value" {
		t.Errorf("NotionClientsDBID = %q", cfg.NotionClientsDBID)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q", cfg.Port)
	}

	if cfg.CacheRefreshInterval != 1*time.Hour {
		t.Errorf("CacheRefreshInterval = %v", cfg.CacheRefreshInterval)
	}
}

// TestLoad_ValidatesOnReturn tests that Load calls Validate and returns early on validation error
func TestLoad_ValidatesOnReturn(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-token")
	setEnv(t, "NOTION_API_KEY", "test-api-key")
	// Missing NOTION_DATABASE_ID to trigger validation error
	unsetEnv(t, "NOTION_DATABASE_ID")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() should have returned an error due to validation failure")
	}

	if cfg != nil {
		t.Error("Load() should return nil when validation fails")
	}
}

// TestLoad_MultipleRequiredFieldsMissing tests Load with multiple required fields missing
func TestLoad_MultipleRequiredFieldsMissing(t *testing.T) {
	unsetEnv(t, "SLACK_SIGNING_SECRET")
	unsetEnv(t, "SLACK_BOT_TOKEN")
	unsetEnv(t, "NOTION_API_KEY")
	unsetEnv(t, "NOTION_DATABASE_ID")
	unsetEnv(t, "NOTION_CLIENTS_DB_ID")
	unsetEnv(t, "PORT")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() should have returned an error for missing required fields")
	}

	if cfg != nil {
		t.Error("Load() should return nil when validation fails")
	}

	// Should report the first missing required field
	if err.Error() != "SLACK_SIGNING_SECRET is required" {
		t.Errorf("error message = %q, want %q", err.Error(), "SLACK_SIGNING_SECRET is required")
	}
}

// TestValidate_CheckOrderOfValidation tests that validation checks required fields in expected order
func TestValidate_CheckOrderOfValidation(t *testing.T) {
	tests := []struct {
		name             string
		config           Config
		expectedErrorMsg string
	}{
		{
			name: "missing SlackSigningSecret",
			config: Config{
				SlackSigningSecret:   "",
				SlackBotToken:        "token",
				NotionAPIKey:         "key",
				NotionDatabaseID:     "db",
				NotionClientsDBID:    "clients",
				CacheRefreshInterval: 1 * time.Hour,
			},
			expectedErrorMsg: "SLACK_SIGNING_SECRET is required",
		},
		{
			name: "missing SlackBotToken",
			config: Config{
				SlackSigningSecret:   "secret",
				SlackBotToken:        "",
				NotionAPIKey:         "key",
				NotionDatabaseID:     "db",
				NotionClientsDBID:    "clients",
				CacheRefreshInterval: 1 * time.Hour,
			},
			expectedErrorMsg: "SLACK_BOT_TOKEN is required",
		},
		{
			name: "missing NotionAPIKey",
			config: Config{
				SlackSigningSecret:   "secret",
				SlackBotToken:        "token",
				NotionAPIKey:         "",
				NotionDatabaseID:     "db",
				NotionClientsDBID:    "clients",
				CacheRefreshInterval: 1 * time.Hour,
			},
			expectedErrorMsg: "NOTION_API_KEY is required",
		},
		{
			name: "missing NotionDatabaseID",
			config: Config{
				SlackSigningSecret:   "secret",
				SlackBotToken:        "token",
				NotionAPIKey:         "key",
				NotionDatabaseID:     "",
				NotionClientsDBID:    "clients",
				CacheRefreshInterval: 1 * time.Hour,
			},
			expectedErrorMsg: "NOTION_DATABASE_ID is required",
		},
		{
			name: "missing NotionClientsDBID",
			config: Config{
				SlackSigningSecret:   "secret",
				SlackBotToken:        "token",
				NotionAPIKey:         "key",
				NotionDatabaseID:     "db",
				NotionClientsDBID:    "",
				CacheRefreshInterval: 1 * time.Hour,
			},
			expectedErrorMsg: "NOTION_CLIENTS_DB_ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatal("Validate() should have returned an error")
			}

			if err.Error() != tt.expectedErrorMsg {
				t.Errorf("error message = %q, want %q", err.Error(), tt.expectedErrorMsg)
			}
		})
	}
}

// TestLoad_CacheRefreshInterval_Default tests default cache refresh interval (1 hour)
func TestLoad_CacheRefreshInterval_Default(t *testing.T) {
	setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
	setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
	setEnv(t, "NOTION_API_KEY", "test-notion-key")
	setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
	setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
	unsetEnv(t, "CACHE_REFRESH_INTERVAL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	expectedInterval := 1 * time.Hour
	if cfg.CacheRefreshInterval != expectedInterval {
		t.Errorf("CacheRefreshInterval = %v, want %v (default)", cfg.CacheRefreshInterval, expectedInterval)
	}
}

// TestLoad_CacheRefreshInterval_Custom tests custom cache refresh interval
func TestLoad_CacheRefreshInterval_Custom(t *testing.T) {
	tests := []struct {
		name             string
		envValue         string
		expectedInterval time.Duration
	}{
		{
			name:             "30 minutes",
			envValue:         "30",
			expectedInterval: 30 * time.Minute,
		},
		{
			name:             "60 minutes (1 hour)",
			envValue:         "60",
			expectedInterval: 60 * time.Minute,
		},
		{
			name:             "360 minutes (6 hours)",
			envValue:         "360",
			expectedInterval: 360 * time.Minute,
		},
		{
			name:             "1 minute",
			envValue:         "1",
			expectedInterval: 1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
			setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
			setEnv(t, "NOTION_API_KEY", "test-notion-key")
			setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
			setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
			setEnv(t, "CACHE_REFRESH_INTERVAL", tt.envValue)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned unexpected error: %v", err)
			}

			if cfg.CacheRefreshInterval != tt.expectedInterval {
				t.Errorf("CacheRefreshInterval = %v, want %v", cfg.CacheRefreshInterval, tt.expectedInterval)
			}
		})
	}
}

// TestLoad_CacheRefreshInterval_Invalid tests invalid cache refresh interval values
func TestLoad_CacheRefreshInterval_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{
			name:     "non-numeric value",
			envValue: "invalid",
		},
		{
			name:     "negative value",
			envValue: "-30",
		},
		{
			name:     "float value",
			envValue: "30.5",
		},
		{
			name:     "empty spaces",
			envValue: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, "SLACK_SIGNING_SECRET", "test-slack-secret")
			setEnv(t, "SLACK_BOT_TOKEN", "test-slack-token")
			setEnv(t, "NOTION_API_KEY", "test-notion-key")
			setEnv(t, "NOTION_DATABASE_ID", "test-db-id")
			setEnv(t, "NOTION_CLIENTS_DB_ID", "test-clients-db-id")
			setEnv(t, "CACHE_REFRESH_INTERVAL", tt.envValue)

			cfg, err := Load()
			if err == nil {
				t.Fatal("Load() should have returned an error for invalid CACHE_REFRESH_INTERVAL")
			}

			if cfg != nil {
				t.Error("Load() should have returned nil config for invalid CACHE_REFRESH_INTERVAL")
			}
		})
	}
}

// TestValidate_CacheRefreshInterval_Zero tests validation with zero cache refresh interval
func TestValidate_CacheRefreshInterval_Zero(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "test-secret",
		SlackBotToken:        "test-token",
		NotionAPIKey:         "test-api-key",
		NotionDatabaseID:     "test-db-id",
		NotionClientsDBID:    "test-clients-db-id",
		Port:                 "8080",
		CacheRefreshInterval: 0,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for zero CacheRefreshInterval")
	}

	if err.Error() != "CACHE_REFRESH_INTERVAL must be greater than 0" {
		t.Errorf("error message = %q, want %q", err.Error(), "CACHE_REFRESH_INTERVAL must be greater than 0")
	}
}

// TestValidate_CacheRefreshInterval_Negative tests validation with negative cache refresh interval
func TestValidate_CacheRefreshInterval_Negative(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "test-secret",
		SlackBotToken:        "test-token",
		NotionAPIKey:         "test-api-key",
		NotionDatabaseID:     "test-db-id",
		NotionClientsDBID:    "test-clients-db-id",
		Port:                 "8080",
		CacheRefreshInterval: -1 * time.Hour,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should have returned an error for negative CacheRefreshInterval")
	}

	if err.Error() != "CACHE_REFRESH_INTERVAL must be greater than 0" {
		t.Errorf("error message = %q, want %q", err.Error(), "CACHE_REFRESH_INTERVAL must be greater than 0")
	}
}

// TestValidate_CacheRefreshInterval_Valid tests validation with valid cache refresh interval
func TestValidate_CacheRefreshInterval_Valid(t *testing.T) {
	cfg := &Config{
		SlackSigningSecret:   "test-secret",
		SlackBotToken:        "test-token",
		NotionAPIKey:         "test-api-key",
		NotionDatabaseID:     "test-db-id",
		NotionClientsDBID:    "test-clients-db-id",
		Port:                 "8080",
		CacheRefreshInterval: 1 * time.Hour,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() returned unexpected error for valid CacheRefreshInterval: %v", err)
	}
}
