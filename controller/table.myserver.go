package controller

import (
	"fmt"
	"net/http"
	"server_monitor/model"
	"server_monitor/utils"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetServerList() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Assuming you expect to get multiple records, so you might want to iterate over the results
		var data []map[string]interface{}
		for _, server := range model.ServerCache {

			logo := "/assets/self/img/ubuntu.svg"
			switch strings.ToLower(server.OS) {
			case "windows":
				logo = "/assets/self/img/windows.svg"
			}
			data = append(data, map[string]interface{}{
				"id":                  server.ID,
				"logo":                logo,
				"is_self":             server.IP == utils.IP,
				"ip":                  server.IP,
				"group_ip":            server.GroupIP,
				"server_name":         server.ServerName,
				"status":              server.Status,
				"uptime_since":        server.UptimeSince,
				"cpu_usage":           fmt.Sprintf("%.2f", server.CPUUsage),
				"cpu_cores":           server.CPUCores,
				"cpu_model":           server.CPUModel,
				"load_average_1":      fmt.Sprintf("%.2f", server.LoadAverage1),
				"load_average_5":      fmt.Sprintf("%.2f", server.LoadAverage5),
				"load_average_15":     fmt.Sprintf("%.2f", server.LoadAverage15),
				"memory_total":        server.MemoryTotal,
				"memory_used":         server.MemoryUsed,
				"memory_free":         server.MemoryFree,
				"memory_available":    server.MemoryAvailable,
				"memory_usage":        fmt.Sprintf("%.2f", server.MemoryUsage),
				"swap_total":          server.SwapTotal,
				"swap_used":           server.SwapUsed,
				"swap_free":           server.SwapFree,
				"disk_total":          server.DiskTotal,
				"disk_used":           server.DiskUsed,
				"disk_read":           server.DiskRead,
				"disk_write":          server.DiskWrite,
				"disk_read_ops":       server.DiskReadOps,
				"disk_write_ops":      server.DiskWriteOps,
				"ssh_connections":     server.SSHConnections,
				"active_users":        server.ActiveUsers,
				"os":                  server.OS,
				"os_version":          server.OSVersion,
				"kernel_version":      server.KernelVersion,
				"architecture":        server.Architecture,
				"temperature":         fmt.Sprintf("%.2f", server.Temperature),
				"use_gui":             server.UseGUI,
				"last_mouse_movement": server.LastMouseMovement.Format(utils.T_YYYYMMDD_HHmmss),
				"last_cli_activity":   server.LastCLIActivity.Format(utils.T_YYYYMMDD_HHmmss),
				"total_port_opens":    server.TotalPortOpens,
				"services_running":    server.ServicesRunning,
				"services_failed":     server.ServicesFailed,
				"last_check_time":     server.LastCheckTime,
				"response_time":       server.ResponseTime,
				"created_at":          server.CreatedAt,
				"updated_at":          server.UpdatedAt,
			})
		}

		// Check if data is empty
		if len(data) == 0 {
			c.SecureJSON(http.StatusOK, gin.H{
				"data": []map[string]interface{}{},
			})
		} else {
			c.SecureJSON(http.StatusOK, gin.H{
				"data": data,
			})
		}

	}
}

