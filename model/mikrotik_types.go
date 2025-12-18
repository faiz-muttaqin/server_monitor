package model

import "time"

// MikroTikConfig represents a single MikroTik device configuration
type MikroTikConfig struct {
	ID   string `json:"id,omitempty"`   // Auto-generated unique ID
	Host string `json:"host"`           // IP:PORT
	User string `json:"user"`           // Username
	Pass string `json:"pass"`           // Password
	Name string `json:"name,omitempty"` // Optional friendly name
}

// NatRule represents a NAT rule from MikroTik
type NatRule struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"` // Which device this rule belongs to
	Chain      string    `json:"chain"`
	DstAddress string    `json:"dst_address"`
	DstPort    string    `json:"dst_port"`
	ToAddress  string    `json:"to_address"`
	ToPort     string    `json:"to_port"`
	Action     string    `json:"action"`
	Comment    string    `json:"comment"`
	Timestamp  time.Time `json:"timestamp"`
}

// ConnectionStats represents connection statistics
type ConnectionStats struct {
	DeviceID         string    `json:"device_id"`
	TotalConnections int       `json:"total_connections"`
	Timestamp        time.Time `json:"timestamp"`
}

// BandwidthUsage represents bandwidth usage per port
type BandwidthUsage struct {
	DeviceID   string    `json:"device_id"`
	DstPort    string    `json:"dst_port"`
	BytesIn    uint64    `json:"bytes_in"`
	BytesOut   uint64    `json:"bytes_out"`
	PacketsIn  uint64    `json:"packets_in"`
	PacketsOut uint64    `json:"packets_out"`
	Timestamp  time.Time `json:"timestamp"`
	Date       string    `json:"date"`  // YYYY-MM-DD format for daily aggregation
	Month      string    `json:"month"` // YYYY-MM format for monthly aggregation
}

// DailyBandwidthSummary represents daily summary of bandwidth usage
type DailyBandwidthSummary struct {
	Date       string                     `json:"date"`
	PortUsages map[string]*BandwidthUsage `json:"port_usages"`
	TotalIn    uint64                     `json:"total_in"`
	TotalOut   uint64                     `json:"total_out"`
}

// MonthlyBandwidthSummary represents monthly summary of bandwidth usage
type MonthlyBandwidthSummary struct {
	Month      string                     `json:"month"`
	PortUsages map[string]*BandwidthUsage `json:"port_usages"`
	TotalIn    uint64                     `json:"total_in"`
	TotalOut   uint64                     `json:"total_out"`
}

// Connection represents a single connection
type Connection struct {
	DeviceID   string    `json:"device_id"`
	SrcAddress string    `json:"src_address"`
	SrcPort    string    `json:"src_port"`
	DstAddress string    `json:"dst_address"`
	DstPort    string    `json:"dst_port"`
	Protocol   string    `json:"protocol"`
	State      string    `json:"state"`
	BytesIn    uint64    `json:"bytes_in"`
	BytesOut   uint64    `json:"bytes_out"`
	PacketsIn  uint64    `json:"packets_in"`
	PacketsOut uint64    `json:"packets_out"`
	Timestamp  time.Time `json:"timestamp"`
}

// CacheData represents cached NAT rules data
type CacheData struct {
	NatRules   map[string]map[string]NatRule `json:"nat_rules"` // deviceID -> ruleID -> NatRule
	LastUpdate time.Time                     `json:"last_update"`
	Version    string                        `json:"version"`
}

// NatRulesResponse represents direct NAT rules response format (without wrapper)
type NatRulesResponse map[string][]NatRule // deviceID -> slice of NatRule

// APIResponse represents standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// MonitorStatus represents monitoring service status
type MonitorStatus struct {
	DeviceID          string    `json:"device_id,omitempty"`
	IsRunning         bool      `json:"is_running"`
	ConnectedDevice   string    `json:"connected_device"`
	LastUpdate        time.Time `json:"last_update"`
	TotalNatRules     int       `json:"total_nat_rules"`
	ActiveConnections int       `json:"active_connections"`
	Uptime            string    `json:"uptime"`
}

// MultiDeviceStatus represents status for all devices
type MultiDeviceStatus struct {
	Devices      map[string]*MonitorStatus `json:"devices"`
	TotalDevices int                       `json:"total_devices"`
	Timestamp    time.Time                 `json:"timestamp"`
}
