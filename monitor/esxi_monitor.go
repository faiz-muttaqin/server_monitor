package monitor

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"server_monitor/model/modelESXI"
	"server_monitor/ws"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Global storage untuk data monitoring
var (
	MultiHostData    modelESXI.MultiHostData
	previousHostData modelESXI.MultiHostData
	dataFilePath     string
	dataMutex        sync.RWMutex
)

// setHostStatusOffline sets host status to offline and broadcasts if changed
func setHostStatusOffline(hostIP string, err error) {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	// Initialize if needed
	if MultiHostData == nil {
		MultiHostData = make(modelESXI.MultiHostData)
	}
	if previousHostData == nil {
		previousHostData = make(modelESXI.MultiHostData)
	}

	// Get current data or create new
	var currentData *modelESXI.MonitoringData
	if existing := MultiHostData[hostIP]; existing != nil {
		currentData = existing
	} else {
		currentData = modelESXI.NewMonitoringData()
		currentData.ID = hostIP
	}

	// Store previous status for comparison
	previousStatus := currentData.Status

	// Set status to offline
	currentData.Status = "offline"
	currentData.Timestamp = time.Now()

	// Update storage
	MultiHostData[hostIP] = currentData

	// Broadcast if status changed
	if previousStatus != "offline" {
		log.Printf("🔴 Host %s changed status: %s -> offline (error: %v)", hostIP, previousStatus, err)
		broadcastStatusChange(hostIP, "offline", previousStatus)
	}
}

// setHostStatusOnline sets host status to online and broadcasts if changed
func setHostStatusOnline(hostIP string, data *modelESXI.MonitoringData) {
	// Store previous status for comparison
	previousStatus := ""
	if previousData := MultiHostData[hostIP]; previousData != nil {
		previousStatus = previousData.Status
	}

	// Set status to online
	data.Status = "online"

	// Broadcast if status changed
	if previousStatus != "online" {
		log.Printf("🟢 Host %s changed status: %s -> online", hostIP, previousStatus)
		broadcastStatusChange(hostIP, "online", previousStatus)
	}
}

// broadcastStatusChange sends WebSocket broadcast when status changes
func broadcastStatusChange(hostIP, newStatus, oldStatus string) {
	// Format IP for broadcast (192.168.1.1 -> 192_168_1_1)
	formattedIP := strings.ReplaceAll(hostIP, ".", "_")

	// Create broadcast message in required format
	message := fmt.Sprintf("esxi:status-%s::%s", formattedIP, newStatus)

	// Send broadcast (1 = websocket.TextMessage)
	ws.BroadcastMessage(1, message)
	logrus.Printf("🔄 Broadcasting status change for %s: %s -> %s\n", hostIP, oldStatus, newStatus)
} // MonitorService handles periodic monitoring
type MonitorESXIService struct {
	cfg                *ESXIConfig
	ctx                context.Context
	cancel             context.CancelFunc
	configReloadTicker *time.Ticker
}

