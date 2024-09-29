package tdengine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosRestful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

var (
	DB *sql.DB
)

type DataBaseConfig struct {
	TDEngineClientConfig *TDEngineClientConfig `json:"config,omitempty"`
}
type TDEngineClientConfig struct {
	Addr   string `json:"addr,omitempty"`
	DBName string `json:"dbName,omitempty"`
}

func NewDataBaseClient(config json.RawMessage) (*DataBaseConfig, error) {
	configdata := new(TDEngineClientConfig)
	err := json.Unmarshal(config, configdata)
	if err != nil {
		return nil, err
	}
	return &DataBaseConfig{
		TDEngineClientConfig: configdata,
	}, nil
}
func (d *DataBaseConfig) InitDbClient() error {
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	dsn := fmt.Sprintf("%s:%s@http(%s)/%s", username, password, d.TDEngineClientConfig.Addr, d.TDEngineClientConfig.DBName)
	var err error
	DB, err = sql.Open("taosRestful", dsn)
	if err != nil {
		klog.Errorf("init TDEngine db fail, err= %v:", err)
	}
	klog.V(1).Infof("init TDEngine database successfully")
	return nil
}

func (d *DataBaseConfig) CloseSessio() {
	err := DB.Close()
	if err != nil {
		klog.Errorf("close TDEngine failed")
	}
}

func (d *DataBaseConfig) AddData(data *common.DataModel) error {

	tableName := data.Namespace + "/" + data.DeviceName
	legalTable := strings.Replace(tableName, "-", "_", -1)
	legalTag := strings.Replace(data.PropertyName, "-", "_", -1)

	stableName := fmt.Sprintf("SHOW STABLES LIKE '%s'", legalTable)
	stabel := fmt.Sprintf("CREATE STABLE %s (ts timestamp, deviceid binary(64), propertyname binary(64), data binary(64),type binary(64)) TAGS (localtion binary(64));", legalTable)

	datatime := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	insertSQL := fmt.Sprintf("INSERT INTO %s USING %s TAGS ('%s') VALUES('%v','%s', '%s', '%s', '%s');",
		legalTag, legalTable, legalTag, datatime, tableName, data.PropertyName, data.Value, data.Type)

	rows, _ := DB.Query(stableName)
	defer rows.Close()

	if err := rows.Err(); err != nil {
		klog.Errorf("query stable failed：%v", err)
	}

	switch rows.Next() {
	case false:
		_, err := DB.Exec(stabel)
		if err != nil {
			klog.Errorf("create stable failed %v\n", err)
		}
		_, err = DB.Exec(insertSQL)
		if err != nil {
			klog.Errorf("failed add data to TdEngine:%v", err)
		}
	case true:
		_, err := DB.Exec(insertSQL)
		if err != nil {
			klog.Errorf("failed add data to TdEngine:%v", err)
		}
	default:
		klog.Infoln("failed add data to TdEngine")
	}

	return nil
}
func (d *DataBaseConfig) GetDataByDeviceID(deviceID string) ([]*common.DataModel, error) {
	querySql := fmt.Sprintf("SELECT ts, deviceid, propertyname, data, type FROM %s", deviceID)
	rows, err := DB.Query(querySql)
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

	legalTable := strings.Replace(deviceID, "-", "_", -1)
	startTime := time.Unix(start, 0).UTC().Format("2006-01-02 15:04:05")
	endTime := time.Unix(end, 0).UTC().Format("2006-01-02 15:04:05")
	//Query data within a specified time range
	querySQL := fmt.Sprintf("SELECT ts, deviceid, propertyname, data, type FROM %s WHERE ts >= '%s' AND ts <= '%s'", legalTable, startTime, endTime)
	fmt.Println(querySQL)
	rows, err := DB.Query(querySQL)
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
