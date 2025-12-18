package controller

import (
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"server_monitor/model"
	"server_monitor/monitor"
	"server_monitor/utils"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var NATRulesData = make(map[string][]map[string]interface{})
var NatRuleUpdateChannelTrigger = make(chan struct{}, 1)
var RWMutexNATRulesData sync.RWMutex

//	func init() {
//		go AsyncHandleGetDevices()
//	}
func AsyncHandleGetDevices() {
	// try to preload NAT rules from cache if present and valid
	cachePath := "./.cache/nat_rule_data.json"
	if fi, err := os.Stat(cachePath); err == nil && !fi.IsDir() {
		if b, err := os.ReadFile(cachePath); err != nil {
			logrus.WithError(err).Warn("[NATRULES_CACHE]failed to read nat rules cache, continuing without it")
		} else {
			logrus.Warn("[NATRULES_CACHE]", string(b))
			var tmp map[string][]map[string]interface{}
			if err := json.Unmarshal(b, &tmp); err != nil {
				logrus.WithError(err).Warn("[NATRULES_CACHE]failed to unmarshal nat rules cache, ignoring file")
			} else {
				// basic format check: ensure map keys are strings and values are slices
				valid := true
				for k, v := range tmp {
					if k == "" {
						valid = false
						break
					}
					if v == nil {
						valid = false
						break
					}
				}
				if valid {
					RWMutexNATRulesData.Lock()
					maps.Copy(NATRulesData, tmp)
					logrus.Warn("[NATRULES_CACHE]", NATRulesData)
					RWMutexNATRulesData.Unlock()
					logrus.Warn("[NATRULES_CACHE]loaded nat rules cache from " + cachePath)
				} else {
					logrus.Warn("[NATRULES_CACHE]nat rules cache has unexpected format, ignoring")
				}

			}
		}
	} else if err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Warn("[NATRULES_CACHE]failed to stat nat rules cache")
	}
	for {
		if monitor.MikrotikMultiService == nil {
			time.Sleep(5 * time.Second)
			continue
		}
		AllNATrules := monitor.MikrotikMultiService.GetAllNATRules()
		logrus.Debugf("[NATRULES_CACHE] Retrieved rules for %d devices", len(AllNATrules))

		// allData := make(map[string][]map[string]interface{})
		for NATDeviceID, rules := range AllNATrules {
			logrus.Debugf("[NATRULES_CACHE] Processing device %s with %d rules", NATDeviceID, len(rules))
			var data []map[string]interface{}
			for _, rule := range rules {
				serviceName := "unreachable"
				processPath := ""
				isService := false
				status := ""
				description := ""
				var startedAt time.Time
				if serverServices, ok := model.ServerServices[rule.ToAddress]; ok && rule.Chain == "dstnat" {
					found := false
					serviceName = "not found"
					// search for matching service
					for _, service := range serverServices {
						// search for matching port
						for ipPort := range strings.SplitSeq(service.IPAddressPort, ",") {
							if ipPorts := strings.Split(ipPort, ":"); len(ipPorts) == 3 {
								protocol := ipPorts[0]
								ip := ipPorts[1]
								port := ipPorts[2]
								if port == rule.ToPort {
									// switch ip {
									// case "[::]", "*", "0.0.0.0":
									// 	serviceName = service.Name + " (" + protocol + ", all)"
									// default:
									// }
									serviceName = service.Name + " (" + protocol + ", " + ip + ")"
									processPath = service.Process
									isService = service.IsService
									status = service.Status
									startedAt = service.StartedAt
									description = service.Description
									found = true
									break
								}
							}
						}
						if found {
							break
						}
					}
				}
				if serviceName == "unreachable" {
					// attempt a TCP connect to the target with a 3s timeout; mark as not connected on failure
					if rule.ToAddress != "" && rule.ToPort != "" {
						addr := net.JoinHostPort(rule.ToAddress, rule.ToPort)
						conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
						if err != nil {
							serviceName = "timeout:" + time.Now().Format(utils.T_YYYYMMDD_HHmmss)
						} else {
							serviceName = "alive:" + time.Now().Format(utils.T_YYYYMMDD_HHmmss)
							_ = conn.Close()
						}
					}
				}
				// Sanitize sensitive fields if any (none in this case)
				data = append(data, map[string]interface{}{
					"id":                  rule.ID,
					"service_name":        serviceName,
					"service_path":        processPath,
					"is_service":          isService,
					"service_status":      status,
					"service_started_at":  startedAt.Format(utils.T_YYYYMMDD_HHmmss),
					"service_description": description,
					"device_id":           rule.DeviceID,
					"chain":               rule.Chain,
					"dst_address":         rule.DstAddress,
					"dst_port":            rule.DstPort,
					"to_address":          rule.ToAddress,
					"to_port":             rule.ToPort,
					"action":              rule.Action,
					"comment":             rule.Comment,
					"timestamp":           rule.Timestamp,
				})
			}
			// Always update the map, even with empty data to ensure consistency
			logrus.Warnf("[NATRULES_CACHE] Storing %d processed rules for device %s", len(data), NATDeviceID)
			if len(data) > 0 {
				RWMutexNATRulesData.Lock()
				NATRulesData[NATDeviceID] = data
				RWMutexNATRulesData.Unlock()
			}
		}
		go func() {
			// marshal and write to a temp file then atomically rename
			RWMutexNATRulesData.RLock()
			allData := make(map[string][]map[string]interface{})
			maps.Copy(allData, NATRulesData)
			RWMutexNATRulesData.RUnlock()

			// Filter out devices with no rules (empty arrays)
			dataToSave := make(map[string][]map[string]interface{})
			for deviceID, rules := range allData {
				if len(rules) > 0 {
					dataToSave[deviceID] = rules
					logrus.Warnf("[NATRULES_CACHE] Device %s has %d rules, will be saved", deviceID, len(rules))
				} else {
					logrus.Warnf("[NATRULES_CACHE] Device %s has no rules, skipping from cache", deviceID)
				}
			}

			logrus.Warnf("[NATRULES_CACHE] Preparing to save cache with %d devices (filtered from %d total)", len(dataToSave), len(allData))

			// Only proceed with saving if there's at least one device with rules
			if len(dataToSave) == 0 {
				logrus.Debug("[NATRULES_CACHE] No devices with rules found, skipping cache save")
				return
			}

			b, err := json.MarshalIndent(dataToSave, "", "  ")
			if err != nil {
				logrus.WithError(err).Error("failed to marshal nat rules snapshot")
				return
			}

			if err := os.MkdirAll("./.cache", 0755); err != nil {
				logrus.WithError(err).Error("failed to create cache directory")
				return
			}

			tmpPath := "./.cache/nat_rule_data.json.tmp"
			if err := os.WriteFile(tmpPath, b, 0644); err != nil {
				logrus.WithError(err).Error("failed to write nat rules temp file")
				return
			}

			if err := os.Rename(tmpPath, cachePath); err != nil {
				logrus.WithError(err).Error("failed to rename nat rules file")
				return
			}

			logrus.Debug("nat rules snapshot saved to " + cachePath)
		}()

		select {
		case <-NatRuleUpdateChannelTrigger:
			logrus.Println("[info][NATRULE] trigger received, skipping sleep and running immediately")
			continue
		case <-time.After(20 * time.Minute):
			// lanjut otomatis ke iterasi berikutnya
		}
	}
}

