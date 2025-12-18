package controller

import (
	"os"
	"server_monitor/model"
	"server_monitor/webguibuilder"

	"github.com/gin-gonic/gin"
)

func ComponentPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		componentID := c.Param("component")
		access_token_param := c.Param("access")
		validComponent := false
		for _, m := range model.Menus {
			if m.Path == componentID {
				validComponent = true
				break
			}
		}
		if validComponent {
			replacements := map[string]any{
				"APP_NAME":         os.Getenv("APP_NAME"),
				"APP_LOGO":         os.Getenv("APP_LOGO"),
				"APP_VERSION":      os.Getenv("APP_VERSION"),
				"APP_VERSION_NO":   os.Getenv("APP_VERSION_NO"),
				"APP_VERSION_CODE": os.Getenv("APP_VERSION_CODE"),
				"APP_VERSION_NAME": os.Getenv("APP_VERSION_NAME"),
				"TABLE_SERVER":     webguibuilder.TABLE_SERVER(access_token_param),
				"TOTAL_SERVERS":    len(model.ServerCache),
				// "TABLE_SERVER_LIST": webguibuilder.TABLE_TRANSACTION_LIST(access_token_param),
				"TOTAL_ESXI": len(os.Getenv("ESXI")),
			}
			c.HTML(200, componentID+".html", replacements)
		} else {
			c.HTML(200, "404.html", nil)
		}
	}
}
