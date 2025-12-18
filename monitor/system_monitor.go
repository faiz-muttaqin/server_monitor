package monitor

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"server_monitor/model"
	"server_monitor/utils"

	"github.com/sirupsen/logrus"
)

// SystemMonitor handles system monitoring operations
type SystemMonitor struct {
	ServerID string
}

// NewSystemMonitor creates a new system monitor instance
func NewSystemMonitor() *SystemMonitor {
	// Get primary IP address as server ID
	utils.IP = getLocalIP()
	return &SystemMonitor{
		ServerID: utils.IP,
	}
}

// CollectSystemData collects all system data and returns a Server struct
func (sm *SystemMonitor) CollectSystemData() (*model.Server, error) {
	server := &model.Server{
		ID:            sm.ServerID,
		IP:            sm.ServerID,
		LastCheckTime: time.Now(),
		Status:        "online",
	}

	var err error

	// Collect data based on OS
	switch runtime.GOOS {
	case "windows":
		err = sm.collectWindowsData(server)
	case "linux":
		err = sm.collectLinuxData(server)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		log.Printf("Error collecting system data: %v", err)
		server.Status = "error"
	}

	return server, nil
}

// CollectSystemDataPartial collects system data and returns as map for partial updates
func (sm *SystemMonitor) CollectSystemDataPartial() (map[string]interface{}, error) {
	updates := make(map[string]interface{})

	// Set basic server info
	updates["id"] = sm.ServerID
	updates["ip"] = sm.ServerID

	var err error

	// Collect data based on OS
	switch runtime.GOOS {
	case "windows":
		err = sm.collectWindowsDataPartial(updates)
	case "linux":
		err = sm.collectLinuxDataPartial(updates)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		log.Printf("Error collecting partial system data: %v", err)
		updates["status"] = "error"
	}

	return updates, nil
}

// getLocalIP gets the primary local IP address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// collectWindowsData collects system data for Windows
func (sm *SystemMonitor) collectWindowsData(server *model.Server) error {
	server.OS = "Windows"

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		server.ServerName = hostname
	}

	// Get OS version
	if output, err := execCommand("ver"); err == nil {
		server.OSVersion = strings.TrimSpace(output)
	}

	// Get system info
	if output, err := execCommand("systeminfo"); err == nil {
		sm.parseWindowsSystemInfo(server, output)
	}

	// Get CPU usage using wmic
	if output, err := execCommand("wmic", "cpu", "get", "loadpercentage", "/value"); err == nil {
		sm.parseWindowsCPUUsage(server, output)
	}

	// Get memory info
	if output, err := execCommand("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/value"); err == nil {
		sm.parseWindowsMemoryInfo(server, output)
	}

	// Get disk info
	if output, err := execCommand("wmic", "logicaldisk", "get", "size,freespace", "/value"); err == nil {
		sm.parseWindowsDiskInfo(server, output)
	}

	// Get network info
	sm.collectWindowsNetworkInfo(server)

	// Get process count
	if output, err := execCommand("tasklist", "/fo", "csv"); err == nil {
		server.ProcessCount = len(strings.Split(output, "\n")) - 2 // Subtract header and empty line
	}

	// Get active users
	if output, err := execCommand("query", "user"); err == nil {
		server.ActiveUsers = len(strings.Split(strings.TrimSpace(output), "\n")) - 1
	}

	// Get uptime
	sm.getWindowsUptime(server)

	// Get GPU and Display information
	sm.collectWindowsGPUInfo(server)
	sm.detectWindowsGUI(server)

	// Get temperature information
	sm.getWindowsTemperature(server)

	// Get open ports information
	sm.collectWindowsPortInfo(server)

	return nil
}

// collectLinuxData collects system data for Linux
func (sm *SystemMonitor) collectLinuxData(server *model.Server) error {
	server.OS = "Linux"

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		server.ServerName = hostname
	}

	// Get OS version
	if content, err := os.ReadFile("/etc/os-release"); err == nil {
		sm.parseLinuxOSRelease(server, string(content))
	}

	// Get kernel version
	if output, err := execCommand("uname", "-r"); err == nil {
		server.KernelVersion = strings.TrimSpace(output)
	}

	// Get architecture
	if output, err := execCommand("uname", "-m"); err == nil {
		server.Architecture = strings.TrimSpace(output)
	}

	// Get CPU info
	sm.collectLinuxCPUInfo(server)

	// Get memory info
	sm.collectLinuxMemoryInfo(server)

	// Get disk info
	sm.collectLinuxDiskInfo(server)

	// Get network info
	sm.collectLinuxNetworkInfo(server)

	// Get load average
	sm.getLinuxLoadAverage(server)

	// Get uptime
	sm.getLinuxUptime(server)

	// Get process count
	if output, err := execCommand("ps", "aux"); err == nil {
		server.ProcessCount = len(strings.Split(output, "\n")) - 2
	}

	// Get active users
	if output, err := execCommand("who"); err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 && lines[0] != "" {
			server.ActiveUsers = len(lines)
		}
	}

	// Get SSH connections
	if output, err := execCommand("ss", "-t", "state", "established", "sport", ":22"); err == nil {
		server.SSHConnections = len(strings.Split(strings.TrimSpace(output), "\n")) - 1
		if server.SSHConnections < 0 {
			server.SSHConnections = 0
		}
	}

	// Get GPU and Display information
	sm.collectLinuxGPUInfo(server)
	sm.detectLinuxGUI(server)

	// Get temperature information
	sm.getLinuxTemperature(server)

	// Get open ports information
	sm.collectLinuxPortInfo(server)

	return nil
}

