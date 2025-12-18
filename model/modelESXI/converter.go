package modelESXI

import (
	"time"

	"github.com/vmware/govmomi/vim25/mo"
)

// ConvertToMonitoringData mengkonversi data dari govmomi ke model struct
func ConvertToMonitoringData(hostID string, host mo.HostSystem, vmInfos []VMInfoInput, datastores []mo.Datastore) *MonitoringData {
	data := NewMonitoringData()

	// Set ID dari host connection
	data.ID = hostID

	// Host Info
	data.HostInfo = HostInfo{
		HostName:            host.Summary.Config.Name,
		HostConnectionState: string(host.Summary.Runtime.ConnectionState),
		HostOverallStatus:   string(host.Summary.OverallStatus),
	}

	// CPU Info
	totalMhz := int64(host.Summary.Hardware.CpuMhz) * int64(host.Summary.Hardware.NumCpuThreads)
	usedMhz := int64(host.Summary.QuickStats.OverallCpuUsage)
	cpuPercent := float64(usedMhz) / float64(totalMhz) * 100

	data.CPUInfo = CPUInfo{
		CPUModel:        host.Summary.Hardware.CpuModel,
		CPUCores:        int32(host.Summary.Hardware.NumCpuCores),
		CPUThreads:      int32(host.Summary.Hardware.NumCpuThreads),
		CPUSpeedMHz:     host.Summary.Hardware.CpuMhz,
		CPUUsedMHz:      usedMhz,
		CPUTotalMHz:     totalMhz,
		CPUUsagePercent: cpuPercent,
	}

	// Memory Info
	totalMemoryBytes := host.Summary.Hardware.MemorySize
	usedMemoryMB := int64(host.Summary.QuickStats.OverallMemoryUsage)
	totalMemoryMB := totalMemoryBytes / (1024 * 1024)
	freeMemoryMB := totalMemoryMB - usedMemoryMB
	memPercent := float64(usedMemoryMB) / float64(totalMemoryMB) * 100

	data.MemoryInfo = MemoryInfo{
		MemoryUsedBytes:    usedMemoryMB * 1024 * 1024,
		MemoryFreeBytes:    freeMemoryMB * 1024 * 1024,
		MemoryTotalBytes:   totalMemoryBytes,
		MemoryUsagePercent: memPercent,
		MemoryUsedMB:       usedMemoryMB,
		MemoryFreeMB:       freeMemoryMB,
		MemoryTotalMB:      totalMemoryMB,
	}

	// Storage Info
	var totalCapacity, totalUsed, totalFree int64
	var datastoreInfos []DatastoreInfo

	for _, ds := range datastores {
		capacity := ds.Summary.Capacity
		free := ds.Summary.FreeSpace
		used := capacity - free
		usedPercent := float64(used) / float64(capacity) * 100

		datastoreInfo := DatastoreInfo{
			Name:         ds.Summary.Name,
			Capacity:     capacity,
			Used:         used,
			Free:         free,
			UsagePercent: usedPercent,
		}

		datastoreInfos = append(datastoreInfos, datastoreInfo)
		totalCapacity += capacity
		totalUsed += used
		totalFree += free
	}

	totalUsagePercent := float64(totalUsed) / float64(totalCapacity) * 100

	data.StorageInfo = StorageInfo{
		StorageDatastores:        datastoreInfos,
		StorageTotalCapacity:     totalCapacity,
		StorageTotalUsed:         totalUsed,
		StorageTotalFree:         totalFree,
		StorageTotalUsagePercent: totalUsagePercent,
	}

	// VM Info
	var vmInfosConverted []VMInfo
	for _, vm := range vmInfos {
		var memUsagePercent float64
		if vm.MemoryMB > 0 {
			memUsagePercent = float64(vm.MemoryUsageMB) / float64(vm.MemoryMB) * 100
		}

		vmInfo := VMInfo{
			Name:               vm.Name,
			UUID:               vm.UUID,
			PowerState:         vm.PowerState,
			IPs:                vm.IPs,
			CPUCores:           vm.CPUCores,
			CPUUsageMHz:        vm.CPUUsageMHz,
			MemoryMB:           vm.MemoryMB,
			MemoryUsageMB:      vm.MemoryUsageMB,
			MemoryUsagePercent: memUsagePercent,
		}

		vmInfosConverted = append(vmInfosConverted, vmInfo)
	}
	data.VMsInfo = vmInfosConverted

	// System Info
	var uptime time.Duration
	var uptimeText string
	var bootTime *time.Time

	if host.Runtime.BootTime != nil {
		bootTime = host.Runtime.BootTime
		uptime = time.Since(*host.Runtime.BootTime).Truncate(time.Second)
		uptimeText = uptime.String()
	}

	var vendor, model string
	if host.Hardware != nil && host.Hardware.SystemInfo.Vendor != "" {
		vendor = host.Hardware.SystemInfo.Vendor
		model = host.Hardware.SystemInfo.Model
	}

	data.SystemInfo = SystemInfo{
		SystemVendor:     vendor,
		SystemModel:      model,
		SystemBootTime:   bootTime,
		SystemUptime:     uptime,
		SystemUptimeText: uptimeText,
	}

	return data
}

// VMInfoInput adalah struct input untuk VM info (untuk menghindari circular import)
type VMInfoInput struct {
	Name          string
	UUID          string
	PowerState    string
	IPs           []string
	CPUCores      int32
	CPUUsageMHz   int32
	MemoryMB      int32
	MemoryUsageMB int32
}

// ConvertVMInfos mengkonversi slice VMInfo dari main ke VMInfoInput
func ConvertVMInfos(vmInfos []interface{}) []VMInfoInput {
	var converted []VMInfoInput

	for _, vm := range vmInfos {
		// Type assertion to extract fields
		if vmMap, ok := vm.(map[string]interface{}); ok {
			vmInput := VMInfoInput{}

			if name, ok := vmMap["Name"].(string); ok {
				vmInput.Name = name
			}
			if uuid, ok := vmMap["UUID"].(string); ok {
				vmInput.UUID = uuid
			}
			if powerState, ok := vmMap["PowerState"].(string); ok {
				vmInput.PowerState = powerState
			}
			if ips, ok := vmMap["IPs"].([]string); ok {
				vmInput.IPs = ips
			}
			if cpuCores, ok := vmMap["CPUCores"].(int32); ok {
				vmInput.CPUCores = cpuCores
			}
			if cpuUsage, ok := vmMap["CPUUsageMHz"].(int32); ok {
				vmInput.CPUUsageMHz = cpuUsage
			}
			if memMB, ok := vmMap["MemoryMB"].(int32); ok {
				vmInput.MemoryMB = memMB
			}
			if memUsage, ok := vmMap["MemoryUsageMB"].(int32); ok {
				vmInput.MemoryUsageMB = memUsage
			}

			converted = append(converted, vmInput)
		}
	}

	return converted
}
