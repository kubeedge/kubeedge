package models

type SubTopics struct {
	Topic string `gorm:"column:topic;type:text;primaryKey"`
}

// TableName returns the name of the table in the DB
func (SubTopics) TableName() string {
	return SubTopicsName
}