// collectLinuxDataPartial collects key system data for Linux and returns as map
func (sm *SystemMonitor) collectLinuxDataPartial(updates map[string]interface{}) error {
	updates["os"] = "Linux"

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		updates["server_name"] = hostname
	}

	// Get OS version
	if output, err := execCommand("lsb_release", "-d", "-s"); err == nil {
		updates["os_version"] = strings.Trim(strings.TrimSpace(output), "\"")
	}

	// Get kernel version
	if output, err := execCommand("uname", "-r"); err == nil {
		updates["kernel_version"] = strings.TrimSpace(output)
	}

	// Get architecture
	if output, err := execCommand("uname", "-m"); err == nil {
		updates["architecture"] = strings.TrimSpace(output)
	}

	// Get uptime
	if content, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) > 0 {
			if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
				updates["uptime_since"] = time.Now().Add(-time.Duration(uptime) * time.Second)
			}
		}
	}

	// Get CPU model and cores
	if content, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(content), "\n")
		cores := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") && updates["cpu_model"] == nil {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					updates["cpu_model"] = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "processor") {
				cores++
			}
		}
		if cores > 0 {
			updates["cpu_cores"] = cores
		}
	}

	// Get load average
	if content, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) >= 3 {
			if load1, err := strconv.ParseFloat(fields[0], 64); err == nil {
				updates["load_average_1"] = load1
			}
			if load5, err := strconv.ParseFloat(fields[1], 64); err == nil {
				updates["load_average_5"] = load5
			}
			if load15, err := strconv.ParseFloat(fields[2], 64); err == nil {
				updates["load_average_15"] = load15
			}
		}
	}

	// Get CPU usage (simplified)
	if cpuUsage := sm.getLinuxCPUUsageSimple(); cpuUsage > 0 {
		updates["cpu_usage"] = cpuUsage
	}

	// Get memory usage (simplified)
	if memInfo := sm.getLinuxMemorySimple(); len(memInfo) > 0 {
		for key, value := range memInfo {
			updates[key] = value
		}
	}

	// Get disk usage (simplified)
	if diskInfo := sm.getLinuxDiskSimple(); len(diskInfo) > 0 {
		for key, value := range diskInfo {
			updates[key] = value
		}
	}

	// Get network I/O (simplified)
	if netInfo := sm.getLinuxNetworkSimple(); len(netInfo) > 0 {
		for key, value := range netInfo {
			updates[key] = value
		}
	}

	// Get process count
	if output, err := execCommand("ps", "aux"); err == nil {
		updates["process_count"] = len(strings.Split(output, "\n")) - 2
	}

	// Get active users
	if output, err := execCommand("who"); err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 && lines[0] != "" {
			updates["active_users"] = len(lines)
		} else {
			updates["active_users"] = 0
		}
	}

	// Get SSH connections
	if output, err := execCommand("ss", "-t", "state", "established", "sport", ":22"); err == nil {
		sshConnections := len(strings.Split(strings.TrimSpace(output), "\n")) - 1
		if sshConnections < 0 {
			sshConnections = 0
		}
		updates["ssh_connections"] = sshConnections
	}

	// Get listening ports using ss
	if output, err := execCommand("ss", "-tuln"); err == nil {
		sm.parseLinuxPortInfoPartial(updates, output)
	}

	// Get GUI information
	updates["use_gui"] = false
	updates["has_desktop_session"] = false
	updates["desktop_environment"] = ""

	// Check if GUI is running
	if output, err := execCommand("ps", "aux"); err == nil {
		if strings.Contains(output, "Xorg") || strings.Contains(output, "wayland") ||
			strings.Contains(output, "gnome") || strings.Contains(output, "kde") ||
			strings.Contains(output, "xfce") || strings.Contains(output, "lxde") {
			updates["use_gui"] = true
			updates["has_desktop_session"] = true

			// Try to detect desktop environment
			if strings.Contains(output, "gnome") {
				updates["desktop_environment"] = "GNOME"
			} else if strings.Contains(output, "kde") {
				updates["desktop_environment"] = "KDE"
			} else if strings.Contains(output, "xfce") {
				updates["desktop_environment"] = "XFCE"
			} else if strings.Contains(output, "lxde") {
				updates["desktop_environment"] = "LXDE"
			}
		}
	}

	// Set timestamps
	updates["last_check_time"] = time.Now()
	updates["status"] = "online"

	return nil
}

