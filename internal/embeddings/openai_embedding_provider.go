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
	OpenAIAPIURL       = "https://api.openai.com/v1/embeddings"
	OpenAIDefaultModel = "text-embedding-3-small"
)

type OpenAIEmbeddingProvider struct {
	apiKey     string
	modelName  string
	httpClient *http.Client
}

type openAIRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type openAIEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

type openAIResponse struct {
	Data   []openAIEmbeddingData `json:"data"`
	Model  string                `json:"model"`
	Object string                `json:"object"`
}

func NewOpenAIEmbeddingProvider(apiKey string, modelName ...string) (*OpenAIEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key cannot be empty")
	}
	mn := OpenAIDefaultModel
	if len(modelName) > 0 && modelName[0] != "" {
		mn = modelName[0]
	}
	log.Printf("OpenAIEmbeddingProvider: Initializing for model: '%s'", mn)
	return &OpenAIEmbeddingProvider{
		apiKey:     apiKey,
		modelName:  mn,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (p *OpenAIEmbeddingProvider) CreateEmbedding(text string) ([]float32, error) {
	payload := openAIRequest{
		Input: text,
		Model: p.modelName,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequest("POST", OpenAIAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAI response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: OpenAI API returned status %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("OpenAI API error (Status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %w. Body: %s", err, string(body))
	}

	if len(apiResp.Data) == 0 || len(apiResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no embedding data")
	}
	log.Printf("Generated OpenAI embedding with %d dimensions using model %s", len(apiResp.Data[0].Embedding), p.modelName)
	return apiResp.Data[0].Embedding, nil
}

func (p *OpenAIEmbeddingProvider) ModelIdentifier() string {
	return fmt.Sprintf("openai:%s", p.modelName)
}
