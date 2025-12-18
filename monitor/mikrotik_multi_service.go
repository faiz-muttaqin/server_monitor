package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"server_monitor/cache"
	"server_monitor/model"

	"github.com/go-routeros/routeros/v3"
)

var MikrotikMultiService *MultiService

// DeviceMonitor represents a single device monitor
type DeviceMonitor struct {
	config          *model.MikroTikConfig
	client          *routeros.Client
	prevNATs        map[string]model.NatRule
	currentNATs     map[string]model.NatRule
	bandwidthData   map[string]*model.BandwidthUsage
	connectionStats *model.ConnectionStats
	deviceName      string
	isRunning       bool
	lastUpdate      time.Time
	parent          *MultiService // Reference to parent MultiService for cache operations
	mu              sync.RWMutex
}

// MultiService manages multiple MikroTik device monitors
type MultiService struct {
	configs        []*model.MikroTikConfig
	monitors       map[string]*DeviceMonitor
	dailySummary   map[string]map[string]*model.DailyBandwidthSummary   // deviceID -> date -> summary
	monthlySummary map[string]map[string]*model.MonthlyBandwidthSummary // deviceID -> month -> summary
	cacheManager   *cache.Manager
	startTime      time.Time
	mu             sync.RWMutex
}

func NewMultiService(configs []*model.MikroTikConfig) *MultiService {
	return &MultiService{
		configs:        configs,
		monitors:       make(map[string]*DeviceMonitor),
		dailySummary:   make(map[string]map[string]*model.DailyBandwidthSummary),
		monthlySummary: make(map[string]map[string]*model.MonthlyBandwidthSummary),
		cacheManager:   cache.NewManager(),
		startTime:      time.Now(),
	}
}

func (ms *MultiService) ConnectAll(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, config := range ms.configs {
		monitor := &DeviceMonitor{
			config:        config,
			prevNATs:      make(map[string]model.NatRule),
			currentNATs:   make(map[string]model.NatRule),
			bandwidthData: make(map[string]*model.BandwidthUsage),
			parent:        ms, // Add reference to parent MultiService
		}

		if err := monitor.connect(ctx); err != nil {
			log.Printf("Failed to connect to device %s (%s): %v", config.Name, config.Host, err)
			continue
		}

		ms.monitors[config.ID] = monitor

		// Initialize summaries for this device
		ms.dailySummary[config.ID] = make(map[string]*model.DailyBandwidthSummary)
		ms.monthlySummary[config.ID] = make(map[string]*model.MonthlyBandwidthSummary)
	}

	if len(ms.monitors) == 0 {
		return fmt.Errorf("failed to connect to any MikroTik devices")
	}

	// Load cached data
	if err := ms.loadCache(); err != nil {
		log.Printf("Warning: Failed to load cache: %v", err)
	}

	log.Printf("Successfully connected to %d out of %d MikroTik devices", len(ms.monitors), len(ms.configs))
	return nil
}

func (dm *DeviceMonitor) connect(ctx context.Context) error {
	fmt.Printf("Connecting to %s (%s) as %s...\n", dm.config.Name, dm.config.Host, dm.config.User)

	// Check TCP reachability
	conn, err := net.DialTimeout("tcp", dm.config.Host, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP connect failed: %v", err)
	}
	conn.Close()

	// Try API connection with TLS fallback
	client, err := routeros.DialContext(ctx, dm.config.Host, dm.config.User, dm.config.Pass)
	if err != nil {
		fmt.Printf("Standard API failed for %s: %v\nTrying TLS...\n", dm.config.Name, err)
		client, err = routeros.DialTLSContext(ctx, dm.config.Host, dm.config.User, dm.config.Pass, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return fmt.Errorf("TLS connection failed: %v", err)
		}
		fmt.Printf("Connected to %s via TLS\n", dm.config.Name)
	} else {
		fmt.Printf("Connected to %s successfully\n", dm.config.Name)
	}

	// Test API and get device name
	reply, err := client.RunArgs([]string{"/system/identity/print"})
	if err != nil {
		client.Close()
		return fmt.Errorf("API test failed: %v", err)
	}
	defer reply.Done.String()

	if len(reply.Re) > 0 {
		if name, ok := reply.Re[0].Map["name"]; ok {
			dm.deviceName = name
			fmt.Printf("✓ Connected to device: %s (%s)\n", name, dm.config.Name)
		}
	}

	dm.client = client
	return nil
}

