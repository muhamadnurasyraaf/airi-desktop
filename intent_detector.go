package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// DetectIntent uses AI to determine if the user is asking about their activities
func DetectIntent(userMessage string) bool {
	prompt := fmt.Sprintf(`You are an intent classifier. Determine if the user is asking about their PERSONAL ACTIVITIES, ACTIONS, or HISTORY.

Examples of activity queries (return true):
- "what am I working on right now?"
- "what im currently doing"
- "show me my browser"
- "what did I do today"
- "what files did I edit"
- "what did I copy"
- "tell me about my day"
- "how productive was I"
- "what apps am I using"
- "what's my current task"
- "summarize my work"

Examples of NON-activity queries (return false):
- "how are you"
- "tell me a joke"
- "what's the weather"
- "help me with this code"
- "explain this concept"
- "write a function"

User message: "%s"

Respond with ONLY one word: "true" or "false"`, userMessage)

	// Get Claude API key from environment
	claudeKey := os.Getenv("CLAUDE_API_KEY")
	if claudeKey == "" {
		return false // Fallback if no API key
	}

	body := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 10,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(
		"POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return false // Fallback to no activity context
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", claudeKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	type ClaudeContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type ClaudeResponse struct {
		Content []ClaudeContent `json:"content"`
	}

	var parsed ClaudeResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return false
	}

	if len(parsed.Content) == 0 {
		return false
	}

	text := parsed.Content[0].Text

	// Check if response contains "true"
	return contains(text, "true")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
