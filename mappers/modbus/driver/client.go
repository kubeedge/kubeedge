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
	"time"

	//. "github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"github.com/sailorvii/modbus"
	"k8s.io/klog"
)

// ModbusTcp is the configurations of modbus TCP.
type ModbusTcp struct {
	SlaveId  byte
	DeviceIp string
	TcpPort  string
	Timeout  time.Duration
}

// ModbusRtu is the configurations of modbus RTU.
type ModbusRtu struct {
	SlaveId      byte
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
}

// NewClient allocate and return a modbus client.
// Client type includes TCP and RTU.
func NewClient(config interface{}) (ModbusClient, error) {
	switch config.(type) {
	case ModbusTcp:
		c, _ := config.(ModbusTcp)
		handler := modbus.NewTCPClientHandler(c.DeviceIp + ":" + c.TcpPort)
		handler.Timeout = c.Timeout
		handler.IdleTimeout = c.Timeout
		handler.SlaveId = c.SlaveId
		return ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: c}, nil
	case ModbusRtu:
		c, _ := config.(ModbusRtu)
		handler := modbus.NewRTUClientHandler(c.SerialName)
		handler.BaudRate = c.BaudRate
		handler.DataBits = c.DataBits
		handler.Parity = parity(c.Parity)
		handler.StopBits = c.StopBits
		handler.SlaveId = c.SlaveId
		handler.Timeout = c.Timeout
		handler.IdleTimeout = c.Timeout
		handler.RS485.Enabled = c.RS485Enabled
		return ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: c}, nil
	default:
		return ModbusClient{}, errors.New("Wrong modbus type")
	}
}

// GetStatus get device status.
// Now we could only get the connection status.
func (c *ModbusClient) GetStatus() string {
	err := c.Client.Connect()
	if err == nil {
		return DEVSTOK
	} else {
		return DEVSTDISCONN
	}
}

// Get get register.
func (c *ModbusClient) Get(registerType string, addr uint16, quantity uint16) (results []byte, err error) {
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
	klog.Info("Get result: ", results)
	return results, err
}

// Set set register.
func (c *ModbusClient) Set(registerType string, addr uint16, value uint16) (results []byte, err error) {
	klog.Info("Set:", registerType, addr, value)

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
	case "DiscreteInputRegister":
		results, err = c.Client.WriteSingleRegister(addr, value)
	default:
		return nil, errors.New("Bad register type")
	}
	klog.Info("Set result:", err, results)
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
