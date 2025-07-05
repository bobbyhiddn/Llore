package main

import (
	ragcontext "Llore/internal/context"
	"Llore/internal/database"
	"Llore/internal/embeddings"
	"Llore/internal/llm"
	"Llore/internal/vault"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	// Import official OpenAI and Gemini SDKs
	openai "github.com/openai/openai-go" // Official OpenAI SDK
	"github.com/openai/openai-go/option" // OpenAI SDK options
	"google.golang.org/genai"            // Corrected new SDK import path
)

// ProcessStoryResult holds the separated lists of new, updated, and existing entries.
type ProcessStoryResult struct {
	NewEntries      []database.CodexEntry `json:"newEntries"`
	UpdatedEntries  []database.CodexEntry `json:"updatedEntries"`
	ExistingEntries []database.CodexEntry `json:"existingEntries"`
}

// Embedding queue system to process embeddings sequentially
var (
	// Create a channel for embedding requests
	embeddingQueue     = make(chan embeddingRequest, 100)
	embeddingQueueOnce sync.Once
)

type embeddingRequest struct {
	entryID int64
	text    string
}

// GetEmbedding retrieves embedding providers for a codex entry
func (a *App) GetEmbedding(entryID int64) ([]string, error) {
	if a.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not initialized")
	}

	// Get all providers that have embeddings for this entry
	providers := make([]string, 0)

	// First verify the entry exists
	var exists bool
	err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM codex_entries WHERE id = ?)", entryID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if entry %d exists: %v", entryID, err)
		return nil, fmt.Errorf("failed to check entry existence: %w", err)
	}
	if !exists {
		log.Printf("Entry %d does not exist", entryID)
		return nil, fmt.Errorf("entry does not exist")
	}

	// Query database for all embeddings of this entry
	rows, err := a.db.Query(`
		SELECT DISTINCT vector_version 
		FROM codex_embeddings 
		WHERE codex_entry_id = ?`, entryID)
	if err != nil {
		log.Printf("Error querying embeddings for entry %d: %v", entryID, err)
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var vectorVersion string
		if err := rows.Scan(&vectorVersion); err != nil {
			log.Printf("Error scanning vector_version for entry %d: %v", entryID, err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		log.Printf("Found embedding with vector_version '%s' for entry %d", vectorVersion, entryID)
		providers = append(providers, vectorVersion)
	}

	if len(providers) == 0 {
		log.Printf("No embeddings found for entry %d", entryID)
		return nil, fmt.Errorf("no embeddings found for entry")
	}

	log.Printf("Found %d embeddings for entry %d: %v", len(providers), entryID, providers)
	return providers, nil
}

// GetAllEntries retrieves all codex entries from the SQLite database
func (a *App) GetAllEntries() ([]database.CodexEntry, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	rows, err := a.db.Query("SELECT id, name, type, content, created_at, updated_at FROM codex_entries ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []database.CodexEntry
	for rows.Next() {
		var e database.CodexEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Content, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// CreateEntry creates a new codex entry in the SQLite database
func (a *App) CreateEntry(name, entryType, content string) (database.CodexEntry, error) {
	if a.db == nil {
		return database.CodexEntry{}, fmt.Errorf("database is not initialized")
	}
	id, err := database.DBInsertEntry(a.db, name, entryType, content)
	if err != nil {
		return database.CodexEntry{}, fmt.Errorf("failed to insert entry: %w", err)
	}

	// Fetch the created entry to return it
	row := a.db.QueryRow("SELECT id, name, type, content, created_at, updated_at FROM codex_entries WHERE id = ?", id)
	var entry database.CodexEntry
	if err := row.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Content, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		log.Printf("Warning: Failed to fetch newly created entry (ID: %d) after insert: %v", id, err)
		entry.ID = id
		entry.Name = name
		entry.Type = entryType
		entry.Content = content
	}

	// Generate embedding for new entry using the queue system
	if a.embeddingService != nil && entry.ID != 0 {
		// Initialize worker if not already done
		a.initEmbeddingWorker()

		// Prepare text for embedding
		text := fmt.Sprintf("Name: %s\nType: %s\nContent: %s",
			entry.Name, entry.Type, entry.Content)

		// Send to channel with non-blocking behavior
		select {
		case embeddingQueue <- embeddingRequest{entryID: entry.ID, text: text}:
			log.Printf("Queued embedding generation for entry %d", entry.ID)
		default:
			log.Printf("Warning: Embedding queue full, skipping embedding for entry %d", entry.ID)
		}
	} else {
		log.Printf("Skipping embedding generation for new entry %d (service nil: %v, ID zero: %v)",
			entry.ID, a.embeddingService == nil, entry.ID == 0)
	}

	return entry, nil
}

// UpdateEntry updates an existing codex entry in the SQLite database
func (a *App) UpdateEntry(entry database.CodexEntry) error {
	if a.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	// Fetch current entry's content and type to compare
	var currentName, currentType, currentContent string
	err := a.db.QueryRow("SELECT name, type, content FROM codex_entries WHERE id = ?", entry.ID).Scan(&currentName, &currentType, &currentContent)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entry with ID %d not found for update", entry.ID)
		}
		return fmt.Errorf("failed to fetch current data for entry %d: %w", entry.ID, err)
	}

	// Determine if an update is actually needed (name, type, or content changed)
	nameChanged := entry.Name != currentName
	typeChanged := entry.Type != currentType
	contentChanged := entry.Content != currentContent

	if !nameChanged && !typeChanged && !contentChanged {
		log.Printf("Skipping update for entry %d ('%s'): no changes to name, type, or content.", entry.ID, entry.Name)
		return nil // No actual change, so no DB update or embedding regeneration needed
	}

	log.Printf("Updating entry %d ('%s'). Changes - Name: %v, Type: %v, Content: %v", entry.ID, entry.Name, nameChanged, typeChanged, contentChanged)
	err = database.DBUpdateEntry(a.db, entry) // DBUpdateEntry already sets updated_at
	if err != nil {
		return err // Return early if DB update failed
	}

	// Only queue embedding update if textual content (name, type, content) relevant to embedding has changed
	if (nameChanged || typeChanged || contentChanged) && a.embeddingService != nil && entry.ID != 0 {
		// Initialize worker if not already done
		a.initEmbeddingWorker()

		// Prepare text for embedding
		textForEmbedding := fmt.Sprintf("Name: %s\nType: %s\nContent: %s",
			entry.Name, entry.Type, entry.Content)

		// Send to channel with non-blocking behavior
		select {
		case embeddingQueue <- embeddingRequest{entryID: entry.ID, text: textForEmbedding}:
			log.Printf("Queued embedding update for entry %d ('%s') due to changes.", entry.ID, entry.Name)
		default:
			log.Printf("Warning: Embedding queue full, skipping embedding update for entry %d", entry.ID)
		}
	} else if !(nameChanged || typeChanged || contentChanged) {
		log.Printf("Skipping embedding update for entry %d ('%s'): content relevant to embedding did not change.", entry.ID, entry.Name)
	} else {
		log.Printf("Skipping embedding update for entry %d (service nil: %v, ID zero: %v)",
			entry.ID, a.embeddingService == nil, entry.ID == 0)
	}

	return nil
}

// DeleteEntry deletes a codex entry by ID using SQLite
func (a *App) DeleteEntry(id int64) error {
	if a.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	return database.DBDeleteEntry(a.db, id)
}

// GetCurrentVaultPath returns the path of the currently loaded vault
func (a *App) GetCurrentVaultPath() string {
	if a.db == nil {
		return ""
	}
	return a.dbPath
}

// App struct holds application state
type App struct {
	ctx              context.Context
	db               *sql.DB // Database connection handle
	dbPath           string  // Current database path
	geminiApiKey     string  // Store Gemini API key for embeddings
	embeddingService *embeddings.EmbeddingService
	contextBuilder   *ragcontext.ContextBuilder // Use alias
	promptBuilder    *llm.PromptBuilder
	// TODO: Add mutex if concurrent access to these services becomes an issue
}

// --- Vault Management ---

// SelectVaultFolder opens a dialog for the user to select an existing vault folder
func (a *App) SelectVaultFolder() (string, error) {
	return vault.SelectVaultFolder(a.ctx)
}

// CreateNewVault creates a new vault folder with the required structure
func (a *App) CreateNewVault(vaultName string) (string, error) {
	return vault.CreateNewVault(a.ctx, vaultName)
}

// SwitchVault switches to a different vault folder
func (a *App) SwitchVault(path string) error {
	// Verify the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access vault path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("specified path is not a directory: %s", path)
	}

	// Verify required subdirectories exist
	requiredDirs := []string{"Library", "Codex", "Chat"}
	for _, dir := range requiredDirs {
		subdir := filepath.Join(path, dir)
		if info, err := os.Stat(subdir); err != nil || !info.IsDir() {
			return fmt.Errorf("invalid vault structure: missing %s directory", dir)
		}
	}

	// Close previous DB connection if open
	if a.db != nil {
		database.DBClose(a.db)
	}

	// Initialize SQLite DB for this vault (store under Codex folder)
	codexDBPath := filepath.Join(path, "Codex", "codex_data.db")
	dbConn, err := database.DBInitialize(codexDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize codex database: %w", err)
	}

	a.db = dbConn
	a.dbPath = path

	// --- Ensure embeddings table exists ---
	_, err = a.db.Exec(`CREATE TABLE IF NOT EXISTS codex_embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		codex_entry_id INTEGER NOT NULL,
		embedding BLOB NOT NULL,
		vector_version TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY(codex_entry_id) REFERENCES codex_entries(id) ON DELETE CASCADE,
		UNIQUE (codex_entry_id, vector_version)
	)`)
	if err != nil {
		log.Printf("Warning: Failed to create index on embeddings table: %v", err)
	}

	// Get current config and initialize services
	currentConfig := llm.GetConfig()
	a.geminiApiKey = currentConfig.GeminiApiKey

	// Initialize LLM client for general LLM tasks
	llmProviderForChat := currentConfig.ActiveMode
	if llmProviderForChat == "local" { // Local embeddings might still use OpenRouter for LLM chat
		llmProviderForChat = "openrouter"
	}

	log.Printf("SwitchVault: Initializing LLM client for mode '%s' (derived for LLM tasks)", llmProviderForChat)
	if err := llm.Init(path); err != nil {
		log.Printf("Warning: Failed to initialize LLM package for vault '%s': %v", path, err)
	}

	// Initialize embedding services using the helper
	if err := a.initializeEmbeddingServices(currentConfig); err != nil {
		log.Printf("Warning: Failed to initialize embedding services during SwitchVault: %v", err)
	}

	// Load library files
	if err := a.refreshLibraryFiles(); err != nil {
		log.Printf("Warning: Failed to load library files: %v", err)
	}

	log.Printf("Successfully switched to vault: %s", path)
	return nil
}

