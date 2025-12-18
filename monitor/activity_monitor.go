package monitor

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"server_monitor/model"
	"server_monitor/utils"
	"server_monitor/ws"
)

// ActivityMonitor handles real-time activity monitoring
type ActivityMonitor struct {
	serverID           string
	mouseCheckInterval time.Duration
	cliCheckInterval   time.Duration
	lastMouseTime      time.Time
	lastCLITime        time.Time
	lastMousePos       string
	lastCLISession     string // Store last CLI session info to detect changes
}

// NewActivityMonitor creates a new activity monitor
func NewActivityMonitor(serverID string, mouseInterval, cliInterval time.Duration) *ActivityMonitor {
	return &ActivityMonitor{
		serverID:           serverID,
		mouseCheckInterval: mouseInterval,
		cliCheckInterval:   cliInterval,
		lastMouseTime:      time.Now(),
		lastCLITime:        time.Now(),
		lastMousePos:       "",
		lastCLISession:     "",
	}
}

// StartActivityMonitoring starts monitoring both mouse and CLI activity
func (am *ActivityMonitor) StartActivityMonitoring(ctx context.Context) {
	log.Println("Starting activity monitoring...")

	// Start mouse activity monitoring in a separate goroutine
	go am.monitorMouseActivity(ctx)

	// Start CLI activity monitoring in a separate goroutine
	go am.monitorCLIActivity(ctx)

	log.Println("Activity monitoring initialized")
}

// monitorMouseActivity monitors mouse movement events
func (am *ActivityMonitor) monitorMouseActivity(ctx context.Context) {
	ticker := time.NewTicker(am.mouseCheckInterval)
	defer ticker.Stop()

	log.Println("Mouse activity monitoring started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Mouse activity monitoring stopped")
			return
		case <-ticker.C:
			if am.detectMouseActivity() {
				now := time.Now()
				if now.Sub(am.lastMouseTime) > 1*time.Second { // Avoid too frequent updates
					am.lastMouseTime = now
					am.updateLastMouseMovement(now)
				}
			}
		}
	}
}

// monitorCLIActivity monitors command line activity
func (am *ActivityMonitor) monitorCLIActivity(ctx context.Context) {
	ticker := time.NewTicker(am.cliCheckInterval)
	defer ticker.Stop()

	log.Println("CLI activity monitoring started")

	for {
		select {
		case <-ctx.Done():
			log.Println("CLI activity monitoring stopped")
			return
		case <-ticker.C:
			cliSession, hasActivity := am.detectCLIActivity()
			if hasActivity && cliSession != "" {
				// Only update if the CLI session info has changed
				if cliSession != am.lastCLISession {
					cliTime := time.Now()

					// Only try to parse SSH session timestamps (not local session indicators)
					if cliSession != "local_session_active" {
						// Try to parse the session time if it's in a recognizable format
						parsedTime, err := time.Parse("2006-01-02 15:04:05", cliSession)
						if err == nil {
							// If we successfully parsed the time, use it
							cliTime = parsedTime
						} else {
							// Try SSH who format: "yyyy-mm-dd hh:mm"
							parsedTime2, err2 := time.Parse("2006-01-02 15:04", cliSession)
							if err2 == nil {
								cliTime = parsedTime2
							}
							// If all parsing failed, use current time (cliTime = time.Now() already set above)
						}
					}

					am.lastCLITime = cliTime
					am.lastCLISession = cliSession
					am.updateLastCLIActivity(cliTime)
				}
			}
		}
	}
}

// detectMouseActivity detects mouse movement (platform-specific)
func (am *ActivityMonitor) detectMouseActivity() bool {
	switch runtime.GOOS {
	case "windows":
		return am.detectWindowsMouseActivity()
	case "linux":
		return am.detectLinuxMouseActivity()
	default:
		return false
	}
}

// detectCLIActivity detects command line activity (platform-specific)
func (am *ActivityMonitor) detectCLIActivity() (string, bool) {
	switch runtime.GOOS {
	case "windows":
		return am.detectWindowsCLIActivity()
	case "linux":
		return am.detectLinuxCLIActivity()
	default:
		return "", false
	}
}

// detectWindowsMouseActivity detects mouse activity on Windows
func (am *ActivityMonitor) detectWindowsMouseActivity() bool {
	// Use PowerShell to get mouse position with proper assembly loading
	cmd := exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Cursor]::Position")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Get current mouse position
	outputStr := strings.TrimSpace(string(output))
	if len(outputStr) == 0 || !strings.Contains(outputStr, "X") || !strings.Contains(outputStr, "Y") {
		return false
	}

	// Compare with last known position
	if am.lastMousePos != outputStr {
		am.lastMousePos = outputStr
		return true
	}

	return false
}

// detectLinuxMouseActivity detects mouse activity on Linux
func (am *ActivityMonitor) detectLinuxMouseActivity() bool {
	// Check if mouse device exists and has recent activity
	mouseDevices := []string{"/dev/input/mouse0", "/dev/input/mouse1", "/dev/input/mice"}

	for _, device := range mouseDevices {
		if stat, err := os.Stat(device); err == nil {
			// Check if device was accessed recently (within last 2 seconds)
			if time.Since(stat.ModTime()) < 2*time.Second {
				return true
			}
		}
	}
	return false
}