func NewESXiMonitoring() *MonitorESXIService {
	ctx, cancel := context.WithCancel(context.Background())
	// Implementation of ESXi monitoring logic goes here
	// Load configuration
	config, err := LoadESXIConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dataFilePath = config.DataFilePath

	// Load existing data from file
	logrus.Println("📂 Loading existing monitoring data...")
	MultiHostData, err = modelESXI.LoadMultiHostDataFromFile(dataFilePath)
	if err != nil {
		log.Printf("⚠️ Warning: Could not load existing data: %v", err)
		MultiHostData = modelESXI.NewMultiHostData()
	} else {
		logrus.Printf("✅ Loaded data for %d hosts from: %s\n", MultiHostData.GetHostCount(), dataFilePath)
	}

	// Initialize previous data storage for change detection
	previousHostData = modelESXI.NewMultiHostData()

	logrus.Printf("📊 Monitoring interval: %v\n", config.PollInterval)
	logrus.Printf("🎯 Monitoring %d ESXi hosts\n", len(config.ESXiHosts))
	for i, host := range config.ESXiHosts {
		logrus.Printf("   %d. %s (%s)\n", i+1, host.Host, host.User)
	}
	logrus.Println("🚀 Starting VMware ESXi monitoring...")
	logrus.Println()

	return &MonitorESXIService{
		cfg:    config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// NewESXiMonitoringWithRetry creates monitoring service with retry mechanism
func NewESXiMonitoringWithRetry() *MonitorESXIService {
	ctx, cancel := context.WithCancel(context.Background())

	// Wait for valid configuration with retry
	config := WaitForValidConfig()

	dataFilePath = config.DataFilePath

	// Load existing data from file
	logrus.Println("📂 Loading existing monitoring data...")
	var err error
	MultiHostData, err = modelESXI.LoadMultiHostDataFromFile(dataFilePath)
	if err != nil {
		log.Printf("⚠️ Warning: Could not load existing data: %v", err)
		MultiHostData = make(modelESXI.MultiHostData)
	} else {
		logrus.Printf("✅ Loaded data for %d hosts from: %s\n", MultiHostData.GetHostCount(), dataFilePath)
	}

	// Initialize previous data storage for change detection
	previousHostData = make(modelESXI.MultiHostData)

	logrus.Printf("📊 Monitoring interval: %v\n", config.PollInterval)
	logrus.Println("🚀 Starting VMware ESXi monitoring with auto-retry...")
	logrus.Println()

	// Create config reload ticker (check every minute)
	configReloadTicker := time.NewTicker(1 * time.Minute)

	return &MonitorESXIService{
		cfg:                config,
		ctx:                ctx,
		cancel:             cancel,
		configReloadTicker: configReloadTicker,
	}
}

func (ms *MonitorESXIService) Start() {
	// Start config hot-reload monitoring if ticker is available
	if ms.configReloadTicker != nil {
		go ms.configReloadLoop()
	}

	// Start activity monitoring in background with restart capability
	go ms.monitoringLoop()
}

// monitoringLoop handles monitoring with restart capability
func (ms *MonitorESXIService) monitoringLoop() {
	for {
		select {
		case <-ms.ctx.Done():
			return
		default:
			// Start monitoring with current config
			if err := monitorMultipleESXIHosts(ms.ctx, ms.cfg); err != nil {
				if ms.ctx.Err() != nil {
					// Context was cancelled, exit gracefully
					return
				}
				log.Printf("❌ Monitor error: %v", err)
				log.Println("⏳ Waiting 1 minute before restart...")

				// Wait before restart, but allow for context cancellation
				select {
				case <-ms.ctx.Done():
					return
				case <-time.After(1 * time.Minute):
					log.Println("🔄 Restarting monitoring...")
					continue
				}
			}
		}
	}
} // configReloadLoop handles hot reloading of configuration
func (ms *MonitorESXIService) configReloadLoop() {
	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ms.configReloadTicker.C:
			newConfig := WaitForValidConfigUpdate(ms.cfg)
			if newConfig != ms.cfg {
				log.Println("🔄 Configuration changed, updating monitoring service...")
				ms.cfg = newConfig
				dataFilePath = newConfig.DataFilePath
			}
		}
	}
}

// Stop stops the monitoring service
func (ms *MonitorESXIService) Stop() {
	log.Println("Stopping monitoring service...")
	if ms.configReloadTicker != nil {
		ms.configReloadTicker.Stop()
	}
	if ms.ctx != nil {
		ms.cancel()
	}
}

func newClient(ctx context.Context, rawurl string, insecure bool) (*govmomi.Client, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return govmomi.NewClient(ctx, u, insecure)
}

