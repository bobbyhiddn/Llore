package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveOpenRouterApiKey updates config.json with the given API key
func (a *App) SaveOpenRouterApiKey(apiKey string) error {
	config := OpenRouterConfig{APIKey: apiKey}
	file, err := os.Create("config.json")
	if err != nil {
		return fmt.Errorf("failed to open config.json: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config.json: %w", err)
	}
	return nil
}
