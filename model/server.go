package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"server_monitor/utils"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// In-memory cache for fast access (thread-safe)
var (
	Servers     []Server
	ServerCache = make(map[string]*Server)
	CacheMutex  sync.RWMutex
)

// Cache file configuration
const (
	cacheDir  = ".cache"
	cacheFile = "server.json"
)

// getCacheFilePath returns the full path to the cache file
func getCacheFilePath() string {
	return filepath.Join(cacheDir, cacheFile)
}

// InitServerCache loads cache from file if exists
func init() {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("failed to create cache directory: %v", err)
		return
	}

	cachePath := getCacheFilePath()

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Cache file doesn't exist, start with empty cache
		fmt.Printf("Cache file does not exist, starting with empty cache: %v", err)
		return
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		fmt.Printf("failed to read cache file: %v", err)
		return
	}

	// Parse JSON into temporary map
	var tempCache map[string]*Server
	if err := json.Unmarshal(data, &tempCache); err != nil {
		fmt.Printf("failed to parse cache file: %v", err)
		return
	}

	// Load into ServerCache
	for id, server := range tempCache {
		if server.LastMouseMovement.Before(server.UptimeSince) {
			server.LastMouseMovement = server.UptimeSince
		}
		ServerCache[id] = server
	}

}

// FlushServerCache saves current cache to file
func FlushServerCache() error {
	CacheMutex.RLock()
	defer CacheMutex.RUnlock()

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Marshal cache to JSON
	data, err := json.MarshalIndent(ServerCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %v", err)
	}

	// Write to cache file
	cachePath := getCacheFilePath()
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	return nil
}

// SaveCacheAsync saves cache to file asynchronously (non-blocking)
func SaveCacheAsync() {
	go func() {
		if err := FlushServerCache(); err != nil {
			fmt.Printf("Warning: Failed to save cache: %v\n", err)
		}
	}()
}