// initializeEmbeddingServices sets up the embedding provider and related services.
// It should be called on vault switch and after settings are saved.
func (a *App) initializeEmbeddingServices(cfg llm.OpenRouterConfig) error {
	log.Printf("Initializing/Re-initializing embedding services for ActiveMode: '%s'", cfg.ActiveMode)

	var chosenProvider embeddings.EmbeddingProvider
	var errProv error

	switch cfg.ActiveMode {
	case "local":
		// Pure Ollama mode for both LLM and embeddings (offline mode)
		providerName := cfg.LocalEmbeddingModelName
		if providerName == "" {
			errProv = fmt.Errorf("Ollama model name (LocalEmbeddingModelName) not set in config for 'local' mode. Please set it in Settings")
		} else {
			chosenProvider, errProv = embeddings.NewLocalEmbeddingProvider(providerName)
		}
	case "hybrid":
		// Hybrid mode: OpenRouter LLM + Ollama embeddings
		providerName := cfg.LocalEmbeddingModelName
		if providerName == "" {
			errProv = fmt.Errorf("Ollama model name (LocalEmbeddingModelName) not set in config for 'hybrid' mode. Please set it in Settings")
		} else {
			chosenProvider, errProv = embeddings.NewLocalEmbeddingProvider(providerName)
		}
	case "gemini":
		if cfg.GeminiApiKey == "" {
			errProv = fmt.Errorf("Gemini API key missing for 'gemini' mode")
		} else {
			chosenProvider = embeddings.NewGeminiEmbeddingProvider(cfg.GeminiApiKey)
		}
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			errProv = fmt.Errorf("OpenAI API key missing for 'openai' mode")
		} else {
			// Use the new OpenAIEmbeddingProvider.
			// The default model "text-embedding-3-small" will be used as defined in the provider.
			chosenProvider, errProv = embeddings.NewOpenAIEmbeddingProvider(cfg.OpenAIAPIKey)
		}
	case "openrouter":
		// For 'openrouter' mode, embeddings might still use Gemini or local,
		// depending on your backend logic. Current code defaults to Gemini if key available.
		log.Println("ActiveMode is 'openrouter'. Embedding provider will be Gemini if API key is set, otherwise check local.")
		if cfg.GeminiApiKey != "" {
			chosenProvider = embeddings.NewGeminiEmbeddingProvider(cfg.GeminiApiKey)
		} else if cfg.LocalEmbeddingModelName != "" {
			log.Printf("Gemini key not found for 'openrouter' mode embeddings, falling back to local Ollama model: %s", cfg.LocalEmbeddingModelName)
			chosenProvider, errProv = embeddings.NewLocalEmbeddingProvider(cfg.LocalEmbeddingModelName)
		} else {
			errProv = fmt.Errorf("for 'openrouter' mode, either Gemini API key (for embeddings) or a Local Embedding Model Name must be set")
		}
	default: // Including empty string if ActiveMode not set
		log.Printf("Warning: ActiveMode '%s' is not explicitly handled or is empty. Attempting to default to local embeddings if LocalEmbeddingModelName is set.", cfg.ActiveMode)
		if cfg.LocalEmbeddingModelName != "" {
			chosenProvider, errProv = embeddings.NewLocalEmbeddingProvider(cfg.LocalEmbeddingModelName)
		} else {
			errProv = fmt.Errorf("no suitable embedding provider for ActiveMode '%s'. Ensure LocalEmbeddingModelName is set for local mode, or configure another mode", cfg.ActiveMode)
		}
	}

	if errProv != nil || chosenProvider == nil {
		log.Printf("CRITICAL: Failed to initialize embedding provider for mode '%s': %v. RAG/Embedding features will be disabled.", cfg.ActiveMode, errProv)
		a.embeddingService = nil
		a.contextBuilder = nil
		a.promptBuilder = nil
		return fmt.Errorf("failed to initialize embedding provider: %w. RAG/Embedding features disabled", errProv)
	}

	log.Printf("Embedding provider successfully initialized: %s", chosenProvider.ModelIdentifier())
	a.embeddingService = embeddings.NewEmbeddingService(a.db, chosenProvider)
	a.contextBuilder = ragcontext.NewContextBuilder(a.embeddingService)
	a.promptBuilder = llm.NewPromptBuilder(a.contextBuilder)

	// Load cache and process missing embeddings only if DB is available
	if a.db != nil {
		go func() {
			if err := a.GenerateMissingEmbeddings(); err != nil {
				log.Printf("Warning: Failed to generate missing embeddings in background: %v", err)
			}
		}()
	} else {
		log.Println("Database not available, skipping missing embedding generation for now.")
	}

	// Initialize LLM service
	if err := a.initializeLLM(cfg, a.dbPath); err != nil {
		log.Printf("Warning: Failed to initialize LLM service: %v", err)
	}

	// Load initial library data
	if err := a.refreshLibraryFiles(); err != nil {
		log.Printf("Warning: Failed to load library files: %v", err)
	}

	log.Printf("Successfully initialized embedding services for vault: %s", a.dbPath)
	return nil
}

// initializeLLM sets up the LLM service based on the current configuration
func (a *App) initializeLLM(cfg llm.OpenRouterConfig, vaultPath string) error {
	log.Printf("Initializing LLM services for ActiveMode: '%s'", cfg.ActiveMode)
	// No specific client instances to store on App struct for now,
	// clients will be created on-demand in GenerateLLMContent.

	// Common llm.Init for OpenRouter cache, which might be generally useful
	if err := llm.Init(vaultPath); err != nil {
		log.Printf("Warning: Failed to initialize LLM package for vault '%s': %v", vaultPath, err)
	}

	// Log API key status for each mode
	switch cfg.ActiveMode {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			log.Println("OpenAI API key not set. OpenAI LLM features will be disabled.")
		} else {
			log.Println("OpenAI API key is set. OpenAI LLM client will be created on demand.")
		}
	case "gemini":
		if cfg.GeminiApiKey == "" {
			log.Println("Gemini API key not set. Gemini LLM features will be disabled.")
		} else {
			log.Println("Gemini API key is set. Gemini LLM client will be created on demand.")
		}
	case "openrouter", "hybrid":
		if cfg.APIKey == "" { // APIKey here is OpenRouter API Key
			log.Println("OpenRouter API key not set. OpenRouter LLM features will be disabled.")
		} else {
			log.Println("OpenRouter API key is set.")
		}
	case "local":
		if cfg.LocalEmbeddingModelName == "" {
			log.Println("Ollama model name not set. Local Ollama LLM features will be disabled.")
		} else {
			log.Printf("Ollama model name is set to '%s' for local mode.", cfg.LocalEmbeddingModelName)
		}
	default:
		log.Printf("LLM for ActiveMode '%s' not specifically handled for client initialization.", cfg.ActiveMode)
	}
	return nil
}

