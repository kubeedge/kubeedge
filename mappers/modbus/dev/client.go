package dev

import (
	"errors"
	"net"
	"time"

	"github.com/goburrow/modbus"
	. "github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

func NewClient(config interface{}) (ModbusClient, error) {
	switch config.(type) {
	case ModbusTcp:
		c, _ := config.(ModbusTcp)
		handler := modbus.NewTCPClientHandler(c.DeviceIp + ":" + c.TcpPort)
		handler.Timeout = 10 * time.Second
		handler.IdleTimeout = 0 // Never idle timeout
		handler.SlaveId = c.SlaveId
		return ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: c}, nil
	case ModbusRtu:
		c, _ := config.(ModbusRtu)
		handler := modbus.NewRTUClientHandler(c.SerialName)
		handler.BaudRate = c.BaudRate
		handler.DataBits = c.DataBits
		handler.Parity = c.Parity
		handler.StopBits = c.StopBits
		handler.SlaveId = c.SlaveId
		handler.Timeout = c.Timeout
		return ModbusClient{Client: modbus.NewClient(handler), Handler: handler, Config: c}, nil
	default:
		c := ModbusClient{}
		return c, errors.New("Wrong modbus type")
	}
}

func GetStatus(c ModbusClient) DevStatus {
	switch c.Config.(type) {
	case ModbusTcp:
		dialer := net.Dialer{Timeout: 500 * time.Millisecond}
		_, err := dialer.Dial("tcp", c.Config.(ModbusTcp).DeviceIp)
		if err != nil {
			return DEVSTDISCONN

		} else {
			return DEVSTOK

		}
	case ModbusRtu:
		return DEVSTUNKNOWN
	default:
		return DEVSTUNKNOWN
	}
}

func Get(c ModbusClient, registerType string, addr uint16, quantity uint16) (results []byte, err error) {
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
	klog.Error("Get")
	klog.Error(results)
	return results, err
}

func Set(c ModbusClient, registerType string, addr uint16, value uint16) (results []byte, err error) {
	klog.Error("Set:", registerType, addr, value)

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
	klog.Error("Set result:", err, results)
	return results, err
}
