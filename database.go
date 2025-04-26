package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// CodexEntry represents an entry in the codex database.
// Timestamps are stored as strings for easier frontend handling.
type CodexEntry struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// DBInitialize sets up the database connection and ensures the table exists.
// Returns the connection handle or an error.
func DBInitialize(dbPath string) (*sql.DB, error) {
	log.Printf("Initializing database connection for: %s\n", dbPath)
	dbConn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbPath, err)
	}

	// Check connection
	err = dbConn.Ping()
	if err != nil {
		dbConn.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database %s: %w", dbPath, err)
	}

	// Create table if it doesn't exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS codex_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		type TEXT,
		content TEXT,
		created_at TEXT,
		updated_at TEXT
	);`

	_, err = dbConn.Exec(createTableSQL)
	if err != nil {
		dbConn.Close() // Close the connection if table creation fails
		return nil, fmt.Errorf("failed to create table in %s: %w", dbPath, err)
	}

	log.Printf("Database connection for %s initialized successfully.", dbPath)
	return dbConn, nil // Return the connection handle
}

// DBClose closes the provided database connection.
func DBClose(dbConn *sql.DB) {
	if dbConn != nil {
		err := dbConn.Close()
		if err != nil {
			log.Printf("Error closing a database connection: %v\n", err)
		} else {
			log.Println("Database connection closed successfully.")
		}
	}
}
