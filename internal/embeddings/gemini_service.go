// internal/embeddings/gemini_service.go
package embeddings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	GeminiEmbeddingsAPI = "https://generativelanguage.googleapis.com/v1/models/embedding-001:embedContent"
	GeminiModelVersion  = "gemini-embedding-001"
)

// GeminiEmbeddingProvider implements the EmbeddingProvider interface for Gemini
type GeminiEmbeddingProvider struct {
	apiKey string
}

// NewGeminiEmbeddingProvider creates a new Gemini embedding provider
func NewGeminiEmbeddingProvider(apiKey string) *GeminiEmbeddingProvider {
	return &GeminiEmbeddingProvider{
		apiKey: apiKey,
	}
}

// CreateEmbedding generates an embedding for the given text using Gemini API
func (p *GeminiEmbeddingProvider) CreateEmbedding(text string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is not configured")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"model": "models/embedding-001",
		"content": map[string]interface{}{
			"parts": []map[string]interface{}{
				{
					"text": text,
				},
			},
		},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request URL with API key
	url := fmt.Sprintf("%s?key=%s", GeminiEmbeddingsAPI, p.apiKey)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, strings.NewReader(string(requestJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error
	if resp.StatusCode != http.StatusOK {
		// Try to parse potential error message from Google API
		var errorResponse struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errorResponse) == nil && errorResponse.Error.Message != "" {
			return nil, fmt.Errorf("API error (%d %s): %s", errorResponse.Error.Code, errorResponse.Error.Status, errorResponse.Error.Message)
		}
		// Fallback to raw body if parsing fails
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Embedding.Values) == 0 {
		return nil, fmt.Errorf("API returned empty embedding values")
	}

	log.Printf("Generated Gemini embedding with %d dimensions", len(response.Embedding.Values))
	return response.Embedding.Values, nil
}

// ModelIdentifier returns the identifier for this embedding model
func (p *GeminiEmbeddingProvider) ModelIdentifier() string {
	return GeminiModelVersion
}
