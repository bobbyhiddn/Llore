// internal/llm/prompt.go
package llm

import (
	"Llore/internal/context" // Use the context package
	"fmt"
	"log" // Added for logging
	"strings"
)

// PromptBuilder constructs LLM prompts, potentially incorporating context
type PromptBuilder struct {
	contextBuilder *context.ContextBuilder
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(contextBuilder *context.ContextBuilder) *PromptBuilder {
	if contextBuilder == nil {
		log.Fatal("FATAL: ContextBuilder cannot be nil in NewPromptBuilder") // Critical dependency
	}
	return &PromptBuilder{
		contextBuilder: contextBuilder,
	}
}

// BuildPromptWithContext creates a prompt string including relevant context retrieved based on the user query
func (b *PromptBuilder) BuildPromptWithContext(userQuery string) (string, error) {
	if b.contextBuilder == nil {
		return "", fmt.Errorf("context builder is not initialized in PromptBuilder")
	}

	// Get context string for the query
	contextStr, err := b.contextBuilder.BuildContextForQuery(userQuery)
	if err != nil {
		// Log the error but proceed without context if retrieval fails
		log.Printf("Warning: Failed to build context for prompt, proceeding without it: %v", err)
		contextStr = "" // Ensure contextStr is empty if there was an error
		// Depending on requirements, you might want to return the error instead:
		// return "", fmt.Errorf("failed to build context: %w", err)
	}

	var sb strings.Builder

	// --- System Instructions ---
	// Provide clear instructions to the LLM on its role and how to use the context.
	sb.WriteString("SYSTEM INSTRUCTIONS:\n")
	sb.WriteString("You are an AI assistant helping a fiction writer manage their worldbuilding codex (characters, locations, lore, etc.). ")
	sb.WriteString("Your goal is to answer the user's query based on the provided CONTEXT INFORMATION below. ")
	sb.WriteString("The context contains relevant entries from the writer's codex, ordered by relevance to the query. ")
	sb.WriteString("If the context contains information relevant to the query, prioritize using it in your answer. ")
	sb.WriteString("If the context does not seem relevant or is insufficient to answer the query fully, clearly state that and then use your general knowledge to provide the best possible response. ")
	sb.WriteString("Be creative and helpful, adopting the persona of a knowledgeable assistant for a writer.\n\n")
	// --- End System Instructions ---

	// --- Context Section ---
	if contextStr != "" {
		sb.WriteString(contextStr) // Context already includes a header "CONTEXT INFORMATION..."
		// sb.WriteString("\n") // Add extra newline for separation if needed
	} else {
		// Explicitly state if no context was found or used
		sb.WriteString("CONTEXT INFORMATION:\n")
		sb.WriteString("(No relevant context found in the codex for this query.)\n\n")
	}
	// --- End Context Section ---

	// --- User Query ---
	sb.WriteString("USER QUERY:\n")
	sb.WriteString(userQuery)
	// --- End User Query ---

	finalPrompt := sb.String()
	log.Printf("Built prompt with context (context length: %d chars)", len(contextStr)) // Log prompt creation

	return finalPrompt, nil
}

// BuildSimplePrompt creates a basic prompt without context retrieval (useful for other tasks)
func BuildSimplePrompt(systemInstruction, userQuery string) string {
	var sb strings.Builder
	if systemInstruction != "" {
		sb.WriteString("SYSTEM INSTRUCTIONS:\n")
		sb.WriteString(systemInstruction)
		sb.WriteString("\n\n")
	}
	sb.WriteString("USER QUERY:\n")
	sb.WriteString(userQuery)
	return sb.String()
}
