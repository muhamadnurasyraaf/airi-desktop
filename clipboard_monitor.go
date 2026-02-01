package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var stopClipboardMonitor chan struct{}
var lastClipboardHash string

func StartClipboardMonitor() {
	stopClipboardMonitor = make(chan struct{})

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopClipboardMonitor:
				return
			case <-ticker.C:
				trackClipboard()
			}
		}
	}()
}

func trackClipboard() {
	var content string

	if runtime.GOOS == "darwin" {
		// macOS: use pbpaste
		content = getClipboardMac()
	} else if runtime.GOOS == "windows" {
		// Windows: use PowerShell
		content = getClipboardWindows()
	} else {
		// Linux: not supported for now
		return
	}

	if content == "" {
		return
	}

	// Hash the content to detect changes without storing full text
	hash := hashString(content)
	if hash == lastClipboardHash {
		return
	}

	lastClipboardHash = hash

	// Truncate content for storage (first 200 chars)
	preview := content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	err := InsertClipboardEvent(preview, len(content))
	if err != nil {
		log.Printf("Failed to insert clipboard event: %v", err)
	}
}

func getClipboardMac() string {
	cmd := exec.Command("pbpaste")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getClipboardWindows() string {
	script := `Get-Clipboard -Raw`
	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func StopClipboardMonitor() {
	if stopClipboardMonitor != nil {
		close(stopClipboardMonitor)
	}
}
