package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"server_monitor/model"
)

type Manager struct {
	filePath string
}

func NewManager() *Manager {
	appDir := os.Getenv("APP_DIR")
	if appDir == "" {
		pwd, _ := os.Getwd()
		appDir = pwd
	}

	// Create .cache directory if it doesn't exist
	cacheDir := filepath.Join(appDir, ".cache")
	os.MkdirAll(cacheDir, 0755)

	return &Manager{
		filePath: filepath.Join(cacheDir, "nat_rule.json"),
	}
}

// Save saves NAT rules to cache file (legacy single device support)
func (m *Manager) Save(natRules map[string]model.NatRule) error {
	// Convert single device format to multi-device format
	multiDeviceRules := make(map[string]map[string]model.NatRule)
	multiDeviceRules["default"] = natRules

	return m.SaveMulti(multiDeviceRules)
}

// Load loads NAT rules from cache file (legacy single device support)
func (m *Manager) Load() (map[string]model.NatRule, error) {
	multiDeviceRules, err := m.LoadMulti()
	if err != nil {
		return nil, err
	}

	// Return rules from default device or first available device
	if defaultRules, exists := multiDeviceRules["default"]; exists {
		return defaultRules, nil
	}

	// If no default device, return first available device's rules
	for _, deviceRules := range multiDeviceRules {
		return deviceRules, nil
	}

	return make(map[string]model.NatRule), nil
}

// Exists checks if cache file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.filePath)
	return !os.IsNotExist(err)
}

// Remove removes the cache file
func (m *Manager) Remove() error {
	if !m.Exists() {
		return nil
	}
	return os.Remove(m.filePath)
}

// SaveMulti saves NAT rules from multiple devices to cache file
func (m *Manager) SaveMulti(natRules map[string]map[string]model.NatRule) error {
	fmt.Printf("DEBUG: SaveMulti called with %d devices\n", len(natRules))
	fmt.Printf("DEBUG: Cache file path: %s\n", m.filePath)

	cacheData := &model.CacheData{
		NatRules:   natRules,
		LastUpdate: time.Now(),
		Version:    "2.0", // Updated version for multi-device support
	}

	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache data: %v", err)
	}

	fmt.Printf("DEBUG: Writing cache file to %s\n", m.filePath)
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %v", err)
	}

	fmt.Printf("DEBUG: Cache file successfully written\n")
	return nil
}

// LoadMulti loads NAT rules for multiple devices from cache file
func (m *Manager) LoadMulti() (map[string]map[string]model.NatRule, error) {
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// Cache file doesn't exist, return empty map
		return make(map[string]map[string]model.NatRule), nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("read cache file: %v", err)
	}

	var cacheData model.CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil, fmt.Errorf("unmarshal cache data: %v", err)
	}

	// Check if cache is not too old (older than 24 hours)
	if time.Since(cacheData.LastUpdate) > 24*time.Hour {
		fmt.Println("Cache is older than 24 hours, ignoring")
		return make(map[string]map[string]model.NatRule), nil
	}

	if cacheData.NatRules == nil {
		return make(map[string]map[string]model.NatRule), nil
	}

	return cacheData.NatRules, nil
}
func (m *Manager) GetLastUpdate() (time.Time, error) {
	if !m.Exists() {
		return time.Time{}, fmt.Errorf("cache file does not exist")
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("read cache file: %v", err)
	}

	var cacheData model.CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return time.Time{}, fmt.Errorf("unmarshal cache data: %v", err)
	}

	return cacheData.LastUpdate, nil
}
