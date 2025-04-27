package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type OpenRouterConfig struct {
	APIKey string `json:"openrouter_api_key"`
}

type OpenRouterCache struct {
	PromptCache map[string]string `json:"prompt_cache"`
	mutex       sync.Mutex        `json:"-"`
}

var openRouterConfig OpenRouterConfig
var openRouterCache OpenRouterCache

// getConfigPath returns the absolute path to the config file (~/.llore/config.json)
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".llore")
	return filepath.Join(configDir, "config.json"), nil
}

// LoadOpenRouterConfig loads the OpenRouter API key from ~/.llore/config.json
func LoadOpenRouterConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		log.Printf("Error getting config path: %v", err)
		return err // Return error if we can't determine path
	}

	log.Printf("Attempting to load config from: %s", configPath)

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil { // Use 0750 for permissions
		log.Printf("Error creating config directory '%s': %v", configDir, err)
		// If we can't create the dir, we likely can't read/write the file either
		return fmt.Errorf("failed to ensure config directory exists: %w", err)
	}

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file '%s' does not exist. Using default empty config.", configPath)
			// File doesn't exist, initialize with default empty config
			openRouterConfig = OpenRouterConfig{}
			return nil // Not an error if file doesn't exist yet
		}
		log.Printf("Error opening config file '%s': %v", configPath, err)
		return fmt.Errorf("failed to open config file: %w", err) // Other error opening file
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&openRouterConfig); err != nil {
		log.Printf("Error decoding config file '%s': %v", configPath, err)
		return fmt.Errorf("failed to decode config file: %w", err)
	}
	log.Printf("Successfully loaded config from %s", configPath)
	return nil
}

// SaveOpenRouterConfig saves the current OpenRouter configuration to ~/.llore/config.json
func SaveOpenRouterConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err // Return error if we can't determine path
	}

	log.Printf("Attempting to save config to: %s", configPath)

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to ensure config directory exists: %w", err)
	}

	file, err := os.Create(configPath) // Create or truncate the file
	if err != nil {
		return fmt.Errorf("failed to create/open config file for writing: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print
	if err := encoder.Encode(openRouterConfig); err != nil {
		return fmt.Errorf("failed to encode config to file: %w", err)
	}

	log.Printf("Successfully saved config to %s", configPath)
	return nil
}

// LoadOpenRouterCache loads the prompt cache from openrouter_cache.json
func LoadOpenRouterCache() error {
	file, err := os.Open("openrouter_cache.json")
	if err != nil {
		openRouterCache.PromptCache = make(map[string]string)
		return nil
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	return decoder.Decode(&openRouterCache)
}

// SaveOpenRouterCache saves the prompt cache to openrouter_cache.json
func SaveOpenRouterCache() error {
	openRouterCache.mutex.Lock()
	defer openRouterCache.mutex.Unlock()
	file, err := os.Create("openrouter_cache.json")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(openRouterCache)
}

// GetOpenRouterCompletion returns a completion from OpenRouter API, using cache if available
func GetOpenRouterCompletion(prompt, model string) (string, error) {
	if openRouterCache.PromptCache == nil {
		openRouterCache.PromptCache = make(map[string]string)
	}
	cacheKey := model + "::" + prompt
	if cached, ok := openRouterCache.PromptCache[cacheKey]; ok {
		return cached, nil
	}
	if openRouterConfig.APIKey == "" {
		return "", fmt.Errorf("OpenRouter API key not loaded")
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"model":   model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+openRouterConfig.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenRouter API error: %s", string(body))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("No choices returned from OpenRouter")
	}
	output := result.Choices[0].Message.Content
	// Save to cache
	openRouterCache.PromptCache[cacheKey] = output
	_ = SaveOpenRouterCache()
	return output, nil
}