// detectWindowsCLIActivity detects CLI activity on Windows
func (am *ActivityMonitor) detectWindowsCLIActivity() (string, bool) {
	// Check for active console processes
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq cmd.exe", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	lines := strings.Split(string(output), "\n")
	// If more than header line, there are active CMD processes
	activeProcesses := len(lines) - 1

	// Also check PowerShell
	cmd2 := exec.Command("tasklist", "/FI", "IMAGENAME eq powershell.exe", "/FO", "CSV")
	output2, err2 := cmd2.Output()
	if err2 == nil {
		lines2 := strings.Split(string(output2), "\n")
		activeProcesses += len(lines2) - 1
	}

	if activeProcesses > 0 {
		return "Windows CLI Active", true
	}
	return "", false
}

// detectLinuxCLIActivity detects CLI activity on Linux
func (am *ActivityMonitor) detectLinuxCLIActivity() (string, bool) {
	// Check for SSH sessions (oldest one)
	cmd := exec.Command("sh", "-c", "who | awk '{print $3, $4}' | sort | head -n 1")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	sessionInfo := strings.TrimSpace(string(output))
	if sessionInfo != "" {
		return sessionInfo, true
	}

	// If no SSH sessions, just return current time indicator for local session
	cmd2 := exec.Command("sh", "-c", "ps aux | grep -E '(bash|zsh|sh|fish)' | grep -v grep | head -n 1")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		return "", false
	}

	localSession := strings.TrimSpace(string(output2))
	if localSession != "" {
		// Return current time as string to indicate local activity
		return "local_session_active", true
	}

	// log.Printf("No CLI activity detected")
	return "", false
}

// updateLastMouseMovement updates the cache with new mouse movement time
func (am *ActivityMonitor) updateLastMouseMovement(timestamp time.Time) {
	// Validate timestamp is not empty/zero
	if timestamp.IsZero() {
		log.Printf("Skipping mouse movement update: timestamp is zero/empty")
		return
	}

	// Get current server data from cache to check uptime_since
	currentServer, exists := model.GetServerCache(am.serverID)
	if !exists {
		log.Printf("Skipping mouse movement update: server %s not found in cache", am.serverID)
		return
	}

	// Validate timestamp is not older than uptime_since
	if timestamp.Before(currentServer.UptimeSince) {
		log.Printf("Skipping mouse movement update: timestamp %s is older than uptime_since %s",
			timestamp.Format("2006-01-02 15:04:05"),
			currentServer.UptimeSince.Format("2006-01-02 15:04:05"))
		return
	}

	updates := map[string]interface{}{
		"last_mouse_movement": timestamp,
	}

	if err := model.UpdateServerCache(am.serverID, updates); err != nil {
		log.Printf("Failed to update mouse movement time: %v", err)
	} else {
		convertedID := strings.ReplaceAll(am.serverID, ".", "_")
		ws.BroadcastMessage(1, "server:last_mouse_movement-"+convertedID+"::"+timestamp.Format(utils.T_YYYYMMDD_HHmmss))
	}
}

// updateLastCLIActivity updates the cache with new CLI activity time
func (am *ActivityMonitor) updateLastCLIActivity(timestamp time.Time) {
	// Validate timestamp is not empty/zero
	if timestamp.IsZero() {
		log.Printf("Skipping CLI activity update: timestamp is zero/empty")
		return
	}

	// Get current server data from cache to check uptime_since
	currentServer, exists := model.GetServerCache(am.serverID)
	if !exists {
		log.Printf("Skipping CLI activity update: server %s not found in cache", am.serverID)
		return
	}

	// Validate timestamp is not older than uptime_since
	if timestamp.Before(currentServer.UptimeSince) {
		log.Printf("Skipping CLI activity update: timestamp %s is older than uptime_since %s",
			timestamp.Format("2006-01-02 15:04:05"),
			currentServer.UptimeSince.Format("2006-01-02 15:04:05"))
		return
	}

	// Validate timestamp is not too far in the future (more than 5 minutes ahead)
	// or too far in the past (more than the current session is reasonable)
	now := time.Now()
	if timestamp.After(now.Add(5 * time.Minute)) {
		log.Printf("Skipping CLI activity update: timestamp %s is too far in the future",
			timestamp.Format("2006-01-02 15:04:05"))
		return
	}

	updates := map[string]interface{}{
		"last_cli_activity": timestamp,
	}

	if err := model.UpdateServerCache(am.serverID, updates); err != nil {
		log.Printf("Failed to update CLI activity time: %v", err)
	} else {
		log.Printf("CLI activity updated for server %s at %s", am.serverID, timestamp.Format("2006-01-02 15:04:05"))
		convertedID := strings.ReplaceAll(am.serverID, ".", "_")
		ws.BroadcastMessage(1, "server:last_cli_activity-"+convertedID+"::"+timestamp.Format(utils.T_YYYYMMDD_HHmmss))
	}
}
