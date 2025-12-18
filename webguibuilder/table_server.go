package webguibuilder

import (
	"html/template"
	"reflect"
	"server_monitor/model"
	"server_monitor/utils"
	"server_monitor/webgui"
)

func TABLE_SERVER(access_token string) template.HTML {
	// Handling Manufactures
	var tableHeaders []webgui.Column
	tableColumn := model.Server{}
	// Use reflection to get the type of the struct
	t := reflect.TypeOf(tableColumn)
	// Loop through the fields of the struct
	for i := 0; i < t.NumField(); i++ {
		if i == 0 {
			tableHeaders = append(tableHeaders, webgui.Column{Data: "", Header: "", Type: "", Visible: true})
			continue
		}

		field := t.Field(i)
		// Get the variable name
		varName := field.Name
		varName = utils.AddSpaceBeforeUppercase(varName)
		// Get the data type
		dataType := field.Type.String()
		// Get the JSON key
		jsonKey := field.Tag.Get("json")
		if jsonKey == "" || jsonKey == "-" {
			continue
		}
		VISIBLE := true
		switch jsonKey {
		case "open_ports_list":
			continue
		}

		is_column_editable := false
		// switch jsonKey {
		// case "secret_key":
		// 	is_column_editable = true
		// }
		selectSrc := ""

		tableHeaders = append(tableHeaders,
			webgui.Column{
				Data:          jsonKey,
				Header:        template.HTML(varName),
				Type:          dataType,
				Visible:       VISIBLE,
				Editable:      is_column_editable,
				Insertable:    false,
				Orderable:     true,
				SelectableSrc: template.URL(selectSrc),
			},
		)
	}

	templates := webgui.Table(
		`Servers`,
		"dt_server",
		"web/"+access_token+"/tab-server-list/server/table",
		5,
		[]int{5, 10, 25, 50, 100, 200, 500, 1000},
		[]any{[]any{1, "desc"}},
		tableHeaders,
		not(INSERTABLE), not(EDITABLE), not(DELETABLE), not(HIDE_HEADER), not(PASSWORDABLE),
		not(SCROLL_UP_DOWN), SCROLL_LEFT_RIGHT,
		[]string{EXPORT_PRINT, EXPORT_CSV, EXPORT_ALL},
	)
	return templates
}
