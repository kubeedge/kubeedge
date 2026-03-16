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

func (d *DataBaseConfig) CloseSessio() {
	if DBPool != nil {
		DBPool.Close()
		klog.V(1).Info("postgresql connection pool closed")
	}
}

func (d *DataBaseConfig) AddData(data *common.DataModel) error {

	ctx := context.Background()
	tableName := data.Namespace + "_" + data.DeviceName
	legalTable := strings.Replace(tableName, "-", "_", -1)
	legalTag := strings.Replace(data.PropertyName, "-", "_", -1)

	tableDDL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (ts timestamp not null, propertyname varchar(64), data varchar(64),type varchar(64)) TAGS (deviceid varchar(64) not null) PRIMARY TAGS(deviceid);", legalTable)

	datatime := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	insertSQL := fmt.Sprintf("INSERT INTO %s VALUES('%v','%s', '%s', '%s', '%s');",
		legalTable, datatime, data.PropertyName, data.Value, data.Type, legalTag)

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
	ctx := context.Background()
	querySql := fmt.Sprintf("SELECT ts, deviceid, propertyname, data, type FROM %s WHERE deviceid = '%s'", deviceID)
	rows, err := DBPool.Query(ctx, querySql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dataModel []*common.DataModel
	for rows.Next() {
		var data common.DataModel
		var ts time.Time
		err := rows.Scan(&ts, &data.DeviceName, &data.PropertyName, &data.Value, &data.Type)
		if err != nil {
			klog.Errorf(" data scan error: %v\n", err)
			//fmt.Printf("scan error:\n", err)
			return nil, err
		}
		data.TimeStamp = ts.Unix()
		dataModel = append(dataModel, &data)
	}
	return dataModel, nil
}
func (d *DataBaseConfig) GetPropertyDataByDeviceID(deviceID string, propertyData string) ([]*common.DataModel, error) {
	//TODO implement me
	return nil, errors.New("implement me")
}
func (d *DataBaseConfig) GetDataByTimeRange(deviceID string, start int64, end int64) ([]*common.DataModel, error) {

	ctx := context.Background()
	legalTable := strings.Replace(deviceID, "-", "_", -1)
	startTime := time.Unix(start, 0).UTC().Format("2006-01-02 15:04:05")
	endTime := time.Unix(end, 0).UTC().Format("2006-01-02 15:04:05")
	//Query data within a specified time range
	querySQL := fmt.Sprintf("SELECT ts, deviceid, propertyname, data, type FROM %s WHERE ts >= '%s' AND ts <= '%s'", legalTable, startTime, endTime)
	fmt.Println(querySQL)
	rows, err := DBPool.Query(ctx, querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataModels []*common.DataModel
	for rows.Next() {
		var data common.DataModel
		var ts time.Time
		err := rows.Scan(&ts, &data.DeviceName, &data.PropertyName, &data.Value, &data.Type)
		if err != nil {
			klog.V(4).Infof("data scan failed：%v", err)
			continue
		}
		dataModels = append(dataModels, &data)
	}
	return dataModels, nil
}
func (d *DataBaseConfig) DeleteDataByTimeRange(start int64, end int64) ([]*common.DataModel, error) {
	//TODO implement me
	return nil, errors.New("implement me")
}
