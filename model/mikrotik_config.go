package model

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
)

// ParseMikroTikConfigs parses the MIKROTIK environment variable JSON string
func ParseMikroTikConfigs(envValue string) ([]*MikroTikConfig, error) {
	if envValue == "" {
		return nil, fmt.Errorf("MIKROTIK environment variable is empty")
	}

	var configs []*MikroTikConfig
	if err := json.Unmarshal([]byte(envValue), &configs); err != nil {
		return nil, fmt.Errorf("failed to parse MIKROTIK JSON: %v", err)
	}

	// Generate IDs and names if not provided
	for i, config := range configs {
		if config.ID == "" {
			// Generate ID from host using MD5 hash (first 8 chars)
			hash := md5.Sum([]byte(config.Host))
			config.ID = fmt.Sprintf("%x", hash)[:8]
		}

		if config.Name == "" {
			// Extract hostname/IP from host:port
			hostPart := strings.Split(config.Host, ":")[0]
			config.Name = fmt.Sprintf("MikroTik-%s", hostPart)
		}

		// Ensure we have required fields
		if config.Host == "" || config.User == "" || config.Pass == "" {
			return nil, fmt.Errorf("config %d: missing required fields (host, user, pass)", i)
		}
	}

	return configs, nil
}

// ValidateMikroTikConfig validates a single MikroTik configuration
func ValidateMikroTikConfig(config *MikroTikConfig) error {
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}
	if config.User == "" {
		return fmt.Errorf("user is required")
	}
	if config.Pass == "" {
		return fmt.Errorf("pass is required")
	}

	// Validate host format (should contain port)
	if !strings.Contains(config.Host, ":") {
		return fmt.Errorf("host should be in format 'ip:port'")
	}

	return nil
}
