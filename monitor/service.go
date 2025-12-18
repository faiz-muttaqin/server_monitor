package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server_monitor/model"
	"server_monitor/utils"
	"server_monitor/ws"
	"server_monitor/wsclient"

	"github.com/sirupsen/logrus"
)

// MonitorService handles periodic monitoring
type MonitorService struct {
	monitor         *SystemMonitor
	activityMonitor *ActivityMonitor
	serviceDetector *ServiceDetector
	interval        time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
}

// LoadServerServices loads data from ./.cache/server_services.json into ServerServices
func LoadServerServices() {
	filePath := filepath.Join(".cache", "server_services.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("⚠️ File %s not found, skipping load.\n", filePath)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("❌ Error reading file: %v\n", err)
		return
	}

	var loaded map[string][]*model.ServerService
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("❌ Error unmarshalling JSON: %v\n", err)
		return
	}

	model.MuServerServices.Lock()
	model.ServerServices = loaded
	model.MuServerServices.Unlock()

	logrus.Printf("✅ Loaded %d groups of server services from cache.\n", len(model.ServerServices))
}

// NewMonitorService creates a new monitor service
func NewMonitorService(interval time.Duration) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	// Create system monitor first to get server ID
	systemMonitor := NewSystemMonitor()

	return &MonitorService{
		monitor:         systemMonitor,
		activityMonitor: NewActivityMonitor(systemMonitor.ServerID, 1*time.Second, 5*time.Second),
		serviceDetector: NewServiceDetector(systemMonitor.ServerID),
		interval:        interval,
		ctx:             ctx,
		cancel:          cancel,
	}
} // Start begins the monitoring service
func (ms *MonitorService) Start() {
	log.Printf("Starting system monitoring service with %v interval", ms.interval)
	log.Printf("Monitoring server: %s", ms.monitor.ServerID)

	// Start activity monitoring in background
	go ms.activityMonitor.StartActivityMonitoring(ms.ctx)
	log.Println("Activity monitoring started")

	// Start service detection in background (runs every 30 seconds)
	go ms.startServiceDetection()
	log.Println("Service detection started")

	// Initial collection
	if err := ms.collectAndStore(); err != nil {
		log.Printf("Initial data collection failed: %v", err)
	}
	go func() {
		for {
			for _, sc := range model.ServerCache {
				if sc.LastCheckTime.IsZero() {
					sc.LastCheckTime = time.Now()
				}
				convertedID := strings.ReplaceAll(sc.ID, ".", "_")
				if sc.LastCheckTime.Before(time.Now().Add(-30 * time.Second)) {
					model.ServerCache[sc.ID].Status = "offline"
					ws.BroadcastMessage(1, "server:status-"+convertedID+"::offline")
				} else {
					ws.BroadcastMessage(1, "server:status-"+convertedID+"::online")
				}
			}
			time.Sleep(30 * time.Second)
		}
	}()
	// Start periodic monitoring
	ticker := time.NewTicker(ms.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			log.Println("Monitoring service stopped")
			return
		case <-ticker.C:
			if err := ms.collectAndStore(); err != nil {
				log.Printf("Data collection failed: %v", err)
			}
		}
	}
}

// Stop stops the monitoring service
func (ms *MonitorService) Stop() {
	log.Println("Stopping monitoring service...")
	ms.cancel()
}