// Helper function to send JSON responses
func sendResponse(c *gin.Context, data interface{}, err error) {
	response := &model.APIResponse{
		Timestamp: time.Now(),
	}

	if err != nil {
		if data != nil {
			response.Data = data
		}
		response.Success = false
		response.Error = err.Error()
		c.JSON(http.StatusInternalServerError, response)
	} else {
		response.Success = true
		response.Data = data
		c.JSON(http.StatusOK, response)
	}
}

// Device management handlers
func HandleGetDevices(c *gin.Context) {
	if monitor.MikrotikMultiService == nil {
		sendResponse(c, nil, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	configs := monitor.MikrotikMultiService.GetDeviceConfigs()

	// Remove sensitive information (passwords)
	safeConfigs := make([]*model.MikroTikConfig, len(configs))
	for i, config := range configs {
		safeConfigs[i] = &model.MikroTikConfig{
			ID:   config.ID,
			Host: config.Host,
			User: config.User,
			Name: config.Name,
			// Pass is omitted for security
		}
	}

	sendResponse(c, safeConfigs, nil)
}

func HandleGetDeviceNATRules(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		sendResponse(c, nil, fmt.Errorf("device ID is required"))
		return
	}
	go func() {
		NatRuleUpdateChannelTrigger <- struct{}{}
	}()
	var data []map[string]interface{}
	RWMutexNATRulesData.RLock()
	NATRules, ok := NATRulesData[deviceID]
	logrus.Println("NATRulesData")
	logrus.Println(NATRulesData)
	RWMutexNATRulesData.RUnlock()
	if ok {
		data = NATRules
	} else {
		if monitor.MikrotikMultiService == nil {
			sendResponse(c, nil, fmt.Errorf("MikroTik multi-service is not initialized"))
			return
		}
		rules := monitor.MikrotikMultiService.GetCurrentNATRules(deviceID)
		for _, rule := range rules {
			serviceName := "unreachable"
			processPath := ""
			isService := false
			status := ""
			description := ""
			var startedAt time.Time
			if serverServices, ok := model.ServerServices[rule.ToAddress]; ok && rule.Chain == "dstnat" {
				found := false
				serviceName = "not found"
				for _, service := range serverServices {
					for ipPort := range strings.SplitSeq(service.IPAddressPort, ",") {
						if ipPorts := strings.Split(ipPort, ":"); len(ipPorts) == 3 {
							protocol := ipPorts[0]
							ip := ipPorts[1]
							port := ipPorts[2]
							if port == rule.ToPort {
								// switch ip {
								// case "[::]", "*", "0.0.0.0":
								// 	serviceName = service.Name + " (" + protocol + ", all)"
								// default:
								// }
								serviceName = service.Name + " (" + protocol + ", " + ip + ")"
								processPath = service.Process
								isService = service.IsService
								status = service.Status
								startedAt = service.StartedAt
								description = service.Description
								found = true
								break
							}
						}
					}
					if found {
						break
					}
				}
			}
			// Sanitize sensitive fields if any (none in this case)
			data = append(data, map[string]interface{}{
				"id":                  rule.ID,
				"service_name":        serviceName,
				"service_path":        processPath,
				"is_service":          isService,
				"service_status":      status,
				"service_started_at":  startedAt.Format(utils.T_YYYYMMDD_HHmmss),
				"service_description": description,
				"device_id":           rule.DeviceID,
				"chain":               rule.Chain,
				"dst_address":         rule.DstAddress,
				"dst_port":            rule.DstPort,
				"to_address":          rule.ToAddress,
				"to_port":             rule.ToPort,
				"action":              rule.Action,
				"comment":             rule.Comment,
				"timestamp":           rule.Timestamp,
			})
		}
	}

	sendResponse(c, data, nil)
}

func HandleGetDeviceStatus(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		sendResponse(c, nil, fmt.Errorf("device ID is required"))
		return
	}
	if monitor.MikrotikMultiService == nil {
		sendResponse(c, nil, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	status := monitor.MikrotikMultiService.GetDeviceStatus(deviceID)
	if status == nil {
		sendResponse(c, nil, fmt.Errorf("device not found: %s", deviceID))
		return
	}

	sendResponse(c, status, nil)
}

// Multi-device handlers
func HandleGetAllNATRules(c *gin.Context) {
	if monitor.MikrotikMultiService == nil {
		sendResponse(c, nil, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	allRules := monitor.MikrotikMultiService.GetAllNATRules()

	// Return direct format without nat_rules wrapper
	c.JSON(http.StatusOK, model.NatRulesResponse(allRules))
}

func HandleGetAllStatus(c *gin.Context) {
	if monitor.MikrotikMultiService == nil {
		data := gin.H{
			"devices":       0,
			"total_devices": 0,
			"timestamp":     time.Now(),
		}
		sendResponse(c, data, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	status := monitor.MikrotikMultiService.GetAllDevicesStatus()
	sendResponse(c, status, nil)
}

func HandleHealth(c *gin.Context) {
	if monitor.MikrotikMultiService == nil {
		health := gin.H{
			"status":         "ok",
			"timestamp":      time.Now(),
			"total_devices":  0,
			"online_devices": 0,
			"health_ratio":   fmt.Sprintf("%d/%d", 0, 0),
		}
		sendResponse(c, health, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	status := monitor.MikrotikMultiService.GetAllDevicesStatus()

	totalDevices := status.TotalDevices
	onlineDevices := 0
	for _, deviceStatus := range status.Devices {
		if deviceStatus.IsRunning {
			onlineDevices++
		}
	}

	health := gin.H{
		"status":         "ok",
		"timestamp":      time.Now(),
		"total_devices":  totalDevices,
		"online_devices": onlineDevices,
		"health_ratio":   fmt.Sprintf("%d/%d", onlineDevices, totalDevices),
	}
	sendResponse(c, health, nil)
}

// Backward compatibility handlers (use first device)
func HandleGetNATRulesCount(c *gin.Context) {
	if monitor.MikrotikMultiService == nil {
		sendResponse(c, nil, fmt.Errorf("MikroTik multi-service is not initialized"))
		return
	}
	configs := monitor.MikrotikMultiService.GetDeviceConfigs()
	if len(configs) == 0 {
		sendResponse(c, gin.H{"count": 0}, nil)
		return
	}

	rules := monitor.MikrotikMultiService.GetCurrentNATRules(configs[0].ID)
	count := gin.H{"count": len(rules)}
	sendResponse(c, count, nil)
}

func HandleGetBandwidth(c *gin.Context) {
	// For backward compatibility, return empty for now
	// This could be enhanced to aggregate data from all devices
	sendResponse(c, gin.H{}, nil)
}

func HandleGetDailyBandwidth(c *gin.Context) {
	// For backward compatibility, return empty for now
	sendResponse(c, gin.H{}, nil)
}

func HandleGetMonthlyBandwidth(c *gin.Context) {
	// For backward compatibility, return empty for now
	sendResponse(c, gin.H{}, nil)
}
