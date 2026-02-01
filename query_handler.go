package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func ParseDateRange(query string) (time.Time, time.Time) {
	now := time.Now()
	query = strings.ToLower(query)

	// Default to today
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	if strings.Contains(query, "yesterday") {
		start = start.AddDate(0, 0, -1)
		end = end.AddDate(0, 0, -1)
	} else if strings.Contains(query, "this week") || strings.Contains(query, "week") {
		// Go back to Monday
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = start.AddDate(0, 0, -(weekday - 1))
	} else if strings.Contains(query, "last hour") {
		start = now.Add(-1 * time.Hour)
		end = now
	}

	return start, end
}

// UnifiedActivity represents a single activity event
type UnifiedActivity struct {
	Timestamp   time.Time
	Type        string // "file", "window", "clipboard"
	Description string
	Category    string
	Duration    int
}

func FormatActivities(fileEvents []FileEvent, windowEvents []WindowEvent, clipboardEvents []ClipboardEvent) string {
	// Create unified timeline
	var activities []UnifiedActivity

	// Add file events
	for _, e := range fileEvents {
		activities = append(activities, UnifiedActivity{
			Timestamp:   e.Timestamp,
			Type:        "file",
			Description: fmt.Sprintf("%s: %s", e.Action, e.Filepath),
			Category:    "file_management",
		})
	}

	// Add window events
	for _, e := range windowEvents {
		title := e.WindowTitle
		if title == "" {
			title = "(no title)"
		}
		durationStr := ""
		if e.DurationSeconds > 0 {
			durationStr = fmt.Sprintf(" (%dm %ds)", e.DurationSeconds/60, e.DurationSeconds%60)
		}
		activities = append(activities, UnifiedActivity{
			Timestamp:   e.Timestamp,
			Type:        "window",
			Description: fmt.Sprintf("%s - %s%s", e.AppName, title, durationStr),
			Category:    e.AppCategory,
			Duration:    e.DurationSeconds,
		})
	}

	// Add clipboard events
	for _, e := range clipboardEvents {
		sizeStr := ""
		if e.ContentLength > 1000 {
			sizeStr = fmt.Sprintf(" [%d KB]", e.ContentLength/1000)
		} else {
			sizeStr = fmt.Sprintf(" [%d bytes]", e.ContentLength)
		}
		activities = append(activities, UnifiedActivity{
			Timestamp:   e.Timestamp,
			Type:        "clipboard",
			Description: fmt.Sprintf("Copied: %s%s", e.ContentPreview, sizeStr),
			Category:    "clipboard",
		})
	}

	// Sort by timestamp
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[i].Timestamp.After(activities[j].Timestamp) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Format as timeline
	var sb strings.Builder
	sb.WriteString("=== UNIFIED ACTIVITY TIMELINE ===\n\n")

	if len(activities) == 0 {
		sb.WriteString("No activities recorded.\n")
		return sb.String()
	}

	lastDate := ""
	for _, a := range activities {
		dateStr := a.Timestamp.Format("2006-01-02")
		if dateStr != lastDate {
			sb.WriteString(fmt.Sprintf("\n📅 %s\n", a.Timestamp.Format("Monday, January 2, 2006")))
			lastDate = dateStr
		}

		icon := "📄"
		switch a.Type {
		case "window":
			icon = "🪟"
		case "clipboard":
			icon = "📋"
		}

		sb.WriteString(fmt.Sprintf("[%s] %s [%s] %s\n",
			a.Timestamp.Format("15:04:05"),
			icon,
			a.Category,
			a.Description,
		))
	}

	return sb.String()
}

func QueryActivities(query string) (string, error) {
	// Parse date range from query
	start, end := ParseDateRange(query)

	// Get events from database
	fileEvents, err := GetFileEvents(start, end)
	if err != nil {
		return "", err
	}

	windowEvents, err := GetWindowEvents(start, end)
	if err != nil {
		return "", err
	}

	clipboardEvents, err := GetClipboardEvents(start, end)
	if err != nil {
		return "", err
	}

	// Format activities as text
	activities := FormatActivities(fileEvents, windowEvents, clipboardEvents)

	// If no activities, return early
	if len(fileEvents) == 0 && len(windowEvents) == 0 && len(clipboardEvents) == 0 {
		return "No activities recorded for the requested time period.", nil
	}

	// Send to Gemini for formatting
	summary, err := SummarizeWithGemini(activities, query)
	if err != nil {
		// Fallback to raw activities if Gemini fails
		return activities, nil
	}

	return summary, nil
}

func SummarizeWithGemini(activities string, userQuery string) (string, error) {
	prompt := fmt.Sprintf(`You are a timesheet assistant for someone with ADHD. They asked: "%s"

Here are their recorded activities:

%s

Format these activities into a clear, easy-to-read summary. Group by time blocks and task type. Be concise but helpful. If there are many similar file operations, summarize them (e.g., "Modified 5 files in project X"). Highlight the main focus areas.`, userQuery, activities)

	// Build request body
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
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=AIzaSyBLJYFfaZygUvOpD_YOVPre4SJ-p0uyCE8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse response
	var parsed map[string]interface{}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", err
	}

	candidates, ok := parsed["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	text := parts[0].(map[string]interface{})["text"].(string)

	return text, nil
}
