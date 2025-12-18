package appInstaller

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// handleMasterFlag processes the master flag value
func handleMasterFlag(master string) {
	if master == "" {
		fmt.Println("No master flag provided")
		return
	}

	// Set environment variable
	os.Setenv("MASTER_HOST", master)

	// Handle .env file
	envPath := ".env"
	var envContent string
	var err error

	if _, err = os.Stat(envPath); os.IsNotExist(err) {
		// .env does not exist, create it
		envContent = fmt.Sprintf("MASTER_HOST=%s\n", master)
		os.WriteFile(envPath, []byte(envContent), 0644)
		return
	}

	// .env exists, read and update
	data, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Println("Error reading .env:", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "MASTER_HOST=") {
			lines[i] = fmt.Sprintf("MASTER_HOST=%s", master)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("MASTER_HOST=%s", master))
	}
	envContent = strings.Join(lines, "\n")
	err = os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		fmt.Println("Error writing .env:", err)
	}
}

func ListenMasterFlag() {
	master := flag.String("master", "", "Master host URL")
	flag.Parse()

	if *master == "" {
		fmt.Println("No master flag provided")
		return
	}

	// Set environment variable
	os.Setenv("MASTER_HOST", *master)

	// Handle .env file
	envPath := ".env"
	var envContent string
	var err error

	if _, err = os.Stat(envPath); os.IsNotExist(err) {
		// .env does not exist, create it
		envContent = fmt.Sprintf("MASTER_HOST=%s\n", *master)
		os.WriteFile(envPath, []byte(envContent), 0644)
		return
	}

	// .env exists, read and update
	data, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Println("Error reading .env:", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "MASTER_HOST=") {
			lines[i] = fmt.Sprintf("MASTER_HOST=%s", *master)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("MASTER_HOST=%s", *master))
	}
	envContent = strings.Join(lines, "\n")
	err = os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		fmt.Println("Error writing .env:", err)
	}
}
