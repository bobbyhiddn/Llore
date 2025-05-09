// internal/embeddings/local_embedding_provider.go
package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	// OllamaDefaultAPIEndpoint is the standard local endpoint for Ollama.
	OllamaDefaultAPIEndpoint = "http://localhost:11434/api/embeddings"
)

// ollamaEmbeddingRequest defines the JSON structure for the request to Ollama.
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	// Options map[string]interface{} `json:"options,omitempty"` // For advanced Ollama options if needed
}

// ollamaEmbeddingResponse defines the JSON structure for a successful embedding response from Ollama.
type ollamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// LocalEmbeddingProvider implements the EmbeddingProvider interface by communicating
// with a locally running Ollama instance.
type LocalEmbeddingProvider struct {
	modelName       string // The Ollama model tag (e.g., "nomic-embed-text")
	apiEndpoint     string
	httpClient      *http.Client
	modelIdentifier string
}

// NewLocalEmbeddingProvider creates a new local embedding provider that connects to Ollama.
// `ollamaModelTag` MUST be the tag of a model already pulled in your local Ollama
// (e.g., "nomic-embed-text", "mxbai-embed-large").
func NewLocalEmbeddingProvider(ollamaModelTag string) (*LocalEmbeddingProvider, error) {
	if ollamaModelTag == "" {
		// This error will be caught by App.svelte, prompting the user in settings.
		// Or, we could default here, but it's better for the user to be explicit.
		return nil, fmt.Errorf("Ollama model tag cannot be empty. Please specify a model (e.g., 'nomic-embed-text') in Llore's settings for local embedding mode")
	}

	log.Printf("LocalEmbeddingProvider: Initializing for Ollama model tag: '%s'", ollamaModelTag)
	log.Println("INFO: This provider requires a local Ollama instance to be running.")
	log.Printf("INFO: Ensure Ollama is running and that model '%s' has been pulled (e.g., `ollama pull %s`)", ollamaModelTag, ollamaModelTag)

	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout as local models can sometimes be slow on first call.
	}

	// A preliminary check to see if Ollama is responsive can be added here,
	// but it's often better to let the first CreateEmbedding call handle connection errors,
	// as Ollama might be started *after* Llore.

	return &LocalEmbeddingProvider{
		modelName:       ollamaModelTag,
		apiEndpoint:     OllamaDefaultAPIEndpoint,
		httpClient:      client,
		modelIdentifier: fmt.Sprintf("ollama:%s", ollamaModelTag), // Unique ID for this provider configuration
	}, nil
}

// CreateEmbedding generates an embedding for the given text by calling the local Ollama /api/embeddings endpoint.
func (p *LocalEmbeddingProvider) CreateEmbedding(text string) ([]float32, error) {
	if p.httpClient == nil {
		return nil, fmt.Errorf("LocalEmbeddingProvider not initialized (httpClient is nil)")
	}

	requestPayload := ollamaEmbeddingRequest{
		Model:  p.modelName,
		Prompt: text,
	}

	bodyBytes, err := json.Marshal(requestPayload)
	if err != nil {
		log.Printf("ERROR: LocalEmbeddingProvider (Ollama) failed to marshal request for model '%s': %v", p.modelName, err)
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	// Create Request
	req, err := http.NewRequest("POST", p.apiEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("ERROR: LocalEmbeddingProvider (Ollama) failed to create HTTP request for model '%s': %v", p.modelName, err)
		return nil, fmt.Errorf("failed to create ollama HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute Request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("ERROR: LocalEmbeddingProvider (Ollama) request failed for model '%s'. Is Ollama running at %s? Error: %v", p.modelName, p.apiEndpoint, err)
		return nil, fmt.Errorf("failed to connect to Ollama at %s (model: %s). Please ensure Ollama is running. Error: %w", p.apiEndpoint, p.modelName, err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ollama response body: %w", err)
	}

	// Check for Ollama API errors
	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Ollama API returned status %d for model '%s'. Body: %s", resp.StatusCode, p.modelName, string(respBody))
		// Try to parse a potential Ollama error message from the JSON response
		var ollamaErrorResp struct {
			Error string `json:"error"`
		}
		apiErrorMsg := string(respBody) // Default to the full body if parsing fails
		if json.Unmarshal(respBody, &ollamaErrorResp) == nil && ollamaErrorResp.Error != "" {
			apiErrorMsg = ollamaErrorResp.Error
		}
		return nil, fmt.Errorf("Ollama API error (Status %d) for model '%s': %s. (Ensure model is pulled and Ollama is running)", resp.StatusCode, p.modelName, apiErrorMsg)
	}

	// Parse Successful Response
	var ollamaSuccessResp ollamaEmbeddingResponse
	if err := json.Unmarshal(respBody, &ollamaSuccessResp); err != nil {
		log.Printf("ERROR: LocalEmbeddingProvider (Ollama) failed to unmarshal successful response for model '%s': %v. Body: %s", p.modelName, err, string(respBody))
		return nil, fmt.Errorf("failed to parse successful ollama response: %w", err)
	}

	if len(ollamaSuccessResp.Embedding) == 0 {
		log.Printf("WARNING: Ollama API returned successfully for model '%s' but with an empty embedding vector.", p.modelName)
		return nil, fmt.Errorf("Ollama API returned an empty embedding vector for model '%s'", p.modelName)
	}

	// log.Printf("LocalEmbeddingProvider: Generated Ollama embedding via '%s' (Dimensions: %d)", p.modelName, len(ollamaSuccessResp.Embedding))
	return ollamaSuccessResp.Embedding, nil
}

// ModelIdentifier returns a string uniquely identifying the Ollama model being used.
// This is crucial for caching and ensuring embeddings are compatible.
func (p *LocalEmbeddingProvider) ModelIdentifier() string {
	return p.modelIdentifier
}

// Destroy is a No-Op for the Ollama provider, as there are no external resources like
// C-bindings or ONNX sessions to clean up. The HTTP client will be garbage collected.
func (p *LocalEmbeddingProvider) Destroy() {
	// No specific resources to release for an HTTP client-based provider.
	// Ollama itself manages the lifecycle of its models.
	log.Printf("LocalEmbeddingProvider (Ollama, model: %s): Destroy called (No-Op).", p.modelName)
}