// parseLinuxPortInfoPartial parses port information for partial updates
func (sm *SystemMonitor) parseLinuxPortInfoPartial(updates map[string]interface{}, output string) {
	lines := strings.Split(output, "\n")
	var openPorts []string
	listeningCount := 0

	for _, line := range lines {
		if strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				// Extract port from local address (format: IP:PORT)
				address := fields[4]
				if strings.Contains(address, ":") {
					parts := strings.Split(address, ":")
					if len(parts) >= 2 {
						port := parts[len(parts)-1]
						openPorts = append(openPorts, port)
						listeningCount++
					}
				}
			}
		}
	}

	updates["total_port_opens"] = listeningCount
	updates["listening_services"] = listeningCount

	// Convert ports list to JSON string
	if len(openPorts) > 0 {
		portsJSON := fmt.Sprintf("[\"%s\"]", strings.Join(openPorts, "\",\""))
		updates["open_ports_list"] = portsJSON
	} else {
		updates["open_ports_list"] = ""
	}

	// Get Linux services information
	servicesRunning, servicesFailed := sm.getLinuxServicesInfo()
	updates["services_running"] = servicesRunning
	updates["services_failed"] = servicesFailed
}

// getLinuxCPUUsageSimple gets simple CPU usage percentage
func (sm *SystemMonitor) getLinuxCPUUsageSimple() float64 {
	if content, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 8 && fields[0] == "cpu" {
				var total, idle uint64
				for i := 1; i < len(fields); i++ {
					if val, err := strconv.ParseUint(fields[i], 10, 64); err == nil {
						total += val
						if i == 4 { // idle time
							idle = val
						}
					}
				}
				if total > 0 {
					return float64(total-idle) / float64(total) * 100.0
				}
			}
		}
	}
	return 0
}

// getLinuxMemorySimple gets simple memory information
func (sm *SystemMonitor) getLinuxMemorySimple() map[string]interface{} {
	memInfo := make(map[string]interface{})

	if content, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(content), "\n")
		memData := make(map[string]uint64)

		for _, line := range lines {
			if strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					valueStr := strings.TrimSpace(parts[1])
					valueStr = strings.Replace(valueStr, "kB", "", -1)
					valueStr = strings.TrimSpace(valueStr)

					if value, err := strconv.ParseUint(valueStr, 10, 64); err == nil {
						memData[key] = value * 1024 // Convert to bytes
					}
				}
			}
		}

		if total, ok := memData["MemTotal"]; ok {
			memInfo["memory_total"] = total
		}
		if available, ok := memData["MemAvailable"]; ok {
			memInfo["memory_available"] = available
		}
		if free, ok := memData["MemFree"]; ok {
			memInfo["memory_free"] = free
		}

		// Calculate used memory
		if total, ok := memData["MemTotal"]; ok {
			if available, ok := memData["MemAvailable"]; ok {
				used := total - available
				memInfo["memory_used"] = used
				if total > 0 {
					memInfo["memory_usage"] = float64(used) / float64(total) * 100.0
				}
			}
		}
	}

	return memInfo
}

// getLinuxDiskSimple gets simple disk information
func (sm *SystemMonitor) getLinuxDiskSimple() map[string]interface{} {
	diskInfo := make(map[string]interface{})

	if output, err := execCommand("df", "/", "-B1"); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					diskInfo["disk_total"] = total
				}
				if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
					diskInfo["disk_used"] = used
				}
				if available, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
					diskInfo["disk_free"] = available
				}
			}
		}
	}

	return diskInfo
}

// getLinuxNetworkSimple gets simple network I/O information
func (sm *SystemMonitor) getLinuxNetworkSimple() map[string]interface{} {
	netInfo := make(map[string]interface{})

	if content, err := os.ReadFile("/proc/net/dev"); err == nil {
		lines := strings.Split(string(content), "\n")
		var totalRx, totalTx uint64

		for _, line := range lines {
			if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					fields := strings.Fields(parts[1])
					if len(fields) >= 9 {
						if rx, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
							totalRx += rx
						}
						if tx, err := strconv.ParseUint(fields[8], 10, 64); err == nil {
							totalTx += tx
						}
					}
				}
			}
		}

		netInfo["net_in"] = totalRx
		netInfo["net_out"] = totalTx
	}

	return netInfo
}

