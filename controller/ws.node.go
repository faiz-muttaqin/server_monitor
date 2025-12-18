package controller

import (
	"server_monitor/utils"
	"server_monitor/ws"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func WsNodeServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("key")
		if key != utils.SERVER_NODE_AUTH_KEY {
			logrus.Error("Key header is missing")
			c.JSON(400, gin.H{"error": "Key header is required"})
			return
		}
		id := c.GetHeader("id")
		ws.HandleWebSocketNode(c.Writer, c.Request, id)
	}
}
