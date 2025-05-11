// internal/llm/ollama_llm.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	OllamaDefaultLLMAPIEndpoint = "http://localhost:11434/api/generate"
	OllamaDefaultChatAPIEndpoint = "http://localhost:11434/api/chat"
	OllamaDefaultTagsAPIEndpoint = "http://localhost:11434/api/tags"
)

// OllamaGenerateRequest defines the JSON structure for the /api/generate request to Ollama.
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	// System string `json:"system,omitempty"` // Optional system prompt
	// KeepAlive string `json:"keep_alive,omitempty"` // Optional: controls how long the model stays loaded in memory
}

// OllamaGenerateResponse defines the JSON structure for a successful non-streaming /api/generate response.
type OllamaGenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	// Context   []int     `json:"context,omitempty"` // If you want to manage conversation context
	// Other fields like total_duration, load_duration, etc., can be added if needed.
}

// OllamaChatRequest defines the JSON structure for the /api/chat request.
type OllamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaChatMessage    `json:"messages"`
	Stream   bool                   `json:"stream"`
	// Options  map[string]interface{} `json:"options,omitempty"`
	// KeepAlive string `json:"keep_alive,omitempty"`
}

// OllamaChatMessage is a single message in a chat.
type OllamaChatMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
	// Images []string `json:"images,omitempty"` // For multimodal models
}

// OllamaChatResponse defines the JSON structure for a successful non-streaming /api/chat response.
type OllamaChatResponse struct {
	Model     string            `json:"model"`
	CreatedAt time.Time         `json:"created_at"`
	Message   OllamaChatMessage `json:"message"`
	Done      bool              `json:"done"`
	// Other fields like total_duration, load_duration, etc.
}


