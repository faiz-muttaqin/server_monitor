package controller

import (
	"net/http"
	"os"
	"server_monitor/kvstore"

	"github.com/gin-gonic/gin"
)

func AuthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := os.Getenv("AUTH")
		if auth != "" {
			cookie, err := c.Cookie(COOKIE_NAME)
			if err != nil || cookie == "" {
				// Redirect to /login
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			username, err := kvstore.GetKey(cookie)
			if err != nil || username == "" {
				// Redirect to /login
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
