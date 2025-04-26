package main

import (
	"log"
	"os"

	"github.com/joho/godotenv" // Using godotenv for local .env loading
)

// AppConfig holds application configuration values.
type AppConfig struct {
	DatabasePath   string
	AWSRegion      string
	// AWS Keys are typically handled by the SDK's credential provider chain (env vars, shared config, IAM role)
	// AWS_ACCESS_KEY_ID string // Avoid storing directly if possible
	// AWS_SECRET_ACCESS_KEY string // Avoid storing directly if possible
	BedrockModelID string
}

// LoadConfig loads configuration from environment variables.
// It attempts to load a .env file first for local development.
func LoadConfig() (*AppConfig, error) {
	// Attempt to load .env file, ignore error if it doesn't exist
	err := godotenv.Load()
	if err != nil {
		log.Println("Info: No .env file found or error loading it, proceeding with environment variables.")
	}

	cfg := &AppConfig{
		DatabasePath:   getEnv("DB_PATH", "./codex_prototype.db"), // Default SQLite path
		AWSRegion:      getEnv("AWS_REGION", "us-east-1"),      // Default AWS region
		BedrockModelID: getEnv("BEDROCK_MODEL_ID", "us.anthropic.claude-3-5-sonnet-20241022-v2:0"), // Default model
		// SDK will pick up AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from env if set
	}

	// Add validation if needed (e.g., check if required vars are set)
	if cfg.AWSRegion == "" {
        log.Println("Warning: AWS_REGION environment variable not set, using default.")
	}
    if cfg.BedrockModelID == "" {
        log.Println("Warning: BEDROCK_MODEL_ID environment variable not set, using default.")
    }

	if modelID := os.Getenv("BEDROCK_MODEL_ID"); modelID != "" {
		cfg.BedrockModelID = modelID
		log.Printf("Using Bedrock Model ID from env: %s", cfg.BedrockModelID)
	}

	log.Printf("Configuration loaded: DBPath=%s, Region=%s, BedrockModel=%s",
		cfg.DatabasePath, cfg.AWSRegion, cfg.BedrockModelID)

	return cfg, nil
}

// Helper function to get environment variable or return default.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
