# Server Monitor

A comprehensive cross-platform server monitoring application built with Go that provides real-time monitoring for system resources, ESXi hosts, MikroTik routers, and network services.

## Features

### 🖥️ System Monitoring
- **Real-time Resource Tracking**: CPU, Memory, Disk, and Network usage
- **Process Monitoring**: Track running processes and resource consumption
- **Service Detection**: Automatically detect and monitor system services
- **Network Port Monitoring**: Track listening ports and connections
- **Activity Monitoring**: Mouse movement and CLI activity detection
- **Cross-Platform**: Supports both Windows and Linux

### 🌐 ESXi Monitoring
- **Multi-Host Support**: Monitor multiple VMware ESXi hosts simultaneously
- **VM Management**: Track virtual machine status, resource usage, and configurations
- **Resource Analytics**: CPU, memory, and storage utilization per host and VM
- **Storage Monitoring**: Datastore capacity and usage tracking
- **Real-time Status**: Live connection status with automatic offline/online detection
- **Change Detection**: Automatic detection and broadcast of VM state changes

### 🔧 MikroTik Router Monitoring
- **Multi-Device Support**: Monitor multiple MikroTik routers
- **NAT Rule Tracking**: Monitor firewall NAT rules and changes
- **Bandwidth Monitoring**: Track bandwidth usage per port and service
- **Connection Statistics**: Active connection tracking
- **Daily/Monthly Summaries**: Bandwidth usage reports
- **Auto-Reconnect**: Automatic reconnection on failure

### 🔌 Network Service Scanner
- **Port Detection**: Scan and identify listening network ports
- **Service Mapping**: Map ports to running processes and systemd services
- **Protocol Support**: Both TCP and UDP monitoring
- **Real-time Updates**: Continuous scanning with configurable intervals

### 📊 Web Dashboard
- **Real-time Updates**: WebSocket-based live data streaming
- **Interactive UI**: Modern web interface with responsive design
- **Multi-Server View**: Monitor multiple servers from a single dashboard
- **Service Tables**: Detailed service and process listings
- **ESXi Tables**: Comprehensive ESXi host and VM information

## Installation

### Prerequisites
- Go 1.21 or higher
- For ESXi monitoring: Access to VMware ESXi hosts
- For MikroTik monitoring: MikroTik router with API access
- For Linux: `sudo` access for full service detection

### Build from Source

```bash
# Clone the repository
git clone https://github.com/faiz-muttaqin/server_monitor.git
cd server_monitor

# Install dependencies
go mod download

# Build the application
go build -o server_monitor main.go

# Run
./server_monitor
```

### Windows Installation
```bash
# Build for Windows
go build -o server_monitor.exe main.go

# Run
.\server_monitor.exe
```

### Linux Installation
```bash
# Build for Linux
go build -o server_monitor main.go

# Make executable
chmod +x server_monitor

# Run with sudo for full service detection
sudo ./server_monitor
```

## Configuration

Create a `.env` file in the application directory. **All configuration options are optional** - the application will run with defaults if not specified.

### Example Configuration

```bash
# Copy the example file
cp .env.example .env

# Edit with your settings
nano .env
```

### Configuration Options

#### Application Settings (Optional)
```env
# Application display name (default: MyServer)
APP_NAME=MyServer

# Application logo path (default: /assets/self/img/server_monitor.jpg)
APP_LOGO=/assets/self/img/server_monitor.jpg

# Version information
APP_VERSION=1.0.0
APP_VERSION_NO=1
APP_VERSION_CODE=1.0.0
APP_VERSION_NAME=MyServer

# Web interface host and port (default: :28888)
APP_LOCAL_HOST=:28888

# Application directory (auto-detected if empty)
APP_DIR=
```

#### Authentication (Optional)
If not set, web dashboard will be accessible without authentication.

```env
# Basic auth users (JSON array of username:password pairs)
AUTH=["admin:password123","user:userpass"]

# JWT secret key for token generation
JWT_SECRET_KEY=your-secret-key-min-32-characters-long
```

#### Database (Optional)
All database fields must be set together or left empty.

```env
DB_WEB_USER=dbuser
DB_WEB_PASSWORD=dbpassword
DB_WEB_HOST=localhost
DB_WEB_PORT=3306
DB_WEB_NAME=server_monitor
```

#### Redis Cache (Optional)
All Redis fields must be set together or left empty.

```env
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

#### Logging (Optional)
```env
# Log level: trace, debug, info, warn, error, fatal, panic (default: info)
LOG_LEVEL=info

# Log format: json or csv (default: json)
LOG_FORMAT=json

# Log file location
LOG_PATH=./log/app/
LOG_FILE=app.log