// FetchOllamaModels returns a list of available local Ollama models.
func (a *App) FetchOllamaModels() ([]llm.OpenRouterModel, error) {
	log.Println("App.FetchOllamaModels called")
	models, err := llm.FetchOllamaModels()
	if err != nil {
		log.Printf("Error fetching Ollama models from app: %v", err)
		return nil, fmt.Errorf("failed to fetch local Ollama models: %w. Ensure Ollama is running and accessible", err)
	}
	return models, nil
}

// FetchOpenAIModels returns a list of available OpenAI models.
func (a *App) FetchOpenAIModels() ([]llm.OpenRouterModel, error) {
	// Get the OpenAI API key from the config
	cfg := llm.GetConfig()
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not set")
	}

	// Create a new OpenAI client with the API key
	client := openai.NewClient(
		option.WithAPIKey(cfg.OpenAIAPIKey),
	)

	// Fetch the list of models
	modelList, err := client.Models.List(context.Background())
	if err != nil {
		log.Printf("Error fetching OpenAI models: %v", err)
		return nil, fmt.Errorf("failed to fetch OpenAI models: %w", err)
	}

	// Filter for chat completion models (typically GPT models)
	var models []llm.OpenRouterModel
	for _, model := range modelList.Data {
		// Only include GPT models that support chat completions
		if strings.HasPrefix(model.ID, "gpt") && !strings.Contains(model.ID, "instruct") && !strings.Contains(model.ID, "vision") {
			models = append(models, llm.OpenRouterModel{
				ID:   model.ID,
				Name: model.ID, // Use ID as name as friendly names aren't always distinct or present
			})
		}
	}

	// Sort models by ID for consistency
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	log.Printf("Fetched %d OpenAI models", len(models))
	return models, nil
}

