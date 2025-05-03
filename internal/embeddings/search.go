// internal/embeddings/search.go
package embeddings

import (
	"Llore/internal/database" // Assuming database types are here
	"database/sql"            // Need this for sql.RawBytes
	"encoding/binary"
	"fmt"
	"log" // Added for logging potential errors
	"math"
	"sort"
)

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Entry database.CodexEntry
	Score float32
}

// FindSimilarEntries finds entries similar to the query using cosine similarity
func (s *EmbeddingService) FindSimilarEntries(query string, limit int) ([]SearchResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if s.apiKey == "" {
		// It might be better to check this during initialization, but added here for safety
		return nil, fmt.Errorf("Gemini API key is not configured")
	}

	// Generate embedding for query
	queryEmbedding, err := s.CreateEmbedding(query)
	if err != nil {
		// Log this error specifically
		log.Printf("ERROR in FindSimilarEntries: failed to create query embedding: %v", err)
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}
	// log.Printf("Successfully generated query embedding for: '%s'", query) // Remove success log

	// Get all entries with embeddings from the database
	rows, err := s.db.Query(`
		SELECT
			e.id, e.name, e.type, e.content, e.created_at, e.updated_at,
			em.embedding
		FROM
			codex_entries e
		LEFT JOIN -- Changed to LEFT JOIN
			codex_embeddings em ON e.id = em.codex_entry_id
	`)
	if err != nil {
		// Log this error
		log.Printf("ERROR in FindSimilarEntries: failed to query entries with embeddings: %v", err)
		return nil, fmt.Errorf("failed to query entries with embeddings: %w", err)
	}
	defer rows.Close()
	// log.Println("Successfully queried database for entries with embeddings.") // Remove success log

	var results []SearchResult
	rowCount := 0 // Add counter

	for rows.Next() {
		rowCount++ // Increment counter
		var entry database.CodexEntry
		var embeddingBytes sql.RawBytes // Use sql.RawBytes to handle potential NULL

		err := rows.Scan(
			&entry.ID, &entry.Name, &entry.Type, &entry.Content,
			&entry.CreatedAt, &entry.UpdatedAt, &embeddingBytes, // Scan into RawBytes
		)
		if err != nil {
			log.Printf("Warning: Failed to scan entry row during search: %v", err)
			continue // Skip this entry if scanning fails
		}

		// Check if embedding is NULL (not generated yet)
		if embeddingBytes == nil {
			log.Printf("Skipping entry ID %d ('%s'): Embedding not generated yet.", entry.ID, entry.Name)
			continue
		}

		// Deserialize embedding
		entryEmbedding := deserializeEmbedding(embeddingBytes)
		if len(entryEmbedding) == 0 {
			// This case should be less common now, but keep check for data integrity
			log.Printf("Warning: Skipping entry ID %d ('%s') due to invalid non-NULL embedding data (length %d)", entry.ID, entry.Name, len(embeddingBytes))
			continue // Skip if embedding is invalid
		}

		// Calculate similarity
		similarity := cosineSimilarity(queryEmbedding, entryEmbedding)

		// Only include results if similarity is valid (e.g., vectors had same dimension)
		if !math.IsNaN(float64(similarity)) {
			results = append(results, SearchResult{
				Entry: entry,
				Score: similarity,
			})
		} else {
			log.Printf("Warning: NaN similarity score for entry ID %d, skipping", entry.ID)
		}
	}
	// log.Printf("Processed %d rows from database query.", rowCount) // Remove row count log

	if err = rows.Err(); err != nil {
		log.Printf("Warning: Error during row iteration in FindSimilarEntries: %v", err)
		// Depending on the error, you might want to return it, but often logging is sufficient
	}

	// Sort by similarity (highest first)
	sort.Slice(results, func(i, j int) bool {
		// Handle NaN scores if any slipped through, putting them at the end
		if math.IsNaN(float64(results[i].Score)) {
			return false
		}
		if math.IsNaN(float64(results[j].Score)) {
			return true
		}
		return results[i].Score > results[j].Score
	})

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// log.Printf("FindSimilarEntries returning %d results.", len(results)) // Remove final count log
	return results, nil
}

// deserializeEmbedding converts bytes to float32 slice
func deserializeEmbedding(data []byte) []float32 {
	if len(data) == 0 || len(data)%4 != 0 {
		// Return empty slice for invalid data, handled in the calling function
		return []float32{}
	}

	result := make([]float32, len(data)/4)
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}

	return result
}

// cosineSimilarity calculates similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		log.Printf("Warning: Cosine similarity length mismatch or zero length (%d vs %d)", len(a), len(b))
		return 0 // Or potentially NaN, but 0 might be safer for sorting
	}

	var dotProduct, normA, normB float64 // Use float64 for intermediate calculations to improve precision

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		log.Println("Warning: Cosine similarity calculated with zero norm vector")
		return 0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))

	// Clamp similarity to [-1, 1] due to potential floating point inaccuracies
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return float32(similarity)
}