# Database and Gin framework log modes
LOG_DB_MODE=silent
LOG_GIN_MODE=release
GIN_MODE=release
```

#### MikroTik Router Monitoring (Optional)
If not configured, MikroTik monitoring features will be disabled.

```env
# Multiple MikroTik routers (JSON array - recommended)
MIKROTIK=[{"host":"192.168.88.1:8728","user":"admin","pass":"password","name":"MikroTik-Main"}]

# Single MikroTik connection (legacy)
MIKROTIK_HOST=192.168.88.1:8728
MIKROTIK_USER=admin
MIKROTIK_PASS=password

# Retry and monitoring intervals
MIKROTIK_RETRY_INTERVAL=30s
MONITOR_MIKROTIK_INTERVAL=60s
```

#### ESXi Monitoring (Optional)
If not configured, ESXi monitoring features will be disabled.

```env
# Multiple ESXi hosts (JSON array)
# Format: https://username:password@host/sdk
ESXI=["https://root:password@192.168.1.10/sdk","https://root:password@192.168.1.11/sdk"]

# Single ESXi connection (alternative)
ESXI_URL=https://192.168.1.10/sdk
ESXI_HOST=192.168.1.10
ESXI_USER=root
ESXI_PASS=password

# Skip SSL certificate verification (1=true, 0=false)
ESXI_INSECURE=0

# Polling interval in seconds
POLL_INTERVAL_SECONDS=60

# Data cache file path
DATA_FILE_PATH=./data/esxi_data.json
```

#### Master/Slave Mode (Optional)
Configure this to send monitoring data to a master server.

```env
# Master server URL to report to
# If set, this instance acts as a slave and sends data to the master
# If not set, this instance runs as standalone
MASTER_HOST=http://192.168.1.100:28888
```

**Note**: Typically, when `MASTER_HOST` is configured, this instance acts as a monitoring agent (slave) and other monitoring features (MikroTik, ESXi) might be disabled on this instance.

#### Service Installation (Optional)
```env
# Service name for OS service installation
# Windows: ServerMonitor
# Linux systemd: server-monitor.service
SERVICE_NAME=
```

## Usage

### Starting the Application

```bash
# Standalone mode (no master server)
./server_monitor

# With configuration file
./server_monitor
```

### Accessing the Web Dashboard

Open your browser and navigate to:
```
http://localhost:28888
```

If authentication is configured, you'll be prompted to login with credentials from the `AUTH` environment variable.

### WebSocket Endpoints

The application provides real-time data through WebSocket connections:

- **Main WebSocket**: `ws://localhost:28888/ws`
- **Node WebSocket**: `ws://localhost:28888/ws/node`

### REST API Endpoints

#### Monitor API
```bash
# Get all monitored servers
GET /api/v1/monitor/servers

# Get servers summary
GET /api/v1/monitor/servers/summary

# Get specific server status
GET /api/v1/monitor/servers/:id

# Update server data
PUT /api/v1/monitor/servers/:id
```

#### MikroTik API (if configured)
```bash
# Get all devices
GET /api/v1/devices

# Get device NAT rules
GET /api/v1/devices/:id/nat-rules

# Get device status
GET /api/v1/devices/:id/status

# Get all NAT rules from all devices
GET /api/v1/nat-rules

# Get bandwidth statistics
GET /api/v1/bandwidth
GET /api/v1/bandwidth/daily
GET /api/v1/bandwidth/monthly
```

#### Services API
```bash
# Get all services from all servers
GET /services

# Get services for specific server IP
GET /services/:ip
```

#### ESXi API
```bash
# Get ESXi data table
GET /esxi/table

# Get specific host data
GET /esxi/table/:hostId
```

## Architecture

### Master-Slave Mode

The application supports two operational modes:

#### Standalone Mode
- Runs all monitoring features independently
- Provides web dashboard for direct access
- Stores all data locally

#### Slave Mode (Agent Mode)
When `MASTER_HOST` is configured:
- Monitors local system resources
- Sends monitoring data to the master server via WebSocket
- Receives commands from master server
- Typically used for distributed monitoring setups

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Server Monitor                           │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   System     │  │    ESXi      │  │  MikroTik    │      │
│  │  Monitoring  │  │  Monitoring  │  │  Monitoring  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                 │                  │               │
│         └─────────────────┴──────────────────┘               │
│                           │                                   │
│                    ┌──────▼──────┐                           │
│                    │   Cache &   │                           │
│                    │   Storage   │                           │
│                    └──────┬──────┘                           │
│                           │                                   │
│         ┌─────────────────┴─────────────────┐               │
│         │                                     │               │
│  ┌──────▼──────┐                    ┌────────▼────────┐     │
│  │ Web Dashboard│                    │   WebSocket     │     │
│  │  (Gin/HTML)  │                    │   Broadcasting  │     │
│  └──────────────┘                    └─────────────────┘     │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Service Detection

