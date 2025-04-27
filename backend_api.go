package main

import (
	"fmt"
	"log"
)

// SaveOpenRouterApiKey updates the API key in the global config and saves it to ~/.llore/config.json
func (a *App) SaveOpenRouterApiKey(apiKey string) error {
	log.Println("Backend API: Attempting to save OpenRouter API key...")
	openRouterConfig.APIKey = apiKey // Update the global config variable
	if err := SaveOpenRouterConfig(); err != nil { // Call the central save function
		log.Printf("Backend API: Error saving OpenRouter config: %v", err)
		return fmt.Errorf("failed to save OpenRouter configuration: %w", err)
	}
	log.Println("Backend API: OpenRouter API key saved successfully via SaveOpenRouterConfig.")
	return nil
}

// FetchOpenRouterModels retrieves the list of models from OpenRouter
