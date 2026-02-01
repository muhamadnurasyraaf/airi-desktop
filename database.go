package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func InitDatabase() error {
	// Create data directory in user's home
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".airi")
	os.MkdirAll(dataDir, 0755)

	dbPath := filepath.Join(dataDir, "airi.db")

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Create file_events table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			filepath TEXT NOT NULL,
			action TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create window_events table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS window_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			app_name TEXT NOT NULL,
			window_title TEXT NOT NULL,
			duration_seconds INTEGER DEFAULT 0,
			app_category TEXT DEFAULT 'unknown'
		)
	`)
	if err != nil {
		return err
	}

	// Create clipboard_events table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clipboard_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			content_preview TEXT NOT NULL,
			content_length INTEGER NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create idle_events table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS idle_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			event_type TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Run migrations to update existing tables
	err = runMigrations()
	if err != nil {
		return err
	}

	return nil
}

func runMigrations() error {
	// Check if window_events has duration_seconds column
	var hasDuration bool
	rows, err := db.Query("PRAGMA table_info(window_events)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue interface{}
		var pk int
		rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if name == "duration_seconds" {
			hasDuration = true
		}
	}

	// Add duration_seconds column if missing
	if !hasDuration {
		_, err = db.Exec("ALTER TABLE window_events ADD COLUMN duration_seconds INTEGER DEFAULT 0")
		if err != nil {
			return err
		}
	}

	// Check if window_events has app_category column
	var hasCategory bool
	rows, err = db.Query("PRAGMA table_info(window_events)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue interface{}
		var pk int
		rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if name == "app_category" {
			hasCategory = true
		}
	}

	// Add app_category column if missing
	if !hasCategory {
		_, err = db.Exec("ALTER TABLE window_events ADD COLUMN app_category TEXT DEFAULT 'unknown'")
		if err != nil {
			return err
		}
	}

	return nil
}

func InsertFileEvent(filepath string, action string) error {
	_, err := db.Exec(
		"INSERT INTO file_events (timestamp, filepath, action) VALUES (?, ?, ?)",
		time.Now(), filepath, action,
	)
	return err
}

func InsertWindowEvent(appName string, windowTitle string) error {
	category := categorizeApplication(appName)
	_, err := db.Exec(
		"INSERT INTO window_events (timestamp, app_name, window_title, app_category) VALUES (?, ?, ?, ?)",
		time.Now(), appName, windowTitle, category,
	)
	return err
}

func UpdateLastWindowDuration(duration int) error {
	_, err := db.Exec(
		"UPDATE window_events SET duration_seconds = ? WHERE id = (SELECT MAX(id) FROM window_events WHERE id < (SELECT MAX(id) FROM window_events))",
		duration,
	)
	return err
}

func InsertClipboardEvent(contentPreview string, contentLength int) error {
	_, err := db.Exec(
		"INSERT INTO clipboard_events (timestamp, content_preview, content_length) VALUES (?, ?, ?)",
		time.Now(), contentPreview, contentLength,
	)
	return err
}

func InsertIdleEvent(eventType string) error {
	_, err := db.Exec(
		"INSERT INTO idle_events (timestamp, event_type) VALUES (?, ?)",
		time.Now(), eventType,
	)
	return err
}

type FileEvent struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Filepath  string    `json:"filepath"`
	Action    string    `json:"action"`
}

type WindowEvent struct {
	ID              int64     `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	AppName         string    `json:"app_name"`
	WindowTitle     string    `json:"window_title"`
	DurationSeconds int       `json:"duration_seconds"`
	AppCategory     string    `json:"app_category"`
}

type ClipboardEvent struct {
	ID             int64     `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	ContentPreview string    `json:"content_preview"`
	ContentLength  int       `json:"content_length"`
}

func GetFileEvents(start time.Time, end time.Time) ([]FileEvent, error) {
	rows, err := db.Query(
		"SELECT id, timestamp, filepath, action FROM file_events WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp",
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []FileEvent
	for rows.Next() {
		var e FileEvent
		rows.Scan(&e.ID, &e.Timestamp, &e.Filepath, &e.Action)
		events = append(events, e)
	}
	return events, nil
}

func GetWindowEvents(start time.Time, end time.Time) ([]WindowEvent, error) {
	rows, err := db.Query(
		"SELECT id, timestamp, app_name, window_title, duration_seconds, app_category FROM window_events WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp",
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []WindowEvent
	for rows.Next() {
		var e WindowEvent
		rows.Scan(&e.ID, &e.Timestamp, &e.AppName, &e.WindowTitle, &e.DurationSeconds, &e.AppCategory)
		events = append(events, e)
	}
	return events, nil
}

func GetClipboardEvents(start time.Time, end time.Time) ([]ClipboardEvent, error) {
	rows, err := db.Query(
		"SELECT id, timestamp, content_preview, content_length FROM clipboard_events WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp",
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ClipboardEvent
	for rows.Next() {
		var e ClipboardEvent
		rows.Scan(&e.ID, &e.Timestamp, &e.ContentPreview, &e.ContentLength)
		events = append(events, e)
	}
	return events, nil
}

func CleanupOldEvents(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	// Delete old file events
	_, err := db.Exec("DELETE FROM file_events WHERE timestamp < ?", cutoffDate)
	if err != nil {
		return err
	}

	// Delete old window events
	_, err = db.Exec("DELETE FROM window_events WHERE timestamp < ?", cutoffDate)
	if err != nil {
		return err
	}

	// Delete old clipboard events
	_, err = db.Exec("DELETE FROM clipboard_events WHERE timestamp < ?", cutoffDate)
	if err != nil {
		return err
	}

	// Delete old idle events
	_, err = db.Exec("DELETE FROM idle_events WHERE timestamp < ?", cutoffDate)
	if err != nil {
		return err
	}

	// Vacuum database to reclaim space
	_, err = db.Exec("VACUUM")
	return err
}

func GetDatabaseStats() (map[string]int, error) {
	stats := make(map[string]int)

	// Count file events
	var fileCount int
	err := db.QueryRow("SELECT COUNT(*) FROM file_events").Scan(&fileCount)
	if err != nil {
		return nil, err
	}
	stats["file_events"] = fileCount

	// Count window events
	var windowCount int
	err = db.QueryRow("SELECT COUNT(*) FROM window_events").Scan(&windowCount)
	if err != nil {
		return nil, err
	}
	stats["window_events"] = windowCount

	// Count clipboard events
	var clipboardCount int
	err = db.QueryRow("SELECT COUNT(*) FROM clipboard_events").Scan(&clipboardCount)
	if err != nil {
		return nil, err
	}
	stats["clipboard_events"] = clipboardCount

	return stats, nil
}

func CloseDatabase() {
	if db != nil {
		db.Close()
	}
}
