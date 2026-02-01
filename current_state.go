package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// GetCurrentState returns the current system state (active window, idle time, etc.)
func GetCurrentState() string {
	var sb strings.Builder

	sb.WriteString("=== CURRENT STATE ===\n")
	sb.WriteString(fmt.Sprintf("Time: %s\n\n", time.Now().Format("15:04:05 Monday, January 2, 2006")))

	// Get current active window
	var appName, windowTitle string
	if runtime.GOOS == "darwin" {
		appName, windowTitle = getActiveWindowMac()
	} else if runtime.GOOS == "windows" {
		appName, windowTitle = getActiveWindowWindows()
	}

	if appName != "" {
		category := categorizeApplication(appName)
		sb.WriteString(fmt.Sprintf("Active Application: %s\n", appName))
		sb.WriteString(fmt.Sprintf("Window Title: %s\n", windowTitle))
		sb.WriteString(fmt.Sprintf("Category: %s\n\n", category))
	} else {
		sb.WriteString("Active Application: Unable to detect\n\n")
	}

	// Get idle time
	idleSeconds := GetIdleTime()
	if idleSeconds > 0 {
		if idleSeconds < 60 {
			sb.WriteString(fmt.Sprintf("Idle Time: %d seconds (User is active)\n", idleSeconds))
		} else {
			minutes := idleSeconds / 60
			seconds := idleSeconds % 60
			sb.WriteString(fmt.Sprintf("Idle Time: %d minutes %d seconds", minutes, seconds))
			if idleSeconds > idleThresholdSeconds {
				sb.WriteString(" (User is IDLE/Away)\n")
			} else {
				sb.WriteString(" (User is active)\n")
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}
