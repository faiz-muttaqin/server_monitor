package model

type Menu struct {
	Create    int8   `json:"create" gorm:"column:create"`
	Read      int8   `json:"read" gorm:"column:read"`
	Update    int8   `json:"update" gorm:"column:update"`
	Delete    int8   `json:"delete" gorm:"column:delete"`
	ParentID  uint   `json:"parent_id" gorm:"column:parent_id"`
	Title     string `json:"title" gorm:"column:title"`
	Path      string `json:"path" gorm:"column:path"`
	MenuOrder uint   `json:"menu_order" gorm:"column:menu_order"`
	Status    uint   `json:"status" gorm:"column:status"`
	Level     uint   `json:"level" gorm:"column:level"`
	Icon      string `json:"icon" gorm:"column:icon"`
}

var Menus = []Menu{
	{ParentID: 0, Title: "Dashboard", Path: "tab-dashboard", MenuOrder: 1, Status: 1, Level: 0, Icon: "bx-home", Create: 1, Read: 1, Update: 1, Delete: 1},
	{ParentID: 0, Title: "Server List", Path: "tab-server-list", MenuOrder: 2, Status: 1, Level: 0, Icon: "bx-server", Create: 1, Read: 1, Update: 1, Delete: 1},
	{ParentID: 0, Title: "Network Routes", Path: "tab-network-routes", MenuOrder: 3, Status: 1, Level: 0, Icon: "bx-network-chart", Create: 1, Read: 1, Update: 1, Delete: 1},
}