// getLinuxServicesInfo gets systemd services statistics
func (sm *SystemMonitor) getLinuxServicesInfo() (int, int) {
	servicesRunning := 0
	servicesFailed := 0

	// Get list of all systemd services
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Failed to get systemd services: %v", err)
		return 0, 0
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle lines that start with ● (failed services marker)
		if strings.HasPrefix(line, "●") {
			line = strings.TrimPrefix(line, "●")
			line = strings.TrimSpace(line)
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// Fields: UNIT LOAD ACTIVE SUB DESCRIPTION
			active := fields[2] // ACTIVE status
			sub := fields[3]    // SUB status

			switch active {
			case "active":
				if sub == "running" {
					servicesRunning++
				}
			case "failed":
				servicesFailed++
			default:
				// Check if sub field indicates failure
				if sub == "failed" {
					servicesFailed++
				}
			}
		}
	}

	// logrus.Debugf("Service stats: running=%d, failed=%d", servicesRunning, servicesFailed)

	return servicesRunning, servicesFailed
}

// collectWindowsDataPartial collects key system data for Windows and returns as map
func (sm *SystemMonitor) collectWindowsDataPartial(updates map[string]interface{}) error {
	updates["os"] = "Windows"

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		updates["server_name"] = hostname
	}

	// Get OS version
	if output, err := execCommand("ver"); err == nil {
		updates["os_version"] = strings.TrimSpace(output)
	}

	// For Windows, we'll use simplified data collection for now
	// More complex Windows monitoring can be added later

	return nil
}

// execCommand executes a command and returns output
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	return string(output), err
}

// Helper functions for Windows parsing
func (sm *SystemMonitor) parseWindowsSystemInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Total Physical Memory:") {
			// Extract memory info
		} else if strings.Contains(line, "Processor(s):") {
			// Extract CPU info
		}
	}
}

func (sm *SystemMonitor) parseWindowsCPUUsage(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "LoadPercentage=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if usage, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					server.CPUUsage = usage
				}
			}
		}
	}
}

func (sm *SystemMonitor) parseWindowsMemoryInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "TotalVisibleMemorySize=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if total, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					server.MemoryTotal = total * 1024 // Convert KB to bytes
				}
			}
		} else if strings.Contains(line, "FreePhysicalMemory=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if free, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					server.MemoryFree = free * 1024 // Convert KB to bytes
				}
			}
		}
	}
	if server.MemoryTotal > 0 {
		server.MemoryUsed = server.MemoryTotal - server.MemoryFree
		server.MemoryUsage = float64(server.MemoryUsed) / float64(server.MemoryTotal) * 100
	}
}

func (sm *SystemMonitor) parseWindowsDiskInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	var totalSize, totalFree uint64
	for _, line := range lines {
		if strings.Contains(line, "Size=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if size, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					totalSize += size
				}
			}
		} else if strings.Contains(line, "FreeSpace=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if free, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					totalFree += free
				}
			}
		}
	}
	server.DiskTotal = totalSize
	server.DiskFree = totalFree
	server.DiskUsed = totalSize - totalFree
}

func (sm *SystemMonitor) collectWindowsNetworkInfo(server *model.Server) {
	// Get network statistics using netstat
	if output, err := execCommand("netstat", "-e"); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) >= 4 {
			// Parse bytes received and sent
			parts := strings.Fields(lines[3])
			if len(parts) >= 2 {
				if bytesReceived, err := strconv.ParseUint(parts[0], 10, 64); err == nil {
					server.NetIn = bytesReceived
				}
				if bytesSent, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					server.NetOut = bytesSent
				}
			}
		}
	}
}

func (sm *SystemMonitor) getWindowsUptime(server *model.Server) {
	if output, err := execCommand("wmic", "os", "get", "lastbootuptime", "/value"); err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "LastBootUpTime=") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					timeStr := strings.TrimSpace(parts[1])
					// Parse Windows time format: 20231015103000.000000+420
					if len(timeStr) >= 14 {
						year, _ := strconv.Atoi(timeStr[0:4])
						month, _ := strconv.Atoi(timeStr[4:6])
						day, _ := strconv.Atoi(timeStr[6:8])
						hour, _ := strconv.Atoi(timeStr[8:10])
						min, _ := strconv.Atoi(timeStr[10:12])
						sec, _ := strconv.Atoi(timeStr[12:14])
						server.UptimeSince = time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
					}
				}
			}
		}
	}
}

