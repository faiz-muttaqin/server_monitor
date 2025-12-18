package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"server_monitor/cache"
	"server_monitor/model"

	"github.com/go-routeros/routeros/v3"
)

type Service struct {
	client          *routeros.Client
	prevNATs        map[string]model.NatRule
	currentNATs     map[string]model.NatRule
	bandwidthData   map[string]*model.BandwidthUsage
	dailySummary    map[string]*model.DailyBandwidthSummary
	monthlySummary  map[string]*model.MonthlyBandwidthSummary
	cacheManager    *cache.Manager
	deviceName      string
	isRunning       bool
	startTime       time.Time
	lastUpdate      time.Time
	mu              sync.RWMutex
	connectionStats *model.ConnectionStats
}

func NewService() *Service {
	return &Service{
		prevNATs:       make(map[string]model.NatRule),
		currentNATs:    make(map[string]model.NatRule),
		bandwidthData:  make(map[string]*model.BandwidthUsage),
		dailySummary:   make(map[string]*model.DailyBandwidthSummary),
		monthlySummary: make(map[string]*model.MonthlyBandwidthSummary),
		cacheManager:   cache.NewManager(),
		startTime:      time.Now(),
	}
}

func (s *Service) Connect(ctx context.Context) error {
	rhost := os.Getenv("MIKROTIK_HOST")
	user := os.Getenv("MIKROTIK_USER")
	pass := os.Getenv("MIKROTIK_PASS")

	if rhost == "" || user == "" || pass == "" {
		return fmt.Errorf("missing environment variables: MIKROTIK_HOST, MIKROTIK_USER, MIKROTIK_PASS")
	}

	fmt.Printf("Connecting to %s as %s...\n", rhost, user)

	// Check TCP reachability
	conn, err := net.DialTimeout("tcp", rhost, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP connect failed: %v", err)
	}
	conn.Close()

	// Try API connection with TLS fallback
	client, err := routeros.DialContext(ctx, rhost, user, pass)
	if err != nil {
		fmt.Printf("Standard API failed: %v\nTrying TLS...\n", err)
		client, err = routeros.DialTLSContext(ctx, rhost, user, pass, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return fmt.Errorf("TLS connection failed: %v", err)
		}
		fmt.Printf("Connected to %s via TLS\n", rhost)
	} else {
		fmt.Printf("Connected to %s successfully\n", rhost)
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
			s.deviceName = name
			fmt.Printf("✓ Connected to device: %s\n", name)
		}
	}

	s.client = client

	// Load cached NAT rules
	if err := s.loadCache(); err != nil {
		log.Printf("Warning: Failed to load cache: %v", err)
	}

	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isRunning = false

	// Save current NAT rules to cache
	if err := s.saveCache(); err != nil {
		log.Printf("Warning: Failed to save cache: %v", err)
	}

	if s.client != nil {
		s.client.Close()
	}
}

func (s *Service) loadCache() error {
	cachedNATs, err := s.cacheManager.Load()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.prevNATs = cachedNATs
	s.mu.Unlock()

	if len(cachedNATs) > 0 {
		fmt.Printf("Loaded %d NAT rules from cache\n", len(cachedNATs))
	}

	return nil
}

func (s *Service) saveCache() error {
	return s.cacheManager.Save(s.currentNATs)
}

