package main

import (
	ragcontext "Llore/internal/context" // Added for RAG context building (aliased)
	"Llore/internal/database"
	"Llore/internal/embeddings" // Added for RAG embeddings
	"Llore/internal/llm"
	"Llore/internal/vault"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	err := database.DBUpdateEntry(a.db, entry)
	if err != nil {
		return err // Return early if DB update failed
	}

	// Update embedding for the entry using the queue system
	if a.embeddingService != nil && entry.ID != 0 {
		// Initialize worker if not already done
		a.initEmbeddingWorker()

		// Prepare text for embedding
		text := fmt.Sprintf("Name: %s\nType: %s\nContent: %s",
			entry.Name, entry.Type, entry.Content)

		// Send to channel with non-blocking behavior
		select {
		case embeddingQueue <- embeddingRequest{entryID: entry.ID, text: text}:
			log.Printf("Queued embedding update for entry %d", entry.ID)
		default:
			log.Printf("Warning: Embedding queue full, skipping embedding update for entry %d", entry.ID)
		}
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
		a.db = nil
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
	       codex_entry_id INTEGER NOT NULL UNIQUE, -- Ensure one embedding per entry
	       embedding BLOB NOT NULL,
	       vector_version TEXT NOT NULL,
	       created_at TEXT NOT NULL,
	       updated_at TEXT NOT NULL,
	       FOREIGN KEY(codex_entry_id) REFERENCES codex_entries(id) ON DELETE CASCADE
	   )`)
	if err != nil {
		// Log warning but don't fail the vault switch
		log.Printf("Warning: Failed to create or verify embeddings table: %v", err)
	}
	_, err = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_embeddings_entry_id ON codex_embeddings(codex_entry_id);`)
	if err != nil {
		log.Printf("Warning: Failed to create index on embeddings table: %v", err)
	}
	// --- End embeddings table check ---

	// --- Initialize Embedding and Context Services ---
	config := llm.GetConfig() // Assumes config includes GeminiApiKey now
	a.geminiApiKey = config.GeminiApiKey
	if a.geminiApiKey == "" {
		log.Println("Warning: Gemini API Key is not set in config. Embedding features will be disabled.")
		a.embeddingService = nil
		a.contextBuilder = nil
		a.promptBuilder = nil
	} else {
		a.embeddingService = embeddings.NewEmbeddingService(a.db, a.geminiApiKey)
		a.contextBuilder = ragcontext.NewContextBuilder(a.embeddingService) // Use alias
		a.promptBuilder = llm.NewPromptBuilder(a.contextBuilder)

		// Pre-load existing embeddings into cache in the background
		go func() {
			if err := a.embeddingService.LoadEmbeddingsIntoCache(); err != nil {
				log.Printf("Warning: Failed to pre-load embeddings into cache: %v", err)
			}
		}()

		// Process missing embeddings in background
		go func() {
			// Add a small delay to allow cache loading to potentially start first
			time.Sleep(2 * time.Second)
			if err := a.GenerateMissingEmbeddings(); err != nil {
				log.Printf("Warning: Failed to generate missing embeddings in background: %v", err)
			}
		}()
	}
	// --- End Service Initialization ---

	// Initialize LLM package with the new vault path (this will load the cache)
	if err := llm.Init(path); err != nil {
		// Log warning but don't fail the vault switch entirely, cache might just be missing/corrupt
		log.Printf("Warning: Failed to initialize LLM package for vault '%s': %v", path, err)
	}

	// Load initial library data
	if err := a.refreshLibraryFiles(); err != nil {
		log.Printf("Warning: Failed to load library files: %v", err)
	}

	log.Printf("Successfully switched to vault: %s", path)
	return nil
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
func (a *App) ImportStoryTextAndFile(text string, providedFilename string) ([]database.CodexEntry, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
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
		return nil, fmt.Errorf("failed to create Library directory: %w", err)
	}
	// Save the file
	filePath := filepath.Join(libraryDir, filename)
	if err := os.WriteFile(filePath, []byte(text), 0644); err != nil {
		return nil, fmt.Errorf("failed to save story file: %w", err)
	}

	// Update library files cache
	if err := a.refreshLibraryFiles(); err != nil {
		log.Printf("Warning: Failed to refresh library files after import: %v", err)
	}

	// Process the story into codex entries
	result, err := a.ProcessStory(text) // Now returns ProcessStoryResult
	if err != nil {
		return nil, fmt.Errorf("failed to process story into codex entries: %w", err)
	}

	// Use counts from the result struct
	newCount := len(result.NewEntries)
	updatedCount := len(result.UpdatedEntries)
	totalProcessed := newCount + updatedCount

	// Log summary of the import operation
	log.Printf("Import complete. Processed %d entries: %d new, %d updated.",
		totalProcessed, newCount, updatedCount)

	if updatedCount > 0 {
		log.Printf("Note: %d entries already existed and were updated with merged content.", updatedCount)
	}

	// Combine new and updated entries for the return value (as the function signature still expects []database.CodexEntry)
	// TODO: Consider changing ImportStoryTextAndFile return type to ProcessStoryResult in the future if needed.
	allEntries := append(result.NewEntries, result.UpdatedEntries...)

	return allEntries, nil
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "openrouter_cache.json" {
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
	// REMOVED: Load OpenRouter cache - This will be handled by llm.Init called during SwitchVault
	// if err := llm.LoadOpenRouterCache(); err != nil {
	// 	log.Printf("Warning: Failed to load OpenRouter cache: %v", err)
	// }

	log.Println("App startup complete.")
}

