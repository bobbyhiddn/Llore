package main

import (
	"database/sql"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	bedrockruntime "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// CodexCodexEntry represents an entry in the codex database.
// Timestamps are stored as strings for easier frontend handling.
type CodexCodexEntry struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// GetAllEntries retrieves all codex entries from the SQLite database
func (a *App) GetAllEntries() ([]CodexEntry, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	rows, err := a.db.Query("SELECT id, name, type, content, created_at, updated_at FROM codex_entries ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []CodexEntry
	for rows.Next() {
		var e CodexEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Content, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// CreateEntry creates a new codex entry in the SQLite database
func (a *App) CreateEntry(name, entryType, content string) (CodexEntry, error) {
	if a.db == nil {
		return CodexEntry{}, fmt.Errorf("database is not initialized")
	}
	id, err := DBInsertEntry(a.db, name, entryType, content)
	if err != nil {
		return CodexEntry{}, err
	}
	// Fetch the created entry
	row := a.db.QueryRow("SELECT id, name, type, content, created_at, updated_at FROM codex_entries WHERE id = ?", id)
	var entry CodexEntry
	if err := row.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Content, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		return CodexEntry{}, err
	}
	return entry, nil
}

// UpdateEntry updates an existing codex entry in the SQLite database
func (a *App) UpdateEntry(entry CodexEntry) error {
	if a.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	return DBUpdateEntry(a.db, entry)
}

// DeleteEntry deletes a codex entry by ID using SQLite
func (a *App) DeleteEntry(id int64) error {
	if a.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	return DBDeleteEntry(a.db, id)
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
	ctx       context.Context
	llmClient *bedrockruntime.Client // AWS Bedrock client
	config    *AppConfig             // Application configuration
	db        *sql.DB                // Database connection handle
	dbPath    string                 // Current database path
}

// --- Vault Management ---

// SelectVaultFolder opens a dialog for the user to select an existing vault folder
func (a *App) SelectVaultFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Lore Vault Folder",
	})
	if err != nil {
		log.Printf("Error opening directory dialog: %v", err)
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}
	log.Printf("Vault folder selected: %s", selection)
	return selection, nil
}

// CreateNewVault creates a new vault folder with the required structure
func (a *App) CreateNewVault(vaultName string) (string, error) {
	// First, let the user select where to create the vault
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Location for New Lore Vault",
	})
	if err != nil {
		log.Printf("Error opening directory dialog: %v", err)
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}

	if vaultName == "" {
		vaultName = "LoreVault"
	}
	// Create the vault directory structure
	vaultPath := filepath.Join(selection, vaultName)
	subdirs := []string{
		filepath.Join(vaultPath, "Library"),
		filepath.Join(vaultPath, "Codex"),
		filepath.Join(vaultPath, "Chat"),
	}

	// Create the main vault directory
	if err := os.MkdirAll(vaultPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Create subdirectories
	for _, dir := range subdirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create subdirectory %s: %w", dir, err)
		}
	}

	log.Printf("Created new vault at: %s", vaultPath)
	return vaultPath, nil
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
		DBClose(a.db)
		a.db = nil
	}

	// Initialize SQLite DB for this vault (store under Codex folder)
	codexDBPath := filepath.Join(path, "Codex", "codex_data.db")
	dbConn, err := DBInitialize(codexDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize codex database: %w", err)
	}
	a.db = dbConn
	a.dbPath = path

	// Load initial data
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

// ImportStoryTextAndFile saves story text to a file and processes it for codex entries
func (a *App) ImportStoryTextAndFile(text string) ([]CodexEntry, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}

	// Generate a filename based on the first line or default
	firstLine := strings.Split(strings.TrimSpace(text), "\n")[0]
	filename := "story.txt"
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
	entries, err := a.ProcessStory(text)
	if err != nil {
		return nil, fmt.Errorf("failed to process story into codex entries: %w", err)
	}
	created := make([]CodexEntry, 0, len(entries))
	for _, entry := range entries {
		_, err := a.CreateEntry(entry.Name, entry.Type, entry.Content)
		if err != nil {
			log.Printf("Warning: Failed to insert codex entry '%s': %v", entry.Name, err)
			continue
		}
		created = append(created, entry)
	}
	return created, nil
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

	// Load configuration
	cfg, err := LoadConfig() // Assuming LoadConfig is defined elsewhere (e.g., config.go)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	a.config = cfg
	log.Println("Configuration loaded successfully.")

	// Initialize AWS Bedrock client
	a.llmClient, err = initializeBedrockClient(ctx, a.config.AWSRegion)
	if err != nil {
		log.Fatalf("Failed to initialize Bedrock client: %v", err)
	}
	log.Println("AWS Bedrock client initialized successfully.")

	log.Println("App startup complete.")
}

 // shutdown is called when the app terminates.
func (a *App) shutdown(ctx context.Context) {
	log.Println("Llore application shutting down...")
}

// --- LLM Interaction ---

// GenerateOpenRouterContent calls OpenRouter with prompt/model, uses cache, and returns the response.
func (a *App) GenerateOpenRouterContent(prompt, model string) (string, error) {
	if err := LoadOpenRouterConfig(); err != nil {
		return "", fmt.Errorf("failed to load OpenRouter config: %w", err)
	}
	if err := LoadOpenRouterCache(); err != nil {
		return "", fmt.Errorf("failed to load OpenRouter cache: %w", err)
	}
	return GetOpenRouterCompletion(prompt, model)
}