func formatBytesEsxi(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

type ResourceUsage struct {
	Used  int64
	Total int64
}

func getVMDetails(ctx context.Context, c *govmomi.Client, vmRefs []types.ManagedObjectReference) ([]VMInfo, error) {
	if len(vmRefs) == 0 {
		return nil, nil
	}

	pc := property.DefaultCollector(c.Client)
	var vms []mo.VirtualMachine

	vmProps := []string{
		"summary.config.name",
		"summary.config.uuid",
		"summary.runtime.powerState",
		"summary.quickStats",
		"guest.net",
		"guest.toolsStatus",
		"config.hardware",
	}

	err := pc.Retrieve(ctx, vmRefs, vmProps, &vms)
	if err != nil {
		return nil, err
	}

	var vmInfos []VMInfo
	for _, vm := range vms {
		vmInfo := VMInfo{
			Name:       vm.Summary.Config.Name,
			UUID:       vm.Summary.Config.Uuid,
			PowerState: string(vm.Summary.Runtime.PowerState),
			IPs:        []string{},
		}

		// Get CPU and Memory info
		if vm.Summary.QuickStats.OverallCpuUsage > 0 || vm.Summary.QuickStats.GuestMemoryUsage > 0 {
			vmInfo.CPUUsageMHz = vm.Summary.QuickStats.OverallCpuUsage
			vmInfo.MemoryUsageMB = vm.Summary.QuickStats.GuestMemoryUsage
		}

		if vm.Config != nil && vm.Config.Hardware.NumCPU > 0 {
			vmInfo.CPUCores = vm.Config.Hardware.NumCPU
			vmInfo.MemoryMB = vm.Config.Hardware.MemoryMB
		}

		// Get IP addresses
		if vm.Guest != nil && vm.Guest.Net != nil {
			for _, nic := range vm.Guest.Net {
				if nic.IpConfig != nil && nic.IpConfig.IpAddress != nil {
					for _, ip := range nic.IpConfig.IpAddress {
						if ip.IpAddress != "" && ip.IpAddress != "127.0.0.1" && !strings.HasPrefix(ip.IpAddress, "169.254") {
							vmInfo.IPs = append(vmInfo.IPs, ip.IpAddress)
						}
					}
				}
			}
		}

		vmInfos = append(vmInfos, vmInfo)
	}

	return vmInfos, nil
}

type VMInfo struct {
	Name          string
	UUID          string
	PowerState    string
	IPs           []string
	CPUCores      int32
	CPUUsageMHz   int32
	MemoryMB      int32
	MemoryUsageMB int32
}

// convertVMInfoToInput mengkonversi VMInfo ke modelESXI.VMInfoInput
func convertVMInfoToInput(vmInfos []VMInfo) []modelESXI.VMInfoInput {
	var converted []modelESXI.VMInfoInput
	for _, vm := range vmInfos {
		vmInput := modelESXI.VMInfoInput{
			Name:          vm.Name,
			UUID:          vm.UUID,
			PowerState:    vm.PowerState,
			IPs:           vm.IPs,
			CPUCores:      vm.CPUCores,
			CPUUsageMHz:   vm.CPUUsageMHz,
			MemoryMB:      vm.MemoryMB,
			MemoryUsageMB: vm.MemoryUsageMB,
		}
		converted = append(converted, vmInput)
	}
	return converted
}

// monitorSingleHost memonitor satu ESXi host
func monitorSingleHost(ctx context.Context, hostIP string, c *govmomi.Client, interval time.Duration) error {
	// Create a view of HostSystem objects
	m := view.NewManager(c.Client)
	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return err
	}
	defer v.Destroy(ctx)

	var hosts []mo.HostSystem
	pc := property.DefaultCollector(c.Client)

	// fields we want to retrieve
	props := []string{
		"summary",
		"runtime.bootTime",
		"hardware",
		"datastore",
		"vm",
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Retrieve host systems (refresh)
			hosts = nil
			err = v.Retrieve(ctx, []string{"HostSystem"}, props, &hosts)
			if err != nil {
				log.Printf("❌ Monitor error for %s: %v", hostIP, err)
				// Set status offline jika tidak bisa mengambil data
				setHostStatusOffline(hostIP, err)
				continue
			}

			for _, h := range hosts {
				logrus.Println("=======================================")
				logrus.Printf("🖥️  ESXi Host: %s\n", h.Summary.Config.Name)
				logrus.Printf("📊 Monitoring at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
				logrus.Println("=======================================")

				// CPU Information and Usage
				logrus.Println("\n🔧 CPU INFORMATION:")
				logrus.Printf("   Model: %s\n", h.Summary.Hardware.CpuModel)
				logrus.Printf("   Cores: %d\n", h.Summary.Hardware.NumCpuCores)
				logrus.Printf("   Threads: %d\n", h.Summary.Hardware.NumCpuThreads)
				logrus.Printf("   Speed: %d MHz\n", h.Summary.Hardware.CpuMhz)

				// CPU Usage
				totalMhz := int64(h.Summary.Hardware.CpuMhz) * int64(h.Summary.Hardware.NumCpuThreads)
				usedMhz := int64(h.Summary.QuickStats.OverallCpuUsage)
				cpuPercent := float64(usedMhz) / float64(totalMhz) * 100

				logrus.Printf("\n📈 CPU USAGE:\n")
				logrus.Printf("   Used: %d MHz (%.2f%%)\n", usedMhz, cpuPercent)
				logrus.Printf("   Total: %d MHz\n", totalMhz)

				// Memory Information and Usage
				logrus.Printf("\n💾 MEMORY USAGE:\n")
				totalMemoryBytes := h.Summary.Hardware.MemorySize
				usedMemoryMB := int64(h.Summary.QuickStats.OverallMemoryUsage)
				totalMemoryMB := totalMemoryBytes / (1024 * 1024)
				freeMemoryMB := totalMemoryMB - usedMemoryMB
				memPercent := float64(usedMemoryMB) / float64(totalMemoryMB) * 100

				logrus.Printf("   Used: %s (%.2f%%)\n", formatBytesEsxi(usedMemoryMB*1024*1024), memPercent)
				logrus.Printf("   Free: %s\n", formatBytesEsxi(freeMemoryMB*1024*1024))
				logrus.Printf("   Total: %s\n", formatBytesEsxi(totalMemoryBytes))

				// Datastore (HDD/Storage) Usage
				logrus.Printf("\n💿 STORAGE USAGE:\n")
				if len(h.Datastore) > 0 {
					var dsObjs []mo.Datastore
					err = pc.Retrieve(ctx, h.Datastore, []string{"summary"}, &dsObjs)
					if err != nil {
						logrus.Printf("   ❌ Failed to retrieve datastore info: %v\n", err)
					} else {
						var totalCapacity, totalUsed, totalFree int64

						for _, ds := range dsObjs {
							capacity := ds.Summary.Capacity
							free := ds.Summary.FreeSpace
							used := capacity - free
							usedPercent := float64(used) / float64(capacity) * 100

							logrus.Printf("   📁 %s:\n", ds.Summary.Name)
							logrus.Printf("      Used: %s (%.2f%%)\n", formatBytesEsxi(used), usedPercent)
							logrus.Printf("      Free: %s\n", formatBytesEsxi(free))
							logrus.Printf("      Total: %s\n", formatBytesEsxi(capacity))

							totalCapacity += capacity
							totalUsed += used
							totalFree += free
						}

						if len(dsObjs) > 1 {
							totalUsedPercent := float64(totalUsed) / float64(totalCapacity) * 100
							logrus.Printf("\n   📊 TOTAL STORAGE:\n")
							logrus.Printf("      Used: %s (%.2f%%)\n", formatBytesEsxi(totalUsed), totalUsedPercent)
							logrus.Printf("      Free: %s\n", formatBytesEsxi(totalFree))
							logrus.Printf("      Total: %s\n", formatBytesEsxi(totalCapacity))
						}
					}
				} else {
					logrus.Println("   ❌ No datastores found")
				}

				// Virtual Machines and their details
				logrus.Printf("\n🖼️  VIRTUAL MACHINES (%d total):\n", len(h.Vm))
				if len(h.Vm) > 0 {
					vmInfos, err := getVMDetails(ctx, c, h.Vm)
					if err != nil {
						logrus.Printf("   ❌ Failed to retrieve VM details: %v\n", err)
					} else {
						poweredOnCount := 0
						for i, vm := range vmInfos {
							if vm.PowerState == "poweredOn" {
								poweredOnCount++
							}

							logrus.Printf("   %d. 🖥️  %s\n", i+1, vm.Name)
							logrus.Printf("      State: %s\n", vm.PowerState)

							if len(vm.IPs) > 0 {
								logrus.Printf("      IPs: %s\n", strings.Join(vm.IPs, ", "))
							} else {
								logrus.Printf("      IPs: No IP addresses found\n")
							}

							if vm.CPUCores > 0 {
								logrus.Printf("      CPU: %d cores", vm.CPUCores)
								if vm.CPUUsageMHz > 0 {
									logrus.Printf(" (using %d MHz)", vm.CPUUsageMHz)
								}
								logrus.Printf("\n")
							}

							if vm.MemoryMB > 0 {
								logrus.Printf("      Memory: %s", formatBytesEsxi(int64(vm.MemoryMB)*1024*1024))
								if vm.MemoryUsageMB > 0 {
									memUsagePercent := float64(vm.MemoryUsageMB) / float64(vm.MemoryMB) * 100
									logrus.Printf(" (using %s - %.1f%%)", formatBytesEsxi(int64(vm.MemoryUsageMB)*1024*1024), memUsagePercent)
								}
								logrus.Printf("\n")
							}
							logrus.Println()
						}
						logrus.Printf("   📊 Summary: %d powered on, %d total VMs\n", poweredOnCount, len(vmInfos))
					}
				}

				// System Information
				logrus.Printf("\n🔧 SYSTEM INFO:\n")
				if h.Runtime.BootTime != nil {
					uptime := time.Since(*h.Runtime.BootTime).Truncate(time.Second)
					logrus.Printf("   Uptime: %s\n", uptime)
					logrus.Printf("   Boot Time: %s\n", h.Runtime.BootTime.Format("2006-01-02 15:04:05"))
				}

				if h.Hardware != nil && h.Hardware.SystemInfo.Vendor != "" {
					logrus.Printf("   Vendor: %s\n", h.Hardware.SystemInfo.Vendor)
					logrus.Printf("   Model: %s\n", h.Hardware.SystemInfo.Model)
				}

				logrus.Printf("   Connection: %s\n", h.Summary.Runtime.ConnectionState)
				logrus.Printf("   Status: %s\n", h.Summary.OverallStatus)

				// Generate JSON output dan simpan ke memory
				if len(h.Vm) > 0 && len(h.Datastore) > 0 {
					// Get VM details
					vmInfos, err := getVMDetails(ctx, c, h.Vm)
					if err == nil {
						// Get datastore details
						var dsObjs []mo.Datastore
						err = pc.Retrieve(ctx, h.Datastore, []string{"summary"}, &dsObjs)
						if err == nil {
							// Convert dan simpan data
							vmInputs := convertVMInfoToInput(vmInfos)
							monitoringData := modelESXI.ConvertToMonitoringData(hostIP, h, vmInputs, dsObjs)

							// Simpan ke memory storage dan deteksi perubahan
							dataMutex.Lock()

							// Copy current data as previous data before updating
							if previousHostData == nil {
								previousHostData = make(modelESXI.MultiHostData)
							}

							// Deep copy current data to previous
							if currentData, exists := MultiHostData[hostIP]; exists && currentData != nil {
								previousData := *currentData
								previousHostData[hostIP] = &previousData
							}

							// Set status online since we successfully retrieved data
							setHostStatusOnline(hostIP, monitoringData)

							// Update with new data
							MultiHostData.UpdateHost(hostIP, monitoringData)

							// Detect changes and broadcast
							detectAndBroadcastChanges(hostIP, monitoringData)

							dataMutex.Unlock()

							// Generate JSON output untuk testing
							generateJSONOutput(hostIP, h, vmInfos, dsObjs)
						}
					}
				}

				logrus.Println("=======================================")
			}
		}
	}
}

