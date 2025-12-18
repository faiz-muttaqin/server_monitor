package serviceScanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"server_monitor/model"
	"server_monitor/wsclient"
)

// getHostnameIP gets the primary IP address from hostname -I command
func getHostnameIP() string {
	cmd := exec.Command("hostname", "-I")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("❌ Failed to get hostname IP: %v", err)
		return "unknown"
	}

	// hostname -I returns space-separated IP addresses, take the first one
	ips := strings.Fields(strings.TrimSpace(string(out)))
	if len(ips) > 0 {
		return ips[0]
	}

	return "unknown"
}

// UpdateListServices scans listening ports and processes using ss command
func UpdateListServices() {
	log.Println("🔎 Scanning listening sockets and mapping to systemd...")

	// Get hostname IP addresses
	hostIP := getHostnameIP()

	// Run ss command to get listening sockets with sudo for full PID access
	cmd := exec.Command("sudo", "ss", "-tulnpH") // -t=TCP, -u=UDP, -l=listening, -n=numeric, -p=processes, -H=no header
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Printf("❌ Failed to run ss command: %v", err)
		return
	}

	lines := strings.Split(out.String(), "\n")
	var newServices []*model.ServerService

	log.Printf("🔍 Total lines from ss command: %d", len(lines))

	pidRegex := regexp.MustCompile(`pid=(\d+)`)
	processRegex := regexp.MustCompile(`\(\("([^"]+)"`)

	processedCount := 0
	skippedNoPid := 0
	skippedEmpty := 0
	skippedFields := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			skippedEmpty++
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			skippedFields++
			log.Printf("⚠️ Skipped line (insufficient fields): %s", line)
			continue
		}

		proto := fields[0]     // tcp/udp
		localAddr := fields[4] // address:port

		// Extract PID from users:(("process",pid=1234,fd=5))
		var pid int
		var processName string

		if pidMatch := pidRegex.FindStringSubmatch(line); len(pidMatch) > 1 {
			if p, err := strconv.Atoi(pidMatch[1]); err == nil {
				pid = p
			}
		}

		if procMatch := processRegex.FindStringSubmatch(line); len(procMatch) > 1 {
			processName = procMatch[1]
		}

		// Skip if no PID found
		if pid == 0 {
			skippedNoPid++
			log.Printf("⚠️ Skipped line (no PID): %s", line)
			continue
		}

		processedCount++

		// Get additional process information
		isService := checkSystemdUnit(pid)
		unitName := getSystemdUnit(pid)
		fullCommand := getProcessCommand(pid)
		startTime := getProcessStartTime(pid)

		// Generate unique ID with hostname IP
		id := fmt.Sprintf("%s-%d", hostIP, pid)

		service := &model.ServerService{
			ID:            id,
			PID:           pid,
			IsService:     isService,
			Process:       fullCommand,
			Name:          getServiceName(processName, unitName),
			IPAddressPort: fmt.Sprintf("%s:%s", proto, localAddr),
			Status:        "LISTEN",
			StartedAt:     startTime,
			Description:   getServiceDescription(processName, unitName, isService),
		}

		newServices = append(newServices, service)
	}

	model.MuServerServices.Lock()
	model.ServerServices[hostIP] = newServices
	model.MuServerServices.Unlock()
	if result, err := json.Marshal(map[string]any{hostIP: newServices}); err == nil {
		// go ws.BroadcastMessageNode(1, "server_services:"+string(result))
		if wsclient.WsMasterConn != nil {
			go wsclient.SendMessage(wsclient.WsMasterConn, "server_services:"+string(result))
		}
	} else {
		log.Printf("❌ Failed to marshal server services for websocket broadcast: %v", err)
	}
	if result, err := json.Marshal(model.ServerServices); err == nil {
		go func(b []byte) {
			cacheDir := "./.cache"
			filePath := filepath.Join(cacheDir, "server_services.json")

			// Cek folder dan buat kalau belum ada
			if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					log.Fatalf("❌ Gagal membuat folder cache: %v", err)
				}
			}

			// Simpan file
			if err := os.WriteFile(filePath, b, 0777); err != nil {
				log.Fatalf("❌ Gagal menyimpan file: %v", err)
			}

			fmt.Printf("💾 Berhasil menyimpan data ke %s\n", filePath)
		}(result)
	}
	log.Printf("✅ Updated %d listening services", len(newServices))
	log.Printf("📊 Stats - Processed: %d, Skipped (no PID): %d, Skipped (empty): %d, Skipped (fields): %d",
		processedCount, skippedNoPid, skippedEmpty, skippedFields)
}

