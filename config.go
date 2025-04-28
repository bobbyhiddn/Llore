package main

import (
	"os"
)

// Helper function to get environment variable or return default.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// This file is intentionally left blank for now.
// Configuration for OpenRouter is handled in openrouter.go
// Database paths are handled during vault switching in app.go
