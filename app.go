package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) Calculate(first int32, second int32) int32 {
	// Placeholder for calculation logic
	return first + second
}

type SystemApi struct{}

type RunningApp struct {
	PID  int32  `json:"pid"`
	Name string `json:"name"`
}

func (a *App) GetRunningApps() ([]RunningApp, error) {
	procs, err := process.Processes()

	if err != nil {
		return nil, err
	}

	var apps []RunningApp

	for _, p := range procs {
		name, err := p.Name()

		if err != nil || name == "" {
			continue
		}
		if strings.Contains(strings.ToLower(name), "system") {
			continue
		}

		apps = append(apps, RunningApp{
			PID:  p.Pid,
			Name: name,
		})
	}
	return apps, nil
}

type AiriAction struct {
	Action    string                 `json:"action"`
	Args      map[string]interface{} `json:"args"`
	Reasoning string                 `json:"reasoning"`
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Remove markdown code fences
	if strings.HasPrefix(s, "```") {
		parts := strings.Split(s, "```")
		if len(parts) >= 3 {
			s = strings.TrimSpace(parts[1])
		}
	}

	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return s // fallback
	}

	return strings.TrimSpace(s[start : end+1])
}

func (a *App) SendUserMessage(userMessage string) string {
	// --- 1. Build prompt ---
	prompt := fmt.Sprintf(`
You are Airi — an AI assistant and safe OS layer running inside a sandbox.

Your job:
1. If the user requests an OS-level action (like listing processes, inspecting directories, reading/writing files, or killing a process), respond ONLY in JSON with this format:

{
  "action": "<one of: inspect_directory, list_processes, kill_process, read_file, write_file, none>",
  "args": { ... },
  "reasoning": "<short explanation>"
}

2. If the user asks a general question or wants to have a conversation, respond **normally in plain text**, NOT JSON.

Rules:
- NEVER execute raw shell commands (like rm, ls, kill, cp).
- Only use the allowed action names listed above.
- When responding in JSON, make sure the output is valid and parsable.
- If the user message is conversational, you can ignore the "action" rules and reply naturally.

User message: %s
`, userMessage)

	// --- 2. Build HTTP request ---
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(
		"POST",
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent?key=AIzaSyBLJYFfaZygUvOpD_YOVPre4SJ-p0uyCE8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "Request build error: " + err.Error()
	}

	req.Header.Set("Content-Type", "application/json")

	// --- 3. Send request ---
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "HTTP error: " + err.Error()
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Read error: " + err.Error()
	}

	// --- 4. Extract text ---
	var parsed map[string]interface{}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "JSON decode error: " + err.Error()
	}

	candidates, ok := parsed["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "Empty candidates: " + string(respBytes)
	}

	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	text := parts[0].(map[string]interface{})["text"].(string)

	// --- 5. Parse action JSON ---
	cleanText := extractJSON(text)
	var action AiriAction
	if err := json.Unmarshal([]byte(cleanText), &action); err != nil {
		// Log for debugging if needed
		// fmt.Printf("JSON parse error: %v\nRaw: %s\nCleaned: %s\n", err, text, cleanText)
		return "Airi: Failed to parse action. Response was:\n" + text
	}

	// --- 6. Execute action ---
	switch action.Action {
	case "list_processes":
		procs, err := a.GetRunningApps()
		if err != nil {
			return "Error listing processes: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    procs,
		})
		return string(out)

	case "kill_process":
		pidFloat, ok := action.Args["pid"].(float64)
		if !ok {
			return "Invalid PID"
		}
		pid := int32(pidFloat)
		proc, err := process.NewProcess(pid)
		if err != nil {
			return "Process not found: " + err.Error()
		}
		err = proc.Kill()
		if err != nil {
			return "Failed to kill process: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    fmt.Sprintf("Process %d killed", pid),
		})
		return string(out)

	case "inspect_directory":
		path, ok := action.Args["path"].(string)
		if !ok {
			path = "." // default to current directory
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return "Failed to read directory: " + err.Error()
		}
		var filenames []string
		for _, f := range files {
			filenames = append(filenames, f.Name())
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    filenames,
		})
		return string(out)

	case "read_file":
		path, ok := action.Args["path"].(string)
		if !ok {
			return "No file path provided"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "Failed to read file: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    string(data),
		})
		return string(out)

	case "write_file":
		path, ok := action.Args["path"].(string)
		if !ok {
			return "No file path provided"
		}
		content, ok := action.Args["content"].(string)
		if !ok {
			return "No content to write"
		}
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "Failed to write file: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    fmt.Sprintf("File written: %s", path),
		})
		return string(out)

	case "none":
		return fmt.Sprintf("Airi: %s", action.Reasoning)

	default:
		return "Unknown action: " + action.Action
	}
}