// GetOllamaCompletion sends a prompt to a local Ollama model using the /api/generate endpoint.
// modelTag is the specific Ollama model to use (e.g., "llama3", "mistral").
func GetOllamaCompletion(prompt, modelTag string) (string, error) {
	// Validate inputs to prevent crashes
	if modelTag == "" {
		return "", fmt.Errorf("Ollama model tag cannot be empty")
	}

	// Recover from panics to prevent application crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: Recovered from panic in GetOllamaCompletion: %v", r)
		}
	}()

	httpClient := &http.Client{Timeout: 300 * time.Second} // Extended timeout (5 minutes) for larger local models like Mistral

	requestPayload := OllamaGenerateRequest{
		Model:  modelTag,
		Prompt: prompt,
		Stream: false, // For synchronous response
	}

	bodyBytes, err := json.Marshal(requestPayload)
	if err != nil {
		log.Printf("ERROR: Ollama LLM: Failed to marshal request for model '%s': %v", modelTag, err)
		return "", fmt.Errorf("failed to marshal Ollama generate request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", OllamaDefaultLLMAPIEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("ERROR: Ollama LLM: Failed to create HTTP request for model '%s': %v", modelTag, err)
		return "", fmt.Errorf("failed to create Ollama generate HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// For larger models, show a progress indicator in the logs
	if strings.Contains(modelTag, "mistral") || strings.Contains(modelTag, "llama") {
		log.Printf("Sending request to Ollama for model '%s'. This may take several minutes for larger models...", modelTag)
	}
	
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Ollama LLM: Request failed for model '%s'. Is Ollama running at %s? Error: %v", modelTag, OllamaDefaultLLMAPIEndpoint, err)
		return "", fmt.Errorf("failed to connect to Ollama at %s (model: %s). Please ensure Ollama is running. Error: %w", OllamaDefaultLLMAPIEndpoint, modelTag, err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Ollama generate response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Ollama LLM API (/api/generate) returned status %d for model '%s'. Body: %s", resp.StatusCode, modelTag, string(respBody))
		var ollamaErrorResp struct {
			Error string `json:"error"`
		}
		apiErrorMsg := string(respBody)
		if json.Unmarshal(respBody, &ollamaErrorResp) == nil && ollamaErrorResp.Error != "" {
			apiErrorMsg = ollamaErrorResp.Error
		}
		return "", fmt.Errorf("Ollama LLM API error (Status %d) for model '%s': %s. (Ensure model is pulled: `ollama pull %s`)", resp.StatusCode, modelTag, apiErrorMsg, modelTag)
	}

	var ollamaSuccessResp OllamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaSuccessResp); err != nil {
		log.Printf("ERROR: Ollama LLM: Failed to unmarshal successful response for model '%s': %v. Body: %s", modelTag, err, string(respBody))
		// Try to extract any text content even if JSON parsing fails
		respStr := string(respBody)
		if len(respStr) > 0 {
			log.Printf("Attempting to salvage response content despite JSON parse error")
			// Try to extract response field from partial JSON
			if responseMatch := regexp.MustCompile(`"response"\s*:\s*"([^"]+)"`).FindStringSubmatch(respStr); len(responseMatch) > 1 {
				log.Printf("Salvaged response content from partial JSON")
				return responseMatch[1], nil
			}
			// If all else fails, return the raw response with a warning
			log.Printf("WARNING: Returning raw response due to JSON parse failure")
			return fmt.Sprintf("[Error parsing Ollama response. Raw content: %s]", respStr), nil
		}
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	if !ollamaSuccessResp.Done || ollamaSuccessResp.Response == "" {
		log.Printf("WARNING: Ollama LLM API for model '%s' returned successfully but with 'done: false' or empty response.", modelTag)
		// Return a more helpful message instead of an empty response
		if ollamaSuccessResp.Response == "" {
			return fmt.Sprintf("[Ollama model '%s' returned an empty response. Please try again or use a different model.]", modelTag), nil
		}
	}

	log.Printf("Ollama LLM: Received completion from '%s'", modelTag)
	return ollamaSuccessResp.Response, nil
}

// OllamaModelInfo describes a locally available Ollama model.
type OllamaModelInfo struct {
	Name       string    `json:"name"` // e.g., "llama2:latest"
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	// Digest string `json:"digest"` // Can be added if needed
	// Details ModelDetails `json:"details"` // Can be added if needed
}

// OllamaTagsResponse is the structure for the /api/tags response.
type OllamaTagsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}


// FetchOllamaModels retrieves the list of locally available Ollama models.
func FetchOllamaModels() ([]OpenRouterModel, error) { // Reusing OpenRouterModel for simplicity in frontend
	httpClient := &http.Client{Timeout: 10 * time.Second}
	log.Println("Fetching local Ollama models from:", OllamaDefaultTagsAPIEndpoint)

	req, err := http.NewRequestWithContext(context.Background(), "GET", OllamaDefaultTagsAPIEndpoint, nil)
	if err != nil {
		log.Printf("ERROR: Failed to create request to fetch Ollama models: %v", err)
		return nil, fmt.Errorf("failed to create request for Ollama models: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to fetch Ollama models. Is Ollama running? Error: %v", err)
		return nil, fmt.Errorf("failed to fetch Ollama models from %s: %w. Ensure Ollama is running", OllamaDefaultTagsAPIEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("ERROR: Ollama /api/tags request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("Ollama /api/tags request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read Ollama /api/tags response body: %v", err)
		return nil, fmt.Errorf("failed to read Ollama /api/tags response body: %w", err)
	}

	var apiResponse OllamaTagsResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("ERROR: Failed to unmarshal Ollama /api/tags response: %v. Body: %s", err, string(body))
		return nil, fmt.Errorf("failed to unmarshal Ollama /api/tags response: %w", err)
	}

	var llmModels []OpenRouterModel // Reusing the existing struct for frontend compatibility
	for _, ollamaModel := range apiResponse.Models {
		llmModels = append(llmModels, OpenRouterModel{
			ID:   ollamaModel.Name, // e.g., "llama3:latest"
			Name: ollamaModel.Name, // Display name can also be just the ID
		})
	}

	log.Printf("Successfully fetched %d local Ollama models.", len(llmModels))
	return llmModels, nil
}