// GeminiAPIModelInfo holds detailed information about a model from the Gemini API.
// Based on the output from: https://generativelanguage.googleapis.com/v1beta/models/
type GeminiAPIModelInfo struct {
	Name                       string   `json:"name"`
	Version                    string   `json:"version"`
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	InputTokenLimit            int      `json:"inputTokenLimit"`
	OutputTokenLimit           int      `json:"outputTokenLimit"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	Temperature                float64  `json:"temperature,omitempty"`
	TopP                       float64  `json:"topP,omitempty"`
	TopK                       int      `json:"topK,omitempty"`
}

// GeminiAPIModelListResponse is the top-level structure for the API's model list response.
type GeminiAPIModelListResponse struct {
	Models        []GeminiAPIModelInfo `json:"models"`
	NextPageToken string               `json:"nextPageToken,omitempty"`
}

// FetchGeminiModels dynamically fetches generative models from the Gemini API.
// It filters for models that support "generateContent" as these are suitable
// for use with client.GenerativeModel() in GenerateLLMContent.
func (a *App) FetchGeminiModels() ([]llm.OpenRouterModel, error) {
	cfg := llm.GetConfig()
	if cfg.GeminiApiKey == "" {
		return nil, fmt.Errorf("Gemini API key not set in config")
	}

	// Construct the API URL
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", cfg.GeminiApiKey)

	log.Printf("Fetching Gemini models from: %s", strings.Replace(apiURL, cfg.GeminiApiKey, "[REDACTED_API_KEY]", 1))

	req, err := http.NewRequestWithContext(context.Background(), "GET", apiURL, nil)
	if err != nil {
		log.Printf("FetchGeminiModels: Error creating request: %v", err)
		return nil, fmt.Errorf("error creating request for Gemini models: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second} // Added timeout
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("FetchGeminiModels: Failed to fetch models from API: %v", err)
		return nil, fmt.Errorf("failed to fetch models from Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("FetchGeminiModels: API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("Gemini API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("FetchGeminiModels: Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read Gemini API response body: %w", err)
	}

	var apiResponse GeminiAPIModelListResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("FetchGeminiModels: Failed to unmarshal API response: %v. Body: %s", err, string(body))
		return nil, fmt.Errorf("failed to unmarshal Gemini API response: %w", err)
	}

	var openRouterModels []llm.OpenRouterModel
	for _, model := range apiResponse.Models {
		isGenerativeModel := false
		for _, method := range model.SupportedGenerationMethods {
			if method == "generateContent" { // We need this for client.GenerativeModel()
				isGenerativeModel = true
				break
			}
		}

		if isGenerativeModel {
			// The Go SDK's GenerativeModel() and EmbeddingModel() functions expect the model ID
			// without the "models/" prefix (e.g., "gemini-1.5-pro-latest").
			sdkModelID := strings.TrimPrefix(model.Name, "models/")

			// Ensure the prefix was actually removed and we have a non-empty ID
			if sdkModelID != "" && sdkModelID != model.Name {
				openRouterModels = append(openRouterModels, llm.OpenRouterModel{
					ID:   sdkModelID,
					Name: model.DisplayName,
				})
			} else {
				log.Printf("FetchGeminiModels: Skipping model with potentially malformed or unhandled ID format: '%s' (DisplayName: '%s')", model.Name, model.DisplayName)
			}
		}
	}

	// Sort models by display name for consistent UI presentation
	sort.Slice(openRouterModels, func(i, j int) bool {
		return openRouterModels[i].Name < openRouterModels[j].Name
	})

	if len(openRouterModels) == 0 {
		log.Println("FetchGeminiModels: No generative models (supporting 'generateContent') found after parsing API response.")
	}

	log.Printf("FetchGeminiModels: Successfully fetched and filtered %d generative models from Gemini API.", len(openRouterModels))
	return openRouterModels, nil
}

// GenerateLLMContent is a new wrapper to dispatch to the correct LLM provider
func (a *App) GenerateLLMContent(prompt, modelID string) (string, error) {
	cfg := llm.GetConfig()
	log.Printf("GenerateLLMContent called for mode: %s, model: %s", cfg.ActiveMode, modelID)

	switch cfg.ActiveMode {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return "", fmt.Errorf("OpenAI API key not set. Cannot use OpenAI LLM")
		}
		client := openai.NewClient(
			option.WithAPIKey(cfg.OpenAIAPIKey),
		)
		// If modelID is empty, choose a default from available OpenAI models
		effectiveModelID := modelID
		if effectiveModelID == "" {
			effectiveModelID = "gpt-3.5-turbo" // Using string literal for safety with openai-go v0.1.0-beta.10
			log.Printf("No modelID provided for OpenAI, defaulting to %s", effectiveModelID)
		}
		log.Printf("Sending prompt to OpenAI model %s", effectiveModelID)

		// Create a chat completion using the official SDK
		completion, err := client.Chat.Completions.New(
			context.Background(),
			openai.ChatCompletionNewParams{
				Model: effectiveModelID,
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage(prompt),
				},
			},
		)
		if err != nil {
			return "", fmt.Errorf("OpenAI chat completion error: %w", err)
		}
		if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
			return "", fmt.Errorf("OpenAI returned no choices or empty content")
		}
		return completion.Choices[0].Message.Content, nil

	case "gemini":
		if cfg.GeminiApiKey == "" {
			return "", fmt.Errorf("Gemini API key not set. Cannot use Gemini LLM")
		}
		effectiveModelID := modelID
		if effectiveModelID == "" {
			effectiveModelID = "gemini-1.0-pro" // Default, or use "gemini-1.5-pro-latest"
			log.Printf("No modelID provided for Gemini, defaulting to %s", effectiveModelID)
		}

		genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: cfg.GeminiApiKey})
		if err != nil {
			return "", fmt.Errorf("failed to create Gemini client: %w", err)
		}

		// Use client.Models.GenerateContent directly
		// The prompt is already a string, so genai.Text(prompt) is appropriate.
		resp, err := genaiClient.Models.GenerateContent(context.Background(), effectiveModelID, genai.Text(prompt), nil) // Passing nil for config as per simple examples
		if err != nil {
			return "", fmt.Errorf("failed to generate content with Gemini: %w", err)
		}

		// Process and return the response based on example_test.go patterns
		if resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
			part := resp.Candidates[0].Content.Parts[0]
			if part != nil {
				// Assuming the part is text and *genai.Part has a Text field of type string.
				// The type genai.Text is used for constructing input parts.
				// For output, the Part struct likely holds the text directly.
				// If 'Text' is not the correct field, the compiler will tell us.
				// Based on examples like `result.Candidates[0].Content.Parts[0].Text` in example_test.go for streams.
				// For a non-streaming GenerateContent, the structure should be similar for a text part.
				textValue := part.Text // Accessing the Text field directly.
				if textValue != "" {   // Check if the text content is not empty
					log.Printf("Gemini Raw Response Part: %s", textValue)
					return textValue, nil
				} else {
					log.Printf("Gemini response part.Text was empty")
				}
			} else {
				log.Printf("Gemini response part was nil")
			}
		}
		return "", fmt.Errorf("gemini response was empty or not in expected format")

	case "openrouter", "hybrid": // OpenRouter for LLM in both openrouter and hybrid modes
		if cfg.APIKey == "" { // This is OpenRouter API key
			return "", fmt.Errorf("OpenRouter API key not set. Cannot use OpenRouter LLM")
		}
		if modelID == "" {
			return "", fmt.Errorf("no modelID provided for OpenRouter LLM mode")
		}
		return llm.GetOpenRouterCompletion(prompt, modelID)
	case "local": // Ollama for LLM in pure local mode
		log.Printf("Using local Ollama model '%s' for LLM content generation.", modelID)
		if modelID == "" {
			return "", fmt.Errorf("no modelID provided for Local Ollama LLM mode")
		}

		// Add warning for larger models that might take longer
		if strings.Contains(modelID, "mistral") || strings.Contains(modelID, "llama") {
			log.Printf("WARNING: Using a larger model (%s) which may take longer to respond. Timeout set to 5 minutes.", modelID)
		}

		// Add extra error handling to prevent application crashes
		response, err := llm.GetOllamaCompletion(prompt, modelID)
		if err != nil {
			log.Printf("ERROR: Failed to get Ollama completion: %v", err)
			return fmt.Sprintf("[Error: Unable to get response from Ollama model '%s'. Please ensure Ollama is running and the model is pulled. Error details: %v]", modelID, err), nil
		}
		return response, nil

	default:
		return "", fmt.Errorf("unsupported LLM ActiveMode: %s. Please configure in Settings", cfg.ActiveMode)
	}
}

// ListLibraryFiles returns a list of files in the vault's Library folder
func (a *App) ListLibraryFiles() ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}
	libraryPath := filepath.Join(a.dbPath, "Library")
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Library directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// refreshLibraryFiles updates the cached list of library files
func (a *App) refreshLibraryFiles() error {
	libraryPath := filepath.Join(a.dbPath, "Library")
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		return fmt.Errorf("failed to read Library directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() { // Only include files, not subdirectories
			files = append(files, entry.Name())
		}
	}

	// a.libraryFiles (removed, now handled by refreshLibraryFiles) = files
	return nil
}

// isValidFilename checks if a filename contains only valid characters
func isValidFilename(filename string) bool {
	for _, r := range filename {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == ' ' || r == '.') {
			return false
		}
	}
	return true && filename != ""
}

// ImportStoryTextAndFile saves story text to a file and processes it for codex entries
// If providedFilename is not empty, it will be used instead of generating a filename
func (a *App) ImportStoryTextAndFile(text string, providedFilename string) (ProcessStoryResult, error) {
	if a.db == nil {
		return ProcessStoryResult{}, fmt.Errorf("no vault is currently loaded")
	}

	filename := "story.txt"

	// Use provided filename if it exists
	if providedFilename != "" {
		// Use the provided filename directly if it's already safe
		if isValidFilename(providedFilename) {
			filename = providedFilename
		} else {
			// Clean the provided filename to ensure it's safe
			filename = strings.Map(func(r rune) rune {
				if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == ' ' || r == '.' {
					return r
				}

				return '_'
			}, providedFilename)
			filename = strings.TrimSpace(filename)
		}
	} else {
		// Generate a filename based on the first line if no filename provided
		firstLine := strings.Split(strings.TrimSpace(text), "\n")[0]
		if len(firstLine) > 0 {
			// Clean the filename
			filename = strings.Map(func(r rune) rune {
				if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == ' ' {
					return r
				}
				return '_'
			}, firstLine)
			filename = strings.TrimSpace(filename) + ".txt"
		}
	}

	// Ensure Library directory exists
	libraryDir := filepath.Join(a.dbPath, "Library")
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		return ProcessStoryResult{}, fmt.Errorf("failed to create Library directory: %w", err)
	}
	// Save the file
	filePath := filepath.Join(libraryDir, filename)
	if err := os.WriteFile(filePath, []byte(text), 0644); err != nil {
		return ProcessStoryResult{}, fmt.Errorf("failed to save story file: %w", err)
	}

	// Update library files cache
	if err := a.refreshLibraryFiles(); err != nil {
		log.Printf("Warning: Failed to refresh library files after import: %v", err)
	}

	// Process the story into codex entries
	result, err := a.ProcessStory(text) // This already returns ProcessStoryResult
	if err != nil {
		return ProcessStoryResult{}, fmt.Errorf("failed to process story into codex entries: %w", err)
	}

	// Log summary of the import operation
	newCount := len(result.NewEntries)
	updatedCount := len(result.UpdatedEntries)
	log.Printf("Import of file '%s' complete. Processed entries: %d new, %d updated.",
		filename, newCount, updatedCount)

	if updatedCount > 0 {
		log.Printf("Note: %d entries already existed and were updated with merged content.", updatedCount)
	}

	// Return the ProcessStoryResult directly
	return result, nil
}

// ReadLibraryFile reads the content of a file from the vault's Library folder
func (a *App) ReadLibraryFile(filename string) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault is currently loaded")
	}
	filePath := filepath.Join(a.dbPath, "Library", filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return string(content), nil
}

// SaveLibraryFile writes content to a specified file in the vault's Library folder
func (a *App) SaveLibraryFile(filename string, content string) error {
	if a.db == nil {
		return fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation to prevent path traversal
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return fmt.Errorf("invalid library filename")
	}

	filePath := filepath.Join(a.dbPath, "Library", filename)

	// Check if the file exists before writing
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Warning: Attempting to save to non-existent file: %s", filePath)
		// Decide if we should allow creating new files this way or return an error
		// For now, let's allow it, mirroring os.WriteFile behavior
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v", filePath, err)
		return fmt.Errorf("failed to write library file %s: %w", filename, err)
	}

	log.Printf("Successfully saved library file: %s", filePath)
	return nil
}

// ListTemplates returns a list of .md files in the vault's Templates folder
func (a *App) ListTemplates() ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}
	templatesPath := filepath.Join(a.dbPath, "Templates")
	entries, err := os.ReadDir(templatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Templates directory does not exist, creating it: %s", templatesPath)
			if err := os.MkdirAll(templatesPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create Templates directory: %w", err)
			}
			return []string{}, nil // Return empty list after creating
		}
		return nil, fmt.Errorf("failed to read Templates directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// SaveTemplate writes content to a specified file in the vault's Templates folder
func (a *App) SaveTemplate(filename string, content string) error {
	if a.db == nil {
		return fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return fmt.Errorf("invalid template filename")
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	filePath := filepath.Join(a.dbPath, "Templates", filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write template file %s: %w", filename, err)
	}
	log.Printf("Successfully saved template file: %s", filePath)
	return nil
}

// WeaveEntryIntoText is the core "Llore-weaving" function.
func (a *App) WeaveEntryIntoText(droppedEntry database.CodexEntry, documentText string, cursorPosition int, templateType string) (string, error) {
	log.Printf("Weaving entry '%s' into a '%s' document.", droppedEntry.Name, templateType)

	var goal string
	// Determine the AI's goal based on context
	switch templateType {
	case "character-sheet":
		if droppedEntry.Type == "Character" {
			goal = fmt.Sprintf("The user dropped Character '%s' onto this character sheet. Generate a new 'Relationships' section describing a plausible connection (e.g., friend, family, rival, mentor) between the sheet's character and the dropped character.", droppedEntry.Name)
		} else { // Handle Artifact, Location, etc.
			goal = fmt.Sprintf("The user dropped the %s '%s' onto this character sheet. Generate a new section describing how the character acquired, uses, or is connected to this %s.", droppedEntry.Type, droppedEntry.Name, droppedEntry.Type)
		}
	case "chapter":
		goal = fmt.Sprintf("The user dropped the %s '%s' into this narrative scene. Weave its introduction or a mention of it naturally into the story at the cursor position. It could be a character noticing it, interacting with it, or thinking about it.", droppedEntry.Type, droppedEntry.Name)
	default: // Generic fallback for blank documents or unknown templates
		goal = fmt.Sprintf("The user dropped the %s '%s' into their document. Based on the surrounding text, intelligently integrate this information. This could be a new descriptive sentence, a new paragraph, or an expansion of an existing idea.", droppedEntry.Type, droppedEntry.Name)
	}

	// Prepare the document with a cursor marker
	if cursorPosition > len(documentText) {
		cursorPosition = len(documentText)
	}
	docWithCursor := documentText[:cursorPosition] + "<<CURSOR>>" + documentText[cursorPosition:]

	// Construct the master prompt
	prompt := fmt.Sprintf(
		"SYSTEM: You are an expert fiction writing assistant. Your task is to seamlessly weave a new codex entry into an existing draft. Your response must be ONLY the text to be inserted. Do not include explanations.\n\n"+
			"GOAL: %s\n\n"+
			"DROPPED ENTRY DETAILS:\n- Name: %s\n- Type: %s\n- Content: %s\n\n"+
			"DOCUMENT CONTEXT (with cursor position):\n---\n%s\n---\n\n"+
			"GENERATED TEXT TO INSERT:",
		goal,
		droppedEntry.Name,
		droppedEntry.Type,
		droppedEntry.Content,
		docWithCursor,
	)

	// Use the existing RAG function to get the completion
	cfg := llm.GetConfig()
	modelID := cfg.ChatModelID // Or a more powerful model if desired for this task
	if modelID == "" {
		return "", fmt.Errorf("no chat model configured in settings")
	}

	return a.GetAIResponseWithContext(prompt, modelID)
}

// --- Chat Log Management ---

// ListChatLogs returns a list of .json files in the vault's Chat folder
func (a *App) ListChatLogs() ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}
	chatPath := filepath.Join(a.dbPath, "Chat")
	entries, err := os.ReadDir(chatPath)
	if err != nil {
		// If the chat directory doesn't exist, return an empty list instead of an error
		if os.IsNotExist(err) {
			log.Printf("Chat directory does not exist, returning empty list: %s", chatPath)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read Chat directory '%s': %w", chatPath, err)
	}

	logs := make([]string, 0)
	for _, entry := range entries {
		// Only include files that end with .json AND are NOT the cache file
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			logs = append(logs, entry.Name())
		}
	}
	return logs, nil
}

// LoadChatLog reads a JSON chat log file from the vault's Chat folder
func (a *App) LoadChatLog(filename string) ([]ChatMessage, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation to prevent path traversal
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return nil, fmt.Errorf("invalid chat log filename")
	}

	chatFilePath := filepath.Join(a.dbPath, "Chat", filename)
	content, err := os.ReadFile(chatFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chat log file '%s': %w", filename, err)
	}

	var messages []ChatMessage
	if err := json.Unmarshal(content, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse chat log file '%s': %w", filename, err)
	}

	return messages, nil
}

// SaveChatLog saves chat messages to a JSON file in the vault's Chat folder
func (a *App) SaveChatLog(filename string, messages []ChatMessage) error {
	if a.db == nil {
		return fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation to prevent path traversal
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return fmt.Errorf("invalid chat log filename")
	}
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json" // Ensure .json extension
	}

	chatPath := filepath.Join(a.dbPath, "Chat")
	// Ensure Chat directory exists
	if err := os.MkdirAll(chatPath, 0755); err != nil {
		return fmt.Errorf("failed to create Chat directory '%s': %w", chatPath, err)
	}

	chatFilePath := filepath.Join(chatPath, filename)
	content, err := json.MarshalIndent(messages, "", "  ") // Pretty print JSON
	if err != nil {
		return fmt.Errorf("failed to marshal chat messages for '%s': %w", filename, err)
	}

	if err := os.WriteFile(chatFilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write chat log file '%s': %w", filename, err)
	}

	log.Printf("Saved chat log to: %s", chatFilePath)
	return nil
}

// DeleteChatLog deletes a chat log file from the vault's Chat folder
func (a *App) DeleteChatLog(filename string) error {
	if a.db == nil {
		return fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation to prevent path traversal
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return fmt.Errorf("invalid chat log filename")
	}

	chatFilePath := filepath.Join(a.dbPath, "Chat", filename)
	// Check if file exists before attempting to delete
	if _, err := os.Stat(chatFilePath); os.IsNotExist(err) {
		return fmt.Errorf("chat log file '%s' does not exist", filename)
	}

	if err := os.Remove(chatFilePath); err != nil {
		return fmt.Errorf("failed to delete chat log file '%s': %w", filename, err)
	}

	log.Printf("Deleted chat log: %s", chatFilePath)
	return nil
}

// ChatMessage represents a single message in a chat log.
type ChatMessage struct {
	Sender string `json:"sender"` // "user" or "ai"
	Text   string `json:"text"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. Initializes config, LLM client, and DB.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("Llore application starting up...")

	// Load OpenRouter config (which now reads from ~/.llore/config.json)
	if err := llm.LoadOpenRouterConfig(); err != nil {
		// Log warning but don't necessarily fail startup, user might add key later
		log.Printf("Warning: Failed to load OpenRouter configuration: %v. API key might be missing.", err)
	}

	log.Println("App startup complete.")
}

