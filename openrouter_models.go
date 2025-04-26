package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type OpenRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// FetchOpenRouterModels fetches available models from OpenRouter API
func (a *App) FetchOpenRouterModels() ([]OpenRouterModel, error) {
	if err := LoadOpenRouterConfig(); err != nil {
		return nil, fmt.Errorf("failed to load OpenRouter config: %w", err)
	}
	if openRouterConfig.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key not loaded")
	}

	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+openRouterConfig.APIKey)
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
