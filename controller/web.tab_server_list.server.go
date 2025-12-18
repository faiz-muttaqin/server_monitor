package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"server_monitor/database"
	"server_monitor/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func PostServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Draw       int    `form:"draw"`
			Start      int    `form:"start"`
			Length     int    `form:"length"`
			Search     string `form:"search[value]"`
			SortColumn int    `form:"order[0][column]"`
			SortDir    string `form:"order[0][dir]"`
		}

		// Bind form data to request struct
		if err := c.Bind(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// fmt.Println("request.CreatedAt")
		// fmt.Println(request.CreatedAt)
		// fmt.Println("request.UpdatedAt")
		// fmt.Println(request.UpdatedAt)

		t := reflect.TypeOf(model.Server{})

		// Initialize the map
		columnMap := make(map[int]string)

		// Loop through the fields of the struct
		colNum := 0
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			// Get the JSON key
			jsonKey := field.Tag.Get("json")
			if jsonKey == "" {
				continue
			}
			columnMap[colNum] = jsonKey
			colNum++
		}

		// Get the column name based on SortColumn value
		sortColumnName := columnMap[request.SortColumn]
		orderString := fmt.Sprintf("%s %s", sortColumnName, request.SortDir)
		// Count the total number of records
		var totalRecords int64
		// Count the number of filtered records
		var filteredRecords int64
		var data []gin.H
		if database.DB != nil {
			// Initial query for filtering
			filteredQuery := database.DB.Model(&model.Server{})

			// // Apply filters
			if request.Search != "" {
				// var querySearch []string
				// var querySearchParams []interface{}

				for i := 0; i < t.NumField(); i++ {
					dataField := ""
					field := t.Field(i)
					// Get the variable name
					// varName := field.Name
					// Get the data type
					dataType := field.Type.String()
					// Get the JSON key
					jsonKey := field.Tag.Get("json")
					// Get the GORM tag
					gormTag := field.Tag.Get("gorm")

					// Initialize a variable to hold the column key
					columnKey := ""

					// Manually parse the gorm tag to find the column value
					tags := strings.Split(gormTag, ";")
					for _, tag := range tags {
						if strings.HasPrefix(tag, "column:") {
							columnKey = strings.TrimPrefix(tag, "column:")
							break
						}
					}
					if jsonKey == "" || jsonKey == "-" {
						if columnKey == "" || columnKey == "-" {
							continue
						} else {
							dataField = columnKey
						}
					} else {
						dataField = jsonKey
					}
					if jsonKey == "" {
						continue
					}
					if dataType != "string" {
						continue
					}
					// fmt.Printf("Variable Name: %s, Data Type: %s, JSON Key: %s, GORM Column Key: %s\n", varName, dataType, jsonKey, columnKey)

					filteredQuery = filteredQuery.Or("`"+dataField+"` LIKE ?", "%"+request.Search+"%")

				}

			} else {
				for i := 0; i < t.NumField(); i++ {
					field := t.Field(i)
					formKey := field.Tag.Get("json")
					if formKey == "" || formKey == "-" {
						continue
					}
					formValue := c.PostForm(formKey)
					if formValue != "" {
						filteredQuery = filteredQuery.Debug().Or("`"+formKey+"` LIKE ?", "%"+formValue+"%")
					}
				}

			}

			database.DB.Model(&model.Server{}).Count(&totalRecords)

			filteredQuery.Count(&filteredRecords)

			// Apply sorting and pagination to the filtered query
			query := filteredQuery.Order(orderString)
			var DeviceManufactures []model.Server
			query = query.Offset(request.Start).Limit(request.Length).Find(&DeviceManufactures)

			if query.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"draw":            request.Draw,
					"recordsTotal":    totalRecords,
					"recordsFiltered": 0,
					"data":            []gin.H{},
					"error":           query.Error.Error(),
				})
				return
			}

			for _, person := range DeviceManufactures {
				newData := make(map[string]interface{})

				v := reflect.ValueOf(person)

				for i := 0; i < t.NumField(); i++ {
					field := t.Field(i)
					fieldValue := v.Field(i)

					// Get the JSON key
					theKey := field.Tag.Get("json")
					if theKey == "" {
						theKey = field.Tag.Get("form")
						if theKey == "" {
							continue
						}
					}

					// Handle time.Time fields differently
					if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
						newData[theKey] = fieldValue.Interface().(time.Time).Format("2006-01-02 15:04:05")
					} else {
						newData[theKey] = fieldValue.Interface()
					}
				}

				data = append(data, gin.H(newData))
			}
		} else {
			var servers []model.Server
			totalRecords = int64(len(model.ServerCache))
			i := 0
			for _, us := range model.ServerCache {
				i++
				if i-1 < request.Start {
					continue
				}
				servers = append(servers, *us)

				if len(servers) >= request.Length {
					break
				}
			}
			c.JSON(http.StatusOK, gin.H{
				"draw":            request.Draw,
				"recordsTotal":    totalRecords,
				"recordsFiltered": filteredRecords,
				"data":            servers,
			})
			return

		}

		// Respond with the formatted data for DataTables
		c.JSON(http.StatusOK, gin.H{
			"draw":            request.Draw,
			"recordsTotal":    totalRecords,
			"recordsFiltered": filteredRecords,
			"data":            data,
		})
	}
}
