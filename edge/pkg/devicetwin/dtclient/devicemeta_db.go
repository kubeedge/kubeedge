package dtclient

import (
	"strings"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// DeviceMeta device metadata object
type DeviceMeta struct {
	Key   string `orm:"column(key); size(256); pk"`
	Value string `orm:"column(value); null; type(text)"`
}

// SaveDeviceMeta save device meta to db
func SaveDeviceMeta(meta *DeviceMeta) error {
	num, err := dbm.DBAccess.Insert(meta)
	klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
	if err == nil || IsNonUniqueNameError(err) {
		return nil
	}
	return err
}

// IsNonUniqueNameError tests if the error returned by sqlite is unique.
// It will check various sqlite versions.
func IsNonUniqueNameError(err error) bool {
	str := err.Error()
	if strings.HasSuffix(str, "are not unique") || strings.Contains(str, "UNIQUE constraint failed") || strings.HasSuffix(str, "constraint failed") {
		return true
	}
	return false
}

// DeleteDeviceMetaByKey delete meta by key
func DeleteDeviceMetaByKey(key string) error {
	num, err := dbm.DBAccess.QueryTable(DeviceMetaTableName).Filter("key", key).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return err
}

// UpdateDeviceMeta update meta
func UpdateDeviceMeta(meta *DeviceMeta) error {
	num, err := dbm.DBAccess.Update(meta) // will update all field
	klog.V(4).Infof("Update affected Num: %d, %v", num, err)
	return err
}

// InsertOrUpdate insert or update meta
func InsertOrUpdate(meta *DeviceMeta) error {
	_, err := dbm.DBAccess.Raw("INSERT OR REPLACE INTO meta (key, value) VALUES (?,?)", meta.Key, meta.Value).Exec() // will update all field
	klog.V(4).Infof("Update result %v", err)
	return err
}

// UpdateDeviceMetaField update special field
func UpdateDeviceMetaField(key string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceMetaTableName).Filter("key", key).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %v", num, err)
	return err
}

// UpdateDeviceMetaFields update special fields
func UpdateDeviceMetaFields(key string, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceMetaTableName).Filter("key", key).Update(cols)
	klog.V(4).Infof("Update affected Num: %d, %v", num, err)
	return err
}

// QueryDeviceMeta return only meta's value, if no error, Meta not null
func QueryDeviceMeta(key string, condition string) (*[]string, error) {
	meta := new([]DeviceMeta)
	_, err := dbm.DBAccess.QueryTable(DeviceMetaTableName).Filter(key, condition).All(meta)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, v := range *meta {
		result = append(result, v.Value)
	}
	return &result, nil
}

// QueryAllDeviceMeta return all meta, if no error, Meta not null
func QueryAllDeviceMeta(key string, condition string) (*[]DeviceMeta, error) {
	meta := new([]DeviceMeta)
	_, err := dbm.DBAccess.QueryTable(DeviceMetaTableName).Filter(key, condition).All(meta)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

//AddDeviceMetaTrans the transaction of add device meta
func AddDeviceMetaTrans(devices []DeviceMeta) error {
	var err error
	obm := dbm.DBAccess
	obm.Begin()
	for _, d := range devices {
		err = SaveDeviceMeta(&d)
		if err != nil {
			klog.Errorf("save device meta failed: %v", err)
			obm.Rollback()
			return err
		}
	}
	obm.Commit()
	return nil
}

//DeleteDeviceMetaTrans the transaction of delete device meta
func DeleteDeviceMetaTrans(keys []string) error {
	var err error
	obm := dbm.DBAccess
	obm.Begin()
	for _, key := range keys {
		err = DeleteDeviceMetaByKey(key)
		if err != nil {
			obm.Rollback()
			return err
		}
	}
	obm.Commit()
	return nil
}
