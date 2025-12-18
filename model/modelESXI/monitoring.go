package modelESXI

import (
	"encoding/json"
	"time"
)

// MonitoringData adalah struktur utama untuk data monitoring ESXi
type MonitoringData struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // offline, online, error
	SystemInfo
	HostInfo
	CPUInfo
	MemoryInfo
	StorageInfo
	VMsInfo []VMInfo `json:"vms_info"`
}

// HostInfo berisi informasi dasar host ESXi
type HostInfo struct {
	HostName            string `json:"host_name"`
	HostConnectionState string `json:"host_connection_state"`
	HostOverallStatus   string `json:"host_overall_status"`
}

// CPUInfo berisi informasi dan usage CPU
type CPUInfo struct {
	CPUModel        string  `json:"cpu_model"`
	CPUCores        int32   `json:"cpu_cores"`
	CPUThreads      int32   `json:"cpu_threads"`
	CPUSpeedMHz     int32   `json:"cpu_speed_mhz"`
	CPUUsedMHz      int64   `json:"cpu_used_mhz"`
	CPUTotalMHz     int64   `json:"cpu_total_mhz"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
}

// MemoryInfo berisi informasi dan usage memori
type MemoryInfo struct {
	MemoryUsedBytes    int64   `json:"memory_used_bytes"`
	MemoryFreeBytes    int64   `json:"memory_free_bytes"`
	MemoryTotalBytes   int64   `json:"memory_total_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	MemoryUsedMB       int64   `json:"memory_used_mb"`
	MemoryFreeMB       int64   `json:"memory_free_mb"`
	MemoryTotalMB      int64   `json:"memory_total_mb"`
}

// StorageInfo berisi informasi storage/datastore
type StorageInfo struct {
	StorageTotalCapacity     int64           `json:"storage_total_capacity"`
	StorageTotalUsed         int64           `json:"storage_total_used"`
	StorageTotalFree         int64           `json:"storage_total_free"`
	StorageTotalUsagePercent float64         `json:"storage_total_usage_percent"`
	StorageDatastores        []DatastoreInfo `json:"storage_datastores"`
}

// DatastoreInfo berisi informasi per datastore
type DatastoreInfo struct {
	Name         string  `json:"name"`
	Capacity     int64   `json:"capacity"`
	Used         int64   `json:"used"`
	Free         int64   `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
}

// VMInfo berisi informasi virtual machine
type VMInfo struct {
	Name               string   `json:"name"`
	UUID               string   `json:"uuid"`
	PowerState         string   `json:"power_state"`
	IPs                []string `json:"ips"`
	CPUCores           int32    `json:"cpu_cores"`
	CPUUsageMHz        int32    `json:"cpu_usage_mhz"`
	MemoryMB           int32    `json:"memory_mb"`
	MemoryUsageMB      int32    `json:"memory_usage_mb"`
	MemoryUsagePercent float64  `json:"memory_usage_percent"`
}

// SystemInfo berisi informasi sistem
type SystemInfo struct {
	SystemVendor     string        `json:"system_vendor"`
	SystemModel      string        `json:"system_model"`
	SystemBootTime   *time.Time    `json:"system_boot_time"`
	SystemUptime     time.Duration `json:"system_uptime"`
	SystemUptimeText string        `json:"system_uptime_text"`
}

// ToJSON mengkonversi MonitoringData ke JSON string
func (m *MonitoringData) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ToJSONCompact mengkonversi MonitoringData ke JSON string compact
func (m *MonitoringData) ToJSONCompact() (string, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// NewMonitoringData membuat instance baru MonitoringData dengan timestamp saat ini
func NewMonitoringData() *MonitoringData {
	return &MonitoringData{
		Timestamp: time.Now(),
		Status:    "unknown", // default status
		VMsInfo:   make([]VMInfo, 0),
		StorageInfo: StorageInfo{
			StorageDatastores: make([]DatastoreInfo, 0),
		},
	}
}
