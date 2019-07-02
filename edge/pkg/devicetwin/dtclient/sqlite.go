/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package dtclient
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
