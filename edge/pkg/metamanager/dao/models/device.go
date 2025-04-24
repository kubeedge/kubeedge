package models

type Device struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"type:text"`
	Description string `gorm:"type:text"`
	State       string `gorm:"type:text"`
	LastOnline  string `gorm:"type:text"`
}

func (Device) TableName() string {
	return DeviceTableName
}

type DeviceTwin struct {
	ID              int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID        string `gorm:"column:deviceid"`
	Name            string `gorm:"column:name"`
	Description     string `gorm:"column:description"`
	Expected        string `gorm:"column:expected"`
	Actual          string `gorm:"column:actual"`
	ExpectedMeta    string `gorm:"column:expected_meta"`
	ActualMeta      string `gorm:"column:actual_meta"`
	ExpectedVersion string `gorm:"column:expected_version"`
	ActualVersion   string `gorm:"column:actual_version"`
	Optional        bool   `gorm:"column:optional"`
	AttrType        string `gorm:"column:attr_type"`
	Metadata        string `gorm:"column:metadata"`
}

func (DeviceTwin) TableName() string {
	return DeviceTwinTableName
}

type DeviceAttr struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID    string `gorm:"column:deviceid"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
	Value       string `gorm:"column:value"`
	Optional    bool   `gorm:"column:optional"`
	AttrType    string `gorm:"column:attr_type"`
	Metadata    string `gorm:"column:metadata"`
}

func (DeviceAttr) TableName() string {
	return DeviceAttrTableName
}

type DeviceDelete struct {
	DeviceID string
	Name     string
}

type DeviceAttrUpdate struct {
	DeviceID string
	Name     string
	Cols     map[string]interface{}
}

type DeviceUpdate struct {
	DeviceID string
	Cols     map[string]interface{}
}

type DeviceTwinUpdate struct {
	DeviceID string
	Name     string
	Cols     map[string]interface{}
}