// Helper functions for Linux parsing
func (sm *SystemMonitor) parseLinuxOSRelease(server *model.Server, content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			server.OSVersion = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
}

func (sm *SystemMonitor) collectLinuxCPUInfo(server *model.Server) {
	// Get CPU info from /proc/cpuinfo
	if content, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(content), "\n")
		coreCount := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "processor") {
				coreCount++
			} else if strings.HasPrefix(line, "model name") && server.CPUModel == "" {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					server.CPUModel = strings.TrimSpace(parts[1])
				}
			}
		}
		server.CPUCores = coreCount
	}

	// Get CPU usage from /proc/stat
	if content, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 8 && fields[0] == "cpu" {
				var total, idle uint64
				for i := 1; i < len(fields); i++ {
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					total += val
					if i == 4 { // idle time is the 4th field
						idle = val
					}
				}
				if total > 0 {
					server.CPUUsage = float64(total-idle) / float64(total) * 100
				}
			}
		}
	}
}

func (sm *SystemMonitor) collectLinuxMemoryInfo(server *model.Server) {
	if content, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseUint(fields[1], 10, 64)
				switch fields[0] {
				case "MemTotal:":
					server.MemoryTotal = value * 1024 // Convert KB to bytes
				case "MemFree:":
					server.MemoryFree = value * 1024
				case "MemAvailable:":
					server.MemoryAvailable = value * 1024
				case "SwapTotal:":
					server.SwapTotal = value * 1024
				case "SwapFree:":
					server.SwapFree = value * 1024
				}
			}
		}
		server.MemoryUsed = server.MemoryTotal - server.MemoryFree
		server.SwapUsed = server.SwapTotal - server.SwapFree
		if server.MemoryTotal > 0 {
			server.MemoryUsage = float64(server.MemoryUsed) / float64(server.MemoryTotal) * 100
		}
	}
}

func (sm *SystemMonitor) collectLinuxDiskInfo(server *model.Server) {
	if output, err := execCommand("df", "-B1", "/"); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				server.DiskTotal, _ = strconv.ParseUint(fields[1], 10, 64)
				server.DiskUsed, _ = strconv.ParseUint(fields[2], 10, 64)
				server.DiskFree, _ = strconv.ParseUint(fields[3], 10, 64)
			}
		}
	}

	// Get disk I/O stats
	if content, err := os.ReadFile("/proc/diskstats"); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 14 && (strings.Contains(fields[2], "sda") || strings.Contains(fields[2], "nvme")) {
				server.DiskReadOps, _ = strconv.ParseUint(fields[3], 10, 64)
				server.DiskRead, _ = strconv.ParseUint(fields[5], 10, 64)
				server.DiskRead *= 512 // Convert sectors to bytes
				server.DiskWriteOps, _ = strconv.ParseUint(fields[7], 10, 64)
				server.DiskWrite, _ = strconv.ParseUint(fields[9], 10, 64)
				server.DiskWrite *= 512 // Convert sectors to bytes
				break
			}
		}
	}
}

func (sm *SystemMonitor) collectLinuxNetworkInfo(server *model.Server) {
	if content, err := os.ReadFile("/proc/net/dev"); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					fields := strings.Fields(parts[1])
					if len(fields) >= 16 {
						rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
						rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
						rxErrors, _ := strconv.ParseUint(fields[2], 10, 64)
						txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
						txPackets, _ := strconv.ParseUint(fields[9], 10, 64)
						txErrors, _ := strconv.ParseUint(fields[10], 10, 64)

						server.NetIn += rxBytes
						server.NetPacketsIn += rxPackets
						server.NetErrorsIn += rxErrors
						server.NetOut += txBytes
						server.NetPacketsOut += txPackets
						server.NetErrorsOut += txErrors
					}
				}
			}
		}
	}
}

func (sm *SystemMonitor) getLinuxLoadAverage(server *model.Server) {
	if content, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) >= 3 {
			server.LoadAverage1, _ = strconv.ParseFloat(fields[0], 64)
			server.LoadAverage5, _ = strconv.ParseFloat(fields[1], 64)
			server.LoadAverage15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}
}

func (sm *SystemMonitor) getLinuxUptime(server *model.Server) {
	if content, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) >= 1 {
			if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
				server.UptimeSince = time.Now().Add(-time.Duration(uptime) * time.Second)
			}
		}
	}
}