// shutdown is called when the app terminates.
func (a *App) shutdown(ctx context.Context) {
	log.Println("Llore application shutting down...")
}

// ProcessStory sends a prompt to the LLM and processes the structured response.
func (a *App) ProcessStory(storyText string) (ProcessStoryResult, error) { // Changed return type
	// Construct a simplified prompt asking for JSON output
	simplifiedPrompt := fmt.Sprintf("Analyze the following story text and extract key entities (characters, locations, items, concepts) and their descriptions. Be thorough and try to identify anywhere from 3 to 15 distinct entities. Format the output STRICTLY as a JSON array where each object has 'name', 'type', and 'content' fields. Types should be one of: Character, Location, Item, Concept. Do not include any text before or after the JSON array. Example: [{\"name\": \"Sir Reginald\", \"type\": \"Character\", \"content\": \"A brave knight known for his shiny armor.\"}]. Story text:\n\n%s", storyText)

	log.Println("Sending prompt to OpenRouter for story processing...")

	// --- Model Selection ---
	// Get the model ID from config, with a fallback
	processingModel := llm.GetConfig().StoryProcessingModelID
	if processingModel == "" {
		log.Println("Warning: StoryProcessingModelID not set in config, using default 'anthropic/claude-3.5-sonnet'")
		processingModel = "anthropic/claude-3.5-sonnet" // Fallback model
	}
	log.Printf("Using model '%s' for processing story", processingModel)

	// Call the OpenRouter client
	llmResponse, err := a.GenerateOpenRouterContent(simplifiedPrompt, processingModel)
	if err != nil {
		log.Printf("Error generating content from OpenRouter: %v", err)
		return ProcessStoryResult{}, fmt.Errorf("failed to get OpenRouter response: %w", err) // Changed return
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
		if strings.HasSuffix(llmResponse, "```") {
			llmResponse = strings.TrimSuffix(llmResponse, "```")
		}
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
		// For now, returning empty result.
		log.Println("Returning empty result due to LLM response parsing error.")
		return ProcessStoryResult{}, nil // Changed return
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

		if err == nil {
			// Entry exists, merge the content
			log.Printf("Merging new information into existing entry '%s'", llmEntry.Name)

			// Keep the existing type if the new one is empty or generic
			entryType := llmEntry.Type
			if entryType == "" || entryType == "Generated" {
				entryType = existingEntry.Type
			}

			// Use RAG to intelligently merge the content
			log.Printf("Using RAG to merge content for entry '%s'", existingEntry.Name)
			mergedContent, err := a.MergeEntryContentWithRAG(existingEntry, llmEntry.Content, processingModel)
			if err != nil {
				log.Printf("Error merging content with RAG: %v. Using existing content.", err)
				mergedContent = existingEntry.Content
			}

			updatedEntry := database.CodexEntry{
				ID:        existingEntry.ID,
				Name:      existingEntry.Name,
				Type:      entryType,
				Content:   mergedContent,
				CreatedAt: existingEntry.CreatedAt,
				UpdatedAt: now.Format(time.RFC3339),
			}

			// Update in DB
			err = a.UpdateEntry(updatedEntry) // Use existing UpdateEntry which queues embedding
			if err != nil {
				log.Printf("Warning: Failed to update entry '%s' in database: %v", updatedEntry.Name, err)
				continue
			}
			updatedEntriesResult = append(updatedEntriesResult, updatedEntry) // Add to updated list

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

// GenerateOpenRouterContent calls OpenRouter with prompt/model, uses cache, and returns the response.
func (a *App) GenerateOpenRouterContent(prompt, model string) (string, error) {
	if err := llm.LoadOpenRouterConfig(); err != nil { // Ensure config is loaded (or attempt reload)
		return "", fmt.Errorf("failed to load OpenRouter configuration: %w", err)
	}
	if err := llm.LoadOpenRouterCache(); err != nil {
		return "", fmt.Errorf("failed to load OpenRouter cache: %w", err)
	}
	return llm.GetOpenRouterCompletion(prompt, model)
}

// GenerateMissingEmbeddings ensures all entries have embeddings
func (a *App) GenerateMissingEmbeddings() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if a.embeddingService == nil {
		log.Println("Skipping GenerateMissingEmbeddings: Embedding service not initialized (likely missing API key).")
		return nil // Not an error, just skipping
	}

	log.Println("Starting background check for missing embeddings...")

	// Find entries without embeddings
	rows, err := a.db.Query(`
        SELECT e.id, e.name, e.type, e.content
        FROM codex_entries e
        LEFT JOIN codex_embeddings em ON e.id = em.codex_entry_id
        WHERE em.id IS NULL
    `)
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
			log.Printf("Warning: Failed to create embedding for entry %d ('%s'): %v", entry.ID, entry.Name, err)
			// Consider adding a delay or backoff here if API errors are frequent
			time.Sleep(1 * time.Second) // Simple delay
			continue                    // Skip this entry
		}

		// Save embedding
		if err := a.embeddingService.SaveEmbedding(entry.ID, embedding); err != nil {
			log.Printf("Warning: Failed to save embedding for entry %d ('%s'): %v", entry.ID, entry.Name, err)
			continue // Skip this entry
		}

		processedCount++
		log.Printf("Generated embedding for entry %d ('%s') (%d/%d)", entry.ID, entry.Name, processedCount, len(entriesToProcess))
		time.Sleep(500 * time.Millisecond) // Add a small delay between API calls
	}

	log.Printf("Finished generating missing embeddings. Processed %d entries.", processedCount)
	return nil
}

