package store

type Meta struct {
	Source   string `gorm:"type:varchar(255);not null;primary_key"`
	Name     string `gorm:"type:varchar(255);not null;"`
	HomePage string `gorm:"type:varchar(255);not null;"`
}

func MetaSave(meta *Meta) error {
	return db.Debug().Save(meta).Error
}

func MetaInfo(source string) (*Meta, error) {
	var meta Meta
	if err := db.Where("source = ?", source).First(&meta).Error; err != nil {
		return nil, err
	}
	return &meta, nil
}
