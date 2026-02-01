package main

import (
	"log"
	"time"
)

var stopCleanupScheduler chan struct{}

const (
	// Cleanup runs daily at 3 AM
	cleanupHour   = 3
	cleanupMinute = 0
	daysToRetain  = 30 // Keep 30 days of history
)

func StartCleanupScheduler() {
	stopCleanupScheduler = make(chan struct{})

	go func() {
		// Run cleanup once on startup (if needed)
		runCleanupIfNeeded()

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-stopCleanupScheduler:
				return
			case <-ticker.C:
				runCleanupIfNeeded()
			}
		}
	}()
}

func runCleanupIfNeeded() {
	now := time.Now()

	// Only run if it's the right time (3 AM ± 1 hour)
	if now.Hour() >= cleanupHour-1 && now.Hour() <= cleanupHour+1 {
		log.Println("Running database cleanup...")

		// Get stats before cleanup
		statsBefore, err := GetDatabaseStats()
		if err != nil {
			log.Printf("Failed to get database stats: %v", err)
			return
		}

		// Run cleanup
		err = CleanupOldEvents(daysToRetain)
		if err != nil {
			log.Printf("Failed to cleanup old events: %v", err)
			return
		}

		// Get stats after cleanup
		statsAfter, err := GetDatabaseStats()
		if err != nil {
			log.Printf("Failed to get database stats after cleanup: %v", err)
			return
		}

		// Log results
		log.Printf("Cleanup complete. Before: %v, After: %v", statsBefore, statsAfter)
	}
}

func StopCleanupScheduler() {
	if stopCleanupScheduler != nil {
		close(stopCleanupScheduler)
	}
}
