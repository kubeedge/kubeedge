package opengemini

import (
	"encoding/json"
	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/kubeedge/Template/pkg/common"
)

type DataBaseConfig struct {
	OpengeminiClientConfig *OpengeminiClientConfig `json:"opengeminiClientConfig,omitempty"`
	OpengeminiDataConfig   *OpengeminiDataConfig   `json:"opengeminiDataConfig,omitempty"`
}

type OpengeminiClientConfig struct {
	URL             string `json:"url,omitempty"`
	Database        string `json:"database,omitempty"`
	RetentionPolicy string `json:"retentionPolicy,omitempty"`
}

type OpengeminiDataConfig struct {
	Measurement string            `json:"measurement,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	FieldKey    string            `json:"fieldKey,omitempty"`
}

func NewDataBaseClient(clientConfig json.RawMessage, dataConfig json.RawMessage) (*DataBaseConfig, error) {
	// TODO parse opengemini database config data

	return &DataBaseConfig{}, nil
}

func (d *DataBaseConfig) InitDbClient() (client.Client, error) {
	// TODO add opengemini database initialization code

	conf := client.HTTPConfig{}
	return client.NewHTTPClient(conf)
}

func (d *DataBaseConfig) CloseSession(cli client.Client) error {
	// TODO add opengemini database close code
	return nil
}

func (d *DataBaseConfig) AddData(data *common.DataModel, cli client.Client) error {
	// TODO add opengemini database data push code
	return nil
}
