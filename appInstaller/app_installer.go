package appInstaller

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var serviceName string

func Init() bool {

	// Define command-line flag for install
	installFlag := flag.Bool("install", false, "Install the service")

	// Define command-line flag for master (moved from ListenMasterFlag)
	masterFlag := flag.String("master", "", "Master host URL")

	// Parse the flags
	flag.Parse()

	// Handle master flag if provided
	if *masterFlag != "" {
		handleMasterFlag(*masterFlag)
	}

	// If the install flag is provided, attempt to install the service
	if *installFlag {
		// Get the current directory
		currentDir, err := os.Getwd()
		if err != nil {
			logrus.Error(err)
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
		_, folderName := filepath.Split(currentDir)
		folderName = strings.Trim(folderName, "/\\")
		serviceName = folderName
		if os.Getenv("SERVICE_NAME") != "" {
			serviceName = os.Getenv("SERVICE_NAME")
		}

		// Detect OS and install appropriate service
		var errInstall error
		switch runtime.GOOS {
		case "windows":
			errInstall = installWindowsService(serviceName, currentDir)
		case "linux":
			errInstall = installLinuxService(serviceName, currentDir)
		default:
			errInstall = fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		if errInstall != nil {
			fmt.Printf("Error installing service: %v\n", errInstall)
			os.Exit(1)
		}
		return true
	}
	return false
}

func installLinuxService(serviceDesc, currentDir string) error {
	// Get the current user
	currentUser, err := user.Current()
	if err != nil {
		logrus.Error(err)
		return fmt.Errorf("error getting current user: %v", err)
	}

	// Check if 'main' and '.env' files exist in the current directory
	mainPath := filepath.Join(currentDir, "main")
	envPath := filepath.Join(currentDir, ".env")

	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		return fmt.Errorf("'main' file not found in the current directory")
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("'.env' file not found in the current directory")
	}

	// Define the service file path
	serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	// Create the service file content
	serviceFileContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=%s
Restart=always
RestartSec=3
EnvironmentFile=%s
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
`, serviceDesc, currentUser.Username, currentUser.Username, currentDir, mainPath, envPath)

	// Write the service file
	if err := os.WriteFile(serviceFilePath, []byte(serviceFileContent), 0644); err != nil {
		return fmt.Errorf("error writing service file: %v", err)
	}

	fmt.Printf("Service file created at %s\n", serviceFilePath)

	// Reload systemd daemon
	if err := exec.Command("sudo", "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("error reloading systemd daemon: %v", err)
	}
	fmt.Println("Systemd daemon reloaded.")

	// Enable the service
	if err := exec.Command("sudo", "systemctl", "enable", fmt.Sprintf("%s.service", serviceName)).Run(); err != nil {
		return fmt.Errorf("error enabling service: %v", err)
	}
	fmt.Println("Service enabled.")

	// Restart the service
	if err := exec.Command("sudo", "systemctl", "restart", fmt.Sprintf("%s.service", serviceName)).Run(); err != nil {
		return fmt.Errorf("error restarting service: %v", err)
	}
	fmt.Println("Service restarted.")

	return nil
}

func installWindowsService(serviceDesc, currentDir string) error {
	// Check if executable exists
	exePath := filepath.Join(currentDir, serviceName+".exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		// Try without .exe extension
		exePath = filepath.Join(currentDir, serviceName)
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			// Try main.exe
			exePath = filepath.Join(currentDir, "main.exe")
			if _, err := os.Stat(exePath); os.IsNotExist(err) {
				return fmt.Errorf("executable not found. Tried: %s.exe, %s, main.exe", serviceName, serviceName)
			}
		}
	}

	fmt.Printf("Using executable: %s\n", exePath)

	// Check if .env file exists (optional for Windows)
	envPath := filepath.Join(currentDir, ".env")
	envExists := false
	if _, err := os.Stat(envPath); err == nil {
		envExists = true
		fmt.Printf("Found .env file: %s\n", envPath)
	}

	// Use sc.exe to create Windows service
	fmt.Printf("Installing Windows service: %s\n", serviceName)

	// Create service with sc.exe
	createCmd := exec.Command("sc.exe", "create", serviceName,
		"binPath="+exePath,
		"DisplayName="+serviceDesc,
		"start=auto",
		"type=own")

	createCmd.Dir = currentDir
	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating Windows service: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("Windows service created successfully.\nOutput: %s\n", string(output))

	// Set service description
	descCmd := exec.Command("sc.exe", "description", serviceName, serviceDesc)
	if descOutput, err := descCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Could not set service description: %v\nOutput: %s\n", err, string(descOutput))
	}

	// Set service to restart on failure
	recoveryCmd := exec.Command("sc.exe", "failure", serviceName, "reset=30", "actions=restart/5000/restart/5000/restart/5000")
	if recoveryOutput, err := recoveryCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Could not set service recovery options: %v\nOutput: %s\n", err, string(recoveryOutput))
	}

	// Start the service
	fmt.Printf("Starting Windows service: %s\n", serviceName)
	startCmd := exec.Command("sc.exe", "start", serviceName)
	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Could not start service immediately: %v\nOutput: %s\n", err, string(startOutput))
		fmt.Println("You can start the service manually with: sc.exe start " + serviceName)
	} else {
		fmt.Printf("Service started successfully.\nOutput: %s\n", string(startOutput))
	}

	// Show environment file info
	if envExists {
		fmt.Printf("\nNote: .env file found but Windows services don't automatically load .env files.\n")
		fmt.Printf("Consider setting environment variables in the service configuration or\n")
		fmt.Printf("modify your application to load the .env file from: %s\n", envPath)
	}

	fmt.Printf("\nWindows service installation completed!\n")
	fmt.Printf("Service name: %s\n", serviceName)
	fmt.Printf("Executable: %s\n", exePath)
	fmt.Printf("\nTo manage the service:\n")
	fmt.Printf("  Start:   sc.exe start %s\n", serviceName)
	fmt.Printf("  Stop:    sc.exe stop %s\n", serviceName)
	fmt.Printf("  Delete:  sc.exe delete %s\n", serviceName)
	fmt.Printf("  Status:  sc.exe query %s\n", serviceName)

	return nil
}
