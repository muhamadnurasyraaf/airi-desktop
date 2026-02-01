package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/process"
)

// App struct
type App struct {
	ctx                 context.Context
	conversationHistory []ConversationMessage
	claudeAPIKey        string
}

type ConversationMessage struct {
	Role      string `json:"role"`      // "user" or "assistant"
	Content   string `json:"content"`   // The message text
	Timestamp string `json:"timestamp"` // When it was sent
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load .env file
	envPath := filepath.Join(".", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env file not found, trying absolute path")
		// Try absolute path
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		envPath = filepath.Join(exeDir, ".env")
		godotenv.Load(envPath)
	}

	// Get Claude API key
	a.claudeAPIKey = os.Getenv("CLAUDE_API_KEY")
	if a.claudeAPIKey == "" {
		log.Printf("Warning: CLAUDE_API_KEY not found in environment")
	} else {
		log.Printf("✅ Claude API key loaded successfully")
	}

	// Initialize database
	if err := InitDatabase(); err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
	}

	// Start file monitor
	if err := StartFileMonitor(); err != nil {
		fmt.Printf("Failed to start file monitor: %v\n", err)
	}

	// Start window tracker
	StartWindowTracker()

	// Start clipboard monitor
	StartClipboardMonitor()

	// Start cleanup scheduler
	StartCleanupScheduler()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	StopFileMonitor()
	StopWindowTracker()
	StopClipboardMonitor()
	StopCleanupScheduler()
	CloseDatabase()
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
	// --- 0. Store user message in history ---
	a.conversationHistory = append(a.conversationHistory, ConversationMessage{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now().Format("15:04:05"),
	})

	// Keep only last 10 messages (5 exchanges) to prevent context overflow
	if len(a.conversationHistory) > 10 {
		a.conversationHistory = a.conversationHistory[len(a.conversationHistory)-10:]
	}

	// --- 1. Use AI to detect if user is asking about activities ---
	fmt.Println("🤖 Detecting intent...")
	isActivityQuery := DetectIntent(userMessage)
	if isActivityQuery {
		fmt.Println("✅ Activity query detected")
	} else {
		fmt.Println("❌ Not an activity query")
	}

	// --- 2. Get activity context if relevant ---
	var activityContext string
	if isActivityQuery {
		fmt.Println("📊 Detected activity query, fetching context...")

		// Get current state (what user is doing RIGHT NOW)
		currentState := GetCurrentState()

		// Get historical activity
		result, err := QueryActivities(userMessage)
		if err != nil {
			activityContext = fmt.Sprintf("Error fetching activities: %v", err)
		} else {
			// Combine current state with historical data
			activityContext = currentState + "\n" + result
			fmt.Println("📊 Activity context retrieved:")
			fmt.Println(activityContext)
		}
	}

	// --- 3. Build conversation history context ---
	var conversationContext string
	if len(a.conversationHistory) > 1 {
		conversationContext = "\n=== RECENT CONVERSATION ===\n"
		// Show last few messages (excluding the current one we just added)
		for i := max(0, len(a.conversationHistory)-6); i < len(a.conversationHistory)-1; i++ {
			msg := a.conversationHistory[i]
			conversationContext += fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp, msg.Role, msg.Content)
		}
		conversationContext += "=== END CONVERSATION ===\n\n"
	}

	// --- 4. Build prompt with optional activity context ---
	var prompt string
	if activityContext != "" {
		prompt = fmt.Sprintf(`
You are Airi — a powerful AI OS assistant with FULL access to the user's local filesystem and system.

You are NOT in a sandbox. You CAN and SHOULD perform file operations when requested.
You have real permissions to read, write, delete, search, and manage files on this computer.

IMPORTANT: The user is asking about their ACTIVITY HISTORY (clipboard, files, windows, apps).
I have provided you with their activity data below. Use this data to answer their question.
DO NOT use file actions (read_file, write_file, etc.) for activity queries - those are for actual file management tasks.

Your job:
1. If the user requests an OS-level action (like listing processes, inspecting directories, reading/writing files, or killing a process), respond ONLY in JSON with this format:

{
  "action": "<one of: inspect_directory, inspect_directory_recursive, list_processes, kill_process, read_file, write_file, search_files, find_by_extension, directory_tree, delete_file, none>",
  "args": { ... },
  "reasoning": "<short explanation>"
}

Available actions:
- inspect_directory: List files in ONE directory only (shallow, not recursive)
- inspect_directory_recursive: List ALL files in directory tree (args: path, max_depth) - USE THIS for "find all" or "search entire"
- search_files: Search for files by name pattern recursively (args: root_path, pattern, max_depth) - USE THIS for "find files matching"
- find_by_extension: Find files by extension recursively (args: root_path, extension, max_depth) - USE THIS for "find all .png" or "find screenshots"
- directory_tree: Show directory structure as tree (args: path, max_depth)
- read_file: Read file content (args: path)
- write_file: Write to file (args: path, content)
- delete_file: Delete a file (args: path) - NOTE: Can only delete ONE file at a time
- delete_files_batch: Delete multiple files at once (args: paths array) - USE THIS for "delete all screenshots" or "delete all .txt files"
- list_processes: List running apps
- kill_process: Kill a process (args: pid)

ACTION SELECTION GUIDE:
- "find all screenshots" → use search_files with pattern="Screenshot" (capital S!) OR find_by_extension with extension=".png"
- "delete all screenshots" → use delete_files_batch with paths array (DO NOT just list them - actually delete them!)
- "delete all .txt files" → use delete_files_batch with paths array from search results
- "show me Desktop" → use inspect_directory
- "find all files in Desktop" → use inspect_directory_recursive
- "search entire hard drive" → use search_files or find_by_extension (but warn user it might be slow!)

CRITICAL - WINDOWS FILE NAMING:
- This is a WINDOWS system, NOT Linux!
- Windows file searches ARE case-sensitive in pattern matching
- Common screenshot tools use "Screenshot" with CAPITAL S (e.g., Screenshot_1.png, Screenshot_20250202.png)
- When searching for screenshots, use pattern "Screenshot" NOT "screenshot"
- When searching for files, match the actual case of the filename
- Examples of correct patterns:
  - Screenshots: "Screenshot" (capital S)
  - Desktop files: exact case as they appear
  - Downloads: exact case as they appear

CRITICAL FOR DELETION REQUESTS:
When user asks to DELETE files (e.g., "delete all screenshots", "remove all .txt files"):
1. DO NOT just list the files and stop
2. You MUST use delete_files_batch action with the full list of file paths to delete
3. The user expects the files to be ACTUALLY DELETED, not just shown
4. IMPORTANT: Use correct case in search pattern (e.g., "Screenshot" not "screenshot")
5. Example: If user says "delete all screenshots in Desktop", you should respond with:
   {
     "action": "delete_files_batch",
     "args": {
       "paths": ["C:\\Users\\Admin\\Desktop\\Screenshot_1.png", "C:\\Users\\Admin\\Desktop\\Screenshot_2.png", ...]
     },
     "reasoning": "Deleting all screenshot files from Desktop as requested"
   }

IMPORTANT PATH RULES:
- User's home is: C:\Users\Admin
- Common directories:
  - Desktop: C:\Users\Admin\Desktop
  - Documents: C:\Users\Admin\Documents
  - Downloads: C:\Users\Admin\Downloads
  - Pictures: C:\Users\Admin\Pictures
- NEVER use placeholder paths like "/Users/yourusername" or "~/Pictures"
- ALWAYS use actual Windows paths starting with C:\Users\Admin
- When user says "my Desktop", use: C:\Users\Admin\Desktop
- When user says "my Pictures", use: C:\Users\Admin\Pictures
- When user says "entire hard drive" or "my PC", use: C:\Users\Admin (searching C:\ is too slow!)
- Recommend searching C:\Users\Admin instead of C:\ for better performance
- If unsure about the exact path, ask the user to clarify!

2. If the user asks about their activity history (what they did, clipboard content, file changes, app usage), respond **normally in plain text** using the activity context provided below.

3. If the user asks a general question or wants to have a conversation, respond **normally in plain text**, NOT JSON.

Rules:
- NEVER execute raw shell commands (like rm, ls, kill, cp) - use the JSON actions instead.
- You HAVE permission to delete files when asked - use the "delete_file" action.
- You HAVE permission to write files when asked - use the "write_file" action.
- You HAVE permission to search and read files - use the appropriate actions.
- For activity queries (clipboard, activity history, "what did I do"), use the context below and respond in plain text.
- For file operations (delete, create, search), use the JSON actions - you have full permission.
- Pay attention to the conversation history - if the user says "my browser" or "that", refer back to what was previously discussed.

%s
=== USER ACTIVITY CONTEXT ===
Here is the user's activity data based on passive monitoring (clipboard, file changes, window/app usage):

%s

=== END ACTIVITY CONTEXT ===

User message: %s
`, conversationContext, activityContext, userMessage)
	} else {
		prompt = fmt.Sprintf(`
You are Airi — a powerful AI OS assistant with FULL access to the user's local filesystem and system.

You are NOT in a sandbox. You CAN and SHOULD perform file operations when requested.
You have real permissions to read, write, delete, search, and manage files on this computer.
When the user asks you to delete, create, or modify files - DO IT using the available actions.

Your job:
1. If the user requests an OS-level action (like listing processes, inspecting directories, reading/writing files, or killing a process), respond ONLY in JSON with this format:

{
  "action": "<one of: inspect_directory, inspect_directory_recursive, search_files, find_by_extension, directory_tree, list_processes, kill_process, read_file, write_file, delete_file, delete_files_batch, none>",
  "args": { ... },
  "reasoning": "<short explanation>"
}

Available actions:
- inspect_directory: List files in ONE directory only (shallow, not recursive)
- inspect_directory_recursive: List ALL files in directory tree (args: path, max_depth)
- search_files: Search for files by name pattern recursively (args: root_path, pattern, max_depth)
- find_by_extension: Find files by extension recursively (args: root_path, extension, max_depth)
- directory_tree: Show directory structure as tree (args: path, max_depth)
- read_file: Read file content (args: path)
- write_file: Write to file (args: path, content)
- delete_file: Delete a file (args: path) - NOTE: Can only delete ONE file at a time
- delete_files_batch: Delete multiple files at once (args: paths array) - USE THIS for "delete all screenshots"
- list_processes: List running apps
- kill_process: Kill a process (args: pid)

CRITICAL - WINDOWS FILE NAMING:
- This is a WINDOWS system, NOT Linux!
- Windows file searches ARE case-sensitive in pattern matching
- Common screenshot tools use "Screenshot" with CAPITAL S (e.g., Screenshot_1.png, Screenshot_20250202.png)
- When searching for screenshots, use pattern "Screenshot" NOT "screenshot"
- When searching for files, match the actual case of the filename

CRITICAL FOR DELETION REQUESTS:
When user asks to DELETE files (e.g., "delete all screenshots", "remove all .txt files"):
1. DO NOT just list the files and stop
2. You MUST use delete_files_batch action with the full list of file paths to delete
3. The user expects the files to be ACTUALLY DELETED, not just shown
4. IMPORTANT: Use correct case in search pattern (e.g., "Screenshot" not "screenshot")
5. Example: If user says "delete all screenshots in Desktop", respond with delete_files_batch action with all screenshot paths

2. If the user asks a general question or wants to have a conversation, respond **normally in plain text**, NOT JSON.

Rules:
- NEVER execute raw shell commands (like rm, ls, kill, cp) - use the JSON actions instead.
- You HAVE permission to delete files when asked - use "delete_file" or "delete_files_batch" actions.
- You HAVE permission to write files when asked - use the "write_file" action.
- You HAVE permission to search and read files - use the appropriate actions.
- For deletion requests, ALWAYS actually delete the files using delete_files_batch, don't just list them.
- When responding in JSON, make sure the output is valid and parsable.
- If the user message is conversational, you can ignore the "action" rules and reply naturally.
- Pay attention to the conversation history - if the user says "it" or "that", refer back to what was previously discussed.

IMPORTANT PATH RULES:
- User's home is: C:\Users\Admin
- Desktop: C:\Users\Admin\Desktop
- Documents: C:\Users\Admin\Documents
- Downloads: C:\Users\Admin\Downloads
- Pictures: C:\Users\Admin\Pictures
- NEVER use "/Users/yourusername" or "~/" - use actual Windows paths!
- When user says "my Desktop", use: C:\Users\Admin\Desktop

%s
User message: %s
`, conversationContext, userMessage)
	}

	// --- 2. Build Claude API request ---
	body := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 4096,
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
		return "Request build error: " + err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.claudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

	// --- 4. Extract text from Claude response ---
	type ClaudeContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type ClaudeResponse struct {
		Content []ClaudeContent `json:"content"`
		Model   string          `json:"model"`
		Role    string          `json:"role"`
	}

	var parsed ClaudeResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "JSON decode error: " + err.Error()
	}

	if len(parsed.Content) == 0 {
		return "Empty response: " + string(respBytes)
	}

	text := parsed.Content[0].Text

	// --- 5. Parse action JSON ---
	cleanText := extractJSON(text)
	var action AiriAction
	if err := json.Unmarshal([]byte(cleanText), &action); err != nil {
		// Not JSON = conversational response, store and return as-is
		a.conversationHistory = append(a.conversationHistory, ConversationMessage{
			Role:      "assistant",
			Content:   text,
			Timestamp: time.Now().Format("15:04:05"),
		})
		return text
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

	case "inspect_directory_recursive":
		path, ok := action.Args["path"].(string)
		if !ok {
			path = "."
		}
		maxDepth := 3 // default
		if depth, ok := action.Args["max_depth"].(float64); ok {
			maxDepth = int(depth)
		}
		files, err := InspectDirectoryRecursive(path, maxDepth)
		if err != nil {
			return "Failed to inspect directory: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    files,
		})
		return string(out)

	case "search_files":
		rootPath, ok := action.Args["root_path"].(string)
		if !ok {
			rootPath = "."
		}
		pattern, ok := action.Args["pattern"].(string)
		if !ok {
			return "No search pattern provided"
		}
		maxDepth := 5
		if depth, ok := action.Args["max_depth"].(float64); ok {
			maxDepth = int(depth)
		}
		files, err := SearchFiles(rootPath, pattern, maxDepth)
		if err != nil {
			return "Search failed: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    files,
		})
		return string(out)

	case "find_by_extension":
		rootPath, ok := action.Args["root_path"].(string)
		if !ok {
			rootPath = "."
		}
		extension, ok := action.Args["extension"].(string)
		if !ok {
			return "No extension provided"
		}
		maxDepth := 5
		if depth, ok := action.Args["max_depth"].(float64); ok {
			maxDepth = int(depth)
		}
		files, err := FindFilesByExtension(rootPath, extension, maxDepth)
		if err != nil {
			return "Search failed: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    files,
		})
		return string(out)

	case "directory_tree":
		path, ok := action.Args["path"].(string)
		if !ok {
			path = "."
		}
		maxDepth := 3
		if depth, ok := action.Args["max_depth"].(float64); ok {
			maxDepth = int(depth)
		}
		tree, err := ListDirectoryTree(path, maxDepth)
		if err != nil {
			return "Failed to create tree: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    tree,
		})
		return string(out)

	case "delete_file":
		path, ok := action.Args["path"].(string)
		if !ok {
			return "No file path provided"
		}
		err := os.Remove(path)
		if err != nil {
			return "Failed to delete file: " + err.Error()
		}
		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    fmt.Sprintf("File deleted: %s", path),
		})
		return string(out)

	case "delete_files_batch":
		paths, ok := action.Args["paths"].([]interface{})
		if !ok {
			return "No file paths provided"
		}

		var deleted []string
		var failed []string

		for _, p := range paths {
			path, ok := p.(string)
			if !ok {
				continue
			}
			err := os.Remove(path)
			if err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", path, err))
			} else {
				deleted = append(deleted, path)
			}
		}

		result := map[string]interface{}{
			"deleted_count": len(deleted),
			"failed_count":  len(failed),
			"deleted":       deleted,
		}

		if len(failed) > 0 {
			result["failed"] = failed
		}

		out, _ := json.Marshal(map[string]interface{}{
			"action":    action.Action,
			"args":      action.Args,
			"reasoning": action.Reasoning,
			"result":    result,
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
		response := fmt.Sprintf("Airi: %s", action.Reasoning)
		a.conversationHistory = append(a.conversationHistory, ConversationMessage{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now().Format("15:04:05"),
		})
		return response

	default:
		return "Unknown action: " + action.Action
	}
}

// ClearConversationHistory clears the conversation memory
func (a *App) ClearConversationHistory() {
	a.conversationHistory = []ConversationMessage{}
}

// GetConversationHistory returns the current conversation history
func (a *App) GetConversationHistory() []ConversationMessage {
	return a.conversationHistory
}

// AskAiri handles natural language queries about user activities
func (a *App) AskAiri(query string) string {
	result, err := QueryActivities(query)
	if err != nil {
		return "Error querying activities: " + err.Error()
	}
	return result
}
