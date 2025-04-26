package main

// LLMResponse structure for handling responses from the language model
type LLMResponse struct {
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
	// Add other fields as needed based on the actual Bedrock API response
}

// You might want to add other models relevant to your application logic here.
