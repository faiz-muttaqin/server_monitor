package model

import (
	"sync"
	"time"
)

var ServerServices = make(map[string][]*ServerService)

var MuServerServices sync.Mutex

type ServerService struct {
	ID            string    `json:"id" gorm:"primaryKey;autoIncrement;comment:Unique identifier Service IPv4 address-PID"`
	PID           int       `json:"pid" gorm:"not null;comment:Process ID"`
	IsService     bool      `json:"is_service" gorm:"size:100;not null;comment:Service name"`
	Process       string    `json:"process" gorm:"size:100;not null;comment:Service process name eg. /home/user/app_name"`
	Name          string    `json:"name" gorm:"size:100;not null;comment:Service name eg. nginx, mysql etc."`
	IPAddressPort string    `json:"ip_address_port" gorm:"size:100;not null;comment:Service IP address and Port eg. 0.0.0.0:8080,[::]:80"`
	Status        string    `json:"status" gorm:"size:20;not null;comment:Service status"`
	StartedAt     time.Time `json:"started_at" gorm:"comment:Timestamp of when the service was started"`
	Description   string    `json:"description" gorm:"size:255;comment:Service description"`
}
