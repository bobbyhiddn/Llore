// internal/embeddings/embedding_provider.go
package embeddings

// EmbeddingProvider defines the interface for any embedding generation service.
type EmbeddingProvider interface {
	CreateEmbedding(text string) ([]float32, error)
	ModelIdentifier() string // e.g., "local:all-MiniLM-L6-v2", "gemini:embedding-001"
}
