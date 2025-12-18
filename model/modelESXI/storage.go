package modelESXI

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MultiHostData menyimpan data monitoring untuk multiple hosts
// Langsung berupa map tanpa wrapper, format: {"192.168.1.1": {...}, "192.168.1.2": {...}}
type MultiHostData map[string]*MonitoringData

// NewMultiHostData membuat instance baru MultiHostData
func NewMultiHostData() MultiHostData {
	return make(MultiHostData)
}

// UpdateHost menambahkan atau update data untuk host tertentu
func (m MultiHostData) UpdateHost(hostIP string, data *MonitoringData) {
	m[hostIP] = data
}

// GetHost mengambil data untuk host tertentu
func (m MultiHostData) GetHost(hostIP string) (*MonitoringData, bool) {
	data, exists := m[hostIP]
	return data, exists
}

// GetAllHosts mengambil semua data hosts
func (m MultiHostData) GetAllHosts() map[string]*MonitoringData {
	return map[string]*MonitoringData(m)
}

// SaveToFile menyimpan data ke file JSON
func (m MultiHostData) SaveToFile(filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadMultiHostDataFromFile memuat data dari file JSON
func LoadMultiHostDataFromFile(filePath string) (MultiHostData, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return NewMultiHostData(), nil // Return empty data if file doesn't exist
	}

	// Read file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var data MultiHostData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if data == nil {
		data = NewMultiHostData()
	}

	return data, nil
}

// ToJSON mengkonversi ke JSON string
func (m MultiHostData) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// GetHostCount mengembalikan jumlah host yang dimonitor
func (m MultiHostData) GetHostCount() int {
	return len(m)
}

// RemoveHost menghapus data host tertentu
func (m MultiHostData) RemoveHost(hostIP string) bool {
	if _, exists := m[hostIP]; exists {
		delete(m, hostIP)
		return true
	}
	return false
}
