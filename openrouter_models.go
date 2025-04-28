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
