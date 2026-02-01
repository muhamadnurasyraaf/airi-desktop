package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

var fileWatcher *fsnotify.Watcher
var stopFileMonitor chan struct{}

func StartFileMonitor() error {
	var err error
	fileWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	stopFileMonitor = make(chan struct{})

	// Get directories to watch
	home, _ := os.UserHomeDir()
	watchDirs := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "Development"),
		filepath.Join(home, "OneDrive"),
	}

	// Add directories to watcher
	for _, dir := range watchDirs {
		if _, err := os.Stat(dir); err == nil {
			// Watch the directory itself
			fileWatcher.Add(dir)

			// Also watch subdirectories (one level deep for MVP)
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				if entry.IsDir() {
					subdir := filepath.Join(dir, entry.Name())
					fileWatcher.Add(subdir)
				}
			}
		}
	}

	// Start watching in background
	go func() {
		for {
			select {
			case <-stopFileMonitor:
				return
			case event, ok := <-fileWatcher.Events:
				if !ok {
					return
				}

				var action string
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					action = "created"
				case event.Op&fsnotify.Write == fsnotify.Write:
					action = "modified"
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					action = "deleted"
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					action = "renamed"
				default:
					continue
				}

				// Insert into database
				err := InsertFileEvent(event.Name, action)
				if err != nil {
					log.Printf("Failed to insert file event: %v", err)
				}

			case err, ok := <-fileWatcher.Errors:
				if !ok {
					return
				}
				log.Printf("File watcher error: %v", err)
			}
		}
	}()

	return nil
}

func StopFileMonitor() {
	if stopFileMonitor != nil {
		close(stopFileMonitor)
	}
	if fileWatcher != nil {
		fileWatcher.Close()
	}
}
