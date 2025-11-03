package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	SlackSigningSecret   string
	SlackBotToken        string
	NotionAPIKey         string
	NotionDatabaseID     string
	NotionClientsDBID    string
	Port                 string
	CacheRefreshInterval time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
		NotionAPIKey:       os.Getenv("NOTION_API_KEY"),
		NotionDatabaseID:   os.Getenv("NOTION_DATABASE_ID"),
		NotionClientsDBID:  os.Getenv("NOTION_CLIENTS_DB_ID"),
		Port:               os.Getenv("PORT"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Load cache refresh interval (default: 1 hour)
	cfg.CacheRefreshInterval = 1 * time.Hour
	if refreshIntervalStr := os.Getenv("CACHE_REFRESH_INTERVAL"); refreshIntervalStr != "" {
		refreshMinutes, err := strconv.Atoi(refreshIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("CACHE_REFRESH_INTERVAL must be a number of minutes: %w", err)
		}
		cfg.CacheRefreshInterval = time.Duration(refreshMinutes) * time.Minute
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.SlackSigningSecret == "" {
		return fmt.Errorf("SLACK_SIGNING_SECRET is required")
	}
	if c.SlackBotToken == "" {
		return fmt.Errorf("SLACK_BOT_TOKEN is required")
	}
	if c.NotionAPIKey == "" {
		return fmt.Errorf("NOTION_API_KEY is required")
	}
	if c.NotionDatabaseID == "" {
		return fmt.Errorf("NOTION_DATABASE_ID is required")
	}
	if c.NotionClientsDBID == "" {
		return fmt.Errorf("NOTION_CLIENTS_DB_ID is required")
	}
	if c.CacheRefreshInterval <= 0 {
		return fmt.Errorf("CACHE_REFRESH_INTERVAL must be greater than 0")
	}
	return nil
}