// GetAIResponseWithContext gets AI response using the RAG pipeline
func (a *App) GetAIResponseWithContext(query string, model string) (string, error) {
	if a.promptBuilder == nil {
		log.Println("Warning: GetAIResponseWithContext called but prompt builder not initialized. Falling back to simple generation.")
		// Fallback to non-contextual response if RAG isn't set up
		return a.GenerateOpenRouterContent(query, model)
		// Alternatively, return an error:
		// return "", fmt.Errorf("RAG system (prompt builder) not initialized, likely missing Gemini API key")
	}

	log.Printf("Building prompt with context for query: %s", query)
	// Build prompt with context
	prompt, err := a.promptBuilder.BuildPromptWithContext(query)
	if err != nil {
		log.Printf("Error building prompt with context: %v. Falling back to simple prompt.", err)
		// Fallback to simple prompt if context building fails
		prompt = query // Or use llm.BuildSimplePrompt if you have a standard system message
	}

	// Get OpenRouter completion with the potentially context-enhanced prompt
	log.Printf("Sending context-aware prompt (length: %d) to model: %s", len(prompt), model)
	return a.GenerateOpenRouterContent(prompt, model) // Use existing method for the API call
}

// MergeEntryContentDirect merges existing entry content with new content using direct AI prompting without RAG
func (a *App) MergeEntryContentDirect(existingEntry database.CodexEntry, newContent string, model string) (string, error) {
	// Skip merging if the new content is already contained in the existing content
	if strings.Contains(strings.ToLower(existingEntry.Content), strings.ToLower(newContent)) {
		log.Printf("New content for '%s' is already contained in existing content. No changes needed.", existingEntry.Name)
		return existingEntry.Content, nil
	}

	// Construct a prompt for merging content
	mergePrompt := fmt.Sprintf(
		"You are helping to update a codex entry with new information. Please merge the existing content with the new content to create a single, coherent entry. Incorporate all information from both sources without redundancy. Do not use phrases like 'According to the new information' or 'Additional information'. Just create a seamless, integrated entry.\n\nExisting Entry Name: %s\nExisting Entry Type: %s\nExisting Content: %s\n\nNew Content to Incorporate: %s\n\nPlease provide the complete merged content for this entry:",
		existingEntry.Name,
		existingEntry.Type,
		existingEntry.Content,
		newContent,
	)

	// Get the merged content from the AI
	log.Printf("Sending direct merge prompt for entry '%s' to model: %s", existingEntry.Name, model)
	mergedContent, err := a.GenerateOpenRouterContent(mergePrompt, model)
	if err != nil {
		log.Printf("Error generating merged content: %v. Falling back to simple merge.", err)
		// Fallback to simple merge on error
		return existingEntry.Content + "\n\nAdditional information:\n" + newContent, nil
	}

	log.Printf("Successfully merged content for entry '%s' using direct AI prompting", existingEntry.Name)
	return mergedContent, nil
}

