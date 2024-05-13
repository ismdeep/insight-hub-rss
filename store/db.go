package store

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	tmpDB, err := gorm.Open(sqlite.Open("data.db"))
	if err != nil {
		panic(err)
	}

	db = tmpDB
	if err := db.AutoMigrate(&Record{}, &Meta{}); err != nil {
		panic(err)
	}
}