func (ms *MultiService) Close() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Save cache before closing
	if err := ms.saveCache(); err != nil {
		log.Printf("Warning: Failed to save cache: %v", err)
	}

	// Close all device connections
	for _, monitor := range ms.monitors {
		monitor.mu.Lock()
		monitor.isRunning = false
		if monitor.client != nil {
			monitor.client.Close()
		}
		monitor.mu.Unlock()
	}
}

func (ms *MultiService) loadCache() error {
	// Load cache implementation will be updated for multi-device support
	// For now, return nil
	return nil
}

func (ms *MultiService) saveCache() error {
	fmt.Printf("[DEBUG] saveCache starting - collecting data from monitors\n")

	// Step 1: Collect device IDs without holding main lock
	ms.mu.RLock()
	deviceIDs := make([]string, 0, len(ms.monitors))
	for deviceID := range ms.monitors {
		deviceIDs = append(deviceIDs, deviceID)
	}
	ms.mu.RUnlock()

	// Step 2: Collect NAT rules from each device separately (no nested locks)
	allNATs := make(map[string]map[string]model.NatRule)
	totalRules := 0

	for _, deviceID := range deviceIDs {
		// Get monitor reference safely
		ms.mu.RLock()
		monitor, exists := ms.monitors[deviceID]
		ms.mu.RUnlock()

		if !exists {
			continue
		}

		// Lock only this device monitor
		monitor.mu.RLock()
		deviceRules := make(map[string]model.NatRule)
		for id, rule := range monitor.currentNATs {
			deviceRules[id] = rule
		}
		monitor.mu.RUnlock()

		allNATs[deviceID] = deviceRules
		totalRules += len(deviceRules)
		fmt.Printf("[DEBUG] Device %s: collected %d NAT rules\n", deviceID, len(deviceRules))
	}

	fmt.Printf("[DEBUG] Total devices: %d, Total NAT rules: %d\n", len(allNATs), totalRules)

	if totalRules == 0 {
		fmt.Printf("[DEBUG] No NAT rules to save, skipping SaveMulti\n")
		return nil
	}

	fmt.Printf("[DEBUG] Calling SaveMulti with collected data\n")
	return ms.cacheManager.SaveMulti(allNATs)
}

func (dm *DeviceMonitor) fetchNAT(ctx context.Context) ([]model.NatRule, error) {
	reply, err := dm.client.RunArgs([]string{"/ip/firewall/nat/print"})
	if err != nil {
		return nil, err
	}
	defer reply.Done.String()

	var rules []model.NatRule
	ts := time.Now()

	for _, r := range reply.Re {
		rules = append(rules, model.NatRule{
			ID:         r.Map[".id"],
			DeviceID:   dm.config.ID,
			Chain:      r.Map["chain"],
			DstAddress: r.Map["dst-address"],
			DstPort:    r.Map["dst-port"],
			ToAddress:  r.Map["to-addresses"],
			ToPort:     r.Map["to-ports"],
			Action:     r.Map["action"],
			Comment:    r.Map["comment"],
			Timestamp:  ts,
		})
	}
	return rules, nil
}

func (dm *DeviceMonitor) fetchConnectionStats(ctx context.Context) (*model.ConnectionStats, error) {
	reply, err := dm.client.RunArgs([]string{"/ip/firewall/connection/print", "count-only"})
	if err != nil {
		reply, err = dm.client.RunArgs([]string{"/ip/firewall/connection/print"})
		if err != nil {
			return nil, err
		}
	}
	defer reply.Done.String()

	stats := &model.ConnectionStats{
		DeviceID:         dm.config.ID,
		TotalConnections: len(reply.Re),
		Timestamp:        time.Now(),
	}
	return stats, nil
}

