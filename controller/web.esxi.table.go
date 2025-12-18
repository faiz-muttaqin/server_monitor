package controller

import (
	"net/http"
	"server_monitor/monitor"

	"github.com/gin-gonic/gin"
)

func GetEsxiDataTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		if monitor.MultiHostData == nil {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{}})
		} else {
			var data []interface{}
			for _, esxi := range monitor.MultiHostData {
				if esxi != nil {
					data = append(data, *esxi)
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		}
	}
}
func GetEsxiDataEach() gin.HandlerFunc {
	return func(c *gin.Context) {
		hostId := c.Param("hostId")

		// Cari data berdasarkan host ID
		if data, exists := monitor.MultiHostData[hostId]; exists && data != nil {
			c.JSON(http.StatusOK, gin.H{
				"data": *data,
			})
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Host not found",
				"message": "No monitoring data found for host: " + hostId,
			})
		}
	}
}