// collectAndStore collects system data and stores it in cache
func (ms *MonitorService) collectAndStore() error {
	start := time.Now()

	// Collect system data as partial updates
	updates, err := ms.monitor.CollectSystemDataPartial()
	if err != nil {
		return fmt.Errorf("failed to collect system data: %v", err)
	}

	// Add response time to updates
	updates["response_time"] = int(time.Since(start).Milliseconds())
	updates["last_check_time"] = time.Now()
	updates["status"] = "online"

	// Get old server data for comparison
	oldServerData, validOldServerData := model.GetServerCache(ms.monitor.ServerID)

	// Update cache with partial data (not overwriting existing data)
	if err := model.UpdateServerCache(ms.monitor.ServerID, updates); err != nil {
		return fmt.Errorf("failed to update server cache: %v", err)
	}

	convertedID := strings.ReplaceAll(ms.monitor.ServerID, ".", "_")

	// Compare old and new data for WebSocket broadcasting
	if validOldServerData {
		changed := make([]string, 0)

		// Compare CPU usage
		if cpuUsage, ok := updates["cpu_usage"].(float64); ok {
			if fmt.Sprintf("%.2f", cpuUsage) != fmt.Sprintf("%.2f", oldServerData.CPUUsage) {
				changed = append(changed, fmt.Sprintf("cpu_usage-%s::%.2f", convertedID, cpuUsage))
			}
		}

		// Compare memory usage
		if memUsage, ok := updates["memory_usage"].(float64); ok {
			if fmt.Sprintf("%.2f", memUsage) != fmt.Sprintf("%.2f", oldServerData.MemoryUsage) {
				changed = append(changed, fmt.Sprintf("memory_usage-%s::%.2f", convertedID, memUsage))
			}
		}

		// Compare disk usage
		if diskUsed, ok := updates["disk_used"].(uint64); ok {
			if diskUsed != oldServerData.DiskUsed {
				changed = append(changed, fmt.Sprintf("disk_used-%s::%d", convertedID, diskUsed))
			}
		}

		// Compare network I/O
		if netIn, ok := updates["net_in"].(uint64); ok {
			if netIn != oldServerData.NetIn {
				changed = append(changed, fmt.Sprintf("net_in-%s::%d", convertedID, netIn))
			}
		}
		if netOut, ok := updates["net_out"].(uint64); ok {
			if netOut != oldServerData.NetOut {
				changed = append(changed, fmt.Sprintf("net_out-%s::%d", convertedID, netOut))
			}
		}

		// Compare status
		if status, ok := updates["status"].(string); ok {
			if status != oldServerData.Status {
				changed = append(changed, fmt.Sprintf("status-%s::%s", convertedID, status))
			}
		}

		// Compare response time
		if responseTime, ok := updates["response_time"].(int); ok {
			if responseTime != oldServerData.ResponseTime {
				changed = append(changed, fmt.Sprintf("response_time-%s::%d", convertedID, responseTime))
			}
		}

		// Add last_check_time update
		if lastCheckTime, ok := updates["last_check_time"].(time.Time); ok {
			changed = append(changed, fmt.Sprintf("last_check_time-%s::%s", convertedID, lastCheckTime.Format(utils.T_YYYYMMDD_HHmmss)))
		}

		// Broadcast changes if any
		if len(changed) > 0 {
			dataString := fmt.Sprintf("server:%s", strings.Join(changed, ";;"))
			ws.BroadcastMessage(1, dataString)
			if wsclient.WsMasterConn != nil {
				go wsclient.SendMessage(wsclient.WsMasterConn, dataString)
			}
			model.ServerCache[ms.monitor.ServerID].LastCheckTime = time.Now()
		}
	}

	return nil
}

// startServiceDetection runs service detection periodically
func (ms *MonitorService) startServiceDetection() {
	// Initial service detection
	if err := ms.serviceDetector.UpdateServices(); err != nil {
		log.Printf("Initial service detection failed: %v", err)
	}

	// Periodic service detection (every 30 seconds)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			log.Println("Service detection stopped")
			return
		case <-ticker.C:
			if err := ms.serviceDetector.UpdateServices(); err != nil {
				log.Printf("Service detection failed: %v", err)
			}
		}
	}
}

// GetDetectedServices returns detected services for the current server
func (ms *MonitorService) GetDetectedServices() []*model.ServerService {
	model.MuServerServices.Lock()
	defer model.MuServerServices.Unlock()

	if services, exists := model.ServerServices[ms.monitor.ServerID]; exists {
		return services
	}
	return []*model.ServerService{}
}

