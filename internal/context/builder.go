// internal/context/builder.go
package context

import (
	"Llore/internal/embeddings" // Use the embeddings package
	"fmt"
	"log" // Added for logging
	"sort"
	"strings"
)

// ContextBuilder builds context for LLM prompts using embeddings
type ContextBuilder struct {
	embeddingService    *embeddings.EmbeddingService
	maxEntries          int     // Max number of entries to retrieve
	similarityThreshold float32 // Minimum similarity score to include
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(embeddingService *embeddings.EmbeddingService) *ContextBuilder {
	if embeddingService == nil {
		log.Fatal("FATAL: EmbeddingService cannot be nil in NewContextBuilder") // Use Fatal as this is critical
	}
	return &ContextBuilder{
		embeddingService:    embeddingService,
		maxEntries:          10,  // Default max entries
		similarityThreshold: 0.4, // Default minimum similarity score
	}
}

// SetMaxEntries allows customizing the maximum number of context entries
func (b *ContextBuilder) SetMaxEntries(max int) {
	if max > 0 {
		b.maxEntries = max
	}
}

// SetSimilarityThreshold allows customizing the minimum similarity score
func (b *ContextBuilder) SetSimilarityThreshold(threshold float32) {
	if threshold >= 0.0 && threshold <= 1.0 {
		b.similarityThreshold = threshold
	}
}

// BuildContextForQuery creates a context string based on similarity search results
func (b *ContextBuilder) BuildContextForQuery(query string) (string, error) {
	if b.embeddingService == nil {
		return "", fmt.Errorf("embedding service is not initialized in ContextBuilder")
	}

	// Find similar entries using the embedding service
	results, err := b.embeddingService.FindSimilarEntries(query, b.maxEntries)
	if err != nil {
		// Log the error but don't necessarily stop; maybe return an empty context
		log.Printf("Warning: Failed to find similar entries for context: %v", err)
		// Depending on the desired behavior, you might return the error or just an empty context
		// return "", fmt.Errorf("failed to find similar entries: %w", err)
		return "", nil // Return empty context on search error for now
	}

	// Build context string
	var sb strings.Builder
	includedCount := 0
	var includedEntryInfo []string // Add slice to store info

	if len(results) == 0 {
		log.Println("No relevant context found for query.")
		return "", nil // No relevant context found
	}

	sb.WriteString("CONTEXT INFORMATION (ordered by relevance):\n") // Add header

	// Sort results by score descending (most similar first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	for _, result := range results {
		// Skip entries below the similarity threshold
		if result.Score < b.similarityThreshold {
			break
		}

		// Add entry to context string
		// Using a clear format for the LLM
		sb.WriteString(fmt.Sprintf("--- Entry Start ---\n"))
		sb.WriteString(fmt.Sprintf("Name: %s\n", result.Entry.Name))
		sb.WriteString(fmt.Sprintf("Content:\n%s\n", result.Entry.Content))
		sb.WriteString(fmt.Sprintf("(Relevance Score: %.2f)\n", result.Score))
		sb.WriteString(fmt.Sprintf("--- Entry End ---\n\n"))

		includedCount++
		includedEntryInfo = append(includedEntryInfo, fmt.Sprintf("%s (Score: %.2f)", result.Entry.Name, result.Score)) // Store info
	}

	if includedCount == 0 {
		log.Println("No entries met the similarity threshold for the query.")
		return "", nil // No entries met the threshold
	}

	log.Printf("Built context with %d entries for query: [%s]", includedCount, strings.Join(includedEntryInfo, "; ")) // Modify log message
	return sb.String(), nil
}