// Server represents a Ubuntu server monitoring data
type Server struct {
	ID          string    `json:"id" gorm:"size:20;primaryKey;not null;comment:Server IP address"`
	IP          string    `json:"ip" gorm:"size:20;primaryKey;not null;comment:Server IP address"`
	GroupIP     string    `json:"group_ip" gorm:"size:20;comment:Container or wrapper server address"`
	ServerName  string    `json:"server_name" gorm:"size:255;comment:Server hostname"`
	UptimeSince time.Time `json:"uptime_since" gorm:"size:255;comment:System uptime start time"`

	// CPU Information
	CPUUsage      float64 `json:"cpu_usage" gorm:"type:decimal(5,2);comment:CPU usage percentage"`
	CPUCores      int     `json:"cpu_cores" gorm:"type:int;comment:Number of CPU cores"`
	CPUModel      string  `json:"cpu_model" gorm:"size:255;comment:CPU model name"`
	LoadAverage1  float64 `json:"load_average_1" gorm:"type:decimal(10,2);comment:Load average 1 minute"`
	LoadAverage5  float64 `json:"load_average_5" gorm:"type:decimal(10,2);comment:Load average 5 minutes"`
	LoadAverage15 float64 `json:"load_average_15" gorm:"type:decimal(10,2);comment:Load average 15 minutes"`

	// Memory Information
	MemoryTotal     uint64  `json:"memory_total" gorm:"type:bigint;comment:Total memory in bytes"`
	MemoryUsed      uint64  `json:"memory_used" gorm:"type:bigint;comment:Used memory in bytes"`
	MemoryFree      uint64  `json:"memory_free" gorm:"type:bigint;comment:Free memory in bytes"`
	MemoryAvailable uint64  `json:"memory_available" gorm:"type:bigint;comment:Available memory in bytes"`
	MemoryUsage     float64 `json:"memory_usage" gorm:"type:decimal(5,2);comment:Memory usage percentage"`
	SwapTotal       uint64  `json:"swap_total" gorm:"type:bigint;comment:Total swap in bytes"`
	SwapUsed        uint64  `json:"swap_used" gorm:"type:bigint;comment:Used swap in bytes"`
	SwapFree        uint64  `json:"swap_free" gorm:"type:bigint;comment:Free swap in bytes"`

	// Disk Information
	DiskTotal    uint64 `json:"disk_total" gorm:"type:bigint;comment:Total disk space in bytes"`
	DiskUsed     uint64 `json:"disk_used" gorm:"type:bigint;comment:Used disk space in bytes"`
	DiskFree     uint64 `json:"disk_free" gorm:"type:bigint;comment:Free disk space in bytes"`
	DiskRead     uint64 `json:"disk_read" gorm:"type:bigint;comment:Disk read bytes"`
	DiskWrite    uint64 `json:"disk_write" gorm:"type:bigint;comment:Disk write bytes"`
	DiskReadOps  uint64 `json:"disk_read_ops" gorm:"type:bigint;comment:Disk read operations"`
	DiskWriteOps uint64 `json:"disk_write_ops" gorm:"type:bigint;comment:Disk write operations"`

	// Network Information
	NetIn         uint64 `json:"net_in" gorm:"type:bigint;comment:Network bytes received"`
	NetOut        uint64 `json:"net_out" gorm:"type:bigint;comment:Network bytes transmitted"`
	NetPacketsIn  uint64 `json:"net_packets_in" gorm:"type:bigint;comment:Network packets received"`
	NetPacketsOut uint64 `json:"net_packets_out" gorm:"type:bigint;comment:Network packets transmitted"`
	NetErrorsIn   uint64 `json:"net_errors_in" gorm:"type:bigint;comment:Network input errors"`
	NetErrorsOut  uint64 `json:"net_errors_out" gorm:"type:bigint;comment:Network output errors"`

	// SSH and Connection Information
	SSHConnections int `json:"ssh_connections" gorm:"type:int;comment:Number of active SSH connections"`
	ActiveUsers    int `json:"active_users" gorm:"type:int;comment:Number of active users"`
	ProcessCount   int `json:"process_count" gorm:"type:int;comment:Total number of processes"`

	// System Information
	OS            string  `json:"os" gorm:"size:255;comment:Operating system type"`
	OSVersion     string  `json:"os_version" gorm:"size:255;comment:Operating system version"`
	KernelVersion string  `json:"kernel_version" gorm:"size:255;comment:Kernel version"`
	Architecture  string  `json:"architecture" gorm:"size:50;comment:System architecture"`
	Temperature   float64 `json:"temperature" gorm:"type:decimal(5,2);comment:CPU temperature in Celsius"`

	// GPU and Display Information
	HasGPU            bool    `json:"has_gpu" gorm:"comment:Whether system has dedicated GPU"`
	GPUName           string  `json:"gpu_name" gorm:"size:255;comment:GPU name/model"`
	GPUTemperature    float64 `json:"gpu_temperature" gorm:"type:decimal(5,2);comment:GPU temperature in Celsius"`
	GPUMemoryTotal    uint64  `json:"gpu_memory_total" gorm:"type:bigint;comment:GPU memory total in bytes"`
	GPUMemoryUsed     uint64  `json:"gpu_memory_used" gorm:"type:bigint;comment:GPU memory used in bytes"`
	GPUUsage          float64 `json:"gpu_usage" gorm:"type:decimal(5,2);comment:GPU usage percentage"`
	DisplayCount      int     `json:"display_count" gorm:"type:int;comment:Number of connected displays"`
	DisplayResolution string  `json:"display_resolution" gorm:"size:100;comment:Primary display resolution"`

	// User Activity and GUI Detection
	UseGUI             bool      `json:"use_gui" gorm:"comment:Interface type (GUI/CLI)"`
	HasDesktopSession  bool      `json:"has_desktop_session" gorm:"comment:Whether desktop session is active"`
	DesktopEnvironment string    `json:"desktop_environment" gorm:"size:100;comment:Desktop environment (GNOME, KDE, XFCE, etc)"`
	LastMouseMovement  time.Time `json:"last_mouse_movement" gorm:"comment:Last mouse movement detected"`
	LastCLIActivity    time.Time `json:"last_cli_activity" gorm:"comment:Last CLI activity detected"`

	// Network Ports
	TotalPortOpens    int    `json:"total_port_opens" gorm:"type:int;comment:Number of open ports"`
	OpenPortsList     string `json:"open_ports_list" gorm:"type:text;comment:List of open ports (JSON array)"`
	ListeningServices int    `json:"listening_services" gorm:"type:int;comment:Number of services listening on ports"`

	// Service Status
	ServicesRunning int `json:"services_running" gorm:"type:int;comment:Number of running services"`
	ServicesFailed  int `json:"services_failed" gorm:"type:int;comment:Number of failed services"`

	// Monitoring Status
	Status        string    `json:"status" gorm:"size:50;default:'online';comment:Server status (online/offline/warning/error)"`
	LastCheckTime time.Time `json:"last_check_time" gorm:"type:timestamp;comment:Last monitoring check time"`
	ResponseTime  int       `json:"response_time" gorm:"type:int;comment:Response time in milliseconds"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Cache operations
func SetServerCache(id string, server *Server) {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()
	ServerCache[id] = server

	// Save cache asynchronously after update
	SaveCacheAsync()
}

// UpdateServerCache - partial update using map[string]interface{}
func UpdateServerCache(id string, updates map[string]interface{}) error {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	server, exists := ServerCache[id]
	if !exists || server == nil {
		// If not exists, create a new empty Server and put in cache
		server = &Server{ID: id, IP: id, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		ServerCache[id] = server
	}

	for key, value := range updates {
		switch key {
		case "cpu_usage":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].CPUUsage = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "status":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].Status = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "memory_total":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].MemoryTotal = v
			} else {
				fmt.Printf("Warning: Failed to convert memory_total for %s: %v\n", id, err)
			}
		case "memory_used":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].MemoryUsed = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "memory_free":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].MemoryFree = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "memory_available":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].MemoryAvailable = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "memory_usage":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].MemoryUsage = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "swap_total":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].SwapTotal = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "swap_used":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].SwapUsed = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "swap_free":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].SwapFree = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_total":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskTotal = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_used":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskUsed = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_free":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskFree = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_read":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskRead = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_write":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskWrite = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_read_ops":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskReadOps = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "disk_write_ops":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].DiskWriteOps = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_in":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetIn = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_out":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetOut = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_packets_in":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetPacketsIn = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_packets_out":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetPacketsOut = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_errors_in":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetErrorsIn = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "net_errors_out":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].NetErrorsOut = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "ssh_connections":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].SSHConnections = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "active_users":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ActiveUsers = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "server_name":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				// Only update if current value is empty or zero
				if ServerCache[id].ServerName == "" {
					ServerCache[id].ServerName = v
				}
			}
		case "uptime_since":
			if v, err := utils.TryConvertTo[time.Time](value); err == nil {
				// Only update if current value is zero/empty
				if !v.IsZero() && v.After(ServerCache[id].UptimeSince) {
					ServerCache[id].UptimeSince = v
				}
			} else if s, ok := value.(string); ok {
				if t, ok := convertToTime(s); ok {
					// Only update if current value is zero/empty
					if ServerCache[id].UptimeSince.IsZero() {
						ServerCache[id].UptimeSince = t
					}
				}
			}
		case "cpu_cores":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				// Only update if current value is zero/empty
				if ServerCache[id].CPUCores == 0 {
					ServerCache[id].CPUCores = v
				}
			}
		case "cpu_model":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				// Only update if current value is empty
				if ServerCache[id].CPUModel == "" {
					ServerCache[id].CPUModel = v
				}
			}
		case "load_average_1":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].LoadAverage1 = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "load_average_5":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].LoadAverage5 = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "load_average_15":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].LoadAverage15 = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "process_count":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ProcessCount = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "os":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].OS = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "os_version":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].OSVersion = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "kernel_version":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].KernelVersion = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "architecture":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].Architecture = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "temperature":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].Temperature = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "has_gpu":
			if v, err := utils.TryConvertTo[bool](value); err == nil {
				ServerCache[id].HasGPU = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "gpu_name":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].GPUName = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "gpu_temperature":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].GPUTemperature = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "gpu_memory_total":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].GPUMemoryTotal = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "gpu_memory_used":
			if v, err := utils.TryConvertTo[uint64](value); err == nil {
				ServerCache[id].GPUMemoryUsed = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "gpu_usage":
			if v, err := utils.TryConvertTo[float64](value); err == nil {
				ServerCache[id].GPUUsage = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "display_count":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].DisplayCount = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "display_resolution":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].DisplayResolution = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "use_gui":
			if v, err := utils.TryConvertTo[bool](value); err == nil {
				ServerCache[id].UseGUI = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "has_desktop_session":
			if v, err := utils.TryConvertTo[bool](value); err == nil {
				ServerCache[id].HasDesktopSession = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "desktop_environment":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].DesktopEnvironment = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "last_mouse_movement":
			if v, err := utils.TryConvertTo[time.Time](value); err == nil {
				// Validate timestamp is not zero/empty
				if v.IsZero() {
					logrus.Warnf("Skipping last_mouse_movement update: timestamp is zero/empty")
					continue
				}
				// Validate timestamp is not older than uptime_since
				if v.Before(ServerCache[id].UptimeSince) {
					logrus.Warnf("Skipping last_mouse_movement update: timestamp %s is older than uptime_since %s",
						v.Format("2006-01-02 15:04:05"),
						ServerCache[id].UptimeSince.Format("2006-01-02 15:04:05"))
					continue
				}
				ServerCache[id].LastMouseMovement = v
			} else if s, ok := value.(string); ok {
				if t, ok := convertToTime(s); ok {
					// Validate timestamp is not zero/empty
					if t.IsZero() {
						logrus.Warnf("Skipping last_mouse_movement update: parsed timestamp is zero/empty")
						continue
					}
					// Validate timestamp is not older than uptime_since
					if t.Before(ServerCache[id].UptimeSince) {
						logrus.Warnf("Skipping last_mouse_movement update: parsed timestamp %s is older than uptime_since %s",
							t.Format("2006-01-02 15:04:05"),
							ServerCache[id].UptimeSince.Format("2006-01-02 15:04:05"))
						continue
					}
					ServerCache[id].LastMouseMovement = t
				} else {
					logrus.Errorf("invalid string format for last_mouse_movement")
					continue
				}
			}
		case "last_cli_activity":
			if v, err := utils.TryConvertTo[time.Time](value); err == nil {
				// Validate timestamp is not zero/empty
				if v.IsZero() {
					logrus.Warnf("Skipping last_cli_activity update: timestamp is zero/empty")
					continue
				}
				// Validate timestamp is not older than uptime_since
				if v.Before(ServerCache[id].UptimeSince) {
					logrus.Warnf("Skipping last_cli_activity update: timestamp %s is older than uptime_since %s",
						v.Format("2006-01-02 15:04:05"),
						ServerCache[id].UptimeSince.Format("2006-01-02 15:04:05"))
					continue
				}
				// Only update if newer than current value
				if v.After(ServerCache[id].LastCLIActivity) {
					ServerCache[id].LastCLIActivity = v
					logrus.Infof("Updated last_cli_activity for server %s: %v", id, v)
				}
			} else if s, ok := value.(string); ok {
				if t, ok := convertToTime(s); ok {
					// Validate timestamp is not zero/empty
					if t.IsZero() {
						logrus.Warnf("Skipping last_cli_activity update: parsed timestamp is zero/empty")
						continue
					}
					// Validate timestamp is not older than uptime_since
					if t.Before(ServerCache[id].UptimeSince) {
						logrus.Warnf("Skipping last_cli_activity update: parsed timestamp %s is older than uptime_since %s",
							t.Format("2006-01-02 15:04:05"),
							ServerCache[id].UptimeSince.Format("2006-01-02 15:04:05"))
						continue
					}
					// Only update if newer than current value
					if t.After(ServerCache[id].LastCLIActivity) {
						ServerCache[id].LastCLIActivity = t
						logrus.Infof("Updated last_cli_activity for server %s from string: %v", id, t)
					}
				} else {
					logrus.Errorf("invalid string format for last_cli_activity: %s", s)
					continue
				}
			} else {
				logrus.Errorf("invalid type for last_cli_activity: %T", value)
			}
		case "total_port_opens":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].TotalPortOpens = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "open_ports_list":
			if v, err := utils.TryConvertTo[string](value); err == nil {
				ServerCache[id].OpenPortsList = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "listening_services":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ListeningServices = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "services_running":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ServicesRunning = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "services_failed":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ServicesFailed = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		case "last_check_time":
			if v, err := utils.TryConvertTo[time.Time](value); err == nil {
				ServerCache[id].LastCheckTime = v
			} else if s, ok := value.(string); ok {
				if t, ok := convertToTime(s); ok {
					ServerCache[id].LastCheckTime = t
				} else {
					logrus.Errorf("invalid string format for last_check_time")
					continue
				}
			}
		case "response_time":
			if v, err := utils.TryConvertTo[int](value); err == nil {
				ServerCache[id].ResponseTime = v
			} else {
				fmt.Printf("Warning: Failed to convert %s for %s: %v\n", key, id, err)
			}
		default:
			// logrus.Errorf("field %s not supported for update", key)
		}
	}

	// Update timestamp
	ServerCache[id].UpdatedAt = time.Now()

	// logrus.Debugf("Successfully updated server cache for ID: %s", id)

	// Save cache asynchronously after update
	SaveCacheAsync()

	return nil
}

// // Helper function to set field value with type conversion
// func setFieldValue(fieldValue reflect.Value, value interface{}) error {
// 	valueType := fieldValue.Type()
// 	inputValue := reflect.ValueOf(value)

// 	// Handle nil values
// 	if value == nil {
// 		if fieldValue.Kind() == reflect.Ptr {
// 			fieldValue.Set(reflect.Zero(valueType))
// 			return nil
// 		}
// 		return fmt.Errorf("cannot set nil to non-pointer field")
// 	}

// 	// Direct assignment if types match
// 	if inputValue.Type() == valueType {
// 		fieldValue.Set(inputValue)
// 		return nil
// 	}

// 	// Type conversion based on target type
// 	switch valueType.Kind() {
// 	case reflect.String:
// 		fieldValue.SetString(fmt.Sprintf("%v", value))

// 	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
// 		if intVal, ok := convertToInt64(value); ok {
// 			fieldValue.SetInt(intVal)
// 		} else {
// 			return fmt.Errorf("cannot convert %v to int", value)
// 		}

// 	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
// 		if uintVal, ok := convertToUint64(value); ok {
// 			fieldValue.SetUint(uintVal)
// 		} else {
// 			return fmt.Errorf("cannot convert %v to uint", value)
// 		}

// 	case reflect.Float32, reflect.Float64:
// 		if floatVal, ok := convertToFloat64(value); ok {
// 			fieldValue.SetFloat(floatVal)
// 		} else {
// 			return fmt.Errorf("cannot convert %v to float", value)
// 		}

// 	case reflect.Bool:
// 		if boolVal, ok := convertToBool(value); ok {
// 			fieldValue.SetBool(boolVal)
// 		} else {
// 			return fmt.Errorf("cannot convert %v to bool", value)
// 		}

// 	case reflect.Struct:
// 		if valueType == reflect.TypeOf(time.Time{}) {
// 			if timeVal, ok := convertToTime(value); ok {
// 				fieldValue.Set(reflect.ValueOf(timeVal))
// 			} else {
// 				return fmt.Errorf("cannot convert %v to time.Time", value)
// 			}
// 		} else {
// 			return fmt.Errorf("unsupported struct type %v", valueType)
// 		}

// 	default:
// 		return fmt.Errorf("unsupported field type %v", valueType)
// 	}

// 	return nil
// }

func GetServerCache(id string) (Server, bool) {
	CacheMutex.RLock()
	defer CacheMutex.RUnlock()
	server, exists := ServerCache[id]
	if !exists || server == nil {
		return Server{}, false
	}
	return *server, true
}

func DeleteServerCache(id string) {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()
	delete(ServerCache, id)

	// Save cache asynchronously after delete
	SaveCacheAsync()
}

func GetAllServersCache() map[string]*Server {
	CacheMutex.RLock()
	defer CacheMutex.RUnlock()
	// Return copy to prevent race conditions
	result := make(map[string]*Server)
	for k, v := range ServerCache {
		result[k] = v
	}
	return result
}

// TableName sets the table name for GORM
func (Server) TableName() string {
	return "ubuntu_servers"
}

// Database operations with cache integration
func (u *Server) Create(db interface{}) error {
	// Save to database (you'll implement this with your DB instance)
	// err := db.Create(u).Error
	// if err == nil {
	// 	SetServerCache(u.ID, u) // Update cache
	// }
	// return err

	// For now, just save to cache
	SetServerCache(u.ID, u)
	return nil
}

func (u *Server) Update(db interface{}) error {
	// Update database
	// err := db.Save(u).Error
	// if err == nil {
	// 	SetServerCache(u.ID, u) // Update cache
	// }
	// return err

	// For now, just update cache
	SetServerCache(u.ID, u)
	return nil
}

func (u *Server) Delete(db interface{}) error {
	// Delete from database
	// err := db.Delete(u).Error
	// if err == nil {
	// 	DeleteServerCache(u.ID) // Remove from cache
	// }
	// return err

	// For now, just remove from cache
	DeleteServerCache(u.ID)
	return nil
}

// Static methods
func FindServerByID(db interface{}, id string) (*Server, error) {
	// Try cache first
	if server, exists := ServerCache[id]; exists {
		return server, nil
	}

	// If not in cache, query database
	// var server Server
	// err := db.First(&server, "id = ?", id).Error
	// if err == nil {
	// 	SetServerCache(id, &server) // Cache the result
	// }
	// return &server, err

	return nil, nil // Placeholder
}

func GetAllServers(db interface{}) ([]Server, error) {
	// Query database
	// var servers []Server
	// err := db.Find(&servers).Error
	// if err == nil {
	// 	// Update cache with fresh data
	// 	for _, server := range servers {
	// 		serverCopy := server
	// 		SetServerCache(server.ID, &serverCopy)
	// 	}
	// }
	// return servers, err

	// For now, return from cache
	cache := GetAllServersCache()
	servers := make([]Server, 0, len(cache))
	for _, server := range cache {
		servers = append(servers, *server)
	}
	return servers, nil
}

// Helper functions for type conversion
func convertToInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			return intVal, true
		}
	}
	return 0, false
}

func convertToUint64(value interface{}) (uint64, bool) {
	switch v := value.(type) {
	case uint:
		return uint64(v), true
	case uint8:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	case int:
		if v >= 0 {
			return uint64(v), true
		}
	case int64:
		if v >= 0 {
			return uint64(v), true
		}
	case float64:
		if v >= 0 {
			return uint64(v), true
		}
	case string:
		if uintVal, err := strconv.ParseUint(v, 10, 64); err == nil {
			return uintVal, true
		}
	}
	return 0, false
}

func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal, true
		}
	}
	return 0, false
}

func convertToBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if boolVal, err := strconv.ParseBool(v); err == nil {
			return boolVal, true
		}
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	case float64:
		return v != 0, true
	}
	return false, false
}

func convertToTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		// Try different time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
			"2006/01/02 15:04:05",           // matches utils.T_YYYYMMDD_HHmmss format
			"2006-01-02 15:04:05 -0700 MST", // matches "2025-10-15 19:48:00 +0000 UTC"
		}
		for _, format := range formats {
			if timeVal, err := time.Parse(format, v); err == nil {
				return timeVal, true
			}
		}
	case int64:
		// Unix timestamp
		return time.Unix(v, 0), true
	}
	return time.Time{}, false
}

// Example usage functions
// func ExamplePartialUpdate() {
// 	// Create a server first
// 	server := &Server{
// 		ID:         "192.168.1.100",
// 		IP:         "192.168.1.100",
// 		ServerName: "web-server-01",
// 		CPUUsage:   25.5,
// 		Status:     "online",
// 	}

// 	// Add to cache
// 	SetServerCache(server.ID, server)

// 	// Partial update examples:

// 	// Update CPU usage only
// 	updates1 := map[string]interface{}{
// 		"cpu_usage": 85.2,
// 		"status":    "warning",
// 	}
// 	UpdateServerCache("192.168.1.100", updates1)

// 	// Update memory information
// 	updates2 := map[string]interface{}{
// 		"memory_total": uint64(16777216000), // 16GB
// 		"memory_used":  uint64(8388608000),  // 8GB
// 		"memory_usage": 50.0,
// 	}
// 	UpdateServerCache("192.168.1.100", updates2)

// 	// Update with mixed types including time
// 	updates3 := map[string]interface{}{
// 		"server_name":     "updated-web-server-01",
// 		"use_gui":         true,
// 		"ssh_connections": 5,
// 		"last_check_time": "2025-10-15T10:30:00Z",
// 	}
// 	UpdateServerCache("192.168.1.100", updates3)
// }