// Windows GPU and Display Information
func (sm *SystemMonitor) collectWindowsGPUInfo(server *model.Server) {
	// Get GPU information using wmic
	if output, err := execCommand("wmic", "path", "win32_VideoController", "get", "name,AdapterRAM,CurrentHorizontalResolution,CurrentVerticalResolution", "/value"); err == nil {
		sm.parseWindowsGPUInfo(server, output)
	}

	// Get display information
	if output, err := execCommand("wmic", "desktopmonitor", "get", "screenheight,screenwidth", "/value"); err == nil {
		sm.parseWindowsDisplayInfo(server, output)
	}

	// Try to get GPU temperature using powershell (if available)
	if output, err := execCommand("powershell", "-Command", "Get-WmiObject -Namespace root/wmi -Class MSAcpi_ThermalZoneTemperature | Select-Object CurrentTemperature"); err == nil {
		sm.parseWindowsGPUTemperature(server, output)
	}
}

func (sm *SystemMonitor) parseWindowsGPUInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	var gpuNames []string
	var totalGPUMemory uint64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Name=") && !strings.Contains(line, "Name=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				gpuName := strings.TrimSpace(parts[1])
				if gpuName != "" && !strings.Contains(gpuName, "Microsoft Basic") {
					gpuNames = append(gpuNames, gpuName)
					server.HasGPU = true
				}
			}
		} else if strings.Contains(line, "AdapterRAM=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if ram, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil && ram > 0 {
					totalGPUMemory += ram
				}
			}
		}
	}

	if len(gpuNames) > 0 {
		server.GPUName = strings.Join(gpuNames, ", ")
		server.GPUMemoryTotal = totalGPUMemory
	}
}

func (sm *SystemMonitor) parseWindowsDisplayInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	var width, height string
	displayCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "ScreenWidth=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				width = strings.TrimSpace(parts[1])
				displayCount++
			}
		} else if strings.Contains(line, "ScreenHeight=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				height = strings.TrimSpace(parts[1])
			}
		}
	}

	server.DisplayCount = displayCount
	if width != "" && height != "" {
		server.DisplayResolution = width + "x" + height
	}
}

func (sm *SystemMonitor) parseWindowsGPUTemperature(server *model.Server, output string) {
	// This is a basic implementation - Windows temperature monitoring is complex
	// and requires specific tools or WMI classes that may not be available on all systems
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "CurrentTemperature") {
			// Parse temperature if available
			// Note: This is simplified and may need adjustment based on actual output format
		}
	}
}

// Windows GUI Detection
func (sm *SystemMonitor) detectWindowsGUI(server *model.Server) {
	// Check if desktop session is running
	if output, err := execCommand("query", "session"); err == nil {
		server.HasDesktopSession = strings.Contains(output, "console")
		server.UseGUI = server.HasDesktopSession
	}

	// Try to detect desktop environment
	if output, err := execCommand("powershell", "-Command", "Get-Process -Name 'explorer' -ErrorAction SilentlyContinue"); err == nil {
		if strings.Contains(output, "explorer") {
			server.DesktopEnvironment = "Windows Explorer"
			server.UseGUI = true
		}
	}

	// Check for running GUI processes
	if output, err := execCommand("tasklist", "/fi", "imagename eq dwm.exe"); err == nil {
		if strings.Contains(output, "dwm.exe") {
			server.UseGUI = true
		}
	}
}

// Windows Temperature Detection
func (sm *SystemMonitor) getWindowsTemperature(server *model.Server) {
	// Try multiple methods to get temperature

	// Method 1: WMI Thermal Zone
	if output, err := execCommand("wmic", "/namespace:\\\\root\\wmi", "path", "MSAcpi_ThermalZoneTemperature", "get", "CurrentTemperature", "/value"); err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "CurrentTemperature=") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						// Convert from Kelvin to Celsius (WMI returns temperature in tenths of Kelvin)
						server.Temperature = (temp / 10.0) - 273.15
						break
					}
				}
			}
		}
	}

	// Method 2: Try PowerShell approach
	if server.Temperature == 0 {
		if output, err := execCommand("powershell", "-Command",
			"Get-WmiObject -Class Win32_PerfRawData_Counters_ThermalZoneInformation | Select-Object Temperature"); err == nil {
			// Parse PowerShell output for temperature
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "Temperature" && line != "----------" {
					if temp, err := strconv.ParseFloat(line, 64); err == nil && temp > 0 {
						server.Temperature = temp - 273.15 // Convert Kelvin to Celsius
						break
					}
				}
			}
		}
	}
}

// Windows Port Information
func (sm *SystemMonitor) collectWindowsPortInfo(server *model.Server) {
	// Get listening ports using netstat
	if output, err := execCommand("netstat", "-an"); err == nil {
		sm.parseWindowsPortInfo(server, output)
	}
}

