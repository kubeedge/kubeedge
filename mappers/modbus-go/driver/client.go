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

package driver

import (
	"errors"
	"sync"
	"time"

	"github.com/sailorvii/modbus"
	"k8s.io/klog/v2"
)

// ModbusTCP is the configurations of modbus TCP.
type ModbusTCP struct {
	SlaveID  byte
	DeviceIP string
	TCPPort  string
	Timeout  time.Duration
}

// ModbusRTU is the configurations of modbus RTU.
type ModbusRTU struct {
	SlaveID      byte
	SerialName   string
	BaudRate     int
	DataBits     int
	StopBits     int
	Parity       string
	RS485Enabled bool
	Timeout      time.Duration
}

// ModbusClient is the structure for modbus client.
type ModbusClient struct {
	Client  modbus.Client
	Handler interface{}
	Config  interface{}

	mu sync.Mutex
}

/*
* In modbus RTU mode, devices could connect to one serial port on RS485. However,
* the serial port doesn't support paralleled visit, and for one tcp device, it also doesn't support
* paralleled visit, so we expect one client for one port.
 */
var clients map[string]*ModbusClient

func newTCPClient(config ModbusTCP) *ModbusClient {
	addr := config.DeviceIP + ":" + config.TCPPort

	if client, ok := clients[addr]; ok {
		return client
	}

	if clients == nil {
		clients = make(map[string]*ModbusClient)
	}

	handler := modbus.NewTCPClientHandler(addr)
	handler.Timeout = config.Timeout
	handler.IdleTimeout = config.Timeout
	handler.SlaveId = config.SlaveID
	client := ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: config}
	clients[addr] = &client
	return &client
}

func newRTUClient(config ModbusRTU) *ModbusClient {
	if client, ok := clients[config.SerialName]; ok {
		return client
	}

	if clients == nil {
		clients = make(map[string]*ModbusClient)
	}

	handler := modbus.NewRTUClientHandler(config.SerialName)
	handler.BaudRate = config.BaudRate
	handler.DataBits = config.DataBits
	handler.Parity = parity(config.Parity)
	handler.StopBits = config.StopBits
	handler.SlaveId = config.SlaveID
	handler.Timeout = config.Timeout
	handler.IdleTimeout = config.Timeout
	handler.RS485.Enabled = config.RS485Enabled
	client := ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: config}
	clients[config.SerialName] = &client
	return &client
}

// NewClient allocate and return a modbus client.
// Client type includes TCP and RTU.
func NewClient(config interface{}) (*ModbusClient, error) {
	switch c := config.(type) {
	case ModbusTCP:
		return newTCPClient(c), nil
	case ModbusRTU:
		return newRTUClient(c), nil
	default:
		return &ModbusClient{}, errors.New("Wrong modbus type")
	}
}

// GetStatus get device status.
// Now we could only get the connection status.
func (c *ModbusClient) GetStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.Client.Connect()
	if err == nil {
		return DEVSTOK
	}
	return DEVSTDISCONN
}

// Get get register.
func (c *ModbusClient) Get(registerType string, addr uint16, quantity uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch registerType {
	case "CoilRegister":
		results, err = c.Client.ReadCoils(addr, quantity)
	case "DiscreteInputRegister":
		results, err = c.Client.ReadDiscreteInputs(addr, quantity)
	case "HoldingRegister":
		results, err = c.Client.ReadHoldingRegisters(addr, quantity)
	case "InputRegister":
		results, err = c.Client.ReadInputRegisters(addr, quantity)
	default:
		return nil, errors.New("Bad register type")
	}
	klog.V(2).Info("Get result: ", results)
	return results, err
}

// Set set register.
func (c *ModbusClient) Set(registerType string, addr uint16, value uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	klog.V(1).Info("Set:", registerType, addr, value)

	switch registerType {
	case "CoilRegister":
		var valueSet uint16
		switch value {
		case 0:
			valueSet = 0x0000
		case 1:
			valueSet = 0xFF00
		default:
			return nil, errors.New("Wrong value")
		}
		results, err = c.Client.WriteSingleCoil(addr, valueSet)
	case "HoldingRegister":
		results, err = c.Client.WriteSingleRegister(addr, value)
	default:
		return nil, errors.New("Bad register type")
	}
	klog.V(1).Info("Set result:", err, results)
	return results, err
}

// parity convert into the format that modbus drvier requires.
func parity(ori string) string {
	var p string
	switch ori {
	case "even":
		p = "E"
	case "odd":
		p = "O"
	default:
		p = "N"
	}
	return p
}
