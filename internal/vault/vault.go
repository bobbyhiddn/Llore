// Package vault provides functions for managing vault folders and structure.
package vault

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectVaultFolder opens a dialog for the user to select an existing vault folder.
// Returns the selected path or an error.
func SelectVaultFolder(ctx context.Context) (string, error) {
	selection, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Lore Vault Folder",
	})
	if err != nil {
		log.Printf("Error opening directory dialog: %v", err)
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}
	log.Printf("Vault folder selected: %s", selection)
	return selection, nil
}

// CreateNewVault creates a new vault folder with the required structure.
// Returns the path to the new vault or an error.
func CreateNewVault(ctx context.Context, vaultName string) (string, error) {
	selection, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Choose Location for New Lore Vault",
	})
	if err != nil {
		log.Printf("Error opening directory dialog: %v", err)
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}

	if vaultName == "" {
		vaultName = "LoreVault"
	}
	vaultPath := filepath.Join(selection, vaultName)
	// ADD "Templates" to the list of subdirectories
	subdirs := []string{
		filepath.Join(vaultPath, "Library"),
		filepath.Join(vaultPath, "Codex"),
		filepath.Join(vaultPath, "Chat"),
		filepath.Join(vaultPath, "Templates"), // New line
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

	log.Printf("Created new vault at: %s with Templates directory", vaultPath)
	return vaultPath, nil
}

// SwitchVault switches to a different vault folder, verifying structure.
// Returns the codex database path or an error.
func SwitchVault(path string) (string, error) {
	// Verify the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to access vault path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("specified path is not a directory: %s", path)
	}

	// Verify required subdirectories exist
	requiredDirs := []string{"Library", "Codex", "Chat", "Templates"}
	for _, dir := range requiredDirs {
		subdir := filepath.Join(path, dir)
		if info, err := os.Stat(subdir); err != nil || !info.IsDir() {
			return "", fmt.Errorf("invalid vault structure: missing %s directory", dir)
		}
	}

	// Return the codex database path (under Codex folder)
	codexDBPath := filepath.Join(path, "Codex", "codex_data.db")
	return codexDBPath, nil
}
