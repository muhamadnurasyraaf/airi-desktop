package main

import (
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

var lastIdleState bool
var idleThresholdSeconds = 300 // 5 minutes

// GetIdleTime returns the number of seconds the system has been idle
func GetIdleTime() int {
	if runtime.GOOS == "darwin" {
		return getIdleTimeMac()
	} else if runtime.GOOS == "windows" {
		return getIdleTimeWindows()
	}
	return 0
}

func getIdleTimeMac() int {
	// Use ioreg to get HIDIdleTime
	cmd := exec.Command("ioreg", "-c", "IOHIDSystem")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "HIDIdleTime") {
			// Extract the number
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				numStr := strings.TrimSpace(parts[1])
				num, err := strconv.ParseInt(numStr, 10, 64)
				if err == nil {
					// HIDIdleTime is in nanoseconds, convert to seconds
					return int(num / 1000000000)
				}
			}
		}
	}
	return 0
}

func getIdleTimeWindows() int {
	// Use PowerShell to get idle time
	script := `
Add-Type @'
using System;
using System.Runtime.InteropServices;

public class IdleTime {
    [DllImport("user32.dll")]
    public static extern bool GetLastInputInfo(ref LASTINPUTINFO plii);

    [StructLayout(LayoutKind.Sequential)]
    public struct LASTINPUTINFO {
        public uint cbSize;
        public uint dwTime;
    }

    public static uint GetIdleTime() {
        LASTINPUTINFO lastInputInfo = new LASTINPUTINFO();
        lastInputInfo.cbSize = (uint)Marshal.SizeOf(lastInputInfo);
        GetLastInputInfo(ref lastInputInfo);
        return ((uint)Environment.TickCount - lastInputInfo.dwTime) / 1000;
    }
}
'@
[IdleTime]::GetIdleTime()
`
	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	idleSeconds, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}

	return idleSeconds
}

// IsIdle returns true if the system has been idle for more than the threshold
func IsIdle() bool {
	idleTime := GetIdleTime()
	return idleTime > idleThresholdSeconds
}

// TrackIdleState checks idle state and logs transitions
func TrackIdleState() {
	isIdle := IsIdle()

	// Only log when state changes
	if isIdle != lastIdleState {
		if isIdle {
			log.Println("System became idle")
			err := InsertIdleEvent("idle_start")
			if err != nil {
				log.Printf("Failed to insert idle event: %v", err)
			}
		} else {
			log.Println("System became active")
			err := InsertIdleEvent("idle_end")
			if err != nil {
				log.Printf("Failed to insert idle event: %v", err)
			}
		}
		lastIdleState = isIdle
	}
}
