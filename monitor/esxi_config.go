package monitor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ESXiHost struct {
	Url      string `json:"url"`
	Host     string `json:"host"`
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Insecure int    `json:"insecure"`
}

type ESXIConfig struct {
	ESXiURL        string        `json:"esxi_url,omitempty"`
	ESXiHost       string        `json:"esxi_host,omitempty"`
	ESXiUser       string        `json:"esxi_user,omitempty"`
	ESXiPass       string        `json:"esxi_pass,omitempty"`
	ESXiHosts      []ESXiHost    `json:"esxi_hosts,omitempty"` // Multi-host support
	Insecure       bool          `json:"insecure"`
	PollInterval   time.Duration `json:"poll_interval"`
	ShowPoweredOff bool          `json:"show_powered_off"`
	ShowVMDetails  bool          `json:"show_vm_details"`
	DetailedOutput bool          `json:"detailed_output"`
	DataFilePath   string        `json:"data_file_path"`
}

func LoadESXIConfig() (*ESXIConfig, error) {
	config := &ESXIConfig{}

	// Try to load multiple ESXi hosts from JSON format
	esxiJsonStr := os.Getenv("ESXI")
	if esxiJsonStr != "" {
		raw := strings.TrimSpace(esxiJsonStr)
		var err error
		var hosts []ESXiHost
		if strings.HasPrefix(raw, "[\"") {
			var urlStrings []string
			err = json.Unmarshal([]byte(esxiJsonStr), &urlStrings)
			for i := range urlStrings {
				hosts = append(hosts, ESXiHost{Url: urlStrings[i]})
			}
		} else if strings.HasPrefix(raw, "[{") {
			err = json.Unmarshal([]byte(esxiJsonStr), &hosts)
		}
		if err == nil {
			for i := range hosts {
				// If a URL is provided, try to parse and fill any missing host/user/pass fields
				if hosts[i].Url != "" {
					if parsedURL, err := url.Parse(hosts[i].Url); err == nil {
						if parsedURL.User != nil {
							if hosts[i].Host == "" {
								hosts[i].Host = parsedURL.Host
							}
							if hosts[i].User == "" {
								hosts[i].User = parsedURL.User.Username()
							}
							if hosts[i].Pass == "" {
								if p, ok := parsedURL.User.Password(); ok {
									hosts[i].Pass = p
								}
							}
							if os.Getenv("ESXI_INSECURE") == "0" {
								hosts[i].Insecure = 0
							} else {
								hosts[i].Insecure = 1
							}
						} else {
							// URL has no user info, but we can still fill host if missing
							if hosts[i].Host == "" {
								hosts[i].Host = parsedURL.Host
							}
						}
					}
				}

				// If URL is missing but we have host/user/pass, build the URL
				if hosts[i].Url == "" && hosts[i].Host != "" && hosts[i].User != "" && hosts[i].Pass != "" {
					hosts[i].Url = fmt.Sprintf("https://%s:%s@%s/sdk",
						url.PathEscape(hosts[i].User),
						url.PathEscape(hosts[i].Pass),
						hosts[i].Host)
				}
			}

			// Keep only entries that have host, user and pass (either originally or after parsing/building)
			var filtered []ESXiHost
			for _, h := range hosts {
				if h.Host != "" && h.User != "" && h.Pass != "" {
					filtered = append(filtered, h)
				}
			}
			config.ESXiHosts = filtered
		} else {
			return nil, fmt.Errorf("invalid ESXI JSON format: %v", err)
		}
	} else {
		// Fallback to single host configuration
		config.ESXiURL = os.Getenv("ESXI_URL")
		config.ESXiHost = os.Getenv("ESXI_HOST")
		config.ESXiUser = os.Getenv("ESXI_USER")
		config.ESXiPass = os.Getenv("ESXI_PASS")

		// Build single host entry
		if config.ESXiURL == "" && (config.ESXiHost != "" && config.ESXiUser != "" && config.ESXiPass != "") {
			insecureInt := 0
			if os.Getenv("ESXI_INSECURE") == "1" || os.Getenv("ESXI_INSECURE") == "true" {
				insecureInt = 1
			}

			config.ESXiHosts = []ESXiHost{
				{
					Host:     config.ESXiHost,
					User:     config.ESXiUser,
					Pass:     config.ESXiPass,
					Insecure: insecureInt,
				},
			}
		} else if config.ESXiURL != "" {
			// Parse URL to extract host info
			parsedURL, err := url.Parse(config.ESXiURL)
			if err == nil && parsedURL.User != nil {
				password, _ := parsedURL.User.Password()
				config.ESXiHosts = []ESXiHost{
					{
						Host:     parsedURL.Host,
						User:     parsedURL.User.Username(),
						Pass:     password,
						Insecure: 1, // Assume insecure for URL format
					},
				}
			}
		}
	}

	// Validate that we have at least one host
	if len(config.ESXiHosts) == 0 {
		return nil, fmt.Errorf("no ESXi hosts configured. Set ESXI environment variable with JSON format or individual ESXI_HOST, ESXI_USER, ESXI_PASS")
	}

	// Insecure SSL setting (fallback)
	config.Insecure = os.Getenv("ESXI_INSECURE") == "1" || os.Getenv("ESXI_INSECURE") == "true"

	// Poll interval
	intervalStr := os.Getenv("POLL_INTERVAL_SECONDS")
	if intervalStr == "" {
		config.PollInterval = 5 * time.Second
	} else {
		if seconds, err := strconv.Atoi(intervalStr); err == nil {
			config.PollInterval = time.Duration(seconds) * time.Second
		} else {
			return nil, fmt.Errorf("invalid POLL_INTERVAL_SECONDS: %s", intervalStr)
		}
	}

	// Display options
	config.ShowPoweredOff = getBoolEnv("SHOW_POWERED_OFF_VMS", true)
	config.ShowVMDetails = getBoolEnv("SHOW_VM_DETAILS", true)
	config.DetailedOutput = getBoolEnv("DETAILED_OUTPUT", true)

	// Data file path
	config.DataFilePath = os.Getenv("DATA_FILE_PATH")
	if config.DataFilePath == "" {
		config.DataFilePath = "./.cache/monitoring_data.json"
	}

	return config, nil
}

// GetESXiURL builds URL for specific host
func (c *ESXIConfig) GetESXiURL(host ESXiHost) string {
	return fmt.Sprintf("https://%s:%s@%s/sdk",
		url.PathEscape(host.User),
		url.PathEscape(host.Pass),
		host.Host)
}

// IsInsecure returns whether SSL verification should be skipped for host
func (c *ESXIConfig) IsInsecure(host ESXiHost) bool {
	return host.Insecure == 1 || c.Insecure
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "1" || value == "true"
}