// shutdown is called when the app terminates.
func (a *App) shutdown(ctx context.Context) {
	log.Println("Llore application shutting down...")
}

// ProcessStory sends a prompt to the LLM and processes the structured response.
func (a *App) ProcessStory(storyText string) (ProcessStoryResult, error) {
	// Prepare the prompt based on content type
	simplifiedPrompt := fmt.Sprintf("Analyze the following text and extract key entities (characters, locations, items, concepts) and their descriptions. Be thorough and try to identify anywhere from 3 to 15 distinct entities. Format the output as a JSON array where each object has 'name', 'type', and 'content' fields. Types should be one of: Character, Location, Item, Concept. Do not include any text before or after the JSON array. Example: [{\"name\": \"Sir Reginald\", \"type\": \"Character\", \"content\": \"A brave knight known for his shiny armor.\"}]. Text to analyze:\n\n%s", storyText)

	log.Println("Sending prompt for story processing...")
	cfg := llm.GetConfig()
	processingModelID := cfg.StoryProcessingModelID
	if processingModelID == "" {
		switch cfg.ActiveMode {
		case "openai":
			processingModelID = "gpt-3.5-turbo" // Using string literal
			log.Printf("Warning: StoryProcessingModelID not set for OpenAI mode, using default '%s'", processingModelID)
		case "gemini":
			processingModelID = "gemini-1.0-pro"
			log.Printf("Warning: StoryProcessingModelID not set for Gemini mode, using default '%s'", processingModelID)
		case "openrouter", "local":
			fallthrough
		default:
			processingModelID = "anthropic/claude-3.5-sonnet"
			log.Printf("Warning: StoryProcessingModelID not set for mode '%s', using OpenRouter default '%s'", cfg.ActiveMode, processingModelID)
		}
	}
	log.Printf("Using model '%s' for processing story (ActiveMode: %s)", processingModelID, cfg.ActiveMode)

	llmResponse, err := a.GenerateLLMContent(simplifiedPrompt, processingModelID)
	if err != nil || llmResponse == "" {
		log.Printf("Primary model '%s' failed or returned empty response (err: %v). Attempting fallbacks...", processingModelID, err)

		fallbackModels := []string{
			"google/gemini-2.5-pro",                     // Proven stable free model
			"openai/gpt-3.5-turbo",                       // OpenAI fallback (may require key)
			"mistralai/mistral-small-3.1-24b-instruct:free", // Another free OpenRouter model
		}

		for _, fm := range fallbackModels {
			log.Printf("Trying fallback model: %s", fm)
			resp, ferr := a.GenerateLLMContent(simplifiedPrompt, fm)
			if ferr == nil && resp != "" {
				log.Printf("Successfully generated response using fallback model: %s", fm)
				llmResponse = resp
				err = nil
				processingModelID = fm // Update for logging downstream
				break
			} else {
				log.Printf("Fallback model '%s' failed (err: %v).", fm, ferr)
			}
		}

		if err != nil || llmResponse == "" {
			log.Printf("All fallback attempts failed. Last error: %v", err)
			return ProcessStoryResult{}, fmt.Errorf("failed to get LLM response after fallbacks: %w", err)
		}
	}

	log.Println("Received LLM response, attempting to parse JSON...")
	// Clean up response - remove code block markers if present
	llmResponse = strings.TrimSpace(llmResponse)
	if strings.HasPrefix(llmResponse, "```") {
		// Find the end of the opening code block
		newlineIndex := strings.Index(llmResponse, "\n")
		if newlineIndex > 0 {
			// Remove opening code block line
			llmResponse = llmResponse[newlineIndex+1:]
		}
		// Remove closing code block
		llmResponse = strings.TrimSuffix(llmResponse, "```")
	}
	llmResponse = strings.TrimSpace(llmResponse)

	// Attempt to parse the structured response (expecting a JSON array of objects)
	var llmEntries []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}

	err = json.Unmarshal([]byte(llmResponse), &llmEntries)
	if err != nil {
		// Handle cases where the response isn't valid JSON or the expected structure
		log.Printf("Warning: LLM response was not a valid JSON array of entries. Error: %v", err)
		log.Printf("LLM Response Text:\n%s", llmResponse)
		// Fallback: Treat the entire response as the content of a single entry
		// Remove unused fallbackEntry declaration
		// Fallback: Treat the entire response as the content of a single entry
		log.Println("Falling back to creating a single entry with the raw LLM response.")
		fallbackEntry := struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
		}{
			Name:    "Generated Entry",
			Type:    "Generated",
			Content: llmResponse,
		}
		llmEntries = append(llmEntries, fallbackEntry)
	}

	// Process the structured entries
	now := time.Now().UTC()
	var newEntriesResult []database.CodexEntry     // Initialize new slice
	var updatedEntriesResult []database.CodexEntry // Initialize updated slice
	for _, llmEntry := range llmEntries {
		if llmEntry.Name == "" {
			log.Println("Warning: Skipping entry with empty name from LLM response.")
			continue
		}

		// Check if an entry with this name already exists
		var existingEntry database.CodexEntry
		row := a.db.QueryRow("SELECT id, name, type, content, created_at, updated_at FROM codex_entries WHERE name = ?", llmEntry.Name)
		err := row.Scan(&existingEntry.ID, &existingEntry.Name, &existingEntry.Type, &existingEntry.Content, &existingEntry.CreatedAt, &existingEntry.UpdatedAt)

		if err == nil { // Entry exists, attempt to merge and update
			log.Printf("Existing entry found for '%s' (ID: %d). Attempting to merge content.", llmEntry.Name, existingEntry.ID)

			newContentType := llmEntry.Type
			if newContentType == "" || newContentType == "Generated" {
				newContentType = existingEntry.Type // Keep existing type if new one is generic
				if newContentType == "" {
					newContentType = "Concept" // Default if both are empty
				}
			}

			// Use the refined MergeEntryContentDirect instead of MergeEntryContentWithRAG
			mergedContent, mergeErr := a.MergeEntryContentDirect(existingEntry, llmEntry.Content, processingModelID)
			if mergeErr != nil {
				// This error is from MergeEntryContentDirect setup, not the LLM call (which has its own fallback)
				log.Printf("Critical error in MergeEntryContentDirect function for '%s': %v. Appending new info as failsafe.", existingEntry.Name, mergeErr)
				mergedContent = existingEntry.Content + "\n\n--- (New Information from Import - Merge Function Error) ---\n" + llmEntry.Content
			}

			// Check if content or type actually changed before updating
			contentChanged := strings.TrimSpace(mergedContent) != strings.TrimSpace(existingEntry.Content)
			typeChanged := newContentType != existingEntry.Type

			if !contentChanged && !typeChanged {
				log.Printf("No effective change for entry '%s' after merge and type check. Skipping update.", existingEntry.Name)
				// If you want to inform the frontend it was "found but not changed", you might add it to a different list in ProcessStoryResult.
				// For now, we simply don't add it to updatedEntriesResult.
				continue
			}

			log.Printf("Content or type changed for entry '%s'. Proceeding with update. Content changed: %v, Type changed: %v", existingEntry.Name, contentChanged, typeChanged)

			updatedEntry := database.CodexEntry{
				ID:        existingEntry.ID,
				Name:      existingEntry.Name, // Name doesn't change during this update
				Type:      newContentType,
				Content:   mergedContent,
				CreatedAt: existingEntry.CreatedAt, // Preserve original creation date
				UpdatedAt: now.Format(time.RFC3339),
			}

			// Call UpdateEntry, which now checks for actual changes before updating DB and queueing embedding
			if err := a.UpdateEntry(updatedEntry); err != nil {
				log.Printf("Warning: Failed to update entry '%s' (ID: %d) in database: %v", updatedEntry.Name, updatedEntry.ID, err)
				continue
			}
			updatedEntriesResult = append(updatedEntriesResult, updatedEntry)

		} else if err == sql.ErrNoRows { // Entry does not exist
			// Create new entry in DB
			savedEntry, err := a.CreateEntry(llmEntry.Name, llmEntry.Type, llmEntry.Content) // Use existing CreateEntry which queues embedding
			if err != nil {
				log.Printf("Warning: Failed to create entry '%s' in database: %v", llmEntry.Name, err)
				continue
			}
			newEntriesResult = append(newEntriesResult, savedEntry) // Add to new list
		} else { // Other DB error during check
			log.Printf("Error checking for existing entry '%s': %v", llmEntry.Name, err)
			continue
		}
	}

	log.Printf("Story processing complete. Created %d new entries, updated %d existing entries.", len(newEntriesResult), len(updatedEntriesResult))

	return ProcessStoryResult{NewEntries: newEntriesResult, UpdatedEntries: updatedEntriesResult}, nil // Return the struct
}

