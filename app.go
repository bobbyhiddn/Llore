package main

import (
	"context"
	"database/sql"
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
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// App struct holds application state
type App struct {
	ctx       context.Context
	llmClient *bedrockruntime.Client // AWS Bedrock client
	config    *AppConfig             // Application configuration
	db        *sql.DB                // Database connection handle
	dbPath    string                 // Current database path
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

	// Database will be initialized later by user action (Create/Load)
	log.Println("App startup complete. Waiting for user to load or create a database.")
}

// shutdown is called when the app terminates. Closes the database connection.
func (a *App) shutdown(ctx context.Context) {
	log.Println("Llore application shutting down...")
	DBClose(a.db) // Close the current database connection
}

// --- Database Management ---

// SelectDatabaseFile opens a dialog for the user to select an existing database file.
func (a *App) SelectDatabaseFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Codex Database File",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Databases (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		log.Printf("Error opening file dialog: %v", err)
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}
	log.Printf("File selected: %s", selection)
	return selection, nil
}

// SaveDatabaseFile opens a dialog for the user to specify a new database file path.
func (a *App) SaveDatabaseFile() (string, error) {
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Save Codex Database As...",
		DefaultDirectory: filepath.Dir(a.dbPath), // Start in the current DB directory
		DefaultFilename:  "new_codex.db",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Databases (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		log.Printf("Error opening save file dialog: %v", err)
		return "", fmt.Errorf("failed to open save file dialog: %w", err)
	}
	log.Printf("File path chosen for saving: %s", selection)
	return selection, nil
}

// SwitchDatabase closes the current database connection and opens a new one.
func (a *App) SwitchDatabase(newPath string) error {
	log.Printf("Switching database from %s to %s", a.dbPath, newPath)

	// Close the existing connection, if open
	if a.db != nil {
		DBClose(a.db)
		a.db = nil // Explicitly set to nil after closing
	}

	// Initialize the new database
	newDB, err := DBInitialize(newPath)
	if err != nil {
		// Attempt to restore the original connection if switching fails?
		// For now, just log the error and leave a.db as nil.
		log.Printf("Error initializing new database '%s': %v. No database is active.", newPath, err)
		return fmt.Errorf("failed to initialize new database '%s': %w", newPath, err)
	}

	// Update App state
	a.db = newDB
	a.dbPath = newPath
	log.Printf("Successfully switched database to %s", newPath)
	return nil
}

// GetCurrentDatabasePath returns the path of the currently loaded database.
func (a *App) GetCurrentDatabasePath() string {
	return a.dbPath
}

// IsDatabaseLoaded checks if a database connection is currently active.
func (a *App) IsDatabaseLoaded() bool {
	return a.db != nil
}

// --- Codex Entry Management (Now using a.db) ---

