package influx

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"k8s.io/klog/v2"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/kubeedge/Template/pkg/common"
)

type DataBaseConfig struct {
	Config   *ConfigData   `json:"configdata,omitempty"`
	Standard *DataStandard `json:"dataStandard,omitempty"`
}

type ConfigData struct {
	Url    string `json:"url,omitempty"`
	Org    string `json:"org,omitempty"`
	Bucket string `json:"bucket,omitempty"`
}

type DataStandard struct {
	// broker address, like mqtt://127.0.0.1:1883
	Measurement string `json:"measurement,omitempty"`
	TagKey      string `json:"tagKey,omitempty"`
	TagValue    string `json:"tagValue,omitempty"`
	FieldKey    string `json:"fieldKey,omitempty"`
}

func NewDataBaseClient(config json.RawMessage, standard json.RawMessage) (*DataBaseConfig, error) {
	configdata := new(ConfigData)
	datastandard := new(DataStandard)
	err := json.Unmarshal(config, configdata)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(standard, datastandard)
	if err != nil {
		return nil, err
	}
	return &DataBaseConfig{
		Config:   configdata,
		Standard: datastandard,
	}, nil
}

func (d *DataBaseConfig) InitDbClient() influxdb2.Client {
	var usrtoken string
	usrtoken = os.Getenv("TOKEN")
	client := influxdb2.NewClient(d.Config.Url, usrtoken)

	return client
}

func (d *DataBaseConfig) CloseSession(client influxdb2.Client) {
	client.Close()
}

func (d *DataBaseConfig) AddData(data *common.DataModel, client influxdb2.Client) error {
	writeAPI := client.WriteAPIBlocking(d.Config.Org, d.Config.Bucket)
	// Create point using full params constructor
	p := influxdb2.NewPoint(d.Standard.Measurement,
		map[string]string{d.Standard.TagKey: d.Standard.TagValue},
		map[string]interface{}{d.Standard.FieldKey: data.Value},
		time.Now())
	// write point immediately
	err := writeAPI.WritePoint(context.Background(), p)
	if err != nil {
		klog.V(4).Info("Exit AddData")
		return err
	}
	return nil
}
