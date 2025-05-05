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
}

type OpenRouterCache struct {
	PromptCache map[string]string              `json:"prompt_cache"`
	ChatHistory map[string][]map[string]string `json:"chat_history"`
	mutex       sync.Mutex                     `json:"-"`
}

var (
	openRouterConfig OpenRouterConfig
	openRouterCache  OpenRouterCache
	configMutex      sync.RWMutex
	vaultChatPath    string
)

// Init initializes the LLM package with the vault path and loads the cache.
func Init(vaultPath string) error {
	if vaultPath == "" {
		return fmt.Errorf("vault path cannot be empty for LLM initialization")
	}
	vaultChatPath = filepath.Join(vaultPath, "Chat")
	log.Printf("LLM Initialized. Cache path set to: %s", vaultChatPath)

	// Ensure Chat directory exists
	if err := os.MkdirAll(vaultChatPath, 0755); err != nil {
		log.Printf("Error creating vault chat directory '%s': %v", vaultChatPath, err)
		return fmt.Errorf("failed to ensure vault chat directory exists: %w", err)
	}

	return LoadOpenRouterCache() // Load cache using the new path
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

// LoadOpenRouterCache loads the prompt cache from the vault's Chat directory.
func LoadOpenRouterCache() error {
	if vaultChatPath == "" {
		log.Println("Warning: Vault path not set, cannot load OpenRouter cache.")
		// Initialize empty cache to prevent nil pointer issues downstream
		openRouterCache.PromptCache = make(map[string]string)
		openRouterCache.ChatHistory = make(map[string][]map[string]string)
		return nil // Or return an error? Returning nil allows app to function without cache initially.
	}
	cacheFilePath := filepath.Join(vaultChatPath, "openrouter_cache.json")
	log.Printf("Attempting to load OpenRouter cache from: %s", cacheFilePath)

	file, err := os.Open(cacheFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("OpenRouter cache file '%s' does not exist. Initializing empty cache.", cacheFilePath)
			openRouterCache.PromptCache = make(map[string]string)
			openRouterCache.ChatHistory = make(map[string][]map[string]string)
			return nil // File not existing is not an error, just means no cache yet
		}
		log.Printf("Error opening OpenRouter cache file '%s': %v", cacheFilePath, err)
		return fmt.Errorf("failed to open OpenRouter cache file: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	// No need for mutex here as this should be called during initialization or vault switch before concurrent access starts.
	// If called concurrently later, mutex would be needed.
	if err := decoder.Decode(&openRouterCache); err != nil {
		log.Printf("Error decoding OpenRouter cache file '%s': %v", cacheFilePath, err)
		// Initialize empty cache on decode error to avoid using potentially corrupt data
		openRouterCache.PromptCache = make(map[string]string)
		openRouterCache.ChatHistory = make(map[string][]map[string]string)
		return fmt.Errorf("failed to decode OpenRouter cache file: %w", err)
	}
	log.Printf("Successfully loaded OpenRouter cache from %s", cacheFilePath)
	return nil
}

// SaveOpenRouterCache saves the prompt cache to the vault's Chat directory.
func SaveOpenRouterCache() error {
	if vaultChatPath == "" {
		log.Println("Error: Vault path not set, cannot save OpenRouter cache.")
		return fmt.Errorf("vault path not set, cannot save cache")
	}

	// Ensure the directory exists (might be redundant if Init was called, but safe)
	if err := os.MkdirAll(vaultChatPath, 0755); err != nil {
		log.Printf("Error creating vault chat directory '%s' before saving cache: %v", vaultChatPath, err)
		return fmt.Errorf("failed to ensure vault chat directory exists before saving cache: %w", err)
	}

	cacheFilePath := filepath.Join(vaultChatPath, "openrouter_cache.json")
	log.Printf("Attempting to save OpenRouter cache to: %s", cacheFilePath)

	openRouterCache.mutex.Lock()
	defer openRouterCache.mutex.Unlock()

	// Create/truncate the file
	file, err := os.Create(cacheFilePath)
	if err != nil {
		log.Printf("Error creating OpenRouter cache file '%s': %v", cacheFilePath, err)
		return fmt.Errorf("failed to create OpenRouter cache file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	// Use the existing structure for saving PromptCache and ChatHistory
	dataToSave := struct {
		PromptCache map[string]string              `json:"prompt_cache"`
		ChatHistory map[string][]map[string]string `json:"chat_history"`
	}{
		PromptCache: openRouterCache.PromptCache,
		ChatHistory: openRouterCache.ChatHistory,
	}
	if err := encoder.Encode(dataToSave); err != nil {
		log.Printf("Error encoding OpenRouter cache data to '%s': %v", cacheFilePath, err)
		return fmt.Errorf("failed to encode cache data: %w", err)
	}
	log.Printf("Successfully saved OpenRouter cache to %s", cacheFilePath)
	return nil
}

// GetOpenRouterCompletion returns a completion from OpenRouter API, using cache if available
func GetOpenRouterCompletion(prompt, model string) (string, error) {
	if openRouterCache.PromptCache == nil {
		openRouterCache.PromptCache = make(map[string]string)
	}
	if openRouterCache.ChatHistory == nil {
		openRouterCache.ChatHistory = make(map[string][]map[string]string)
	}
	cacheKey := model + "::" + prompt
	if cached, ok := openRouterCache.PromptCache[cacheKey]; ok {
		return cached, nil
	}
	configMutex.RLock()
	apiKey := openRouterConfig.APIKey
	configMutex.RUnlock()
	if apiKey == "" {
		return "", fmt.Errorf("OpenRouter API key not set")
	}

	// Get chat history from cache
	chatHistory := make([]map[string]string, 0)
	if chatHistoryKey := fmt.Sprintf("chat_history_%s", model); openRouterCache.ChatHistory != nil {
		if history, ok := openRouterCache.ChatHistory[chatHistoryKey]; ok {
			chatHistory = history
		}
	} else {
		openRouterCache.ChatHistory = make(map[string][]map[string]string)
	}

	// Add current message
	chatHistory = append(chatHistory, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	// Keep only last 10 messages
	if len(chatHistory) > 10 {
		chatHistory = chatHistory[len(chatHistory)-10:]
	}

	// Save updated history
	openRouterCache.ChatHistory[fmt.Sprintf("chat_history_%s", model)] = chatHistory
	_ = SaveOpenRouterCache()

	// Create request body with chat history
	reqBody := map[string]interface{}{
		"model":    model,
		"messages": chatHistory,
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	configMutex.RLock()
	authKey := openRouterConfig.APIKey
	configMutex.RUnlock()
	req.Header.Set("Authorization", "Bearer "+authKey)
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

	// Add assistant's response to chat history
	if chatHistoryKey := fmt.Sprintf("chat_history_%s", model); openRouterCache.ChatHistory != nil {
		if history, ok := openRouterCache.ChatHistory[chatHistoryKey]; ok {
			history = append(history, map[string]string{
				"role":    "assistant",
				"content": output,
			})
			// Keep only last 10 messages
			if len(history) > 10 {
				history = history[len(history)-10:]
			}
			openRouterCache.ChatHistory[chatHistoryKey] = history
		}
	}

	// Save to cache
	openRouterCache.PromptCache[cacheKey] = output
	_ = SaveOpenRouterCache()
	return output, nil
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
