package dtclient

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	//TwinTableName twin table name
	TwinTableName = "twin"
)

// Twin object
type Twin struct {
	// ID    int64  `orm:"pk; auto; column(id)"`
	DeviceID   string `orm:"column(deviceid); size(64); pk"`
	DeviceName string `orm:"column(devicename); null; type(text)"`
	Expected   string `orm:"column(expected); null; type(text)"`
	Actual     string `orm:"column(actual); null; type(text)"`
	Metadata   string `orm:"column(metadata); null; type(text)"`
	LastState  string `orm:"column(laststate); null; type(text)"`
	Attributes string `orm:"column(attributes); null; type(text)"`
	VersionSet string `orm:"column(versionset); null; type(text)"`
}

//SaveTwin  save twin
func SaveTwin(doc *Twin) error {
	num, err := dbm.DBAccess.Insert(doc)
	log.LOGGER.Debugf("Insert affected Num: %d, %s", num, err)
	return err
}

//DeleteTwinByID delete twin
func DeleteTwinByID(id string) error {
	num, err := dbm.DBAccess.QueryTable(TwinTableName).Filter("deviceid", id).Delete()
	if err != nil {
		log.LOGGER.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	log.LOGGER.Debugf("Delete affected Num: %d, %s", num)
	return nil
}

// UpdateTwinField update special field
func UpdateTwinField(deviceID string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(TwinTableName).Filter("deviceid", deviceID).Update(map[string]interface{}{col: value})
	log.LOGGER.Debugf("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateTwinFields update special fields
func UpdateTwinFields(deviceID string, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(TwinTableName).Filter("deviceid", deviceID).Update(cols)
	log.LOGGER.Debugf("Update affected Num: %d, %s", num, err)
	return err
}

// QueryTwin query twin
func QueryTwin(key string, condition string) (*[]Twin, error) {
	twin := new([]Twin)
	_, err := dbm.DBAccess.QueryTable(TwinTableName).Filter(key, condition).All(twin)
	if err != nil {
		return nil, err
	}
	return twin, nil

}

// QueryTwinAll query twin
func QueryTwinAll() (*[]Twin, error) {
	twin := new([]Twin)
	_, err := dbm.DBAccess.QueryTable(TwinTableName).All(twin)
	if err != nil {
		return nil, err
	}
	return twin, nil
}
