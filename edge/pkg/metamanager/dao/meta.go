package dao

import (
	"github.com/kubeedge/beehive/pkg/common/log"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

//constant metatable name reference
const (
	MetaTableName = "meta"
)

// Meta metadata object
type Meta struct {
	// ID    int64  `orm:"pk; auto; column(id)"`
	Key   string `orm:"column(key); size(256); pk"`
	Type  string `orm:"column(type); size(32)"`
	Value string `orm:"column(value); null; type(text)"`
}

// SaveMeta save meta to db
func SaveMeta(meta *Meta) error {
	num, err := dbm.DBAccess.Insert(meta)
	log.LOGGER.Debugf("Insert affected Num: %d, %v", num, err)
	if err == nil || dbm.IsNonUniqueNameError(err) {
		return nil
	}
	return err
}

// DeleteMetaByKey delete meta by key
func DeleteMetaByKey(key string) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Delete()
	log.LOGGER.Debugf("Delete affected Num: %d, %v", num, err)
	return err
}

// UpdateMeta update meta
func UpdateMeta(meta *Meta) error {
	num, err := dbm.DBAccess.Update(meta) // will update all field
	log.LOGGER.Debugf("Update affected Num: %d, %v", num, err)
	return err
}

// InsertOrUpdate insert or update meta
func InsertOrUpdate(meta *Meta) error {
	_, err := dbm.DBAccess.Raw("INSERT OR REPLACE INTO meta (key, type, value) VALUES (?,?,?)", meta.Key, meta.Type, meta.Value).Exec() // will update all field
	log.LOGGER.Debugf("Update result %v", err)
	return err
}

// UpdateMetaField update special field
func UpdateMetaField(key string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Update(map[string]interface{}{col: value})
	log.LOGGER.Debugf("Update affected Num: %d, %v", num, err)
	return err
}

// UpdateMetaFields update special fields
func UpdateMetaFields(key string, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(MetaTableName).Filter("key", key).Update(cols)
	log.LOGGER.Debugf("Update affected Num: %d, %v", num, err)
	return err
}

// QueryMeta return only meta's value, if no error, Meta not null
func QueryMeta(key string, condition string) (*[]string, error) {
	meta := new([]Meta)
	_, err := dbm.DBAccess.QueryTable(MetaTableName).Filter(key, condition).All(meta)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, v := range *meta {
		result = append(result, v.Value)
	}
	return &result, nil
}

// QueryAllMeta return all meta, if no error, Meta not null
func QueryAllMeta(key string, condition string) (*[]Meta, error) {
	meta := new([]Meta)
	_, err := dbm.DBAccess.QueryTable(MetaTableName).Filter(key, condition).All(meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
