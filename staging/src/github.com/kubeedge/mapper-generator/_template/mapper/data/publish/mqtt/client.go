package mqtt

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/global"
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
	// TODO add init code
	fmt.Println("Init Mqtt")
	return nil
}

func (pm *PushMethod) Push(data *common.DataModel) {
	// TODO add push code
	fmt.Printf("Publish %v to %s on topic: %s, Qos: %d, Retained: %v",
		data.Value, pm.MQTT.Address, pm.MQTT.Topic, pm.MQTT.QoS, pm.MQTT.Retained)
}