func (sm *SystemMonitor) parseWindowsPortInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	var openPorts []string
	listeningCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "LISTENING") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// Extract port from address (format: IP:PORT)
				address := fields[1]
				if strings.Contains(address, ":") {
					parts := strings.Split(address, ":")
					if len(parts) >= 2 {
						port := parts[len(parts)-1]
						openPorts = append(openPorts, port)
						listeningCount++
					}
				}
			}
		}
	}

	server.TotalPortOpens = listeningCount
	server.ListeningServices = listeningCount

	// Convert ports list to JSON string
	if len(openPorts) > 0 {
		portsJSON := fmt.Sprintf("[\"%s\"]", strings.Join(openPorts, "\",\""))
		server.OpenPortsList = portsJSON
	}
}

// Linux GPU and Display Information
func (sm *SystemMonitor) collectLinuxGPUInfo(server *model.Server) {
	// Check for NVIDIA GPU using nvidia-smi
	if output, err := execCommand("nvidia-smi", "--query-gpu=name,memory.total,memory.used,temperature.gpu,utilization.gpu", "--format=csv,noheader,nounits"); err == nil {
		sm.parseLinuxNvidiaGPU(server, output)
	}

	// Check for AMD GPU using rocm-smi
	if !server.HasGPU {
		if output, err := execCommand("rocm-smi", "--showtemp", "--showuse", "--showmeminfo"); err == nil {
			sm.parseLinuxAMDGPU(server, output)
		}
	}

	// Check for Intel GPU using intel_gpu_top
	if !server.HasGPU {
		if output, err := execCommand("intel_gpu_top", "-s", "1000", "-l", "-o", "/dev/stdout"); err == nil {
			sm.parseLinuxIntelGPU(server, output)
		}
	}

	// Fallback: Check lspci for GPU information
	if !server.HasGPU {
		if output, err := execCommand("lspci", "-v"); err == nil {
			sm.parseLinuxGenericGPU(server, output)
		}
	}

	// Get display information
	sm.collectLinuxDisplayInfo(server)
}

func (sm *SystemMonitor) parseLinuxNvidiaGPU(server *model.Server, output string) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) >= 5 {
			server.HasGPU = true
			server.GPUName = strings.TrimSpace(fields[0])

			if memTotal, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64); err == nil {
				server.GPUMemoryTotal = memTotal * 1024 * 1024 // Convert MB to bytes
			}

			if memUsed, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64); err == nil {
				server.GPUMemoryUsed = memUsed * 1024 * 1024 // Convert MB to bytes
			}

			if temp, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
				server.GPUTemperature = temp
			}

			if usage, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
				server.GPUUsage = usage
			}
			break // Use first GPU
		}
	}
}

func (sm *SystemMonitor) parseLinuxAMDGPU(server *model.Server, output string) {
	// Parse rocm-smi output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "GPU") && strings.Contains(line, "Temp") {
			server.HasGPU = true
			server.GPUName = "AMD GPU"
			// Parse AMD GPU specific output format
		}
	}
}

func (sm *SystemMonitor) parseLinuxIntelGPU(server *model.Server, output string) {
	// Parse intel_gpu_top output
	if strings.Contains(output, "intel") {
		server.HasGPU = true
		server.GPUName = "Intel GPU"
	}
}

func (sm *SystemMonitor) parseLinuxGenericGPU(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "VGA compatible controller") || strings.Contains(line, "3D controller") {
			server.HasGPU = true
			// Extract GPU name from lspci output
			if strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) >= 3 {
					server.GPUName = strings.TrimSpace(parts[2])
				}
			}
			break
		}
	}
}

func (sm *SystemMonitor) collectLinuxDisplayInfo(server *model.Server) {
	// Method 1: Try xrandr (for X11)
	if output, err := execCommand("xrandr", "--current"); err == nil {
		sm.parseLinuxXrandrOutput(server, output)
	}

	// Method 2: Try wayland-info (for Wayland)
	if server.DisplayCount == 0 {
		if output, err := execCommand("wayland-info"); err == nil {
			if strings.Contains(output, "wl_output") {
				server.DisplayCount = 1 // Basic detection
			}
		}
	}

	// Method 3: Check /sys/class/drm/
	if server.DisplayCount == 0 {
		if output, err := execCommand("ls", "/sys/class/drm/"); err == nil {
			lines := strings.Split(output, "\n")
			displayCount := 0
			for _, line := range lines {
				if strings.Contains(line, "card") && strings.Contains(line, "DP") || strings.Contains(line, "HDMI") || strings.Contains(line, "VGA") {
					displayCount++
				}
			}
			server.DisplayCount = displayCount
		}
	}
}

