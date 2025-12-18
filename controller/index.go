package controller

import (
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"server_monitor/kvstore"
	"server_monitor/model"
	"server_monitor/monitor"
	"server_monitor/utils"
	"server_monitor/wsclient"
	"strings"

	"github.com/gin-gonic/gin"
)

func Index() gin.HandlerFunc {
	return func(c *gin.Context) {

		fileContent := ""

		fileContentTab := ""
		for _, featurePrivilegeParent := range model.Menus {
			if featurePrivilegeParent.Path == "tab-network-routes" {
				if monitor.MikrotikMultiService == nil {
					continue
				}
			}
			fileContentChild := ""
			menuToggle := ""

			if len(strings.TrimSpace(featurePrivilegeParent.Path)) == 0 {
				for _, featurePrivilege := range model.Menus {
					if featurePrivilege.ParentID == featurePrivilegeParent.MenuOrder {
						if featurePrivilege.Create == 0 &&
							featurePrivilege.Read == 0 &&
							featurePrivilege.Update == 0 &&
							featurePrivilege.Delete == 0 {
							continue
						}
						fileContentChild += `        
							<li class="menu-item">
								<a href="#` + featurePrivilege.Path + `" class="menu-link">
									<div class="text-truncate" data-i18n="` + featurePrivilege.Title + `">` + featurePrivilege.Title + `</div>
								</a>
							</li>`
					}
				}

				if len(fileContentChild) > 0 {
					fileContentChild = `<ul class="menu-sub">` + fileContentChild + `</ul>`
					menuToggle = "menu-toggle"
				}
			}

			if featurePrivilegeParent.Level == 0 && featurePrivilegeParent.Status == 1 {
				hrefPath := ""
				if len(featurePrivilegeParent.Path) != 0 {
					if featurePrivilegeParent.Create == 0 &&
						featurePrivilegeParent.Read == 0 &&
						featurePrivilegeParent.Update == 0 &&
						featurePrivilegeParent.Delete == 0 {
						continue
					}
					hrefPath = `href="#` + featurePrivilegeParent.Path + `"`
				} else {
					if len(fileContentChild) == 0 {
						if featurePrivilegeParent.Create == 0 &&
							featurePrivilegeParent.Read == 0 &&
							featurePrivilegeParent.Update == 0 &&
							featurePrivilegeParent.Delete == 0 {
							continue
						}
					}
				}

				fileContent += `
					<li class="menu-item ">
						<a ` + hrefPath + ` class="menu-link ` + menuToggle + `">
							<i class="menu-icon tf-icons bx ` + featurePrivilegeParent.Icon + `"></i>
							<div class="text-truncate" data-i18n="` + featurePrivilegeParent.Title + `">` + featurePrivilegeParent.Title + `</div>
						</a>
						` + fileContentChild + `
					</li>
				`
			}

			if len(featurePrivilegeParent.Path) > 0 {
				fileContentTab += `<div id="` + featurePrivilegeParent.Path + `" class="tab-content flex-grow-1 container-p-y d-none"></div>` //` + string(fileContent) + `
			}
		}

		randomAccessToken := utils.GenerateRandomString(20 + rand.Intn(30) + 1)
		username := "admin"
		if cookie, err := c.Cookie(COOKIE_NAME); err == nil {
			if user, err := kvstore.GetKey(cookie); err == nil && user != "" {
				username = user
			}
		}
		c.HTML(http.StatusOK, "web.html", gin.H{
			"APP_NAME":            utils.Getenv("APP_NAME", "MySever"),
			"APP_LOGO":            utils.Getenv("APP_LOGO", "/assets/self/img/server_monitor.jpg"),
			"APP_VERSION":         utils.Getenv("APP_VERSION", "1.0.0"),
			"APP_VERSION_NO":      utils.Getenv("APP_VERSION_NO", "1"),
			"APP_VERSION_CODE":    utils.Getenv("APP_VERSION_CODE", "1.0.0"),
			"APP_VERSION_NAME":    utils.Getenv("APP_VERSION_NAME", "MyServer"),
			"ACCESS":              "web/" + randomAccessToken + "/",
			"MASTER_HOST":         os.Getenv("MASTER_HOST"),
			"connected_to_master": wsclient.IsWsMasterConnected.Load(),
			"username":            username,
			"role":                "Admin",
			"profile_image":       "https://ui-avatars.com/api/?name=admin&background=random",
			"sidebar":             template.HTML(string(fileContent)),
			"contents":            template.HTML(string(fileContentTab)),
		})
	}
}
