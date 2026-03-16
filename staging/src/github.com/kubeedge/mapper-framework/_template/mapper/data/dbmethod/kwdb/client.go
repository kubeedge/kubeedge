/*
Copyright 2024 The KubeEdge Authors.

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

package kwdb

import (
	"encoding/json"
	"errors"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

var DBPool *pgxpool.Pool

type DataBaseConfig struct {
	KWDBClientConfig *KWDBClientConfig `json:"config,omitempty"`
}
type KWDBClientConfig struct {
	Addr   string `json:"addr,omitempty"`
	DBName string `json:"dbName,omitempty"`
}

func NewDataBaseClient(config json.RawMessage) (*DataBaseConfig, error) {
	configdata := new(KWDBClientConfig)
	err := json.Unmarshal(config, configdata)
	if err != nil {
		return nil, err
	}
	return &DataBaseConfig{
		KWDBClientConfig: configdata,
	}, nil
}
func (d *DataBaseConfig) InitDbClient() error {
	ctx := context.Background()
	username := os.Getenv("USERNAME")
	if (username == "") { username = "root" }
	password := os.Getenv("PASSWORD")
	if (password== "") { password = "0" }
	connConfig, err := pgxpool.ParseConfig(fmt.Sprintf(
		"postgresql://%s:%s@%s/%s",
		username, password, d.KWDBClientConfig.Addr, d.KWDBClientConfig.DBName,
	))
	if err != nil {
		klog.Errorf("parse pgx config failed: %v", err)
	}
	DBPool, err = pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		klog.Errorf("create pgx pool failed: %v", err)
	}
	klog.V(1).Infof("init KWDB database successfully")
	return nil
}

func (d *DataBaseConfig) CloseSession() {
	if DBPool != nil {
		DBPool.Close()
		klog.V(1).Info("postgresql connection pool closed")
	}
}

func (d *DataBaseConfig) AddData(data *common.DataModel) error {

	ctx := context.Background()
	tsTableName := data.Namespace + "_" + data.DeviceName
	validTableName := strings.Replace(tsTableName, "-", "_", -1)
	validPtagName := strings.Replace(data.PropertyName, "-", "_", -1)

	tableDDL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (ts timestamp not null, propertyname varchar(64), data varchar(64),type varchar(64)) TAGS (deviceid varchar(64) not null) PRIMARY TAGS(deviceid);", validTableName)

	datatime := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	insertSQL := fmt.Sprintf("INSERT INTO %s VALUES('%v','%s', '%s', '%s', '%s');",
		validTableName, datatime, data.PropertyName, data.Value, data.Type, validPtagName)

	_, err := DBPool.Exec(ctx, tableDDL)
	if err != nil {
		klog.Errorf("create table failed %v\n", err)
	}
	_, err = DBPool.Exec(ctx, insertSQL)
	if err != nil {
		klog.Errorf("failed add data to KWDB:%v", err)
	}

	return nil
}

func (d *DataBaseConfig) GetDataByDeviceID(deviceID string) ([]*common.DataModel, error) {
	//TODO: implement
	return nil, errors.New("implement me")
}

func (d *DataBaseConfig) GetPropertyDataByDeviceID(deviceID string, propertyData string) ([]*common.DataModel, error) {
	//TODO: implement
	return nil, errors.New("implement me")
}

func (d *DataBaseConfig) GetDataByTimeRange(deviceID string, start int64, end int64) ([]*common.DataModel, error) {
	//TODO: implement
	return nil, errors.New("implement me")
}

func (d *DataBaseConfig) DeleteDataByTimeRange(start int64, end int64) ([]*common.DataModel, error) {
	//TODO: implement
	return nil, errors.New("implement me")
}
