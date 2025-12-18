package routes

import (
	"server_monitor/controller"
	"server_monitor/model"

	"github.com/gin-gonic/gin"
)

var R *gin.Engine

func Routes() {
	// Initialize controllers
	monitorController := &controller.MonitorController{}
	R.GET("/login", controller.LoginPage())
	R.POST("/login", controller.LoginPost())
	R.DELETE("/login", controller.Logout())
	R.GET("/logout", controller.Logout())

	authRoute := R.Group("", controller.AuthCheck())
	{
		// Main routes
		authRoute.GET("/", controller.Index())
		authRoute.GET("/esxi/table", controller.GetEsxiDataTable())
		authRoute.GET("/esxi/table/:hostId", controller.GetServerSpecificData())

		authRoute.GET("/servers/table", controller.GetServerList())
		authRoute.GET("/servers/table/:hostId", controller.GetServerSpecificData())
		authRoute.PATCH("/servers/table", controller.UpdateServerList())
		authRoute.GET("/services", func(c *gin.Context) {
			// var data []*model.ServerService
			// for _, service := range model.ServerServices {
			// 	data = append(data, service...)
			// }
			c.JSON(200, gin.H{
				"total":    len(model.ServerServices),
				"services": model.ServerServices,
			})
		})
		authRoute.GET("/services/:ip", func(c *gin.Context) {
			ip := c.Param("ip")
			var data []*model.ServerService
			if serverServices, ok := model.ServerServices[ip]; ok {
				data = append(data, serverServices...)
			}
			c.JSON(200, gin.H{
				"total":    len(data),
				"services": data,
			})
		})
		// API routes for monitoring
		api := authRoute.Group("/api/v1")
		{
			// Monitor endpoints
			monitor := api.Group("/monitor")
			{
				monitor.GET("/servers", monitorController.GetAllServers)
				monitor.GET("/servers/summary", monitorController.GetServersSummary)
				monitor.GET("/servers/:id", monitorController.GetServerStatus)
				monitor.PUT("/servers/:id", monitorController.UpdateServerData)
			}

			// MIKROTIK ROUTES
			{
				// Device management endpoints
				api.GET("/devices", controller.HandleGetDevices)
				api.GET("/devices/:id/nat-rules", controller.HandleGetDeviceNATRules)
				api.GET("/devices/:id/status", controller.HandleGetDeviceStatus)

				// Multi-device endpoints
				api.GET("/nat-rules", controller.HandleGetAllNATRules)
				api.GET("/status", controller.HandleGetAllStatus)
				api.GET("/health", controller.HandleHealth)

				// Backward compatibility endpoints (use first device)
				api.GET("/nat-rules/count", controller.HandleGetNATRulesCount)
				api.GET("/bandwidth", controller.HandleGetBandwidth)
				api.GET("/bandwidth/daily", controller.HandleGetDailyBandwidth)
				api.GET("/bandwidth/monthly", controller.HandleGetMonthlyBandwidth)
			}

		}
		web := authRoute.Group("/web/:access")
		{
			web.GET("/components/:component", controller.ComponentPage())
			tsl := web.Group("/tab-server-list")
			{
				tsl.POST("/server/table", controller.PostServer())
				// tsl.PATCH("/server/table", controller.UpdateServerList())
				// tsl.GET("/server/:id", controller.GetServerSpecificData())
			}
		}

	}

	// WebSocket routes
	R.GET("/ws", controller.WebSocketVerify())
	R.GET("/ws/node", controller.WsNodeServer())
}
