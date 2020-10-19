package dev

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"

	"github.com/kubeedge/kubeedge/mappers/modbus/configmap"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	. "github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

var devices map[string]*ModbusDev
var models map[string]mappercommon.DeviceModel
var protocols map[string]mappercommon.Protocol

var wg sync.WaitGroup

func DevInit(configmapPath string) error {
	devices = make(map[string]*ModbusDev)
	models = make(map[string]mappercommon.DeviceModel)
	protocols = make(map[string]mappercommon.Protocol)
	return configmap.Parse(configmapPath, devices, models, protocols)
}

func getDeviceID(topic string) (id string) {
	re := regexp.MustCompile(`hw/events/device/(.+)/twin/update/delta`)
	return re.FindStringSubmatch(topic)[1]
}

func onMessage(client mqtt.Client, message mqtt.Message) {
	klog.Error("Receive message", message.Topic())
	// TopicTwinUpdateDelta
	id := getDeviceID(message.Topic())
	if id == "" {
		klog.Error("Wrong topic")
		return
	}
	klog.Error("Device id: ", id)

	// Get device ID and get device instance
	var device *ModbusDev
	var ok bool
	if device, ok = devices[id]; !ok {
		klog.Error("Device not exist")
		return
	}

	// Get twin map key as the propertyName
	var delta mappercommon.DeviceTwinDelta
	err := json.Unmarshal(message.Payload(), &delta)
	if err != nil {
		klog.Error("Unmarshal message failed: ", err)
		return
	}
	for key, value := range delta.Delta {
		i := 0
		for i = 0; i < len(device.Instance.Twins); i++ {
			if key == device.Instance.Twins[i].PropertyName {
				break
			}
		}
		if i == len(device.Instance.Twins) {
			klog.Error("Twin not found")
			continue
		}
		// Type transfer
		device.Instance.Twins[i].Desired.Value = value
		var r configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(device.Instance.Twins[i].PVisitor.VisitorConfig), &r); err != nil {
			klog.Error("Unmarshal visitor config failed")
		}
		klog.Error("Desired value: ", value, device.Instance.Twins[i].PVisitor.PProperty.DataType)
		v, err := mappercommon.Convert(device.Instance.Twins[i].PVisitor.PProperty.DataType, value)
		vint, _ := v.(int64)
		if err == nil {
			Set(device.ModbusClient, r.Register, r.Offset, uint16(vint))
		} else {
			klog.Error("Convert failed")
		}
	}
}

type TwinData struct {
	Client       ModbusClient
	Name         string
	Type         string
	RegisterType string
	Address      uint16
	Quantity     uint16
	Results      []byte
	Topic        string
}

func (td *TwinData) Run() {
	var err error
	td.Results, err = Get(td.Client, td.RegisterType, td.Address, td.Quantity)
	if err != nil {
		klog.Error("Get register failed")
		return
	}
	// construct payload
	var payload []byte
	if strings.Contains(td.Topic, "update") {
		payload = mappercommon.CreateMessageTwinUpdate(td.Name, td.Type, strconv.Itoa(int(td.Results[0])))
		klog.Error("Input value:", string(int(td.Results[0])))
		var deviceTwinUpdate mappercommon.DeviceTwinUpdate
		err = json.Unmarshal(payload, &deviceTwinUpdate)
		if err != nil {
			klog.Error("Unmarshal error", err)
		} else {
			klog.Error(deviceTwinUpdate)
			klog.Error("The value is:", *deviceTwinUpdate.Twin["temperature-enable"].Actual.Value)
		}
	} else {
		payload = mappercommon.CreateMessageData(td.Name, td.Type, string(td.Results))
		var data mappercommon.DeviceData
		err = json.Unmarshal(payload, &data)
		klog.Error(err, data)
	}
	err = globals.MqttClient.Publish(td.Topic, payload)

	if err != nil {
		klog.Error(err)
	}
}

type GetSt struct {
	Client ModbusClient
	St     DevStatus
	topic  string
}

func (gs *GetSt) Run() {
	gs.St = GetStatus(gs.Client)

	var payload []byte
	payload = mappercommon.CreateMessageState(strconv.Itoa(int(gs.St)))
	globals.MqttClient.Publish(gs.topic, payload)
}

