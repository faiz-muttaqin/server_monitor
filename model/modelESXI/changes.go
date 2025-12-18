package modelESXI

import (
	"fmt"
	"strings"
)

// CompareAndGetChanges membandingkan data lama dengan data baru dan mengembalikan list perubahan
func CompareAndGetChanges(oldData, newData MultiHostData) []string {
	var changes []string

	for hostIP, newHostData := range newData {
		if newHostData == nil {
			continue
		}

		// Convert IP format: 192.101.1.10 -> 192_101_1_10
		formattedIP := strings.ReplaceAll(hostIP, ".", "_")

		// Check if host exists in old data
		oldHostData, exists := oldData[hostIP]
		if !exists || oldHostData == nil {
			// New host, add all key data
			changes = append(changes, getHostKeyData(formattedIP, newHostData)...)
			continue
		}

		// Compare specific fields and detect changes
		changes = append(changes, compareHostData(formattedIP, oldHostData, newHostData)...)
	}

	return changes
}

// getHostKeyData gets key data for a new host
func getHostKeyData(formattedIP string, data *MonitoringData) []string {
	var keyData []string

	// Key metrics to broadcast for new host
	keyData = append(keyData, fmt.Sprintf("status-%s::%s", formattedIP, data.Status))
	keyData = append(keyData, fmt.Sprintf("cpu_used_mhz-%s::%d", formattedIP, data.CPUUsedMHz))
	keyData = append(keyData, fmt.Sprintf("cpu_usage_percent-%s::%.2f", formattedIP, data.CPUUsagePercent))
	keyData = append(keyData, fmt.Sprintf("memory_used_mb-%s::%d", formattedIP, data.MemoryUsedMB))
	keyData = append(keyData, fmt.Sprintf("memory_usage_percent-%s::%.2f", formattedIP, data.MemoryUsagePercent))
	keyData = append(keyData, fmt.Sprintf("storage_total_usage_percent-%s::%.2f", formattedIP, data.StorageTotalUsagePercent))
	keyData = append(keyData, fmt.Sprintf("host_connection_state-%s::%s", formattedIP, data.HostConnectionState))

	return keyData
}

// compareHostData compares old and new host data and returns changes
func compareHostData(formattedIP string, oldData, newData *MonitoringData) []string {
	var changes []string

	// Status changes (online/offline)
	if oldData.Status != newData.Status {
		changes = append(changes, fmt.Sprintf("status-%s::%s", formattedIP, newData.Status))
	}

	// CPU changes
	if oldData.CPUUsedMHz != newData.CPUUsedMHz {
		changes = append(changes, fmt.Sprintf("cpu_used_mhz-%s::%d", formattedIP, newData.CPUUsedMHz))
	}
	if fmt.Sprintf("%.2f", oldData.CPUUsagePercent) != fmt.Sprintf("%.2f", newData.CPUUsagePercent) {
		changes = append(changes, fmt.Sprintf("cpu_usage_percent-%s::%.2f", formattedIP, newData.CPUUsagePercent))
	}

	// Memory changes
	if oldData.MemoryUsedBytes != newData.MemoryUsedBytes {
		changes = append(changes, fmt.Sprintf("memory_used_bytes-%s::%d", formattedIP, newData.MemoryUsedBytes))
	}
	if oldData.MemoryUsedMB != newData.MemoryUsedMB {
		changes = append(changes, fmt.Sprintf("memory_used_mb-%s::%d", formattedIP, newData.MemoryUsedMB))
	}
	if fmt.Sprintf("%.2f", oldData.MemoryUsagePercent) != fmt.Sprintf("%.2f", newData.MemoryUsagePercent) {
		changes = append(changes, fmt.Sprintf("memory_usage_percent-%s::%.2f", formattedIP, newData.MemoryUsagePercent))
	}

	// Storage changes
	if fmt.Sprintf("%.2f", oldData.StorageTotalUsagePercent) != fmt.Sprintf("%.2f", newData.StorageTotalUsagePercent) {
		changes = append(changes, fmt.Sprintf("storage_total_usage_percent-%s::%.2f", formattedIP, newData.StorageTotalUsagePercent))
	}
	if oldData.StorageTotalUsed != newData.StorageTotalUsed {
		changes = append(changes, fmt.Sprintf("storage_total_used-%s::%d", formattedIP, newData.StorageTotalUsed))
	}

	// Host connection state changes
	if oldData.HostConnectionState != newData.HostConnectionState {
		changes = append(changes, fmt.Sprintf("host_connection_state-%s::%s", formattedIP, newData.HostConnectionState))
	}
	if oldData.HostOverallStatus != newData.HostOverallStatus {
		changes = append(changes, fmt.Sprintf("host_overall_status-%s::%s", formattedIP, newData.HostOverallStatus))
	}

	// VM count changes
	oldVMCount := len(oldData.VMsInfo)
	newVMCount := len(newData.VMsInfo)
	if oldVMCount != newVMCount {
		changes = append(changes, fmt.Sprintf("vm_count-%s::%d", formattedIP, newVMCount))
	}

	// VM power state changes
	vmChanges := compareVMChanges(formattedIP, oldData.VMsInfo, newData.VMsInfo)
	changes = append(changes, vmChanges...)

	return changes
}

