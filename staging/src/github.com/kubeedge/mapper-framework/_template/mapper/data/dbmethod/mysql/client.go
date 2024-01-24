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

package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

var (
	DB *sql.DB
)

type DataBaseConfig struct {
	MySQLClientConfig *MySQLClientConfig `json:"mysqlClientConfig"`
}

type MySQLClientConfig struct {
	Addr     string `json:"addr,omitempty"`
	Database string `json:"database,omitempty"`
	UserName string `json:"userName,omitempty"`
}

func NewDataBaseClient(config json.RawMessage) (*DataBaseConfig, error) {
	configdata := new(MySQLClientConfig)
	err := json.Unmarshal(config, configdata)
	if err != nil {
		return nil, err
	}
	return &DataBaseConfig{
		MySQLClientConfig: configdata,
	}, nil
}

func (d *DataBaseConfig) InitDbClient() error {
	password := os.Getenv("PASSWORD")
	usrName := d.MySQLClientConfig.UserName
	addr := d.MySQLClientConfig.Addr
	dataBase := d.MySQLClientConfig.Database
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", usrName, password, addr, dataBase)
	var err error
	DB, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		return fmt.Errorf("connection to %s of mysql faild with err:%v", dataBase, err)
	}

	return nil
}

func (d *DataBaseConfig) CloseSession() {
	err := DB.Close()
	if err != nil {
		klog.Errorf("close mysql failed with err:%v", err)
	}
}

func (d *DataBaseConfig) AddData(data *common.DataModel) error {
	deviceName := data.DeviceName
	propertyName := data.PropertyName
	namespace := data.Namespace
	tableName := namespace + "/" + deviceName + "/" + propertyName
	datatime := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")

	createTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (id INT AUTO_INCREMENT PRIMARY KEY, ts  DATETIME NOT NULL,field TEXT)", tableName)
	_, err := DB.Exec(createTable)
	if err != nil {
		return fmt.Errorf("create tabe into mysql failed with err:%v", err)
	}

	stmt, err := DB.Prepare(fmt.Sprintf("INSERT INTO `%s` (ts,field) VALUES (?,?)", tableName))
	if err != nil {
		return fmt.Errorf("prepare parament failed with err:%v", err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			klog.Errorf("close mysql's statement failed with err:%v", err)
		}
	}(stmt)
	_, err = stmt.Exec(datatime, data.Value)
	if err != nil {
		return fmt.Errorf("insert data into msyql failed with err:%v", err)
	}

	return nil
}
