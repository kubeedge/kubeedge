package models

type TargetUrls struct {
	URL string `gorm:"column:url;type:text;primaryKey"`
}

// TableName overrides the table name used by GORM
func (TargetUrls) TableName() string {
	return TargetUrlsName
}
