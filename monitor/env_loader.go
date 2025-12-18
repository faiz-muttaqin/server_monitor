package monitor

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// ReloadEnvironment reloads environment variables from .env file
func ReloadEnvironment() error {
	// Load .env file
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// Try loading from executable directory first
	if err := godotenv.Load(filepath.Join(exeDir, ".env")); err != nil {
		// If that fails, try current working directory
		pwd, _ := os.Getwd()
		if err := godotenv.Load(filepath.Join(pwd, ".env")); err != nil {
			log.Printf("Error loading .env file from both executable directory and current working directory: %v", err)
			return err
		} else {
			os.Setenv("APP_DIR", pwd)
		}
	} else {
		os.Setenv("APP_DIR", exeDir)
	}

	return nil
}

// WaitForValidConfig waits for valid ESXI configuration with retry mechanism
func WaitForValidConfig() *ESXIConfig {
	for {
		log.Println("🔄 Attempting to load ESXI configuration...")

		// Reload environment variables
		if err := ReloadEnvironment(); err != nil {
			log.Printf("⚠️ Failed to reload environment: %v", err)
		}

		// Try to load config
		config, err := LoadESXIConfig()
		if err != nil {
			log.Printf("❌ Failed to load ESXI config: %v", err)
			log.Println("⏳ Waiting 1 minute before retry...")
			time.Sleep(1 * time.Minute)
			continue
		}

		// Validate that we have at least one host
		if len(config.ESXiHosts) == 0 {
			log.Println("❌ No ESXI hosts found in configuration")
			log.Println("⏳ Waiting 1 minute before retry...")
			time.Sleep(1 * time.Minute)
			continue
		}

		// Log successful configuration
		log.Printf("✅ Successfully loaded ESXI configuration with %d host(s):", len(config.ESXiHosts))
		for i, host := range config.ESXiHosts {
			log.Printf("   %d. %s (%s)", i+1, host.Host, host.User)
		}

		return config
	}
}

// WaitForValidConfigUpdate waits for valid configuration update during runtime
func WaitForValidConfigUpdate(currentConfig *ESXIConfig) *ESXIConfig {
	log.Println("🔄 Checking for configuration updates...")

	// Reload environment variables
	if err := ReloadEnvironment(); err != nil {
		log.Printf("⚠️ Failed to reload environment: %v", err)
		return currentConfig
	}

	// Try to load new config
	newConfig, err := LoadESXIConfig()
	if err != nil {
		log.Printf("⚠️ Failed to load updated config: %v", err)
		return currentConfig
	}

	// Validate that we have at least one host
	if len(newConfig.ESXiHosts) == 0 {
		log.Println("⚠️ No ESXI hosts found in updated configuration")
		return currentConfig
	}

	// Check if configuration actually changed
	if !configChanged(currentConfig, newConfig) {
		log.Println("ℹ️ Configuration unchanged")
		return currentConfig
	}

	// Log configuration update
	log.Printf("🔄 Configuration updated! Now monitoring %d host(s):", len(newConfig.ESXiHosts))
	for i, host := range newConfig.ESXiHosts {
		log.Printf("   %d. %s (%s)", i+1, host.Host, host.User)
	}

	return newConfig
}

// configChanged checks if configuration has changed
func configChanged(oldConfig, newConfig *ESXIConfig) bool {
	if len(oldConfig.ESXiHosts) != len(newConfig.ESXiHosts) {
		return true
	}

	// Simple comparison - in production you might want more sophisticated comparison
	for i, oldHost := range oldConfig.ESXiHosts {
		if i >= len(newConfig.ESXiHosts) {
			return true
		}
		newHost := newConfig.ESXiHosts[i]
		if oldHost.Host != newHost.Host || oldHost.User != newHost.User || oldHost.Pass != newHost.Pass {
			return true
		}
	}

	return false
}
