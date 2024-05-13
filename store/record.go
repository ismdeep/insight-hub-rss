package store

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

type Record struct {
	ID          string `gorm:"type:varchar(255);primaryKey"`
	Source      string `gorm:"type:text"`
	Author      string `gorm:"type:text"`
	Title       string `gorm:"type:text"`
	Link        string `gorm:"type:text"`
	Content     string `gorm:"type:text"`
	PublishedAt int64  `gorm:"type:bigint"`
}

func init() {
	tmpDB, err := gorm.Open(sqlite.Open("data.db"))
	if err != nil {
		panic(err)
	}

	db = tmpDB
	if err := db.AutoMigrate(&Record{}); err != nil {
		panic(err)
	}
}

func RecordExists(id string) (bool, error) {
	var cnt int64
	if err := db.Model(&Record{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func RecordSave(r Record) error {
	exists, err := RecordExists(r.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return db.Create(&r).Error
}

func RecordRecentList() ([]Record, error) {
	var records []Record
	if err := db.Order("published_at desc").Limit(50).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func RecordRecentListBySource(source string) ([]Record, error) {
	var records []Record
	if err := db.Where("source = ?", source).Order("published_at desc").Limit(50).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func RecordSources() ([]string, error) {
	var sources []string
	if err := db.Model(&Record{}).Select("distinct source").Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}
