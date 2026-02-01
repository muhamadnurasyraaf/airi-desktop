package main

import (
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var stopWindowTracker chan struct{}
var lastAppName string
var lastWindowTitle string
var lastWindowTime time.Time

func StartWindowTracker() {
	stopWindowTracker = make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Do first check immediately
		trackActiveWindow()

		for {
			select {
			case <-stopWindowTracker:
				return
			case <-ticker.C:
				trackActiveWindow()
			}
		}
	}()
}

func trackActiveWindow() {
	// Check idle state first
	TrackIdleState()

	// Skip tracking window if system is idle
	if IsIdle() {
		return
	}

	var appName, windowTitle string

	if runtime.GOOS == "darwin" {
		// macOS: use osascript
		appName, windowTitle = getActiveWindowMac()
	} else if runtime.GOOS == "windows" {
		// Windows: use PowerShell
		appName, windowTitle = getActiveWindowWindows()
	} else {
		// Linux: not supported for MVP
		return
	}

	if appName == "" {
		return
	}

	now := time.Now()

	// Only insert if changed (avoid duplicate entries)
	if appName == lastAppName && windowTitle == lastWindowTitle {
		return
	}

	// Update duration for previous window if exists
	if lastAppName != "" && !lastWindowTime.IsZero() {
		duration := int(now.Sub(lastWindowTime).Seconds())
		err := UpdateLastWindowDuration(duration)
		if err != nil {
			log.Printf("Failed to update window duration: %v", err)
		}
	}

	lastAppName = appName
	lastWindowTitle = windowTitle
	lastWindowTime = now

	err := InsertWindowEvent(appName, windowTitle)
	if err != nil {
		log.Printf("Failed to insert window event: %v", err)
	}
}

func getActiveWindowMac() (string, string) {
	// Get frontmost app name
	appCmd := exec.Command("osascript", "-e", `tell application "System Events" to get name of first application process whose frontmost is true`)
	appOutput, err := appCmd.Output()
	if err != nil {
		return "", ""
	}
	appName := strings.TrimSpace(string(appOutput))

	// Get window title
	titleCmd := exec.Command("osascript", "-e", `
		tell application "System Events"
			set frontApp to first application process whose frontmost is true
			tell frontApp
				if (count of windows) > 0 then
					return name of front window
				else
					return ""
				end if
			end tell
		end tell
	`)
	titleOutput, _ := titleCmd.Output()
	windowTitle := strings.TrimSpace(string(titleOutput))

	return appName, windowTitle
}

func getActiveWindowWindows() (string, string) {
	// PowerShell command to get active window
	script := `
Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
public class WinAPI {
    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int count);
    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);
}
"@
$hwnd = [WinAPI]::GetForegroundWindow()
$sb = New-Object System.Text.StringBuilder 256
[WinAPI]::GetWindowText($hwnd, $sb, 256) | Out-Null
$processId = 0
[WinAPI]::GetWindowThreadProcessId($hwnd, [ref]$processId) | Out-Null
$process = Get-Process -Id $processId -ErrorAction SilentlyContinue
"$($process.ProcessName)|$($sb.ToString())"
`
	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func StopWindowTracker() {
	if stopWindowTracker != nil {
		close(stopWindowTracker)
	}
}