// GenerateOpenRouterContent calls OpenRouter with prompt/model and returns the response.
func (a *App) GenerateOpenRouterContent(prompt, model string) (string, error) {
	if err := llm.LoadOpenRouterConfig(); err != nil { // Ensure config is loaded (or attempt reload)
		return "", fmt.Errorf("failed to load OpenRouter configuration: %w", err)
	}
	return llm.GetOpenRouterCompletion(prompt, model)
}

// GenerateMissingEmbeddings ensures all entries have embeddings
func (a *App) GenerateMissingEmbeddings() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if a.embeddingService == nil || a.embeddingService.GetProvider() == nil { // Ensure provider is available
		log.Println("Skipping GenerateMissingEmbeddings: Embedding service or provider not fully initialized.")
		return nil // Not an error, just skipping
	}

	currentProviderIdentifier := a.embeddingService.ModelIdentifier()
	log.Printf("Starting background check for missing embeddings for provider: %s", currentProviderIdentifier)

	// Find entries that do not have an embedding for the current provider
	rows, err := a.db.Query(`
        SELECT e.id, e.name, e.type, e.content
        FROM codex_entries e
        LEFT JOIN codex_embeddings em ON e.id = em.codex_entry_id AND em.vector_version = ?
        WHERE em.id IS NULL
    `, currentProviderIdentifier)
	if err != nil {
		return fmt.Errorf("failed to query entries missing embeddings: %w", err)
	}
	defer rows.Close()

	var processedCount int
	var entriesToProcess []database.CodexEntry // Collect entries first

	for rows.Next() {
		var entry database.CodexEntry
		if err := rows.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Content); err != nil {
			log.Printf("Warning: Failed to scan entry during missing embedding check: %v", err)
			continue
		}
		entriesToProcess = append(entriesToProcess, entry)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Warning: Error iterating rows for missing embeddings: %v", err)
		// Continue processing the entries found so far
	}

	if len(entriesToProcess) == 0 {
		log.Println("No missing embeddings found.")
		return nil
	}

	log.Printf("Found %d entries missing embeddings. Processing...", len(entriesToProcess))

	// Process entries sequentially to avoid overwhelming the API
	for _, entry := range entriesToProcess {
		// Create text for embedding
		text := fmt.Sprintf("Name: %s\nType: %s\nContent: %s",
			entry.Name, entry.Type, entry.Content)

		// Generate embedding
		embedding, err := a.embeddingService.CreateEmbedding(text)
		if err != nil {
			log.Printf("Warning: Failed to create embedding for entry %d: %v", entry.ID, err)
			// Consider adding a delay or backoff here if API errors are frequent
			time.Sleep(1 * time.Second) // Simple delay
			continue                    // Skip this entry
		}

		// Save embedding
		if err := a.embeddingService.SaveEmbedding(entry.ID, embedding); err != nil {
			log.Printf("Warning: Failed to save embedding for entry %d: %v", entry.ID, err)
			continue // Skip this entry
		}

		processedCount++
		log.Printf("Generated embedding for entry %d", entry.ID)
		time.Sleep(500 * time.Millisecond) // Add a small delay between API calls
	}

	log.Printf("Finished generating missing embeddings. Processed %d entries.", processedCount)
	return nil
}

