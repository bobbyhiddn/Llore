// Package llm provides OpenRouter LLM integration and configuration management.
package llm

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
	APIKey                 string `json:"openrouter_api_key"`
	ChatModelID            string `json:"chat_model_id,omitempty"`
	StoryProcessingModelID string `json:"story_processing_model_id,omitempty"`
	GeminiApiKey           string `json:"gemini_api_key,omitempty"`

	// New fields for different modes
	ActiveMode              string `json:"active_mode,omitempty"` // "local", "openrouter", "openai", "gemini"
	OpenAIAPIKey            string `json:"openai_api_key,omitempty"`
	LocalEmbeddingModelName string `json:"local_embedding_model_name,omitempty"`
}

var (
	openRouterConfig OpenRouterConfig
	configMutex      sync.RWMutex
	vaultChatPath    string
)

// Init initializes the LLM package with the vault path.
func Init(vaultPath string) error {
	if vaultPath == "" {
		return fmt.Errorf("vault path cannot be empty for LLM initialization")
	}
	vaultChatPath = filepath.Join(vaultPath, "Chat")
	log.Printf("LLM Initialized. Chat path set to: %s", vaultChatPath)

	// Ensure Chat directory exists
	if err := os.MkdirAll(vaultChatPath, 0755); err != nil {
		log.Printf("Error creating vault chat directory '%s': %v", vaultChatPath, err)
		return fmt.Errorf("failed to ensure vault chat directory exists: %w", err)
	}

	return nil
}

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
		return err
	}

	log.Printf("Attempting to load config from: %s", configPath)

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		log.Printf("Error creating config directory '%s': %v", configDir, err)
		return fmt.Errorf("failed to ensure config directory exists: %w", err)
	}

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file '%s' does not exist. Using default empty config.", configPath)
			openRouterConfig = OpenRouterConfig{}
			return nil
		}
		log.Printf("Error opening config file '%s': %v", configPath, err)
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	configMutex.Lock()
	if err := decoder.Decode(&openRouterConfig); err != nil {
		configMutex.Unlock()
		log.Printf("Error decoding config file '%s': %v", configPath, err)
		return fmt.Errorf("failed to decode config file: %w", err)
	}
	log.Printf("Successfully loaded config from %s", configPath)
	configMutex.Unlock()
	return nil
}

// SaveOpenRouterConfig saves the current OpenRouter configuration to ~/.llore/config.json
func SaveOpenRouterConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("could not get config path: %w", err)
	}

	log.Printf("Attempting to save config to path: %s", configPath)

	configMutex.RLock()
	data, err := json.MarshalIndent(openRouterConfig, "", "  ")
	configMutex.RUnlock()
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0750)
	if err != nil {
		log.Printf("Error writing config file '%s': %v", configPath, err)
		return fmt.Errorf("could not write config file %s: %w", configPath, err)
	}
	log.Printf("Successfully wrote config file: %s", configPath)
	return nil
}

// GetOpenRouterCompletion returns a completion from OpenRouter API
func GetOpenRouterCompletion(prompt, model string) (string, error) {
	configMutex.RLock()
	apiKey := openRouterConfig.APIKey
	configMutex.RUnlock()
	if apiKey == "" {
		return "", fmt.Errorf("OpenRouter API key not set")
	}

	// Create request body
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
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
	return result.Choices[0].Message.Content, nil
}

// OpenRouter model definitions and model-fetching logic

type OpenRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// FetchOpenRouterModels fetches available models from OpenRouter API using the provided key.
func FetchOpenRouterModels(apiKey string) ([]OpenRouterModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key not provided to FetchOpenRouterModels")
	}

	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenRouter API error: %s", string(body))
	}
	var result OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetConfig returns a copy of the current OpenRouterConfig.
func GetConfig() OpenRouterConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return openRouterConfig
}

// SetConfig sets the OpenRouterConfig.
func SetConfig(cfg OpenRouterConfig) {
	configMutex.Lock()
	openRouterConfig = cfg
	configMutex.Unlock()
}