// ProcessStory sends a prompt to the LLM and processes the structured response.
func (a *App) ProcessStory(storyText string) ([]CodexEntry, error) { 
	if a.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not initialized")
	}

	// Construct the specific prompt for story analysis and JSON output
	structuredPrompt := fmt.Sprintf(`Analyze the following story text to identify key entities (like Characters, Locations, Items, or significant Lore concepts). For each significant entity found, generate a concise description based solely on the information presented in the text.\n\nFormat your entire response *strictly* as a JSON array of objects. Each object in the array must represent one entity and contain exactly three keys:\n1. "name": The name of the entity (string).\n2. "type": The type of entity (string, e.g., 'Character', 'Location', 'Item', 'Lore').\n3. "content": The concise description generated from the text (string).\n\n**IMPORTANT: Your response must contain *only* the valid JSON array. Do not include any introductory text, concluding remarks, explanations, apologies, markdown formatting (like backticks or 'json' tags), or anything else outside the JSON array itself.**\n\nIf no significant entities are found in the text, return an empty JSON array: []\n\nStory Text:\n"""\n%s\n"""\n\nJSON Array Output:`, storyText)

	// Generate content using the LLM with the structured prompt
	generatedText, err := a.GenerateContent(structuredPrompt) // Use the constructed prompt
	if err != nil {
		return nil, fmt.Errorf("error generating content from LLM: %w", err)
	}

	// Attempt to parse the structured response (expecting a JSON array of objects)
	var llmEntries []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}

	err = json.Unmarshal([]byte(generatedText), &llmEntries)
	if err != nil {
		// Handle cases where the response isn't valid JSON or the expected structure
		log.Printf("Warning: LLM response was not a valid JSON array of entries. Treating as single entry. Error: %v", err)
		log.Printf("LLM Response Text:\n%s", generatedText)
		// Fallback: Treat the entire response as the content of a single entry
		now := time.Now().UTC()
		fallbackCodexEntry := CodexEntry{
			ID:        time.Now().UnixNano(),
			Name:      "Generated CodexEntry (Unstructured)", // Provide a default name
			Type:      "Generated",                      // Provide a default type
			Content:   generatedText,
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
			
		}
		return []CodexEntry{fallbackCodexEntry}, nil // Return as a slice
	}

	// Process the structured entries
	now := time.Now().UTC()
	var createdEntries []CodexEntry
	for _, llmCodexEntry := range llmEntries {
		if llmCodexEntry.Name == "" {
			log.Println("Warning: Skipping entry with empty name from LLM response.")
			continue
		}
		createdCodexEntry := CodexEntry{
			ID:        time.Now().UnixNano(),
			Name:      llmCodexEntry.Name,
			Type:      llmCodexEntry.Type,
			Content:   llmCodexEntry.Content,
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
			
		}

		createdEntries = append(createdEntries, createdCodexEntry)
	}

	return createdEntries, nil
}

// GenerateContent uses the configured Bedrock client to generate text based on a prompt.
func (a *App) GenerateContent(prompt string) (string, error) {
	if a.llmClient == nil {
		return "", fmt.Errorf("Bedrock client not initialized")
	}

	// Model ID set by user edit
	modelID := "us.anthropic.claude-3-5-sonnet-20241022-v2:0"

	// Prepare the payload using the Messages API format for Claude 3
	payload := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31", // Required for Claude 3 models
		"max_tokens":        1024,                   // Use max_tokens instead of max_tokens_to_sample
		"temperature":       0.7,
		"top_p":             1,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Invoke the model
	output, err := a.llmClient.InvokeModel(a.ctx, &bedrockruntime.InvokeModelInput{
		Body:        payloadBytes,
		ModelId:     &modelID,
		ContentType: Ptr("application/json"),
		Accept:      Ptr("*/*"), // Changed Accept as per Messages API common practice
	})
	if err != nil {
		return "", fmt.Errorf("failed to invoke Bedrock model %s: %w", modelID, err)
	}

	// Parse the response using the Messages API structure
	var respBody struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		// Add other fields like Usage if needed
	}
	if err := json.Unmarshal(output.Body, &respBody); err != nil {
		return "", fmt.Errorf("failed to unmarshal Bedrock Messages API response body: %w", err)
	}

	// Extract the text content
	if len(respBody.Content) > 0 && respBody.Content[0].Type == "text" {
		log.Printf("LLM generated content successfully using Messages API.")
		return strings.TrimSpace(respBody.Content[0].Text), nil
	}

	log.Printf("Warning: LLM response did not contain expected text content.")
	return "", fmt.Errorf("LLM response did not contain expected text content structure")
}

// initializeBedrockClient initializes the AWS Bedrock Runtime client.
func initializeBedrockClient(ctx context.Context, region string) (*bedrockruntime.Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w. Ensure credentials are set (env vars, ~/.aws/credentials, etc.)", err)
	}
	client := bedrockruntime.NewFromConfig(awsCfg)
	return client, nil
}

// Helper function to get a pointer to a string
func Ptr(s string) *string {
	return &s
}
