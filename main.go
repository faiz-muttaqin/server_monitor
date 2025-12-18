package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"server_monitor/appInstaller"
	"server_monitor/controller"
	"server_monitor/database"
	"server_monitor/kvstore"
	"server_monitor/logger"
	"server_monitor/model"
	"server_monitor/monitor"
	"server_monitor/routes"
	"server_monitor/serviceScanner"
	"server_monitor/utils"
	"server_monitor/webgui"
	"server_monitor/wsclient"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

//go:embed views
var viewsFS embed.FS

func loadEnv() {
	// Try to load .env from executable directory first, then current directory
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	os.Setenv("APP_DIR", exeDir)

	if err := godotenv.Load(filepath.Join(exeDir, ".env")); err != nil {
		pwd, _ := os.Getwd()
		os.Setenv("APP_DIR", pwd)
		if err := godotenv.Load(filepath.Join(pwd, ".env")); err != nil {
			log.Printf("Warning: .env file not found in executable or current directory: %v", err)
		}
	}
}
func main() {
	loadEnv()
	logger.InitLogrus()
	if appInstaller.Init() {
		log.Fatal("App Installed")
	}
	go controller.AsyncHandleGetDevices()
	logrus.Println("hehe")
	if os.Getenv("REDIS_HOST") != "" {
		kvstore.InitRedis(
			fmt.Sprintf(`%s:%s`, os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
			os.Getenv("REDIS_PASSWORD"),
			0,
		)
	}
	webgui.Init()

	// Initialize database if configured
	if os.Getenv("DB_WEB_USER") != "" &&
		os.Getenv("DB_WEB_PASSWORD") != "" &&
		os.Getenv("DB_WEB_HOST") != "" &&
		os.Getenv("DB_WEB_PORT") != "" &&
		os.Getenv("DB_WEB_NAME") != "" {
		var err error
		if database.DB, err = database.InitMySqlDB(
			os.Getenv("DB_WEB_USER"),
			os.Getenv("DB_WEB_PASSWORD"),
			os.Getenv("DB_WEB_HOST"),
			os.Getenv("DB_WEB_PORT"),
			os.Getenv("DB_WEB_NAME"),
		); err == nil {
			database.AutoMigrateWeb(database.DB)
		}
	}

	// Setup Gin mode
	if os.Getenv("LOG_GIN_MODE") == "" && os.Getenv("GIN_MODE") == "" {
		gin.SetMode("release")
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Background service scanner
	go func() {
		// Initial scan
		serviceScanner.UpdateListServices()
		for {
			time.Sleep(30 * time.Second) // Scan every 30 seconds
			serviceScanner.UpdateListServices()
		}
	}()
	// Create context with cancel for monitoring
	monitorMikrotikCtx, monitorMikrotikCancel := context.WithCancel(context.Background())
	defer monitorMikrotikCancel()
	go func() {
		for {
			loadEnv()
			// Parse MikroTik configurations from environment
			mikrotikEnv := os.Getenv("MIKROTIK")
			if mikrotikEnv == "" {
				time.Sleep(1 * time.Minute)
				continue
				// log.Fatal("MIKROTIK environment variable is required. Format: [{\"host\":\"ip:port\",\"user\":\"user\",\"pass\":\"pass\"}]")
			}

			configs, err := model.ParseMikroTikConfigs(mikrotikEnv)
			if err != nil {
				time.Sleep(utils.Getenv("MIKROTIK_RETRY_INTERVAL", 30*time.Second))
				continue
				// log.Fatalf("Failed to parse MikroTik configurations: %v", err)
			}

			log.Printf("Loaded %d MikroTik device configurations", len(configs))
			if len(configs) == 0 {
				time.Sleep(utils.Getenv("MIKROTIK_RETRY_INTERVAL", 30*time.Second))
				continue
			}
			for _, config := range configs {
				log.Printf("  - %s (%s)", config.Name, config.Host)
			}

			// Initialize multi-device monitor service
			monitor.MikrotikMultiService = monitor.NewMultiService(configs)

			// Connect to all MikroTik devices
			log.Println("Initializing MikroTik connections...")
			ctx := context.Background()
			if err := monitor.MikrotikMultiService.ConnectAll(ctx); err != nil {
				logrus.Errorf("Failed to connect to MikroTik devices: %v", err)
				time.Sleep(utils.Getenv("MIKROTIK_RETRY_INTERVAL", 30*time.Second))
				continue
			}
			break
		}
		// Start monitoring service
		monitorInterval := 5 * time.Second
		if interval := os.Getenv("MONITOR_MIKROTIK_INTERVAL"); interval != "" {
			if d, err := time.ParseDuration(interval); err == nil {
				monitorInterval = d
			}
		}
		// Start monitoring in separate goroutine
		go monitor.MikrotikMultiService.Start(monitorMikrotikCtx, monitorInterval)
	}()

	// Start system monitoring service
	logrus.Info("Starting system monitoring service...")
	monitor.LoadServerServices()
	var monitorService *monitor.MonitorService
	go func() {
		monitorService = monitor.NewMonitorService(10 * time.Second)
		monitorService.Start()
	}()
	var monitorESXIService *monitor.MonitorESXIService
	go func() {
		monitorESXIService = monitor.NewESXiMonitoringWithRetry()
		monitorESXIService.Start()
	}()
	// Start web server in goroutine
	go func() {
		//HANDLE WEB ENDPOINT
		routes.R = gin.Default()
		routes.R.Use(cors.Default())
		// Cache /assets/* requests
		routes.R.Use(func(c *gin.Context) {
			if c.Request.Method == http.MethodGet && len(c.Request.URL.Path) >= 8 && c.Request.URL.Path[:8] == "/assets/" {
				c.Header("Cache-Control", "public, max-age=86400") // Cache for 1 day
			}
		})

		// Setup static files and templates with fallback for development
		if err := setupStaticFilesAndTemplates(); err != nil {
			logrus.Fatalf("Failed to setup static files and templates: %v", err)
		}

		routes.Routes()
		logrus.Info("Web Hosted at http://localhost" + utils.Getenv("APP_LOCAL_HOST", ":28888") + "/")
		if err := routes.R.Run(os.Getenv("APP_LOCAL_HOST")); err != nil {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()
	go wsclient.Connect()

	// ################################   SIGNAL GRACEFUL SHUTDOWN
	// Wait for shutdown signal
	<-sigChan
	logrus.Info("Shutting down gracefully...")
	// Stop monitoring service
	monitorService.Stop()

	// Flush server cache to file before shutdown
	if err := model.FlushServerCache(); err != nil {
		logrus.Errorf("Failed to flush server cache: %v", err)
	} else {
		logrus.Info("Server cache flushed successfully")
	}
	// Save data before exit
	fmt.Println("\n🛑 Shutdown signal received, saving data...")
	if monitorESXIService != nil {
		monitorESXIService.Stop()
		monitor.SaveESXIDataOnExit()
	}
	log.Println("Shutdown signal received, initiating graceful shutdown...")

	// Cancel monitoring context first
	monitorMikrotikCancel()

	// Create shutdown timeout context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	// Create a done channel to track completion
	done := make(chan bool, 1)

	// Perform shutdown in goroutine
	go func() {
		// Stop monitor service
		log.Println("🛑 Stopping monitor service...")
		if monitor.MikrotikMultiService != nil {
			monitor.MikrotikMultiService.Close()
		}

		done <- true
	}()
	// Wait for shutdown to complete or timeout
	select {
	case <-done:
		log.Println("Multi-Device MikroTik Monitor stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached, forcing exit...")
	}

	logrus.Info("Server stopped")
	// Force exit to ensure clean termination
	os.Exit(0)
}

// setupStaticFilesAndTemplates configures static files and HTML templates
// with embedded files and fallback to filesystem for development
func setupStaticFilesAndTemplates() error {
	// Setup embedded static files
	assetsSubFS, err := fs.Sub(viewsFS, "views/assets")
	if err != nil {
		return fmt.Errorf("failed to create assets sub filesystem: %v", err)
	}
	routes.R.StaticFS("/assets", http.FS(assetsSubFS))

	// Setup embedded HTML templates
	templ, err := template.ParseFS(viewsFS, "views/**/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse embedded templates: %v", err)
	}
	routes.R.SetHTMLTemplate(templ)

	return nil
}
