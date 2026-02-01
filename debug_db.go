package main

import (
	"fmt"
	"time"
)

// GetDatabaseContents returns a summary of what's in the database
func (a *App) GetDatabaseContents() string {
	var output string

	// Get counts
	stats, err := GetDatabaseStats()
	if err != nil {
		return fmt.Sprintf("Error getting stats: %v", err)
	}

	output += "=== DATABASE STATISTICS ===\n"
	output += fmt.Sprintf("File Events: %d\n", stats["file_events"])
	output += fmt.Sprintf("Window Events: %d\n", stats["window_events"])
	output += fmt.Sprintf("Clipboard Events: %d\n", stats["clipboard_events"])
	output += "\n"

	// Get recent window events
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	windowEvents, err := GetWindowEvents(start, now)
	if err == nil && len(windowEvents) > 0 {
		output += "=== RECENT WINDOW EVENTS (Last 24h) ===\n"
		for i, e := range windowEvents {
			if i >= 10 {
				break
			}
			output += fmt.Sprintf("[%s] %s - %s (%s) - %ds\n",
				e.Timestamp.Format("15:04:05"),
				e.AppName,
				e.WindowTitle,
				e.AppCategory,
				e.DurationSeconds)
		}
		output += "\n"
	}

	// Get recent file events
	fileEvents, err := GetFileEvents(start, now)
	if err == nil && len(fileEvents) > 0 {
		output += "=== RECENT FILE EVENTS (Last 24h) ===\n"
		for i, e := range fileEvents {
			if i >= 10 {
				break
			}
			output += fmt.Sprintf("[%s] %s: %s\n",
				e.Timestamp.Format("15:04:05"),
				e.Action,
				e.Filepath)
		}
		output += "\n"
	}

	// Get recent clipboard events
	clipboardEvents, err := GetClipboardEvents(start, now)
	if err == nil && len(clipboardEvents) > 0 {
		output += "=== RECENT CLIPBOARD EVENTS (Last 24h) ===\n"
		for i, e := range clipboardEvents {
			if i >= 10 {
				break
			}
			output += fmt.Sprintf("[%s] %s (%d bytes)\n",
				e.Timestamp.Format("15:04:05"),
				e.ContentPreview,
				e.ContentLength)
		}
		output += "\n"
	}

	return output
}