// printStatus prints current server status for testing
func (ms *MonitorService) printStatus(server *model.Server) {
	fmt.Printf("\n=== System Monitor Update [%s] ===\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Server ID: %s\n", server.ID)
	fmt.Printf("Server Name: %s\n", server.ServerName)
	fmt.Printf("OS: %s %s\n", server.OS, server.OSVersion)
	fmt.Printf("Status: %s\n", server.Status)
	fmt.Printf("Response Time: %dms\n", server.ResponseTime)

	fmt.Printf("\n--- CPU Information ---\n")
	fmt.Printf("CPU Usage: %.2f%%\n", server.CPUUsage)
	fmt.Printf("CPU Cores: %d\n", server.CPUCores)
	fmt.Printf("CPU Model: %s\n", server.CPUModel)
	if server.LoadAverage1 > 0 {
		fmt.Printf("Load Average: %.2f, %.2f, %.2f\n", server.LoadAverage1, server.LoadAverage5, server.LoadAverage15)
	}

	fmt.Printf("\n--- Memory Information ---\n")
	fmt.Printf("Memory Usage: %.2f%% (%s / %s)\n",
		server.MemoryUsage,
		formatBytes(server.MemoryUsed),
		formatBytes(server.MemoryTotal))
	fmt.Printf("Memory Available: %s\n", formatBytes(server.MemoryAvailable))
	if server.SwapTotal > 0 {
		fmt.Printf("Swap: %s / %s\n", formatBytes(server.SwapUsed), formatBytes(server.SwapTotal))
	}

	fmt.Printf("\n--- Disk Information ---\n")
	fmt.Printf("Disk Used: %s / %s (%s free)\n",
		formatBytes(server.DiskUsed),
		formatBytes(server.DiskTotal),
		formatBytes(server.DiskFree))
	if server.DiskRead > 0 || server.DiskWrite > 0 {
		fmt.Printf("Disk I/O: Read %s, Write %s\n", formatBytes(server.DiskRead), formatBytes(server.DiskWrite))
		fmt.Printf("Disk Ops: Read %d, Write %d\n", server.DiskReadOps, server.DiskWriteOps)
	}

	fmt.Printf("\n--- Network Information ---\n")
	if server.NetIn > 0 || server.NetOut > 0 {
		fmt.Printf("Network I/O: In %s, Out %s\n", formatBytes(server.NetIn), formatBytes(server.NetOut))
		fmt.Printf("Network Packets: In %d, Out %d\n", server.NetPacketsIn, server.NetPacketsOut)
		if server.NetErrorsIn > 0 || server.NetErrorsOut > 0 {
			fmt.Printf("Network Errors: In %d, Out %d\n", server.NetErrorsIn, server.NetErrorsOut)
		}
	}

	fmt.Printf("\n--- System Information ---\n")
	if server.Architecture != "" {
		fmt.Printf("Architecture: %s\n", server.Architecture)
	}
	if server.KernelVersion != "" {
		fmt.Printf("Kernel: %s\n", server.KernelVersion)
	}
	fmt.Printf("Process Count: %d\n", server.ProcessCount)
	fmt.Printf("Active Users: %d\n", server.ActiveUsers)
	if server.SSHConnections > 0 {
		fmt.Printf("SSH Connections: %d\n", server.SSHConnections)
	}
	if server.Temperature > 0 {
		fmt.Printf("CPU Temperature: %.1f°C\n", server.Temperature)
	}
	if !server.UptimeSince.IsZero() {
		uptime := time.Since(server.UptimeSince)
		fmt.Printf("Uptime: %s\n", formatDuration(uptime))
	}

	fmt.Printf("\n--- GPU and Display Information ---\n")
	if server.HasGPU {
		fmt.Printf("GPU: %s\n", server.GPUName)
		if server.GPUTemperature > 0 {
			fmt.Printf("GPU Temperature: %.1f°C\n", server.GPUTemperature)
		}
		if server.GPUUsage > 0 {
			fmt.Printf("GPU Usage: %.1f%%\n", server.GPUUsage)
		}
		if server.GPUMemoryTotal > 0 {
			fmt.Printf("GPU Memory: %s / %s\n", formatBytes(server.GPUMemoryUsed), formatBytes(server.GPUMemoryTotal))
		}
	} else {
		fmt.Printf("GPU: Not detected or integrated\n")
	}
	fmt.Printf("Display Count: %d\n", server.DisplayCount)
	if server.DisplayResolution != "" {
		fmt.Printf("Primary Resolution: %s\n", server.DisplayResolution)
	}

	fmt.Printf("\n--- GUI and Desktop Information ---\n")
	fmt.Printf("Uses GUI: %t\n", server.UseGUI)
	fmt.Printf("Desktop Session Active: %t\n", server.HasDesktopSession)
	if server.DesktopEnvironment != "" {
		fmt.Printf("Desktop Environment: %s\n", server.DesktopEnvironment)
	}

	fmt.Printf("\n--- Network Ports Information ---\n")
	fmt.Printf("Total Open Ports: %d\n", server.TotalPortOpens)
	fmt.Printf("Listening Services: %d\n", server.ListeningServices)
	if server.OpenPortsList != "" {
		fmt.Printf("Open Ports: %s\n", server.OpenPortsList)
	}

	// Show cache status
	cache := model.GetAllServersCache()
	fmt.Printf("\n--- Cache Status ---\n")
	fmt.Printf("Total Servers in Cache: %d\n", len(cache))

	fmt.Printf("=====================================\n")
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human readable format
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// GetCachedServerData returns server data from cache for API endpoints
// func GetCachedServerData(serverID string) (*model.Server, bool) {
// 	return model.GetServerCache(serverID)
// }

// GetAllCachedServers returns all servers from cache
func GetAllCachedServers() []model.Server {
	cache := model.GetAllServersCache()
	servers := make([]model.Server, 0, len(cache))
	for _, server := range cache {
		servers = append(servers, *server)
	}
	return servers
}

// GetAllDetectedServices returns all detected services from all servers
func GetAllDetectedServices() map[string][]*model.ServerService {
	model.MuServerServices.Lock()
	defer model.MuServerServices.Unlock()

	// Create a copy to avoid race conditions
	result := make(map[string][]*model.ServerService)
	for serverID, services := range model.ServerServices {
		serviceCopy := make([]*model.ServerService, len(services))
		copy(serviceCopy, services)
		result[serverID] = serviceCopy
	}
	return result
}

// GetDetectedServicesForServer returns detected services for a specific server
func GetDetectedServicesForServer(serverID string) []*model.ServerService {
	model.MuServerServices.Lock()
	defer model.MuServerServices.Unlock()

	if services, exists := model.ServerServices[serverID]; exists {
		// Create a copy to avoid race conditions
		serviceCopy := make([]*model.ServerService, len(services))
		copy(serviceCopy, services)
		return serviceCopy
	}
	return []*model.ServerService{}
}