// GetAIResponseWithContext uses the chat model ID from settings.
func (a *App) GetAIResponseWithContext(query string, modelID string) (string, error) {
	// modelID here is expected to be cfg.ChatModelID
	if modelID == "" {
		cfg := llm.GetConfig()
		modelID = cfg.ChatModelID // Ensure we use the configured chat model
		if modelID == "" {        // If still empty, apply mode-specific default
			switch cfg.ActiveMode {
			case "openai":
				modelID = "gpt-3.5-turbo" // Using string literal
			case "gemini":
				modelID = "gemini-1.0-pro"
			case "openrouter", "local":
				modelID = "openai/gpt-3.5-turbo" // A common OpenRouter default
			default:
				return "", fmt.Errorf("chat model not configured and no default for mode %s", cfg.ActiveMode)
			}
			log.Printf("GetAIResponseWithContext: modelID was empty, defaulted to %s for mode %s", modelID, cfg.ActiveMode)
		}
	}

	if a.promptBuilder == nil {
		log.Println("Warning: GetAIResponseWithContext called but prompt builder not initialized. Falling back to simple generation.")
		return a.GenerateLLMContent(query, modelID) // USE NEW METHOD
	}

	log.Printf("Building prompt with context for query: %s", query)
	prompt, err := a.promptBuilder.BuildPromptWithContext(query)
	if err != nil {
		log.Printf("Error building prompt with context: %v. Falling back to simple prompt.", err)
		prompt = query
	}

	log.Printf("Sending RAG prompt (length: %d) to model: %s", len(prompt), modelID)
	return a.GenerateLLMContent(prompt, modelID) // USE NEW METHOD
}

// MergeEntryContentDirect merges existing entry content with new content using direct AI prompting without RAG
func (a *App) MergeEntryContentDirect(existingEntry database.CodexEntry, newContent string, model string) (string, error) {
	// Simple check to see if new content is already present.
	// More sophisticated diffing could be used, but this is a quick win.
	if strings.Contains(strings.ToLower(existingEntry.Content), strings.ToLower(newContent)) {
		log.Printf("New content for '%s' appears to be already contained in existing content. Skipping AI merge.", existingEntry.Name)
		return existingEntry.Content, nil // Return original content, no actual merge needed by AI
	}

	// Construct a more explicit prompt for intelligent merging
	mergePrompt := fmt.Sprintf(
		"You are an expert editor updating a codex entry. Your task is to intelligently integrate the \"New Information\" into the \"Existing Content\" to create a single, coherent, and improved entry. Preserve all key details from both. Avoid redundancy. Ensure the final merged content flows naturally and maintains a consistent tone.\n\n"+
			"VERY IMPORTANT INSTRUCTIONS:\n"+
			"1. Output ONLY the final merged content text. No conversational filler, explanations, or meta-text.\n"+
			"2. Do NOT use meta-text like \"Additional information:\" or \"Updated content:\" unless it's a natural part of the lore itself.\n"+
			"3. The merged content should read as if it were written as a single, original piece.\n\n"+
			"Existing Entry Name: %s\n"+
			"Existing Entry Type: %s\n"+
			"Existing Content:\n\"\"\"\n%s\n\"\"\"\n\n"+
			"New Information to Incorporate:\n\"\"\"\n%s\n\"\"\"\n\n"+
			"Final Merged Content (provide ONLY the text, ensuring it's a complete and coherent description):",
		existingEntry.Name,
		existingEntry.Type,
		existingEntry.Content,
		newContent,
	)

	log.Printf("Sending direct merge prompt for entry '%s' (ID: %d) to model: %s", existingEntry.Name, existingEntry.ID, model)
	mergedOutput, err := a.GenerateLLMContent(mergePrompt, model) // Use the new generic LLM method
	if err != nil {
		log.Printf("Error generating merged content via AI for '%s': %v. Falling back to appending new information.", existingEntry.Name, err)
		// Fallback to simple append with a clear separator if AI call fails
		return existingEntry.Content + "\n\n--- (New Information from Import - AI Merge Failed) ---\n" + newContent, nil
	}

	mergedOutput = strings.TrimSpace(mergedOutput)
	if mergedOutput == "" {
		log.Printf("AI returned empty string for merged content for '%s'. Falling back to appending.", existingEntry.Name)
		return existingEntry.Content + "\n\n--- (New Information from Import - AI Returned Empty) ---\n" + newContent, nil
	}

	// If AI essentially returns the original content, treat it as no change
	if strings.TrimSpace(mergedOutput) == strings.TrimSpace(existingEntry.Content) {
		log.Printf("AI merge for '%s' resulted in no significant change to content. Using original.", existingEntry.Name)
		return existingEntry.Content, nil
	}

	log.Printf("Successfully merged content for entry '%s' using direct AI prompting. Old length: %d, New length: %d", existingEntry.Name, len(existingEntry.Content), len(mergedOutput))
	return mergedOutput, nil
}

// MergeEntryContentWithRAG uses the RAG system to intelligently merge existing entry content with new content
func (a *App) MergeEntryContentWithRAG(existingEntry database.CodexEntry, newContent string, model string) (string, error) {
	if a.promptBuilder == nil {
		log.Println("Warning: MergeEntryContentWithRAG called but prompt builder not initialized. Falling back to direct merge.")
		// Fallback to direct merge if RAG isn't set up
		return a.MergeEntryContentDirect(existingEntry, newContent, model)
	}

	// Construct a prompt for merging content with very explicit instructions
	mergePrompt := fmt.Sprintf(
		"You are helping to update a codex entry with new information. Your task is to merge the existing content with the new content to create a single, coherent entry.\n\nVERY IMPORTANT INSTRUCTIONS:\n1. Provide ONLY the final merged content with no commentary, explanations, or meta-text.\n2. Do not start with phrases like \"Here's the merged content\" or \"Certainly! Let's weave those details together\".\n3. Do not include phrases like \"According to the new information\" or \"Additional information\".\n4. Create a seamless, integrated entry that reads like a single, coherent description.\n5. Your entire response should be ONLY the merged content text itself.\n\nExisting Entry Name: %s\nExisting Entry Type: %s\nExisting Content: %s\n\nNew Content to Incorporate: %s\n\nMerged Content (provide ONLY the final text with no commentary):",
		existingEntry.Name,
		existingEntry.Type,
		existingEntry.Content,
		newContent,
	)

	log.Printf("Building RAG-enhanced prompt for merging content for entry '%s'", existingEntry.Name)
	// Use the RAG system to enhance the merge with context from other related entries
	enhancedPrompt, err := a.promptBuilder.BuildPromptWithContext(mergePrompt)
	if err != nil {
		log.Printf("Error building context-enhanced prompt for merge: %v. Falling back to direct merge.", err)
		return a.MergeEntryContentDirect(existingEntry, newContent, model)
	}

	// Get the merged content from the AI
	log.Printf("Sending RAG-enhanced merge prompt to model: %s", model)
	mergedContent, err := a.GenerateLLMContent(enhancedPrompt, model)
	if err != nil {
		log.Printf("Error generating merged content with RAG: %v. Falling back to direct merge.", err)
		return a.MergeEntryContentDirect(existingEntry, newContent, model)
	}

	log.Printf("Successfully merged content for entry '%s' using RAG", existingEntry.Name)
	return mergedContent, nil
}

