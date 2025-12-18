package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"server_monitor/kvstore"
	"server_monitor/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var COOKIE_NAME = "credentials"

func LoginPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := os.Getenv("AUTH")
		if auth == "" {
			c.Redirect(302, "/")
			return
		}
		c.HTML(200, "login.html", gin.H{
			"title": "Login",
		})
	}
}
func LoginPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := os.Getenv("AUTH")
		if auth == "" {
			c.Redirect(302, "/")
			return
		}
		fmt.Println("auth")
		fmt.Println("auth")
		fmt.Println("auth")
		fmt.Println("auth")
		fmt.Println("auth")
		fmt.Println(auth)
		var req struct {
			Username   string `json:"username" binding:"required"`
			Password   string `json:"password" binding:"required"`
			RememberMe bool   `json:"remember_me"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid request payload",
			})
			return
		}
		var authData []string
		err := json.Unmarshal([]byte(auth), &authData)
		if err != nil || len(authData) == 0 {
			c.JSON(500, gin.H{
				"error": "Invalid AUTH configuration" + err.Error(),
			})
			return
		}
		for _, cred := range authData {
			parts := strings.SplitN(cred, ":", 2)
			if len(parts) != 2 {
				continue
			}
			if parts[0] == req.Username && parts[1] == req.Password {
				// Set session or token here
				RandomString := utils.GenerateRandomString(64)
				if req.RememberMe {
					// Set a longer expiration for the cookie
					c.SetCookie(COOKIE_NAME, RandomString, 7*24*3600, "/", "", false, true)
					kvstore.SetKey(RandomString, req.Username, 7*24*time.Hour)
				} else {
					// Set session or token here
					c.SetCookie(COOKIE_NAME, RandomString, 3600, "/", "", false, true)
					kvstore.SetKey(RandomString, req.Username, 1*time.Hour)
				}
				c.JSON(200, gin.H{
					"message": "Login successful",
				})
				return
			}
		}
		c.JSON(401, gin.H{
			"error": "Invalid username or password",
		})

	}
}
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(COOKIE_NAME)
		if err == nil && cookie != "" {
			kvstore.DeleteKey(cookie)
		}
		c.SetCookie(COOKIE_NAME, "", -1, "/", "", false, false)
		c.JSON(200, gin.H{
			"message": "Logout successful",
		})
	}
}
