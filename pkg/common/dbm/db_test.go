package dbm_test

import (
	"testing"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/pkg/common/dbm"

	"github.com/astaxie/beego/orm"
)

const (
	//MetaTable name
	MetaTableName = "meta"
	// DocumentTableName document table name
	DocumentTableName = "document"
)

// Document device document
type Document struct {
	Deviceid   string `orm:"column(deviceid); size(64); pk"`
	Expected   string `orm:"column(expected); type(text); null"`
	Actual     string `orm:"column(actual); type(text); null"`
	Metadata   string `orm:"column(metadata); type(text); null"`
	Laststate  string `orm:"column(laststate); type(text); null"`
	Lastsync   string `orm:"column(lastsync); type(text); null"`
	Attributes string `orm:"column(attributes); type(text); null"`
}

// SaveDocument save document to db
func SaveDocument(doc *Document) error {
	_, err := dbm.DBAccess.Insert(doc)
	return err
}

// Meta metadata object
type TestMeta struct {
	// ID    int64  `orm:"pk; auto; column(id)"`
	Key   string `orm:"column(key); size(36); pk"`
	Type  string `orm:"column(type); size(36)"`
	Value string `orm:"column(value); null; type(text)"`
}

// SaveMeta save meta to db
func SaveMeta(meta *TestMeta) error {
	num, err := dbm.DBAccess.Insert(meta)
	log.LOGGER.Debugf("Insert affected Num: %d, %s", num, err)
	return err
}

// DeleteMetaByKey delete meta by key
func DeleteMetaByKey(key string) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Delete()
	if err != nil {
		log.LOGGER.Errorf("Something wrong when inserting data: %v", err)
		return err
	}
	log.LOGGER.Debugf("Delete affected Num: %d", num)
	return nil
}

// UpdateMeta update meta
func UpdateMeta(meta *TestMeta) error {
	num, err := dbm.DBAccess.Update(meta) // will update all field
	log.LOGGER.Debugf("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateMetaField update special field
func UpdateMetaField(key string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Update(orm.Params{col: value})
	log.LOGGER.Debugf("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateMetaFields update special fields
func UpdateMetaFields(key string, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Update(cols)
	log.LOGGER.Debugf("Update affected Num: %d, %s", num, err)
	return err
}

// QueryMetaByKey query meta
func QueryMetaByKey() error {
	return nil
}

func TestInitDB(t *testing.T) {
	dbm.RegisterModel("dbTest", &TestMeta{})
	dbm.InitDBManager()
	//m := &TestMeta{Key: "testKey", Type: "testType", Value: "testValue"}
	//SaveMeta(m)
}
