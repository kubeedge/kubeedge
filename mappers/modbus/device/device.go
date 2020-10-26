/*
Copyright 2020 The KubeEdge Authors.

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

package dev

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"

	"github.com/kubeedge/kubeedge/mappers/modbus/configmap"
	"github.com/kubeedge/kubeedge/mappers/modbus/driver"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

var devices map[string]*globals.ModbusDev
var models map[string]mappercommon.DeviceModel
var protocols map[string]mappercommon.Protocol
var wg sync.WaitGroup

// getDeviceID extract the device ID from Mqtt topic.
func getDeviceID(topic string) (id string) {
	re := regexp.MustCompile(`hw/events/device/(.+)/twin/update/delta`)
	return re.FindStringSubmatch(topic)[1]
}

// onMessage callback function of Mqtt subscribe message.
func onMessage(client mqtt.Client, message mqtt.Message) {
	klog.Info("Receive message", message.Topic())
	// Get device ID and get device instance
	id := getDeviceID(message.Topic())
	if id == "" {
		klog.Error("Wrong topic")
		return
	}
	klog.Info("Device id: ", id)

	var device *globals.ModbusDev
	var ok bool
	if device, ok = devices[id]; !ok {
		klog.Error("Device not exist")
		return
	}

	// Get twin map key as the propertyName
	var delta mappercommon.DeviceTwinDelta
	if err := json.Unmarshal(message.Payload(), &delta); err != nil {
		klog.Error("Unmarshal message failed: ", err)
		return
	}
	for twinName, twinValue := range delta.Delta {
		i := 0
		for i = 0; i < len(device.Instance.Twins); i++ {
			if twinName == device.Instance.Twins[i].PropertyName {
				break
			}
		}
		if i == len(device.Instance.Twins) {
			klog.Error("Twin not found: ", twinName)
			continue
		}
		// Type transfer
		device.Instance.Twins[i].Desired.Value = twinValue
		var visitorConfig configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(device.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Error("Unmarshal visitor config failed")
		}
		valueConverted, err := mappercommon.Convert(device.Instance.Twins[i].PVisitor.PProperty.DataType, twinValue)
		if err == nil {
			valueInt, _ := valueConverted.(int64)
			device.ModbusClient.Set(visitorConfig.Register, visitorConfig.Offset, uint16(valueInt))
		} else {
			klog.Error("Convert failed")
		}
	}
}

// isRS485Enabled is RS485 feature enabled for RTU.
func isRS485Enabled(customizedValue configmap.CustomizedValue) bool {
	isEnabled := false

	if len(customizedValue) != 0 {
		if value, ok := customizedValue["serialType"]; ok {
			if value == "RS485" {
				isEnabled = true
			}
		}
	}
	return isEnabled
}

// initModbus initialize modbus client
func initModbus(protocolConfig configmap.ModbusProtocolCommonConfig, slaveID int16, client *driver.ModbusClient) error {
	if protocolConfig.Com.SerialPort != "" {
		modbusRtu := driver.ModbusRtu{SlaveId: byte(slaveID),
			SerialName:   protocolConfig.Com.SerialPort,
			BaudRate:     int(protocolConfig.Com.BaudRate),
			DataBits:     int(protocolConfig.Com.DataBits),
			StopBits:     int(protocolConfig.Com.StopBits),
			Parity:       protocolConfig.Com.Parity,
			RS485Enabled: isRS485Enabled(protocolConfig.CustomizedValues),
			Timeout:      5 * time.Second}
		*client, _ = driver.NewClient(modbusRtu)
	} else if protocolConfig.Tcp.Ip != "" {
		modbusTcp := driver.ModbusTcp{
			SlaveId:  byte(slaveID),
			DeviceIp: protocolConfig.Tcp.Ip,
			TcpPort:  strconv.FormatInt(protocolConfig.Tcp.Port, 10),
			Timeout:  5 * time.Second}
		*client, _ = driver.NewClient(modbusTcp)
	} else {
		return errors.New("No protocol found")
	}
	return nil
}

// initTwin initialize the timer to get twin value.
func initTwin(dev *globals.ModbusDev) {
	for i := 0; i < len(dev.Instance.Twins); i++ {
		var visitorConfig configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Twins[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Error(err)
			continue
		}
		value, err := mappercommon.Convert(dev.Instance.Twins[i].PVisitor.PProperty.DataType, dev.Instance.Twins[i].Desired.Value)
		if err != nil {
			klog.Error("Convert error. Type: %s, value: %s ", dev.Instance.Twins[i].PVisitor.PProperty.DataType, dev.Instance.Twins[i].Desired.Value)
			continue
		}
		valueInt, _ := value.(int64)
		_, err = dev.ModbusClient.Set(visitorConfig.Register, visitorConfig.Offset, uint16(valueInt))
		if err != nil {
			klog.Error(err, visitorConfig)
			continue
		}

		twinData := TwinData{Client: dev.ModbusClient,
			Name:         dev.Instance.Twins[i].PropertyName,
			Type:         dev.Instance.Twins[i].Desired.Metadatas.Type,
			RegisterType: visitorConfig.Register,
			Address:      visitorConfig.Offset,
			Quantity:     uint16(visitorConfig.Limit),
			Topic:        fmt.Sprintf(mappercommon.TopicTwinUpdate, dev.Instance.ID)}
		collectCycle := time.Duration(dev.Instance.Twins[i].PVisitor.CollectCycle)
		// If the collect cycle is not set, set it to 1 second.
		if collectCycle == 0 {
			collectCycle = 1 * time.Second
		}
		timer := mappercommon.Timer{twinData.Run, collectCycle, 0}
		go timer.Start()
		wg.Add(1)
	}
}

// initData initialize the timer to get data.
func initData(dev *globals.ModbusDev) {
	for i := 0; i < len(dev.Instance.Datas.Properties); i++ {
		var visitorConfig configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(dev.Instance.Datas.Properties[i].PVisitor.VisitorConfig), &visitorConfig); err != nil {
			klog.Error("Unmarshal visitor config failed")
		}
		twinData := TwinData{Client: dev.ModbusClient,
			Name:         dev.Instance.Datas.Properties[i].PropertyName,
			Type:         dev.Instance.Datas.Properties[i].Metadatas.Type,
			RegisterType: visitorConfig.Register,
			Address:      visitorConfig.Offset,
			Quantity:     uint16(visitorConfig.Limit),
			Topic:        dev.Instance.Datas.Topic}
		timer := mappercommon.Timer{twinData.Run, time.Duration(dev.Instance.Twins[i].PVisitor.CollectCycle), 0}
		go timer.Start()
		wg.Add(1)
	}
}

// initSubscribeMqtt subscirbe Mqtt topics.
func initSubscribeMqtt(instanceID string) error {
	topic := fmt.Sprintf(mappercommon.TopicTwinUpdateDelta, instanceID)
	klog.Info("Subscribe topic: ", topic)
	return globals.MqttClient.Subscribe(topic, onMessage)
}

// initGetStatus start timer to get device status and send to eventbus.
func initGetStatus(dev *globals.ModbusDev) {
	getStatus := GetStatus{Client: dev.ModbusClient,
		topic: fmt.Sprintf(mappercommon.TopicStateUpdate, dev.Instance.ID)}
	timer := mappercommon.Timer{getStatus.Run, 1 * time.Second, 0}
	go timer.Start()
	wg.Add(1)
}

// start start the device.
func start(dev *globals.ModbusDev) {
	var protocolConfig configmap.ModbusProtocolCommonConfig
	if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolCommonConfig), &protocolConfig); err != nil {
		klog.Error(err)
		return
	}

	if err := initModbus(protocolConfig, dev.Instance.PProtocol.ProtocolConfigs.SlaveID, &dev.ModbusClient); err != nil {
		klog.Error(err)
		return
	}

	initTwin(dev)
	initData(dev)

	if err := initSubscribeMqtt(dev.Instance.ID); err != nil {
		klog.Error(err)
		return
	}

	initGetStatus(dev)
}

// DevInit initialize the device datas.
func DevInit(configmapPath string) error {
	devices = make(map[string]*globals.ModbusDev)
	models = make(map[string]mappercommon.DeviceModel)
	protocols = make(map[string]mappercommon.Protocol)
	return configmap.Parse(configmapPath, devices, models, protocols)
}

// DevStart start all devices.
func DevStart() {
	for id, dev := range devices {
		klog.Info("Dev: ", id, dev)
		start(dev)
	}
	wg.Wait()
}
