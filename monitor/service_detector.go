package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"server_monitor/model"

	"github.com/sirupsen/logrus"
)

// ServiceDetector handles cross-platform service detection
type ServiceDetector struct {
	serverID string
}

// NewServiceDetector creates a new service detector
func NewServiceDetector(serverID string) *ServiceDetector {
	return &ServiceDetector{
		serverID: serverID,
	}
}

// DetectServices detects running services based on the operating system
func (sd *ServiceDetector) DetectServices() ([]*model.ServerService, error) {
	switch runtime.GOOS {
	case "windows":
		return sd.detectWindowsServices()
	case "linux":
		return sd.detectLinuxServices()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// detectWindowsServices detects services running on Windows
func (sd *ServiceDetector) detectWindowsServices() ([]*model.ServerService, error) {
	logrus.Info("Starting Windows service detection...")

	// Step 1: Get network connections with netstat
	netConnections, err := sd.getWindowsNetConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get network connections: %v", err)
	}

	// Step 2: Get process information
	processInfo, err := sd.getWindowsProcessInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get process info: %v", err)
	}

	// Step 3: Get Windows services information
	windowsServices, err := sd.getWindowsServicesInfo()
	if err != nil {
		logrus.Warnf("Failed to get Windows services info: %v", err)
		windowsServices = make(map[int]WindowsServiceInfo)
	}

	// Step 4: Combine data and create ServerService objects
	services := make([]*model.ServerService, 0)

	for _, conn := range netConnections {
		if conn.PID <= 0 {
			continue
		}

		// Get process info
		proc, hasProcessInfo := processInfo[conn.PID]

		// Get service info
		svcInfo, isService := windowsServices[conn.PID]

		// Create service object
		service := &model.ServerService{
			ID:            fmt.Sprintf("%s-%d", conn.LocalAddress, conn.PID),
			PID:           conn.PID,
			IsService:     isService,
			IPAddressPort: conn.LocalAddress,
			Status:        conn.State,
			StartedAt:     time.Now(), // Will be updated if we can get actual start time
		}

		// Fill process information
		if hasProcessInfo {
			service.Process = proc.ExecutablePath
			service.Name = proc.Name
		} else {
			service.Process = "unknown"
			service.Name = fmt.Sprintf("PID_%d", conn.PID)
		}

		// Fill service information if it's a Windows service
		if isService {
			service.Name = svcInfo.ServiceName
			service.Description = svcInfo.Description
			service.Status = svcInfo.Status
			if !svcInfo.StartTime.IsZero() {
				service.StartedAt = svcInfo.StartTime
			}
		}

		services = append(services, service)
	}

	logrus.Infof("Detected %d Windows services/processes with network connections", len(services))
	return services, nil
}

// WindowsNetConnection represents a network connection on Windows
type WindowsNetConnection struct {
	Protocol      string
	LocalAddress  string
	RemoteAddress string
	State         string
	PID           int
}

// WindowsProcessInfo represents process information on Windows
type WindowsProcessInfo struct {
	PID            int
	Name           string
	ExecutablePath string
	StartTime      time.Time
}

// WindowsServiceInfo represents Windows service information
type WindowsServiceInfo struct {
	ServiceName string
	DisplayName string
	Status      string
	StartType   string
	Description string
	StartTime   time.Time
}

// getWindowsNetConnections gets network connections using netstat
func (sd *ServiceDetector) getWindowsNetConnections() ([]WindowsNetConnection, error) {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat command failed: %v", err)
	}

	var connections []WindowsNetConnection
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "Proto") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		protocol := fields[0]
		localAddr := fields[1]
		remoteAddr := fields[2]
		state := ""
		pidStr := ""

		// Handle different netstat output formats
		if protocol == "TCP" {
			if len(fields) >= 5 {
				state = fields[3]
				pidStr = fields[4]
			}
		} else if protocol == "UDP" {
			if len(fields) >= 4 {
				state = "LISTENING"
				pidStr = fields[3]
			}
		}

		if pidStr == "" {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		connections = append(connections, WindowsNetConnection{
			Protocol:      protocol,
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,
			State:         state,
			PID:           pid,
		})
	}

	logrus.Infof("Found %d network connections", len(connections))
	return connections, nil
}

