package dtclient

import (
	"github.com/beego/beego/v2/client/orm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// DeviceAttr the struct of device attributes
type DeviceAttr struct {
	ID          int64  `orm:"column(id);size(64);auto;pk"`
	DeviceID    string `orm:"column(deviceid); null; type(text)"`
	Name        string `orm:"column(name);null;type(text)"`
	Description string `orm:"column(description);null;type(text)"`
	Value       string `orm:"column(value);null;type(text)"`
	Optional    bool   `orm:"column(optional);null;type(integer)"`
	AttrType    string `orm:"column(attr_type);null;type(text)"`
	Metadata    string `orm:"column(metadata);null;type(text)"`
}

// SaveDeviceAttr save device attributes
func SaveDeviceAttr(to orm.TxOrmer, doc *DeviceAttr) error {
	num, err := to.Insert(doc)
	klog.V(4).Infof("Insert affected Num: %d, %s", num, err)
	return err
}

// DeleteDeviceAttrByDeviceID delete device attr
func DeleteDeviceAttrByDeviceID(to orm.TxOrmer, deviceID string) error {
	num, err := to.QueryTable(DeviceAttrTableName).Filter("deviceid", deviceID).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

// DeleteDeviceAttr delete device attr
func DeleteDeviceAttr(to orm.TxOrmer, deviceID string, name string) error {
	num, err := to.QueryTable(DeviceAttrTableName).Filter("deviceid", deviceID).Filter("name", name).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

// UpdateDeviceAttrField update special field
func UpdateDeviceAttrField(deviceID string, name string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceAttrTableName).Filter("deviceid", deviceID).Filter("name", name).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateDeviceAttrFields update special fields
func UpdateDeviceAttrFields(to orm.TxOrmer, deviceID string, name string, cols map[string]interface{}) error {
	num, err := to.QueryTable(DeviceAttrTableName).Filter("deviceid", deviceID).Filter("name", name).Update(cols)
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// QueryDeviceAttr query Device
func QueryDeviceAttr(key string, condition string) (*[]DeviceAttr, error) {
	attrs := new([]DeviceAttr)
	_, err := dbm.DBAccess.QueryTable(DeviceAttrTableName).Filter(key, condition).All(attrs)
	if err != nil {
		return nil, err
	}
	return attrs, nil
}

// DeviceDelete the struct for deleting device
type DeviceDelete struct {
	DeviceID string
	Name     string
}

// DeviceAttrUpdate the struct for updating device attr
type DeviceAttrUpdate struct {
	DeviceID string
	Name     string
	Cols     map[string]interface{}
}

// UpdateDeviceAttrMulti update device attr multi
func UpdateDeviceAttrMulti(updates []DeviceAttrUpdate) error {
	var err error
	to, err := dbm.DBAccess.Begin()
	for _, update := range updates {
		err = UpdateDeviceAttrFields(to, update.DeviceID, update.Name, update.Cols)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeviceAttrTrans transaction of device attr
func DeviceAttrTrans(adds []DeviceAttr, deletes []DeviceDelete, updates []DeviceAttrUpdate) error {
	obm := dbm.DefaultOrmFunc()
	to, err := obm.Begin()
	if err != nil {
		klog.Errorf("failed to begin transaction: %v", err)
		return err
	}

	defer func() {
		if err != nil {
			dbm.RollbackTransaction(to)
		} else {
			err = to.Commit()
			if err != nil {
				klog.Errorf("failed to commit transaction: %v", err)
			}
		}
	}()

	for _, add := range adds {
		err = SaveDeviceAttr(to, &add)
		if err != nil {
			return err
		}
	}

	for _, delete := range deletes {
		err = DeleteDeviceAttr(to, delete.DeviceID, delete.Name)
		if err != nil {
			return err
		}
	}

	for _, update := range updates {
		err = UpdateDeviceAttrFields(to, update.DeviceID, update.Name, update.Cols)
		if err != nil {
			return err
		}
	}

	return err
}