// GetAllEntries retrieves all codex entries.
func (a *App) GetAllEntries() ([]CodexEntry, error) {
	if a.db == nil { // Check App's db handle
		return nil, fmt.Errorf("database is not initialized")
	}
	rows, err := a.db.Query("SELECT id, name, type, content, created_at, updated_at FROM codex_entries ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []CodexEntry
	for rows.Next() {
		var entry CodexEntry
		var createdAt sql.NullString
		var updatedAt sql.NullString
		if err := rows.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Content, &createdAt, &updatedAt); err != nil {
			log.Printf("Warning: failed to scan row: %v", err)
			continue
		}
		entry.CreatedAt = createdAt.String
		entry.UpdatedAt = updatedAt.String
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// CreateEntry adds a new codex entry.
func (a *App) CreateEntry(entry CodexEntry) (CodexEntry, error) {
	if a.db == nil { // Check App's db handle
		return CodexEntry{}, fmt.Errorf("database is not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := a.db.Prepare("INSERT INTO codex_entries(name, type, content, created_at, updated_at) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return CodexEntry{}, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(entry.Name, entry.Type, entry.Content, now, now)
	if err != nil {
		return CodexEntry{}, fmt.Errorf("failed to execute insert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return CodexEntry{}, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	entry.ID = id
	entry.CreatedAt = now
	entry.UpdatedAt = now
	return entry, nil
}

// GetEntryByID retrieves a single entry by its ID.
func (a *App) GetEntryByID(id int64) (CodexEntry, error) {
	if a.db == nil { // Check App's db handle
		return CodexEntry{}, fmt.Errorf("database is not initialized")
	}
	row := a.db.QueryRow("SELECT id, name, type, content, created_at, updated_at FROM codex_entries WHERE id = ?", id)

	var entry CodexEntry
	var createdAt sql.NullString
	var updatedAt sql.NullString
	if err := row.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Content, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return CodexEntry{}, fmt.Errorf("entry with ID %d not found", id)
		}
		return CodexEntry{}, fmt.Errorf("failed to scan entry with ID %d: %w", id, err)
	}
	entry.CreatedAt = createdAt.String
	entry.UpdatedAt = updatedAt.String

	return entry, nil
}

// UpdateEntry modifies an existing codex entry.
func (a *App) UpdateEntry(entry CodexEntry) error {
	if a.db == nil { // Check App's db handle
		return fmt.Errorf("database is not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := a.db.Prepare("UPDATE codex_entries SET name = ?, type = ?, content = ?, updated_at = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(entry.Name, entry.Type, entry.Content, now, entry.ID)
	if err != nil {
		return fmt.Errorf("failed to execute update for ID %d: %w", entry.ID, err)
	}
	return nil
}

// DeleteEntry removes a codex entry by its ID.
func (a *App) DeleteEntry(id int64) error {
	if a.db == nil { // Check App's db handle
		return fmt.Errorf("database is not initialized")
	}
	stmt, err := a.db.Prepare("DELETE FROM codex_entries WHERE id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("failed to execute delete for ID %d: %w", id, err)
	}
	return nil
}

// --- LLM Interaction ---

// ProcessStory sends a prompt to the LLM and processes the structured response.
func (a *App) ProcessStory(storyText string) ([]CodexEntry, error) { // Renamed parameter
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
		fallbackEntry := CodexEntry{
			Name:    "Generated Entry (Unstructured)", // Provide a default name
			Type:    "Generated",                      // Provide a default type
			Content: generatedText,
		}
		createdEntry, createErr := a.CreateEntry(fallbackEntry) // Use App's CreateEntry method
		if createErr != nil {
			return nil, fmt.Errorf("failed to create fallback database entry: %w", createErr)
		}
		return []CodexEntry{createdEntry}, nil // Return as a slice
	}

	// Process the structured entries
	var createdEntries []CodexEntry
	for _, llmEntry := range llmEntries {
		if llmEntry.Name == "" {
			log.Println("Warning: Skipping entry with empty name from LLM response.")
			continue
		}
		newEntry := CodexEntry{
			Name:    llmEntry.Name,
			Type:    llmEntry.Type,
			Content: llmEntry.Content,
		}

		createdEntry, err := a.CreateEntry(newEntry) // Use App's CreateEntry method
		if err != nil {
			log.Printf("Failed to create entry '%s': %v. Skipping.", newEntry.Name, err)
			continue // Continue processing other potential entries
		}
		// Append the returned value directly
		createdEntries = append(createdEntries, createdEntry)
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

// --- Database Utility ---

// CopyDatabase copies all entries from the current database to a new database file.
func (a *App) CopyDatabase(newPath string) error {
	log.Printf("Attempting to copy current database (%s) to: %s", a.dbPath, newPath)
	if a.db == nil { // Check App's db handle
		return fmt.Errorf("source database connection is not initialized")
	}

	// Ensure the new database file directory exists or create it
	dir := filepath.Dir(newPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for new database '%s': %w", dir, err)
		}
	}

	// Use DBInitialize to set up the destination DB (incl. table creation)
	destDB, err := DBInitialize(newPath)
	if err != nil {
		return fmt.Errorf("failed to initialize destination database '%s': %w", newPath, err)
	}
	defer DBClose(destDB) // Close the destination DB when done

	// Get all entries from the current (source) database
	entries, err := a.GetAllEntries() // Use App's method
	if err != nil {
		return fmt.Errorf("failed to get entries from source database (%s): %w", a.dbPath, err)
	}
	log.Printf("Retrieved %d entries from source database.", len(entries))

	// Prepare insert statement for the destination database
	stmt, err := destDB.Prepare("INSERT INTO codex_entries(name, type, content, created_at, updated_at) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for destination database: %w", err)
	}
	defer stmt.Close()

	// Iterate and insert into the destination database
	copiedCount := 0
	for _, entry := range entries {
		// Generate a NEW timestamp string for the copied entry
		nowStr := time.Now().UTC().Format(time.RFC3339)
		_, err := stmt.Exec(entry.Name, entry.Type, entry.Content, nowStr, nowStr) // Use new timestamp
		if err != nil {
			// Log error but try to continue copying other entries
			log.Printf("Warning: Failed to copy entry ID %d ('%s') to destination database: %v", entry.ID, entry.Name, err)
			continue
		}
		copiedCount++
	}

	log.Printf("Successfully copied %d out of %d entries from %s to %s", copiedCount, len(entries), a.dbPath, newPath)
	if copiedCount != len(entries) {
		log.Printf("Warning: %d entries failed to copy.", len(entries)-copiedCount)
	}

	return nil
}