// getWindowsProcessInfo gets process information using tasklist
func (sd *ServiceDetector) getWindowsProcessInfo() (map[int]WindowsProcessInfo, error) {
	cmd := exec.Command("powershell", "-Command",
		`Get-WmiObject -Class Win32_Process | Select-Object ProcessId,Name,ExecutablePath,CreationDate | ConvertTo-Json`)

	output, err := cmd.Output()
	if err != nil {
		// Fallback to tasklist if PowerShell fails
		return sd.getWindowsProcessInfoTasklist()
	}

	var processes []struct {
		ProcessId      int    `json:"ProcessId"`
		Name           string `json:"Name"`
		ExecutablePath string `json:"ExecutablePath"`
		CreationDate   string `json:"CreationDate"`
	}

	if err := json.Unmarshal(output, &processes); err != nil {
		return sd.getWindowsProcessInfoTasklist()
	}

	processMap := make(map[int]WindowsProcessInfo)
	for _, proc := range processes {
		var startTime time.Time
		if proc.CreationDate != "" {
			// Parse WMI date format
			if parsed, err := time.Parse("20060102150405.000000-0700", proc.CreationDate); err == nil {
				startTime = parsed
			}
		}

		processMap[proc.ProcessId] = WindowsProcessInfo{
			PID:            proc.ProcessId,
			Name:           proc.Name,
			ExecutablePath: proc.ExecutablePath,
			StartTime:      startTime,
		}
	}

	logrus.Infof("Retrieved process information for %d processes", len(processMap))
	return processMap, nil
}

// getWindowsProcessInfoTasklist fallback method using tasklist
func (sd *ServiceDetector) getWindowsProcessInfoTasklist() (map[int]WindowsProcessInfo, error) {
	cmd := exec.Command("tasklist", "/fo", "csv", "/v")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tasklist command failed: %v", err)
	}

	processMap := make(map[int]WindowsProcessInfo)
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		// Parse CSV-like output
		fields := strings.Split(line, "\",\"")
		if len(fields) < 3 {
			continue
		}

		// Clean quotes from fields
		for j := range fields {
			fields[j] = strings.Trim(fields[j], "\"")
		}

		pidStr := fields[1]
		name := fields[0]

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		processMap[pid] = WindowsProcessInfo{
			PID:  pid,
			Name: name,
		}
	}

	logrus.Infof("Retrieved basic process information for %d processes", len(processMap))
	return processMap, nil
}

// getWindowsServicesInfo gets Windows service information
func (sd *ServiceDetector) getWindowsServicesInfo() (map[int]WindowsServiceInfo, error) {
	cmd := exec.Command("powershell", "-Command", `
		Get-WmiObject -Class Win32_Service | ForEach-Object {
			$process = Get-Process -Id $_.ProcessId -ErrorAction SilentlyContinue
			[PSCustomObject]@{
				ServiceName = $_.Name
				DisplayName = $_.DisplayName
				Status = $_.State
				StartMode = $_.StartMode
				ProcessId = $_.ProcessId
				Description = $_.Description
				StartTime = if ($process) { $process.StartTime } else { $null }
			}
		} | ConvertTo-Json`)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("PowerShell service query failed: %v", err)
	}

	var services []struct {
		ServiceName string    `json:"ServiceName"`
		DisplayName string    `json:"DisplayName"`
		Status      string    `json:"Status"`
		StartMode   string    `json:"StartMode"`
		ProcessId   int       `json:"ProcessId"`
		Description string    `json:"Description"`
		StartTime   time.Time `json:"StartTime"`
	}

	if err := json.Unmarshal(output, &services); err != nil {
		return nil, fmt.Errorf("failed to parse service JSON: %v", err)
	}

	serviceMap := make(map[int]WindowsServiceInfo)
	for _, svc := range services {
		if svc.ProcessId > 0 {
			serviceMap[svc.ProcessId] = WindowsServiceInfo{
				ServiceName: svc.ServiceName,
				DisplayName: svc.DisplayName,
				Status:      svc.Status,
				StartType:   svc.StartMode,
				Description: svc.Description,
				StartTime:   svc.StartTime,
			}
		}
	}

	logrus.Infof("Retrieved information for %d Windows services", len(serviceMap))
	return serviceMap, nil
}

// detectLinuxServices detects services running on Linux (your existing logic)
func (sd *ServiceDetector) detectLinuxServices() ([]*model.ServerService, error) {
	logrus.Info("Starting Linux service detection...")

	// Use your existing Linux detection logic
	services, err := sd.detectLinuxNetworkServices()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Detected %d Linux services", len(services))
	return services, nil
}

// detectLinuxNetworkServices implements Linux service detection using ss command
func (sd *ServiceDetector) detectLinuxNetworkServices() ([]*model.ServerService, error) {
	cmd := exec.Command("ss", "-tulnp")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ss command failed: %v", err)
	}

	var services []*model.ServerService
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Netid") {
			continue
		}

		service := sd.parseLinuxSSLine(line)
		if service != nil {
			services = append(services, service)
		}
	}

	return services, nil
}