// MergeEntryContentWithRAG uses the RAG system to intelligently merge existing entry content with new content
func (a *App) MergeEntryContentWithRAG(existingEntry database.CodexEntry, newContent string, model string) (string, error) {
	if a.promptBuilder == nil {
		log.Println("Warning: MergeEntryContentWithRAG called but prompt builder not initialized. Falling back to direct merge.")
		// Fallback to direct merge if RAG isn't set up
		return a.MergeEntryContentDirect(existingEntry, newContent, model)
	}

	// Construct a prompt for merging content
	mergePrompt := fmt.Sprintf(
		"You are helping to update a codex entry with new information. Please merge the existing content with the new content to create a single, coherent entry. Incorporate all information from both sources without redundancy. Do not use phrases like 'According to the new information' or 'Additional information'. Just create a seamless, integrated entry.\n\nExisting Entry Name: %s\nExisting Entry Type: %s\nExisting Content: %s\n\nNew Content to Incorporate: %s\n\nPlease provide the complete merged content for this entry:",
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
	mergedContent, err := a.GenerateOpenRouterContent(enhancedPrompt, model)
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

	// TODO: Allow user to select model for processing in the future.
	processingModel := "anthropic/claude-3.5-sonnet"
	log.Printf("Using model: %s for processing", processingModel)

	// Call the OpenRouter client
	llmResponse, err := a.GenerateOpenRouterContent(simplifiedPrompt, processingModel)
	if err != nil {
		log.Printf("Error generating content from OpenRouter: %v", err)
		return 0, fmt.Errorf("failed to get OpenRouter response: %w", err)
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
		return 0, fmt.Errorf("LLM response was not the expected JSON array format")
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
	log.Printf("SaveSettings called with received config: %+v", config) // Log received config

	// Update the global variable
	llm.SetConfig(config)
	log.Printf("Global openRouterConfig updated to: %+v", config)
	//  is now in llm package, so we don't need to unlock here

	// Save the updated global config to the file
	if err := llm.SaveOpenRouterConfig(); err != nil {
		log.Printf("Error saving settings: %v", err)
		return fmt.Errorf("failed to save OpenRouter configuration: %w", err)
	}
	log.Println("Settings saved successfully.")
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