func (sm *SystemMonitor) parseLinuxXrandrOutput(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	displayCount := 0
	var primaryResolution string

	for _, line := range lines {
		if strings.Contains(line, " connected") {
			displayCount++
			// Extract resolution
			if strings.Contains(line, "primary") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.Contains(part, "x") && strings.Contains(part, "+") {
						primaryResolution = strings.Split(part, "+")[0]
						break
					}
				}
			}
		}
	}

	server.DisplayCount = displayCount
	if primaryResolution != "" {
		server.DisplayResolution = primaryResolution
	}
}

// Linux GUI Detection
func (sm *SystemMonitor) detectLinuxGUI(server *model.Server) {
	// Check for X11 session
	if os.Getenv("DISPLAY") != "" {
		server.UseGUI = true
		server.HasDesktopSession = true
	}

	// Check for Wayland session
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		server.UseGUI = true
		server.HasDesktopSession = true
	}

	// Detect desktop environment
	desktopEnv := os.Getenv("XDG_CURRENT_DESKTOP")
	if desktopEnv == "" {
		desktopEnv = os.Getenv("DESKTOP_SESSION")
	}
	if desktopEnv != "" {
		server.DesktopEnvironment = desktopEnv
		server.UseGUI = true
	}

	// Check for running GUI processes
	guiProcesses := []string{"gnome-session", "kde-session", "xfce4-session", "lxsession", "mate-session"}
	for _, proc := range guiProcesses {
		if output, err := execCommand("pgrep", proc); err == nil && strings.TrimSpace(output) != "" {
			server.UseGUI = true
			server.HasDesktopSession = true
			if server.DesktopEnvironment == "" {
				switch proc {
				case "gnome-session":
					server.DesktopEnvironment = "GNOME"
				case "kde-session":
					server.DesktopEnvironment = "KDE"
				case "xfce4-session":
					server.DesktopEnvironment = "XFCE"
				case "lxsession":
					server.DesktopEnvironment = "LXDE"
				case "mate-session":
					server.DesktopEnvironment = "MATE"
				}
			}
			break
		}
	}

	// Check for display manager
	if !server.UseGUI {
		displayManagers := []string{"gdm", "lightdm", "sddm", "xdm", "kdm"}
		for _, dm := range displayManagers {
			if output, err := execCommand("pgrep", dm); err == nil && strings.TrimSpace(output) != "" {
				server.UseGUI = true
				break
			}
		}
	}
}

// Linux Temperature Detection
func (sm *SystemMonitor) getLinuxTemperature(server *model.Server) {
	// Method 1: Check /sys/class/thermal/
	if content, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		if temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil {
			server.Temperature = temp / 1000.0 // Convert millidegrees to degrees
		}
	}

	// Method 2: Try sensors command
	if server.Temperature == 0 {
		if output, err := execCommand("sensors", "-u"); err == nil {
			sm.parseLinuxSensorsOutput(server, output)
		}
	}

	// Method 3: Check other thermal zones
	if server.Temperature == 0 {
		for i := 1; i <= 10; i++ {
			thermalPath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
			if content, err := os.ReadFile(thermalPath); err == nil {
				if temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil {
					server.Temperature = temp / 1000.0
					break
				}
			}
		}
	}
}

func (sm *SystemMonitor) parseLinuxSensorsOutput(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "temp") && strings.Contains(line, "_input:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if temp, err := strconv.ParseFloat(parts[1], 64); err == nil {
					server.Temperature = temp
					break
				}
			}
		}
	}
}

// Linux Port Information
func (sm *SystemMonitor) collectLinuxPortInfo(server *model.Server) {
	// Get listening ports using ss
	if output, err := execCommand("ss", "-tuln"); err == nil {
		sm.parseLinuxPortInfo(server, output)
	}
}

func (sm *SystemMonitor) parseLinuxPortInfo(server *model.Server, output string) {
	lines := strings.Split(output, "\n")
	var openPorts []string
	listeningCount := 0

	for _, line := range lines {
		if strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				// Extract port from local address (format: IP:PORT)
				address := fields[4]
				if strings.Contains(address, ":") {
					parts := strings.Split(address, ":")
					if len(parts) >= 2 {
						port := parts[len(parts)-1]
						openPorts = append(openPorts, port)
						listeningCount++
					}
				}
			}
		}
	}

	server.TotalPortOpens = listeningCount
	server.ListeningServices = listeningCount

	// Convert ports list to JSON string
	if len(openPorts) > 0 {
		portsJSON := fmt.Sprintf("[\"%s\"]", strings.Join(openPorts, "\",\""))
		server.OpenPortsList = portsJSON
	}
}
