// internal/embeddings/gemini_service.go
package embeddings

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	GeminiEmbeddingsAPI = "https://generativelanguage.googleapis.com/v1/models/embedding-001:embedContent"
	ModelVersion        = "gemini-embedding-001"
)

// EmbeddingService manages vector embeddings
type EmbeddingService struct {
	db         *sql.DB
	apiKey     string
	cache      map[int64][]float32
	cacheMutex sync.RWMutex
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(db *sql.DB, apiKey string) *EmbeddingService {
	return &EmbeddingService{
		db:     db,
		apiKey: apiKey,
		cache:  make(map[int64][]float32),
	}
}

// CreateEmbedding generates an embedding for the given text
func (s *EmbeddingService) CreateEmbedding(text string) ([]float32, error) {
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
	url := fmt.Sprintf("%s?key=%s", GeminiEmbeddingsAPI, s.apiKey)

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

	return response.Embedding.Values, nil
}

// SaveEmbedding stores an embedding for a codex entry using INSERT OR REPLACE
func (s *EmbeddingService) SaveEmbedding(entryID int64, embedding []float32) error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if len(embedding) == 0 {
		return fmt.Errorf("cannot save empty embedding")
	}

	// Serialize embedding to binary
	embeddingBytes := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(embeddingBytes[i*4:], math.Float32bits(v))
	}

	// Use INSERT OR REPLACE to handle both new and existing entries atomically
	// This relies on the UNIQUE constraint on codex_entry_id.
	// It keeps the original created_at timestamp if replacing an existing row.
	_, err := s.db.Exec( // Can use Exec directly for single atomic statement
		`INSERT OR REPLACE INTO codex_embeddings
         (codex_entry_id, embedding, vector_version, created_at, updated_at)
         VALUES (?, ?, ?,
                 COALESCE((SELECT created_at FROM codex_embeddings WHERE codex_entry_id = ?), datetime('now')), -- Keep original created_at
                 datetime('now'))`,
		entryID, embeddingBytes, ModelVersion, entryID, // entryID used twice
	)

	if err != nil {
		// Log the specific error before returning
		log.Printf("ERROR saving embedding for entry ID %d: %v", entryID, err)
		return fmt.Errorf("failed to save embedding to database: %w", err)
	}

	// Update cache only after successful DB operation
	s.cacheMutex.Lock()
	s.cache[entryID] = embedding
	s.cacheMutex.Unlock()

	// Log success only after error check passes
	log.Printf("Saved embedding for entry ID %d", entryID)
	return nil
}

// GetEmbedding retrieves an embedding for a codex entry
func (s *EmbeddingService) GetEmbedding(entryID int64) ([]float32, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	// Check cache first
	s.cacheMutex.RLock()
	cached, ok := s.cache[entryID]
	s.cacheMutex.RUnlock()

	if ok {
		return cached, nil
	}

	// Query database
	var embeddingBytes []byte
	err := s.db.QueryRow(
		"SELECT embedding FROM codex_embeddings WHERE codex_entry_id = ?",
		entryID,
	).Scan(&embeddingBytes)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return a specific error or nil, nil? Let's return an error for clarity.
			return nil, fmt.Errorf("no embedding found for entry %d", entryID)
		}
		return nil, fmt.Errorf("failed to retrieve embedding from database: %w", err)
	}

	// Deserialize from binary
	if len(embeddingBytes)%4 != 0 {
		log.Printf("Warning: Invalid embedding data length (%d bytes) for entry %d", len(embeddingBytes), entryID)
		return nil, fmt.Errorf("invalid embedding data length")
	}

	embedding := make([]float32, len(embeddingBytes)/4)
	for i := range embedding {
		embedding[i] = math.Float32frombits(binary.LittleEndian.Uint32(embeddingBytes[i*4:]))
	}

	// Update cache
	s.cacheMutex.Lock()
	s.cache[entryID] = embedding
	s.cacheMutex.Unlock()

	return embedding, nil
}

// LoadEmbeddingsIntoCache pre-loads embeddings into the cache
func (s *EmbeddingService) LoadEmbeddingsIntoCache() error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	log.Println("Loading embeddings into cache...")
	rows, err := s.db.Query("SELECT codex_entry_id, embedding FROM codex_embeddings")
	if err != nil {
		return fmt.Errorf("failed to query embeddings for cache: %w", err)
	}
	defer rows.Close()

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = make(map[int64][]float32) // Clear existing cache
	count := 0
	for rows.Next() {
		var entryID int64
		var embeddingBytes []byte
		if err := rows.Scan(&entryID, &embeddingBytes); err != nil {
			log.Printf("Warning: Failed to scan embedding row for cache: %v", err)
			continue
		}

		if len(embeddingBytes)%4 != 0 {
			log.Printf("Warning: Invalid embedding data length (%d bytes) for entry %d in cache load", len(embeddingBytes), entryID)
			continue
		}

		embedding := make([]float32, len(embeddingBytes)/4)
		for i := range embedding {
			embedding[i] = math.Float32frombits(binary.LittleEndian.Uint32(embeddingBytes[i*4:]))
		}
		s.cache[entryID] = embedding
		count++
	}
	log.Printf("Loaded %d embeddings into cache.", count)
	return rows.Err() // Check for errors during iteration
}
