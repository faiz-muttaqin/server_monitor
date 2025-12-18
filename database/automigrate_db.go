package database

import (
	"log"
	"server_monitor/model"

	"gorm.io/gorm"
)

func AutoMigrateWeb(db *gorm.DB) {

	// Run migrations
	if err := db.AutoMigrate(
		&model.Server{},
	); err != nil {
		log.Fatal(err)
	}
}
