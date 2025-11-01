package config

import (
	"fmt"
	"os"
)

type Config struct {
	SlackSigningSecret string
	SlackBotToken      string
	NotionAPIKey       string
	NotionDatabaseID   string
	NotionClientsDBID  string
	Port               string
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
	return nil
}