func (s *Service) fetchNAT(ctx context.Context) ([]model.NatRule, error) {
	reply, err := s.client.RunArgs([]string{"/ip/firewall/nat/print"})
	if err != nil {
		return nil, err
	}
	defer reply.Done.String()

	var rules []model.NatRule
	ts := time.Now()

	for _, r := range reply.Re {
		rules = append(rules, model.NatRule{
			ID:         r.Map[".id"],
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

func (s *Service) fetchConnections(ctx context.Context) ([]*model.Connection, error) {
	reply, err := s.client.RunArgs([]string{"/ip/firewall/connection/print"})
	if err != nil {
		return nil, err
	}
	defer reply.Done.String()

	var connections []*model.Connection
	ts := time.Now()

	for _, r := range reply.Re {
		bytesIn, _ := strconv.ParseUint(r.Map["orig-bytes"], 10, 64)
		bytesOut, _ := strconv.ParseUint(r.Map["repl-bytes"], 10, 64)
		packetsIn, _ := strconv.ParseUint(r.Map["orig-packets"], 10, 64)
		packetsOut, _ := strconv.ParseUint(r.Map["repl-packets"], 10, 64)

		connection := &model.Connection{
			SrcAddress: r.Map["src-address"],
			SrcPort:    r.Map["src-port"],
			DstAddress: r.Map["dst-address"],
			DstPort:    r.Map["dst-port"],
			Protocol:   r.Map["protocol"],
			State:      r.Map["connection-state"],
			BytesIn:    bytesIn,
			BytesOut:   bytesOut,
			PacketsIn:  packetsIn,
			PacketsOut: packetsOut,
			Timestamp:  ts,
		}
		connections = append(connections, connection)
	}

	return connections, nil
}

func (s *Service) fetchConnectionStats(ctx context.Context) (*model.ConnectionStats, error) {
	reply, err := s.client.RunArgs([]string{"/ip/firewall/connection/print", "count-only"})
	if err != nil {
		reply, err = s.client.RunArgs([]string{"/ip/firewall/connection/print"})
		if err != nil {
			return nil, err
		}
	}
	defer reply.Done.String()

	stats := &model.ConnectionStats{
		TotalConnections: len(reply.Re),
		Timestamp:        time.Now(),
	}
	return stats, nil
}

func (s *Service) processBandwidthUsage(connections []*model.Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	today := now.Format("2006-01-02")
	thisMonth := now.Format("2006-01")

	// Initialize daily summary if not exists
	if s.dailySummary[today] == nil {
		s.dailySummary[today] = &model.DailyBandwidthSummary{
			Date:       today,
			PortUsages: make(map[string]*model.BandwidthUsage),
		}
	}

	// Initialize monthly summary if not exists
	if s.monthlySummary[thisMonth] == nil {
		s.monthlySummary[thisMonth] = &model.MonthlyBandwidthSummary{
			Month:      thisMonth,
			PortUsages: make(map[string]*model.BandwidthUsage),
		}
	}

	// Process each connection
	for _, conn := range connections {
		if conn.DstPort == "" {
			continue
		}

		// Update port-specific usage
		key := conn.DstPort

		if s.bandwidthData[key] == nil {
			s.bandwidthData[key] = &model.BandwidthUsage{
				DstPort:   conn.DstPort,
				Timestamp: now,
				Date:      today,
				Month:     thisMonth,
			}
		}

		usage := s.bandwidthData[key]
		usage.BytesIn += conn.BytesIn
		usage.BytesOut += conn.BytesOut
		usage.PacketsIn += conn.PacketsIn
		usage.PacketsOut += conn.PacketsOut
		usage.Timestamp = now

		// Update daily summary
		dailyUsage := s.dailySummary[today].PortUsages[key]
		if dailyUsage == nil {
			dailyUsage = &model.BandwidthUsage{
				DstPort: conn.DstPort,
				Date:    today,
			}
			s.dailySummary[today].PortUsages[key] = dailyUsage
		}
		dailyUsage.BytesIn += conn.BytesIn
		dailyUsage.BytesOut += conn.BytesOut
		dailyUsage.PacketsIn += conn.PacketsIn
		dailyUsage.PacketsOut += conn.PacketsOut

		s.dailySummary[today].TotalIn += conn.BytesIn
		s.dailySummary[today].TotalOut += conn.BytesOut

		// Update monthly summary
		monthlyUsage := s.monthlySummary[thisMonth].PortUsages[key]
		if monthlyUsage == nil {
			monthlyUsage = &model.BandwidthUsage{
				DstPort: conn.DstPort,
				Month:   thisMonth,
			}
			s.monthlySummary[thisMonth].PortUsages[key] = monthlyUsage
		}
		monthlyUsage.BytesIn += conn.BytesIn
		monthlyUsage.BytesOut += conn.BytesOut
		monthlyUsage.PacketsIn += conn.PacketsIn
		monthlyUsage.PacketsOut += conn.PacketsOut

		s.monthlySummary[thisMonth].TotalIn += conn.BytesIn
		s.monthlySummary[thisMonth].TotalOut += conn.BytesOut
	}
}

func (s *Service) monitorNATChanges(ctx context.Context) error {
	nats, err := s.fetchNAT(ctx)
	if err != nil {
		return fmt.Errorf("fetch NAT error: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var sortedNewRules []model.NatRule
	current := make(map[string]model.NatRule)

	// Check for new NAT rules
	for _, n := range nats {
		current[n.ID] = n
		if _, exists := s.prevNATs[n.ID]; !exists {
			sortedNewRules = append(sortedNewRules, n)
		}
	}

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

	// Log new rules
	for _, n := range sortedNewRules {
		log.Printf("[NAT ADDED] %s:%s → %s:%s (%s)",
			n.DstAddress, n.DstPort, n.ToAddress, n.ToPort, n.Comment)
	}

	// Check for removed NAT rules
	for id, rule := range s.prevNATs {
		if _, exists := current[id]; !exists {
			log.Printf("[NAT REMOVED] %s (%s)", id, rule.Comment)
		}
	}

	s.prevNATs = current
	s.currentNATs = current
	s.lastUpdate = time.Now()

	return nil
}

func (s *Service) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.mu.Lock()
	s.isRunning = true
	s.mu.Unlock()

	log.Printf("Starting MikroTik monitoring every %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping monitor...")
			return
		case <-ticker.C:
			if err := s.monitorNATChanges(ctx); err != nil {
				log.Printf("NAT monitoring error: %v", err)
			}

			// Fetch connection stats
			stats, err := s.fetchConnectionStats(ctx)
			if err != nil {
				log.Printf("Connection stats error: %v", err)
			} else {
				s.mu.Lock()
				s.connectionStats = stats
				s.mu.Unlock()
				log.Printf("Active connections: %d", stats.TotalConnections)
			}

			// Fetch detailed connections for bandwidth monitoring
			connections, err := s.fetchConnections(ctx)
			if err != nil {
				log.Printf("Connections fetch error: %v", err)
			} else {
				s.processBandwidthUsage(connections)
			}
		}
	}
}

// Public methods for API access
func (s *Service) GetCurrentNATRules() []model.NatRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]model.NatRule, 0, len(s.currentNATs))
	for _, rule := range s.currentNATs {
		rules = append(rules, rule)
	}

	// Sort by port
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

func (s *Service) GetBandwidthUsage() map[string]*model.BandwidthUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]*model.BandwidthUsage)
	for k, v := range s.bandwidthData {
		result[k] = v
	}
	return result
}

func (s *Service) GetDailyBandwidthSummary(date string) *model.DailyBandwidthSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	return s.dailySummary[date]
}

func (s *Service) GetMonthlyBandwidthSummary(month string) *model.MonthlyBandwidthSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if month == "" {
		month = time.Now().Format("2006-01")
	}

	return s.monthlySummary[month]
}

func (s *Service) GetStatus() *model.MonitorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeConnections := 0
	if s.connectionStats != nil {
		activeConnections = s.connectionStats.TotalConnections
	}

	uptime := time.Since(s.startTime).String()

	return &model.MonitorStatus{
		IsRunning:         s.isRunning,
		ConnectedDevice:   s.deviceName,
		LastUpdate:        s.lastUpdate,
		TotalNatRules:     len(s.currentNATs),
		ActiveConnections: activeConnections,
		Uptime:            uptime,
	}
}
