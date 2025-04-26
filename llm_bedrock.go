package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// BedrockClient wraps the AWS Bedrock Runtime client.
type BedrockClient struct {
	client *bedrockruntime.Client
	modelID string // e.g., "anthropic.claude-v2", "amazon.titan-text-express-v1"
}

// NewBedrockClient creates a new Bedrock client helper.
func NewBedrockClient(cfg aws.Config, modelID string) *BedrockClient {
	brClient := bedrockruntime.NewFromConfig(cfg)
	return &BedrockClient{
		client: brClient,
		modelID: modelID, // Store the default model ID
	}
}

// GenerateText sends a prompt to the configured Bedrock model and returns the response.
func (bc *BedrockClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	log.Printf("Sending prompt to Bedrock model %s: %.50s...", bc.modelID, prompt)

	// --- Construct Request Body for Anthropic Messages API --- 
	requestBody := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31", // Required for Claude 3/3.5
		"max_tokens":        4096,                 // Adjust as needed
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		// Optional parameters can be added here if needed:
		// "temperature": 0.7,
		// "top_p": 0.9,
		// "system": "Your system prompt here",
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshalling request body: %v", err)
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// --- Invoke Model --- 
	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(bc.modelID),
		Body:        requestBodyBytes,
		Accept:      aws.String("application/json"), // Recommended
		ContentType: aws.String("application/json"),
	}

	result, err := bc.client.InvokeModel(ctx, input)
	if err != nil {
		// Log detailed error information if possible
		log.Printf("Error invoking Bedrock model %s: %v", bc.modelID, err)
		return "", fmt.Errorf("failed to invoke bedrock model %s: %w", bc.modelID, err)
	}

	log.Printf("Successfully invoked model %s.", bc.modelID)

	// --- Parse Response Body (Messages API format) --- 
	var responseBody map[string]interface{}
	err = json.Unmarshal(result.Body, &responseBody)
	if err != nil {
		log.Printf("Error unmarshalling response body: %v", err)
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Extract content based on Messages API structure
	// responseBody -> content -> [0] -> text
	contentList, ok := responseBody["content"].([]interface{})
	if !ok || len(contentList) == 0 {
		log.Printf("Error parsing response: 'content' array not found or empty. Response: %+v", responseBody)
		return "", fmt.Errorf("failed to parse response: 'content' array not found or empty")
	}

	firstContentItem, ok := contentList[0].(map[string]interface{})
	if !ok {
		log.Printf("Error parsing response: First item in 'content' is not a map. Response: %+v", responseBody)
		return "", fmt.Errorf("failed to parse response: first item in 'content' is not a map")
	}

	responseText, ok := firstContentItem["text"].(string)
	if !ok {
		log.Printf("Error parsing response: 'text' field not found or not a string in content item. Response: %+v", responseBody)
		return "", fmt.Errorf("failed to parse response: 'text' field not found in content item")
	}

	log.Printf("Received response text: %.50s...", responseText)
	return responseText, nil
}
