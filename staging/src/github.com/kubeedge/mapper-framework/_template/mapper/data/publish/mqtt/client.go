package mqtt

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/global"
)

type PushMethod struct {
	MQTT *MQTTConfig `json:"http"`
}

type MQTTConfig struct {
	Address  string `json:"address,omitempty"`
	Topic    string `json:"topic,omitempty"`
	QoS      int    `json:"qos,omitempty"`
	Retained bool   `json:"retained,omitempty"`
}

func NewDataPanel(config json.RawMessage) (global.DataPanel, error) {
	mqttConfig := new(MQTTConfig)
	err := json.Unmarshal(config, mqttConfig)
	if err != nil {
		return nil, err
	}
	return &PushMethod{
		MQTT: mqttConfig,
	}, nil
}

func (pm *PushMethod) InitPushMethod() error {
	klog.V(1).Info("Init MQTT")
	return nil
}

func (pm *PushMethod) Push(data *common.DataModel) {
	klog.V(1).Infof("Publish %v to %s on topic: %s, Qos: %d, Retained: %v",
		data.Value, pm.MQTT.Address, pm.MQTT.Topic, pm.MQTT.QoS, pm.MQTT.Retained)

	opts := mqtt.NewClientOptions().AddBroker(pm.MQTT.Address)
	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	formatTimeStr := time.Unix(data.TimeStamp/1e3, 0).Format("2006-01-02 15:04:05")
	str_time := "time is " + formatTimeStr + "  "
	str_publish := str_time + pm.MQTT.Topic + ": " + data.Value

	token := client.Publish(pm.MQTT.Topic, byte(pm.MQTT.QoS), pm.MQTT.Retained, str_publish)
	token.Wait()

	client.Disconnect(250)
	klog.V(2).Info("###############  Message published.  ###############")
}