// compareVMChanges compares VM data and returns changes
func compareVMChanges(formattedIP string, oldVMs, newVMs []VMInfo) []string {
	var changes []string

	// Create maps for easier comparison
	oldVMMap := make(map[string]VMInfo)
	for _, vm := range oldVMs {
		oldVMMap[vm.UUID] = vm
	}

	newVMMap := make(map[string]VMInfo)
	for _, vm := range newVMs {
		newVMMap[vm.UUID] = vm
	}

	// Check for VM power state changes
	poweredOnCount := 0
	for uuid, newVM := range newVMMap {
		if newVM.PowerState == "poweredOn" {
			poweredOnCount++
		}

		oldVM, exists := oldVMMap[uuid]
		if !exists {
			// New VM
			changes = append(changes, fmt.Sprintf("vm_new-%s::%s_%s", formattedIP, newVM.Name, newVM.PowerState))
			continue
		}

		// Check power state changes
		if oldVM.PowerState != newVM.PowerState {
			changes = append(changes, fmt.Sprintf("vm_power_state-%s::%s_%s", formattedIP, newVM.Name, newVM.PowerState))
		}

		// Check significant CPU usage changes (threshold: 100 MHz)
		if abs(int(oldVM.CPUUsageMHz)-int(newVM.CPUUsageMHz)) > 100 {
			changes = append(changes, fmt.Sprintf("vm_cpu_usage-%s::%s_%d", formattedIP, newVM.Name, newVM.CPUUsageMHz))
		}

		// Check significant memory usage changes (threshold: 100 MB)
		if abs(int(oldVM.MemoryUsageMB)-int(newVM.MemoryUsageMB)) > 100 {
			changes = append(changes, fmt.Sprintf("vm_memory_usage-%s::%s_%d", formattedIP, newVM.Name, newVM.MemoryUsageMB))
		}
	}

	// Check for removed VMs
	for uuid, oldVM := range oldVMMap {
		if _, exists := newVMMap[uuid]; !exists {
			changes = append(changes, fmt.Sprintf("vm_removed-%s::%s", formattedIP, oldVM.Name))
		}
	}

	// Add powered on VM count
	changes = append(changes, fmt.Sprintf("vm_powered_on_count-%s::%d", formattedIP, poweredOnCount))

	return changes
}

// abs returns absolute value of integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// FormatChangesForBroadcast formats changes into broadcast message format
func FormatChangesForBroadcast(changes []string) string {
	if len(changes) == 0 {
		return ""
	}
	return fmt.Sprintf("esxi:%s", strings.Join(changes, ";;"))
}
