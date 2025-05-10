// internal/embeddings/gemini_service.go
package embeddings

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/genai"
)

const (
	GeminiEmbeddingModel = "text-embedding-004" // Updated to a more current model
	TaskTypeRetrievalDocument = "RETRIEVAL_DOCUMENT"
)

// GeminiEmbeddingProvider implements the EmbeddingProvider interface for Gemini
// It now correctly returns []float32 for CreateEmbedding.
type GeminiEmbeddingProvider struct {
	apiKey string
}

// NewGeminiEmbeddingProvider creates a new Gemini embedding provider
func NewGeminiEmbeddingProvider(apiKey string) *GeminiEmbeddingProvider {
	return &GeminiEmbeddingProvider{
		apiKey: apiKey,
	}
}

// CreateEmbedding generates an embedding for the given text using the Gemini SDK.
// It now returns []float32 as expected by the EmbeddingProvider interface.
func (p *GeminiEmbeddingProvider) CreateEmbedding(text string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: p.apiKey})
	if err != nil {
		log.Printf("GeminiEmbeddingProvider: failed to create genai client: %v", err)
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	resp, err := client.Models.EmbedContent(ctx, GeminiEmbeddingModel, genai.Text(text), &genai.EmbedContentConfig{TaskType: TaskTypeRetrievalDocument})
	if err != nil {
		log.Printf("GeminiEmbeddingProvider: failed to embed content: %v", err)
		return nil, fmt.Errorf("failed to embed content with Gemini: %w", err)
	}

	if resp == nil || len(resp.Embeddings) == 0 || resp.Embeddings[0] == nil {
		log.Printf("GeminiEmbeddingProvider: received nil response or no embeddings or nil embedding")
		return nil, fmt.Errorf("gemini embedding response contained no embeddings")
	}

	embeddingValues := resp.Embeddings[0].Values
	if embeddingValues == nil {
		log.Printf("GeminiEmbeddingProvider: embedding values are nil")
		return nil, fmt.Errorf("gemini embedding values are nil")
	}

	// Return []float32 directly, as assumed from SDK and required by interface
	return embeddingValues, nil
}

// ModelIdentifier returns the specific model identifier for this provider.
func (p *GeminiEmbeddingProvider) ModelIdentifier() string {
	return fmt.Sprintf("gemini:%s", GeminiEmbeddingModel)
}