func (dm *DeviceMonitor) monitorNATChanges(ctx context.Context) error {
	nats, err := dm.fetchNAT(ctx)
	if err != nil {
		return fmt.Errorf("fetch NAT error: %v", err)
	}

	var sortedNewRules []model.NatRule
	current := make(map[string]model.NatRule)
	var needsCacheUpdate bool

	// Process data outside of locks first
	for _, n := range nats {
		current[n.ID] = n
	}

	// Critical section: update device state
	dm.mu.Lock()

	// Check for new NAT rules
	for _, n := range nats {
		if _, exists := dm.prevNATs[n.ID]; !exists {
			sortedNewRules = append(sortedNewRules, n)
		}
	}

	// Check for removed NAT rules
	var removedRules []model.NatRule
	for id, rule := range dm.prevNATs {
		if _, exists := current[id]; !exists {
			removedRules = append(removedRules, rule)
		}
	}

	// Update state
	dm.prevNATs = current
	dm.currentNATs = current
	dm.lastUpdate = time.Now()

	// Determine if cache update is needed
	needsCacheUpdate = len(sortedNewRules) > 0 || len(removedRules) > 0

	fmt.Printf("[DEBUG] After update - device %s now has %d NAT rules in currentNATs\n",
		dm.config.Name, len(dm.currentNATs))

	dm.mu.Unlock()
	// End critical section

	// Log changes (outside of locks)
	if len(sortedNewRules) > 0 {
		// Sort new rules by port
		sort.Slice(sortedNewRules, func(i, j int) bool {
			port1, err1 := strconv.Atoi(sortedNewRules[i].DstPort)
			if err1 != nil {
				port1 = 0
			}
			port2, err2 := strconv.Atoi(sortedNewRules[j].DstPort)
			if err2 != nil {
				port2 = 1
			}
			return port1 < port2
		})

		for _, n := range sortedNewRules {
			log.Printf("[%s] [NAT ADDED] %s:%s → %s:%s (%s)",
				dm.config.Name, n.DstAddress, n.DstPort, n.ToAddress, n.ToPort, n.Comment)
		}
	}

	for _, rule := range removedRules {
		log.Printf("[%s] [NAT REMOVED] %s (%s)", dm.config.Name, rule.ID, rule.Comment)
	}

	// Save cache after releasing all locks to avoid deadlock
	if needsCacheUpdate && dm.parent != nil {
		fmt.Printf("[DEBUG] Calling saveCache for device: %s\n", dm.config.Name)
		if err := dm.parent.saveCache(); err != nil {
			log.Printf("[%s] Warning: Failed to save cache: %v", dm.config.Name, err)
		} else {
			fmt.Printf("[DEBUG] saveCache completed successfully for device: %s\n", dm.config.Name)
		}
	}

	return nil
}

func (ms *MultiService) Start(ctx context.Context, interval time.Duration) {
	ms.mu.Lock()
	for _, monitor := range ms.monitors {
		monitor.mu.Lock()
		monitor.isRunning = true
		monitor.mu.Unlock()
	}
	ms.mu.Unlock()

	log.Printf("Starting multi-device MikroTik monitoring every %v", interval)

	// Start monitoring each device in its own goroutine
	var wg sync.WaitGroup
	for _, monitor := range ms.monitors {
		wg.Add(1)
		go func(dm *DeviceMonitor) {
			defer wg.Done()
			ms.monitorDevice(ctx, dm, interval)
		}(monitor)
	}

	wg.Wait()
	log.Println("All device monitors stopped")
}