func Start(md *ModbusDev) {
	var pcc configmap.ModbusProtocolCommonConfig
	if err := json.Unmarshal([]byte(md.Instance.PProtocol.ProtocolCommonConfig), &pcc); err != nil {
		klog.Error(err)
		return
	}

	if pcc.Com.SerialPort != "" {
		modbusRtu := ModbusRtu{SlaveId: byte(md.Instance.PProtocol.ProtocolConfigs.SlaveID),
			SerialName: pcc.Com.SerialPort,
			BaudRate:   int(pcc.Com.BaudRate),
			DataBits:   int(pcc.Com.DataBits),
			StopBits:   int(pcc.Com.StopBits),
			Parity:     pcc.Com.Parity}
		md.ModbusClient, _ = NewClient(modbusRtu)
	} else if pcc.Tcp.Ip != "" {
		modbusTcp := ModbusTcp{
			SlaveId:  byte(md.Instance.PProtocol.ProtocolConfigs.SlaveID),
			DeviceIp: pcc.Tcp.Ip,
			TcpPort:  strconv.FormatInt(pcc.Tcp.Port, 10)}
		md.ModbusClient, _ = NewClient(modbusTcp)
	} else {
		klog.Error("No protocol")
		return
	}

	// set expected value
	for i := 0; i < len(md.Instance.Twins); i++ {
		var r configmap.ModbusVisitorConfig
		klog.Error(md.Instance.Twins[i].PVisitor.VisitorConfig)
		if err := json.Unmarshal([]byte(md.Instance.Twins[i].PVisitor.VisitorConfig), &r); err != nil {
			klog.Error(err)
			continue
		}
		v, err := mappercommon.Convert(md.Instance.Twins[i].PVisitor.PProperty.DataType, md.Instance.Twins[i].Desired.Value)
		if err != nil {
			klog.Error("Convert error. Type: %s, value: %s ", md.Instance.Twins[i].PVisitor.PProperty.DataType, md.Instance.Twins[i].Desired.Value)
			continue
		}
		vint, _ := v.(int64)
		_, err = Set(md.ModbusClient, r.Register, r.Offset, uint16(vint))
		if err != nil {
			klog.Error(err, r)
			continue
		}

		td := TwinData{Client: md.ModbusClient,
			Name:         md.Instance.Twins[i].PropertyName,
			Type:         md.Instance.Twins[i].Desired.Metadatas.Type,
			RegisterType: r.Register,
			Address:      r.Offset,
			Quantity:     uint16(r.Limit),
			Topic:        fmt.Sprintf(mappercommon.TopicTwinUpdate, md.Instance.ID)}
		c := time.Duration(md.Instance.Twins[i].PVisitor.CollectCycle)
		if c == 0 {
			c = 1 * time.Second
		}
		t := mappercommon.Timer{td.Run, c, 0}
		go t.Start()
		wg.Add(1)
	}

	// timer get data
	for i := 0; i < len(md.Instance.Datas.Properties); i++ {
		var r configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(md.Instance.Datas.Properties[i].PVisitor.VisitorConfig), &r); err != nil {
			klog.Error("Unmarshal visitor config failed")
		}
		td := TwinData{Client: md.ModbusClient,
			Name:         md.Instance.Datas.Properties[i].PropertyName,
			Type:         md.Instance.Datas.Properties[i].Metadatas.Type,
			RegisterType: r.Register,
			Address:      r.Offset,
			Quantity:     uint16(r.Limit),
			Topic:        md.Instance.Datas.Topic}
		t := mappercommon.Timer{td.Run, time.Duration(md.Instance.Twins[i].PVisitor.CollectCycle), 0}
		go t.Start()
		wg.Add(1)
	}

	//Subscribe the TwinUpdate topic
	topic := fmt.Sprintf(mappercommon.TopicTwinUpdateDelta, md.Instance.ID)
	klog.Error("subscribe topic:", topic)
	err := globals.MqttClient.Subscribe(topic, onMessage)
	if err != nil {
		klog.Error(err)
		return
	}
	/*
		// timer get status and send to eventbus
		gs := GetSt{Client: md.ModbusClient,
			topic: fmt.Sprintf(mappercommon.TopicStateUpdate, md.Instance.ID)}
		t := mappercommon.Timer{gs.Run, 1 * time.Second, 0}
		go t.Start()
		wg.Add(1)
	*/
}

func DevStart() {
	klog.Error("len: ", len(devices))
	for id, dev := range devices {
		klog.Error("Dev:", id, dev)
		Start(dev)
	}
	wg.Wait()
}