### Linux
Uses the `ss` command with `sudo` for comprehensive service detection:
- Detects listening TCP/UDP ports
- Maps processes to systemd services
- Provides service descriptions and status
- Tracks service start times

### Windows
Uses multiple Windows APIs:
- `netstat` for network connections
- `tasklist` / WMI for process information
- Service Control Manager for Windows services
- PowerShell for detailed service data

## Data Storage

### File-Based Storage
- **Server Cache**: `./data/server_cache.json`
- **ESXi Data**: `./data/esxi_data.json` (configurable)
- **Service Data**: `./data/server_services.json`
- **Logs**: `./log/app/app.log` (configurable)

### Redis Cache (Optional)
When Redis is configured, it's used for:
- Session management
- Temporary data caching
- Cross-instance data sharing

## Monitoring Intervals

Default monitoring intervals (configurable):
- **System Monitoring**: 10 seconds
- **Service Scanner**: 30 seconds
- **ESXi Polling**: 60 seconds (configurable via `POLL_INTERVAL_SECONDS`)
- **MikroTik Monitoring**: 60 seconds (configurable via `MONITOR_MIKROTIK_INTERVAL`)
- **Activity Monitoring**: Mouse (5s), CLI (10s)

## Security Considerations

1. **Authentication**: Use strong passwords in the `AUTH` configuration
2. **JWT Secret**: Use a strong, randomly generated secret key (min 32 characters)
3. **ESXi Credentials**: Consider using dedicated monitoring accounts with read-only access
4. **HTTPS**: For production, use a reverse proxy (nginx/apache) with SSL/TLS
5. **Firewall**: Restrict access to the web interface port (default 28888)
6. **Sudo Access**: On Linux, the service scanner requires sudo for full functionality

## Troubleshooting

### Service Scanner Shows No Services (Linux)
```bash
# Ensure the application has sudo privileges
sudo ./server_monitor
```

### ESXi Connection Fails
```bash
# Check ESXi host reachability
ping 192.168.1.10

# Verify ESXi API is enabled
# In ESXi: Configuration → Security Profile → Services → Enable "ESXi Shell"

# For self-signed certificates, set ESXI_INSECURE=1
```

### MikroTik Connection Fails
```bash
# Verify API port is accessible
telnet 192.168.88.1 8728

# Check MikroTik API service is enabled:
# /ip service print
# Enable if needed: /ip service enable api
```

### WebSocket Connection Issues
- Check firewall allows WebSocket connections
- Verify `APP_LOCAL_HOST` is correctly configured
- Check browser console for detailed error messages

### High Memory Usage
- Reduce monitoring intervals
- Limit number of ESXi hosts or MikroTik devices
- Enable Redis for better memory management

## Development

### Project Structure
```
server_monitor/
├── main.go                 # Application entry point
├── appInstaller/           # OS service installation
├── cache/                  # Cache management
├── controller/             # HTTP/WebSocket controllers
├── database/               # Database connections
├── kvstore/                # Redis key-value store
├── logger/                 # Logging configuration
├── model/                  # Data models
├── monitor/                # Monitoring services
│   ├── system_monitor.go   # System resource monitoring
│   ├── esxi_monitor.go     # ESXi monitoring
│   ├── mikrotik_service.go # MikroTik monitoring
│   └── activity_monitor.go # Activity detection
├── routes/                 # API routing
├── serviceScanner/         # Network service detection
├── utils/                  # Utility functions
├── views/                  # Web UI templates
├── webgui/                 # Web GUI components
├── ws/                     # WebSocket handlers
└── wsclient/               # WebSocket client (slave mode)
```

### Building for Different Platforms

```bash
# Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o server_monitor.exe

# Linux (64-bit)
GOOS=linux GOARCH=amd64 go build -o server_monitor

# Linux (ARM - Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o server_monitor

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o server_monitor

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o server_monitor
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [govmomi](https://github.com/vmware/govmomi) - VMware vSphere API Go bindings
- [go-routeros](https://github.com/go-routeros/routeros) - MikroTik RouterOS API
- [logrus](https://github.com/sirupsen/logrus) - Structured logging
- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation

## Support

For issues, questions, or contributions, please visit:
- **GitHub**: [https://github.com/faiz-muttaqin/server_monitor](https://github.com/faiz-muttaqin/server_monitor)
- **Issues**: [https://github.com/faiz-muttaqin/server_monitor/issues](https://github.com/faiz-muttaqin/server_monitor/issues)

---

**Made with ❤️ for server monitoring enthusiasts**
