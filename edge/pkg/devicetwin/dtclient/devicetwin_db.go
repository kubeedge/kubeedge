package dtclient

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// DeviceTwin the struct of device twin
type DeviceTwin struct {
	ID              int64  `orm:"column(id);size(64);auto;pk"`
	DeviceID        string `orm:"column(deviceid); null; type(text)"`
	Name            string `orm:"column(name);null;type(text)"`
	Description     string `orm:"column(description);null;type(text)"`
	Expected        string `orm:"column(expected);null;type(text)"`
	Actual          string `orm:"column(actual);null;type(text)"`
	ExpectedMeta    string `orm:"column(expected_meta);null;type(text)"`
	ActualMeta      string `orm:"column(actual_meta);null;type(text)"`
	ExpectedVersion string `orm:"column(expected_version);null;type(text)"`
	ActualVersion   string `orm:"column(actual_version);null;type(text)"`
	Optional        bool   `orm:"column(optional);null;type(integer)"`
	AttrType        string `orm:"column(attr_type);null;type(text)"`
	Metadata        string `orm:"column(metadata);null;type(text)"`
}

// SaveDeviceTwin save device twin
func SaveDeviceTwin(o orm.Ormer, doc *DeviceTwin) error {
	err := o.DoTx(func(_ context.Context, txOrm orm.TxOrmer) error {
		// insert data
		// Using txOrm to execute SQL
		_, e := txOrm.Insert(doc)
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})
	if err != nil {
		klog.Errorf("Something wrong when insert DeviceTwin data: %v", err)
		return err
	}
	klog.V(4).Info("insert DeviceTwin data successfully")
	return nil
}

// DeleteDeviceTwinByDeviceID delete device twin
func DeleteDeviceTwinByDeviceID(o orm.Ormer, deviceID string) error {
	err := o.DoTx(func(_ context.Context, txOrm orm.TxOrmer) error {
		// Delete data
		// Using txOrm to execute SQL
		_, e := txOrm.QueryTable(DeviceTwinTableName).Filter("deviceid", deviceID).Delete()
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})

	if err != nil {
		klog.Errorf("Something wrong when deleting Device data: %v", err)
		return err
	}
	klog.V(4).Info("Delete Device data successfully")
	return nil
}

// DeleteDeviceTwin delete device twin
func DeleteDeviceTwin(o orm.Ormer, deviceID string, name string) error {
	err := o.DoTx(func(_ context.Context, txOrm orm.TxOrmer) error {
		// Delete data
		// Using txOrm to execute SQL
		_, e := txOrm.QueryTable(DeviceTwinTableName).Filter("deviceid", deviceID).Filter("name", name).Delete()
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})

	if err != nil {
		klog.Errorf("Something wrong when deleting Device Twin data: %v", err)
		return err
	}
	klog.V(4).Info("Delete Device Twin data successfully")
	return nil
}

// UpdateDeviceTwinField update special field
func UpdateDeviceTwinField(deviceID string, name string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("deviceid", deviceID).Filter("name", name).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateDeviceTwinFields update special fields
func UpdateDeviceTwinFields(o orm.Ormer, deviceID string, name string, cols map[string]interface{}) error {
	num, err := o.QueryTable(DeviceTwinTableName).Filter("deviceid", deviceID).Filter("name", name).Update(cols)
	klog.V(4).Infof("Update affected Num: %d, %v", num, err)
	return err
}

// QueryDeviceTwin query Device
func QueryDeviceTwin(key string, condition string) (*[]DeviceTwin, error) {
	twin := new([]DeviceTwin)
	_, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter(key, condition).All(twin)
	if err != nil {
		return nil, err
	}
	return twin, nil
}

// DeviceTwinUpdate the struct for updating device twin
type DeviceTwinUpdate struct {
	DeviceID string
	Name     string
	Cols     map[string]interface{}
}

// UpdateDeviceTwinMulti update device twin multi
func UpdateDeviceTwinMulti(updates []DeviceTwinUpdate) error {
	for _, update := range updates {
		err := UpdateDeviceTwinFields(dbm.DBAccess, update.DeviceID, update.Name, update.Cols)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeviceTwinTrans transaction of device twin
func DeviceTwinTrans(adds []DeviceTwin, deletes []DeviceDelete, updates []DeviceTwinUpdate) error {
	obm := dbm.DefaultOrmFunc()
	var err error

	for _, add := range adds {
		err = SaveDeviceTwin(obm, &add)
		if err != nil {
			return err
		}
	}

	for _, delete := range deletes {
		err = DeleteDeviceTwin(obm, delete.DeviceID, delete.Name)
		if err != nil {
			return err
		}
	}

	for _, update := range updates {
		err = UpdateDeviceTwinFields(obm, update.DeviceID, update.Name, update.Cols)
		if err != nil {
			return err
		}
	}

	return err
}
