package controller

import (
	"net/http"

	"server_monitor/model"
	"server_monitor/monitor"

	"github.com/gin-gonic/gin"
)

// MonitorController handles monitoring API endpoints
type MonitorController struct{}

// GetServerStatus returns current server status
func (mc *MonitorController) GetServerStatus(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	server, exists := model.ServerCache[serverID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Server not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   server,
	})
}

// GetAllServers returns all cached servers
func (mc *MonitorController) GetAllServers(c *gin.Context) {
	servers := monitor.GetAllCachedServers()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(servers),
		"data":   servers,
	})
}

// GetServersSummary returns summary of all servers
func (mc *MonitorController) GetServersSummary(c *gin.Context) {
	servers := monitor.GetAllCachedServers()

	summary := struct {
		TotalServers   int     `json:"total_servers"`
		OnlineServers  int     `json:"online_servers"`
		OfflineServers int     `json:"offline_servers"`
		ErrorServers   int     `json:"error_servers"`
		AvgCPUUsage    float64 `json:"avg_cpu_usage"`
		AvgMemoryUsage float64 `json:"avg_memory_usage"`
	}{
		TotalServers: len(servers),
	}

	var totalCPU, totalMemory float64
	for _, server := range servers {
		switch server.Status {
		case "online":
			summary.OnlineServers++
		case "offline":
			summary.OfflineServers++
		case "error":
			summary.ErrorServers++
		}
		totalCPU += server.CPUUsage
		totalMemory += server.MemoryUsage
	}

	if len(servers) > 0 {
		summary.AvgCPUUsage = totalCPU / float64(len(servers))
		summary.AvgMemoryUsage = totalMemory / float64(len(servers))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}

// UpdateServerData allows partial updates to server data
func (mc *MonitorController) UpdateServerData(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Server ID is required",
		})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}

	if err := model.UpdateServerCache(serverID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get updated server data
	server, exists := model.ServerCache[serverID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Server not found after update",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Server data updated successfully",
		"data":    server,
	})
}