// parseLinuxSSLine parses a single line from ss command output
func (sd *ServiceDetector) parseLinuxSSLine(line string) *model.ServerService {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return nil
	}

	localAddr := fields[4]
	processInfo := ""

	if len(fields) > 6 {
		processInfo = fields[6]
	}

	// Parse process info (e.g., "users:(("nginx",pid=1234,fd=6))")
	pid := sd.extractPIDFromLinuxProcessInfo(processInfo)
	processName := sd.extractProcessNameFromLinuxProcessInfo(processInfo)

	if pid <= 0 {
		return nil
	}

	// Get additional process information
	isService, serviceName, description := sd.getLinuxServiceInfo(pid, processName)

	return &model.ServerService{
		ID:            fmt.Sprintf("%s-%d", localAddr, pid),
		PID:           pid,
		IsService:     isService,
		Process:       processName,
		Name:          serviceName,
		IPAddressPort: localAddr,
		Status:        "running",
		StartedAt:     sd.getLinuxProcessStartTime(pid),
		Description:   description,
	}
}

// extractPIDFromLinuxProcessInfo extracts PID from Linux ss process info
func (sd *ServiceDetector) extractPIDFromLinuxProcessInfo(processInfo string) int {
	re := regexp.MustCompile(`pid=(\d+)`)
	matches := re.FindStringSubmatch(processInfo)
	if len(matches) > 1 {
		if pid, err := strconv.Atoi(matches[1]); err == nil {
			return pid
		}
	}
	return 0
}

// extractProcessNameFromLinuxProcessInfo extracts process name from Linux ss process info
func (sd *ServiceDetector) extractProcessNameFromLinuxProcessInfo(processInfo string) string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindStringSubmatch(processInfo)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// getLinuxServiceInfo gets additional service information for Linux
func (sd *ServiceDetector) getLinuxServiceInfo(pid int, processName string) (bool, string, string) {
	// Check if it's a systemd service
	if serviceName, description := sd.checkLinuxSystemdService(pid); serviceName != "" {
		return true, serviceName, description
	}

	// Default to process name
	return false, processName, ""
}

// checkLinuxSystemdService checks if a PID belongs to a systemd service
func (sd *ServiceDetector) checkLinuxSystemdService(pid int) (string, string) {
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)
	content, err := os.ReadFile(cgroupPath)
	if err != nil {
		return "", ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, ".service") {
			parts := strings.Split(line, "/")
			for _, part := range parts {
				if strings.HasSuffix(part, ".service") {
					serviceName := strings.TrimSuffix(part, ".service")
					description := sd.getLinuxServiceDescription(serviceName)
					return serviceName, description
				}
			}
		}
	}

	return "", ""
}

// getLinuxServiceDescription gets service description from systemd
func (sd *ServiceDetector) getLinuxServiceDescription(serviceName string) string {
	cmd := exec.Command("systemctl", "show", serviceName, "--property=Description", "--value")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

// getLinuxProcessStartTime gets process start time on Linux
func (sd *ServiceDetector) getLinuxProcessStartTime(pid int) time.Time {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	content, err := os.ReadFile(statPath)
	if err != nil {
		return time.Now()
	}

	fields := strings.Fields(string(content))
	if len(fields) > 21 {
		if starttime, err := strconv.ParseUint(fields[21], 10, 64); err == nil {
			// Convert from clock ticks to time
			return time.Now().Add(-time.Duration(starttime) * time.Millisecond / 100)
		}
	}

	return time.Now()
}

// SaveServicesToCache saves detected services to cache file
func (sd *ServiceDetector) SaveServicesToCache(services []*model.ServerService) error {
	model.MuServerServices.Lock()
	defer model.MuServerServices.Unlock()

	// Update global ServerServices map
	model.ServerServices[sd.serverID] = services

	// Save to file
	cacheDir := ".cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	filePath := filepath.Join(cacheDir, "server_services.json")
	data, err := json.MarshalIndent(model.ServerServices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal services: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	logrus.Infof("Saved %d services to cache for server %s", len(services), sd.serverID)
	return nil
}

// UpdateServices detects and caches current services
func (sd *ServiceDetector) UpdateServices() error {
	services, err := sd.DetectServices()
	if err != nil {
		return fmt.Errorf("failed to detect services: %v", err)
	}

	if err := sd.SaveServicesToCache(services); err != nil {
		return fmt.Errorf("failed to save services to cache: %v", err)
	}

	logrus.Infof("Successfully updated %d services for %s (%s)",
		len(services), sd.serverID, runtime.GOOS)
	return nil
}
