package controller

import (
	"server_monitor/utils"
	"server_monitor/ws"

	"github.com/gin-gonic/gin"
)

func WebSocketVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := utils.GenerateRandomHexaString(32)
		ws.HandleWebSocket(c.Writer, c.Request, clientID)
	}
}
