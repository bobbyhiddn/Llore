package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	// OpenAIDefaultModel is the default model for OpenAI embeddings.
	// "text-embedding-3-small" is a good general-purpose and cost-effective model.
	OpenAIDefaultModel = "text-embedding-3-small"
)

// OpenAIEmbeddingProvider implements the EmbeddingProvider interface for OpenAI.
type OpenAIEmbeddingProvider struct {
	apiKey     string
	modelName  string
	httpClient *http.Client
}

// NewOpenAIEmbeddingProvider creates a new OpenAI embedding provider.
// If no modelName is provided, it defaults to OpenAIDefaultModel.
func NewOpenAIEmbeddingProvider(apiKey string, modelIdentifier ...string) (*OpenAIEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key cannot be empty")
	}

	selectedModel := OpenAIDefaultModel
	if len(modelIdentifier) > 0 && modelIdentifier[0] != "" {
		selectedModel = modelIdentifier[0]
	}

	log.Printf("OpenAIEmbeddingProvider: Initializing for model: '%s'", selectedModel)

	return &OpenAIEmbeddingProvider{
		apiKey:     apiKey,
		modelName:  selectedModel,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// CreateEmbedding generates an embedding for the given text using the OpenAI API.
func (p *OpenAIEmbeddingProvider) CreateEmbedding(text string) ([]float32, error) {
	if p.httpClient == nil {
		return nil, fmt.Errorf("OpenAI HTTP client is not initialized")
	}

	// OpenAI API expects a slice of strings for input, even if it's just one.
	inputTexts := []string{text}

	// Create the request payload
	payload := struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}{
		Input: inputTexts,
		Model: p.modelName,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI HTTP request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAI response body: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: OpenAI API returned status %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("OpenAI API error (Status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var apiResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
			Object    string    `json:"object"`
		} `json:"data"`
		Model  string `json:"model"`
		Object string `json:"object"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %w. Body: %s", err, string(body))
	}

	// Validate the response
	if len(apiResp.Data) == 0 || len(apiResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no embedding data")
	}

	log.Printf("Generated OpenAI embedding with %d dimensions using model %s", len(apiResp.Data[0].Embedding), p.modelName)
	return apiResp.Data[0].Embedding, nil
}

// ModelIdentifier returns a string uniquely identifying the OpenAI model being used.
func (p *OpenAIEmbeddingProvider) ModelIdentifier() string {
	return fmt.Sprintf("openai:%s", p.modelName)
}
