package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

// LoadOpenRouterConfig loads the OpenRouter API key from config.json
func LoadOpenRouterConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	return decoder.Decode(&openRouterConfig)
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