func GetServerSpecificData() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the server ID from the URL parameters
		key := c.Param("hostId")
		splitedKey := strings.SplitN(key, "-", 2)
		if len(splitedKey) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key format"})
			return
		}
		field := splitedKey[0]
		id := strings.ReplaceAll(splitedKey[1], "_", ".")

		server, exists := model.ServerCache[id]
		if !exists || server == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
			return
		}
		// Create a copy of the server as a map
		serverMap := make(map[string]interface{})
		serverMap["id"] = server.ID
		serverMap["ip"] = server.IP
		serverMap["group_ip"] = server.GroupIP
		serverMap["server_name"] = server.ServerName
		serverMap["status"] = server.Status
		serverMap["uptime_since"] = server.UptimeSince
		serverMap["cpu_usage"] = server.CPUUsage
		serverMap["cpu_cores"] = server.CPUCores
		serverMap["cpu_model"] = server.CPUModel
		serverMap["load_average_1"] = server.LoadAverage1
		serverMap["load_average_5"] = server.LoadAverage5
		serverMap["load_average_15"] = server.LoadAverage15
		serverMap["memory_total"] = server.MemoryTotal
		serverMap["memory_used"] = server.MemoryUsed
		serverMap["memory_free"] = server.MemoryFree
		serverMap["memory_available"] = server.MemoryAvailable
		serverMap["memory_usage"] = server.MemoryUsage
		serverMap["swap_total"] = server.SwapTotal
		serverMap["swap_used"] = server.SwapUsed
		serverMap["swap_free"] = server.SwapFree
		serverMap["disk_total"] = server.DiskTotal
		serverMap["disk_used"] = server.DiskUsed
		serverMap["disk_read"] = server.DiskRead
		serverMap["disk_write"] = server.DiskWrite
		serverMap["disk_read_ops"] = server.DiskReadOps
		serverMap["disk_write_ops"] = server.DiskWriteOps
		serverMap["ssh_connections"] = server.SSHConnections
		serverMap["active_users"] = server.ActiveUsers
		serverMap["os"] = server.OS
		serverMap["os_version"] = server.OSVersion
		serverMap["kernel_version"] = server.KernelVersion
		serverMap["architecture"] = server.Architecture
		serverMap["temperature"] = server.Temperature
		serverMap["use_gui"] = server.UseGUI
		serverMap["last_mouse_movement"] = server.LastMouseMovement.Format(utils.T_YYYYMMDD_HHmmss)
		serverMap["last_cli_activity"] = server.LastCLIActivity.Format(utils.T_YYYYMMDD_HHmmss)
		serverMap["total_port_opens"] = server.TotalPortOpens
		serverMap["open_ports_list"] = server.OpenPortsList
		serverMap["services_running"] = server.ServicesRunning
		serverMap["services_failed"] = server.ServicesFailed
		serverMap["last_check_time"] = server.LastCheckTime
		serverMap["response_time"] = server.ResponseTime
		serverMap["created_at"] = server.CreatedAt
		serverMap["updated_at"] = server.UpdatedAt

		// Search using map field
		value, ok := serverMap[field]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "Field not found"})
			return
		}

		// If the field is "open_ports_list", convert string to []string, then []int, sort, and return
		if field == "open_ports_list" {
			var ports []int
			switch v := value.(type) {
			case string:
				// Split by comma, trim spaces, convert to int
				for _, p := range strings.Split(v, ",") {
					p = strings.TrimSpace(p)
					p = strings.ReplaceAll(p, "\"", "")
					p = strings.ReplaceAll(p, "]", "")
					p = strings.ReplaceAll(p, "[", "")
					if p == "" {
						continue
					}
					if portNum, err := strconv.Atoi(p); err == nil {
						ports = append(ports, portNum)
					}
				}
			case []string:
				for _, p := range v {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					if portNum, err := strconv.Atoi(p); err == nil {
						ports = append(ports, portNum)
					}
				}
			case []int:
				ports = v
			}
			// Sort ports
			sort.Ints(ports)
			c.JSON(http.StatusOK, gin.H{"key": field, "value": ports})
			return
		}

		c.JSON(http.StatusOK, gin.H{"key": field, "value": value})

	}
}
func UpdateServerList() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		req.Key = strings.TrimSpace(req.Key)
		req.Value = strings.TrimSpace(req.Value)
		splitedKey := strings.SplitN(req.Key, "-", 2)
		if len(splitedKey) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key format"})
			return
		}
		field := splitedKey[0]
		id := strings.ReplaceAll(splitedKey[1], "_", ".")

		server, exists := model.ServerCache[id]
		if !exists || server == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
			return
		}
		err := model.UpdateServerCache(id, map[string]interface{}{field: req.Value})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update server"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Server updated successfully", "key": req.Key, "value": req.Value})
	}
}