// detectAndBroadcastChanges mendeteksi perubahan dan mengirim broadcast
func detectAndBroadcastChanges(hostIP string, newData *modelESXI.MonitoringData) {
	if previousHostData == nil || newData == nil {
		return
	}

	// Create temporary maps for comparison
	oldDataMap := modelESXI.MultiHostData{}
	newDataMap := modelESXI.MultiHostData{}

	// Add previous data if exists
	if prevData, exists := previousHostData[hostIP]; exists {
		oldDataMap[hostIP] = prevData
	}

	// Add new data
	newDataMap[hostIP] = newData

	// Get changes
	changes := modelESXI.CompareAndGetChanges(oldDataMap, newDataMap)

	// Broadcast changes if any
	if len(changes) > 0 {
		changeMessage := modelESXI.FormatChangesForBroadcast(changes)
		if changeMessage != "" {
			// Print changes for debugging
			logrus.Printf("🔄 Broadcasting %d changes for %s\n", len(changes), hostIP)

			// Broadcast via WebSocket
			ws.BroadcastMessage(1, changeMessage)
		}
	}
}

// generateJSONOutput membuat JSON output untuk testing
func generateJSONOutput(hostIP string, h mo.HostSystem, vmInfos []VMInfo, dsObjs []mo.Datastore) {
	logrus.Println("\n🔍 JSON OUTPUT (for testing):")
	logrus.Println("=======================================")

	// Convert VM info
	vmInputs := convertVMInfoToInput(vmInfos)

	// Create monitoring data model
	monitoringData := modelESXI.ConvertToMonitoringData(hostIP, h, vmInputs, dsObjs)

	// Generate JSON
	jsonOutput, err := monitoringData.ToJSON()
	if err != nil {
		logrus.Printf("❌ Error generating JSON: %v\n", err)
	} else {
		logrus.Println(jsonOutput)
	}

	logrus.Println("JSON Compact:")
	jsonCompact, err := monitoringData.ToJSONCompact()
	if err != nil {
		logrus.Printf("❌ Error generating compact JSON: %v\n", err)
	} else {
		logrus.Println(jsonCompact)
	}

	logrus.Println("=======================================")
}