func (ms *MultiService) monitorDevice(ctx context.Context, dm *DeviceMonitor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting monitoring for device: %s", dm.config.Name)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping monitor for device: %s", dm.config.Name)
			return
		case <-ticker.C:
			if err := dm.monitorNATChanges(ctx); err != nil {
				log.Printf("[%s] NAT monitoring error: %v", dm.config.Name, err)
			}

			// Fetch connection stats
			stats, err := dm.fetchConnectionStats(ctx)
			if err != nil {
				log.Printf("[%s] Connection stats error: %v", dm.config.Name, err)
			} else {
				dm.mu.Lock()
				dm.connectionStats = stats
				dm.mu.Unlock()
				log.Printf("[%s] Active connections: %d", dm.config.Name, stats.TotalConnections)
			}
		}
	}
}

// Public methods for API access
func (ms *MultiService) GetDeviceConfigs() []*model.MikroTikConfig {
	return ms.configs
}

func (ms *MultiService) GetCurrentNATRules(deviceID string) []model.NatRule {
	// Get monitor reference safely
	ms.mu.RLock()
	monitor, exists := ms.monitors[deviceID]
	ms.mu.RUnlock()

	if !exists {
		return []model.NatRule{}
	}

	// Lock only this device monitor
	monitor.mu.RLock()
	rules := make([]model.NatRule, 0, len(monitor.currentNATs))
	for _, rule := range monitor.currentNATs {
		rules = append(rules, rule)
	}
	monitor.mu.RUnlock()

	// Sort by port (no locks needed for local slice)
	sort.Slice(rules, func(i, j int) bool {
		port1, err1 := strconv.Atoi(rules[i].DstPort)
		if err1 != nil {
			port1 = 0
		}
		port2, err2 := strconv.Atoi(rules[j].DstPort)
		if err2 != nil {
			port2 = 1
		}
		return port1 < port2
	})

	return rules
}

func (ms *MultiService) GetAllNATRules() map[string][]model.NatRule {
	// Get device IDs safely
	ms.mu.RLock()
	deviceIDs := make([]string, 0, len(ms.monitors))
	for deviceID := range ms.monitors {
		deviceIDs = append(deviceIDs, deviceID)
	}
	ms.mu.RUnlock()

	// Collect rules from each device separately
	result := make(map[string][]model.NatRule)
	for _, deviceID := range deviceIDs {
		result[deviceID] = ms.GetCurrentNATRules(deviceID)
	}
	return result
}

func (ms *MultiService) GetDeviceStatus(deviceID string) *model.MonitorStatus {
	// Get monitor reference safely
	ms.mu.RLock()
	monitor, exists := ms.monitors[deviceID]
	ms.mu.RUnlock()

	if !exists {
		return nil
	}

	uptime := time.Since(ms.startTime).String()

	// Get device-specific data with minimal lock
	monitor.mu.RLock()
	isRunning := monitor.isRunning
	deviceName := monitor.deviceName
	lastUpdate := monitor.lastUpdate
	totalNatRules := len(monitor.currentNATs)
	activeConnections := 0
	if monitor.connectionStats != nil {
		activeConnections = monitor.connectionStats.TotalConnections
	}
	monitor.mu.RUnlock()

	return &model.MonitorStatus{
		DeviceID:          deviceID,
		IsRunning:         isRunning,
		ConnectedDevice:   deviceName,
		LastUpdate:        lastUpdate,
		TotalNatRules:     totalNatRules,
		ActiveConnections: activeConnections,
		Uptime:            uptime,
	}
}

func (ms *MultiService) GetAllDevicesStatus() *model.MultiDeviceStatus {
	// Get device IDs safely
	ms.mu.RLock()
	deviceIDs := make([]string, 0, len(ms.monitors))
	for deviceID := range ms.monitors {
		deviceIDs = append(deviceIDs, deviceID)
	}
	totalDevices := len(ms.monitors)
	ms.mu.RUnlock()

	// Collect status from each device separately (no nested locks)
	devices := make(map[string]*model.MonitorStatus)
	for _, deviceID := range deviceIDs {
		devices[deviceID] = ms.GetDeviceStatus(deviceID)
	}

	return &model.MultiDeviceStatus{
		Devices:      devices,
		TotalDevices: totalDevices,
		Timestamp:    time.Now(),
	}
}