// checkSystemdUnit checks if process is managed by systemd
func checkSystemdUnit(pid int) bool {
	if pid == 0 {
		return false
	}

	// Check /proc/<pid>/cgroup for systemd unit
	cmd := exec.Command("cat", fmt.Sprintf("/proc/%d/cgroup", pid))
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	cgroupContent := string(out)
	return strings.Contains(cgroupContent, "system.slice") || strings.Contains(cgroupContent, ".service")
}

// getSystemdUnit extracts systemd unit name from cgroup
func getSystemdUnit(pid int) string {
	if pid == 0 {
		return "-"
	}

	cmd := exec.Command("cat", fmt.Sprintf("/proc/%d/cgroup", pid))
	out, err := cmd.Output()
	if err != nil {
		return "-"
	}

	cgroupContent := string(out)

	// Look for system.slice/<unit> pattern
	if strings.Contains(cgroupContent, "system.slice") {
		serviceRegex := regexp.MustCompile(`system\.slice/([^/\s]+\.service)`)
		if match := serviceRegex.FindStringSubmatch(cgroupContent); len(match) > 1 {
			return match[1]
		}
	}

	// Fallback: look for any .service
	serviceRegex := regexp.MustCompile(`([^/\s]+\.service)`)
	if match := serviceRegex.FindStringSubmatch(cgroupContent); len(match) > 1 {
		return match[1]
	}

	return "-"
}

// getProcessCommand gets the full command line of the process
func getProcessCommand(pid int) string {
	if pid == 0 {
		return "-"
	}

	// First try to get comm (process name)
	commCmd := exec.Command("cat", fmt.Sprintf("/proc/%d/comm", pid))
	if commOut, err := commCmd.Output(); err == nil {
		comm := strings.TrimSpace(string(commOut))

		// Then try to get full cmdline
		cmdlineCmd := exec.Command("cat", fmt.Sprintf("/proc/%d/cmdline", pid))
		if cmdlineOut, err := cmdlineCmd.Output(); err == nil {
			cmdline := strings.ReplaceAll(string(cmdlineOut), "\x00", " ")
			cmdline = strings.TrimSpace(cmdline)
			if cmdline != "" {
				// Truncate if too long, keep first 100 characters
				if len(cmdline) > 100 {
					return cmdline[:100] + "..."
				}
				return cmdline
			}
		}
		return comm
	}

	return "-"
}

// getProcessStartTime gets when the process started
func getProcessStartTime(pid int) time.Time {
	if pid == 0 {
		return time.Time{}
	}

	// Try to get start time from ps
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	timeStr := strings.TrimSpace(string(out))
	if timeStr == "" {
		return time.Time{}
	}

	// Parse time format: "Mon Jan _2 15:04:05 2006"
	layouts := []string{
		"Mon Jan _2 15:04:05 2006",
		"Mon Jan 2 15:04:05 2006",
		"Jan _2 15:04:05",
		"Jan 2 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			// If year is missing, assume current year
			if t.Year() == 0 {
				now := time.Now()
				t = t.AddDate(now.Year(), 0, 0)
			}
			return t
		}
	}

	return time.Time{}
}

// getServiceName determines the best name for the service
func getServiceName(processName, unitName string) string {
	if unitName != "-" && unitName != "" {
		// Remove .service suffix for cleaner display
		if strings.HasSuffix(unitName, ".service") {
			return strings.TrimSuffix(unitName, ".service")
		}
		return unitName
	}

	if processName != "" {
		return processName
	}

	return "unknown"
}

// getServiceDescription generates a description for the service
func getServiceDescription(processName, unitName string, isService bool) string {
	if isService && unitName != "-" {
		return fmt.Sprintf("Systemd service: %s", unitName)
	}

	if processName != "" {
		return fmt.Sprintf("Process: %s", processName)
	}

	return "Listening process"
}
