package config

import (
	"fmt"
	"os"
)

type Config struct {
	HatenaID string
	BlogID   string
	APIKey   string
}

func Load() (*Config, error) {
	hatenaID := os.Getenv("HATENA_ID")
	if hatenaID == "" {
		return nil, fmt.Errorf("HATENA_ID environment variable is required")
	}

	blogID := os.Getenv("BLOG_ID")
	if blogID == "" {
		return nil, fmt.Errorf("BLOG_ID environment variable is required")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	return &Config{
		HatenaID: hatenaID,
		BlogID:   blogID,
		APIKey:   apiKey,
	}, nil
}