// SaveESXIDataOnExit menyimpan data saat aplikasi akan keluar
func SaveESXIDataOnExit() {
	dataMutex.RLock()
	defer dataMutex.RUnlock()

	if len(MultiHostData) > 0 {
		logrus.Println("\n💾 Saving monitoring data...")
		if err := MultiHostData.SaveToFile(dataFilePath); err != nil {
			logrus.Printf("❌ Error saving data: %v\n", err)
		} else {
			logrus.Printf("✅ Data saved to: %s\n", dataFilePath)
		}
	}
}

// monitorMultipleESXIHosts memonitor multiple ESXi hosts
func monitorMultipleESXIHosts(ctx context.Context, config *ESXIConfig) error {
	var wg sync.WaitGroup

	for _, host := range config.ESXiHosts {
		wg.Add(1)
		go func(esxiHost ESXiHost) {
			defer wg.Done()

			// Create client for this host
			hostURL := config.GetESXiURL(esxiHost)
			client, err := newClient(ctx, hostURL, config.IsInsecure(esxiHost))
			if err != nil {
				log.Printf("❌ Failed to connect to ESXi %s: %v", esxiHost.Host, err)
				return
			}
			defer client.Logout(ctx)

			logrus.Printf("🔗 Connected to ESXi: %s\n", esxiHost.Host)

			// Start monitoring this host
			if err := monitorSingleHost(ctx, esxiHost.Host, client, config.PollInterval); err != nil {
				log.Printf("❌ Monitor error for %s: %v", esxiHost.Host, err)
			}
		}(host)
	}

	wg.Wait()
	return nil
}