// ProcessAndSaveTextAsEntries takes text, processes it via LLM to extract structured
// codex entries (like ProcessStory), and then saves those entries directly to the DB.
// It returns the number of entries successfully created.
func (a *App) ProcessAndSaveTextAsEntries(textToProcess string) (int, error) {
	log.Printf("Processing text and saving entries...")

	// 1. Process the text using the same logic as ProcessStory
	// Construct a simplified prompt asking for JSON output
	simplifiedPrompt := fmt.Sprintf("Analyze the following text and extract key entities (characters, locations, items, concepts) and their descriptions. Format the output STRICTLY as a JSON array where each object has 'name', 'type', and 'content' fields. Types should be one of: Character, Location, Item, Concept. Do not include any text before or after the JSON array. Example: [{\"name\": \"Sir Reginald\", \"type\": \"Character\", \"content\": \"A brave knight known for his shiny armor.\"}]. Text to analyze:\n\n%s", textToProcess)

	cfg := llm.GetConfig()
	processingModelID := cfg.StoryProcessingModelID
	if processingModelID == "" {
		switch cfg.ActiveMode {
		case "openai":
			processingModelID = "gpt-3.5-turbo" // Using string literal
		case "gemini":
			processingModelID = "gemini-1.0-pro"
		case "openrouter", "local":
			fallthrough
		default:
			processingModelID = "anthropic/claude-3.5-sonnet"
		}
		log.Printf("Warning: StoryProcessingModelID not set, using fallback '%s' for ProcessAndSaveTextAsEntries in mode %s", processingModelID, cfg.ActiveMode)
	}
	log.Printf("Using model: %s for processing in ProcessAndSaveTextAsEntries (ActiveMode: %s)", processingModelID, cfg.ActiveMode)

	llmResponse, err := a.GenerateLLMContent(simplifiedPrompt, processingModelID)
	if err != nil {
		log.Printf("Error generating content from LLM in ProcessAndSaveTextAsEntries: %v", err)
		return 0, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// 2. Parse the LLM response (expecting JSON array)
	var llmEntries []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}

	err = json.Unmarshal([]byte(llmResponse), &llmEntries)
	if err != nil {
		// Handle cases where the response isn't valid JSON or the expected structure
		log.Printf("Warning: LLM response for ProcessAndSaveTextAsEntries was not a valid JSON array. Error: %v", err)
		log.Printf("LLM Response Text:\n%s", llmResponse)
		// Decide if we should save the raw text as one entry or just fail?
		// For now, let's just return an error indicating parsing failure.
		return 0, fmt.Errorf("LLM response was not the expected JSON array format: %w. Raw: %s", err, llmResponse)
	}

	// 3. Save the parsed entries to the database
	createdCount := 0
	if a.db == nil {
		log.Println("Error: Database connection is nil in ProcessAndSaveTextAsEntries")
		return 0, fmt.Errorf("database not initialized")
	}

	for _, entryData := range llmEntries {
		// Basic validation
		if entryData.Name == "" {
			log.Printf("Skipping entry with missing name: %+v", entryData)
			continue
		}
		if entryData.Type == "" {
			log.Printf("Warning: Entry '%s' has missing type, defaulting to 'Concept'", entryData.Name)
			entryData.Type = "Concept" // Default type if missing
		}

		// Check if entry exists
		var existingId int64
		err := a.db.QueryRow("SELECT id FROM codex_entries WHERE name = ?", entryData.Name).Scan(&existingId)

		if err == nil {
			// Entry exists, update it
			updatedEntry := database.CodexEntry{
				ID:      existingId,
				Name:    entryData.Name,
				Type:    entryData.Type,
				Content: entryData.Content,
				// CreatedAt is not updated
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			err = a.UpdateEntry(updatedEntry) // Use the existing UpdateEntry method
			if err != nil {
				log.Printf("Warning: Failed to update existing codex entry '%s' in ProcessAndSave: %v", entryData.Name, err)
				continue // Skip this entry on update failure
			}
			log.Printf("Successfully updated extracted entry: %s (%s)", entryData.Name, entryData.Type)
			createdCount++ // Count updates as well for success metric
		} else if err == sql.ErrNoRows {
			// Entry doesn't exist, create new
			newEntry, err := a.CreateEntry(entryData.Name, entryData.Type, entryData.Content) // Use the existing CreateEntry method
			if err != nil {
				log.Printf("Warning: Failed to insert new codex entry '%s' in ProcessAndSave: %v", entryData.Name, err)
				continue // Skip this entry on insert failure
			}
			log.Printf("Successfully saved extracted entry: %s (%s)", newEntry.Name, newEntry.Type)
			createdCount++
		} else {
			// Other database error during check
			log.Printf("Error checking for existing entry '%s' in ProcessAndSave: %v", entryData.Name, err)
			continue // Skip this entry on check failure
		}
	}

	log.Printf("Finished processing text. Created %d entries.", createdCount)
	return createdCount, nil
}

// --- Settings Management ---

// GetSettings returns the current OpenRouter configuration
func (a *App) GetSettings() llm.OpenRouterConfig {
	// Load config just in case it hasn't been loaded or might have changed externally
	// Although typically it's loaded at startup.
	if err := llm.LoadOpenRouterConfig(); err != nil {
		log.Printf("Warning: Failed to reload OpenRouter config in GetSettings: %v", err)
		// Return the potentially stale global config or an empty one if loading failed badly
	}
	//  is now in llm package, so we don't need to lock here
	config := llm.GetConfig()
	log.Printf("Returning current settings: API Key Set: %v, Chat Model: %s, Story Model: %s", config.APIKey != "", config.ChatModelID, config.StoryProcessingModelID)
	return config
}

// SaveSettings saves the OpenRouter configuration settings
func (a *App) SaveSettings(config llm.OpenRouterConfig) error {
	log.Printf("SaveSettings called with received config: %+v", config)

	// Update the global variable in the llm package
	llm.SetConfig(config)

	// Save the updated global config to the file
	if err := llm.SaveOpenRouterConfig(); err != nil {
		log.Printf("Error saving settings to file: %v", err)
		return fmt.Errorf("failed to save OpenRouter configuration: %w", err)
	}
	log.Println("Settings saved to file successfully. Re-initializing services...")

	// CRITICAL: Re-initialize embedding services with the new config
	// This ensures that if ActiveMode or related keys/models changed, the app uses them.
	if err := a.initializeEmbeddingServices(config); err != nil {
		log.Printf("Warning: Failed to re-initialize embedding services after saving settings: %v", err)
		// Return this error to the frontend so it can display it
		return fmt.Errorf("settings saved, but failed to apply embedding service changes: %w. Embeddings might not work as expected until next vault switch or app restart", err)
	}

	log.Println("Services re-initialized based on new settings.")
	return nil
}

// SaveAPIKeyOnly updates just the API key in the global config and saves it.
// This is specifically for the simpler save flow from the chat modal's API key input.
func (a *App) SaveAPIKeyOnly(apiKey string) error {
	log.Printf("SaveAPIKeyOnly called with key ending: ...%s", getLastNChars(apiKey, 6))
	if apiKey == "" {
		log.Println("Warning: Attempting to save an empty API key via SaveAPIKeyOnly.")
		// Allow saving empty key to clear it if intended
	}
	cfg := llm.GetConfig()
	cfg.APIKey = apiKey // Update only the API key field
	llm.SetConfig(cfg)
	log.Printf("Global openRouterConfig APIKey field updated. Current full config: %+v", cfg)

	if err := llm.SaveOpenRouterConfig(); err != nil {
		log.Printf("Error saving config after updating API key via SaveAPIKeyOnly: %v", err)
		return fmt.Errorf("failed to save OpenRouter configuration after API key update: %w", err)
	}
	log.Println("Configuration saved successfully via SaveAPIKeyOnly.")
	return nil
}

// Fetchllm.OpenRouterModelsWithKey fetches available models using a provided API key.
// This is called directly from the frontend when it knows the key.
func (a *App) FetchOpenRouterModelsWithKey(apiKey string) ([]llm.OpenRouterModel, error) {
	log.Println("FetchOpenRouterModelsWithKey called")
	return llm.FetchOpenRouterModels(apiKey)
}

// --- Utility Functions ---

// getLastNChars returns the last N characters of a string, or the whole string if shorter.
func getLastNChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// initEmbeddingWorker initializes the embedding worker goroutine
func (a *App) initEmbeddingWorker() {
	embeddingQueueOnce.Do(func() {
		go func() {
			for req := range embeddingQueue {
				if a.embeddingService == nil {
					log.Println("Warning: Embedding service not initialized, skipping embedding for entry:", req.entryID)
					continue
				}

				// Create embedding
				embedding, err := a.embeddingService.CreateEmbedding(req.text)
				if err != nil {
					log.Printf("Warning: Failed to create embedding for entry %d: %v", req.entryID, err)
					continue
				}

				// Save embedding (SaveEmbedding now has its own mutex)
				if err := a.embeddingService.SaveEmbedding(req.entryID, embedding); err != nil {
					log.Printf("Warning: Failed to save embedding for entry %d: %v", req.entryID, err)
				} else {
					log.Printf("Successfully generated and saved embedding for entry %d", req.entryID)
				}

				// Add a small delay between operations to reduce contention
				time.Sleep(100 * time.Millisecond)
			}
		}()
	})
}
