package globals

import (
	"time"

	"github.com/goburrow/modbus"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
)

type ModbusTcp struct {
	SlaveId  byte
	DeviceIp string
	TcpPort  string
}

type ModbusRtu struct {
	SlaveId    byte
	SerialName string
	BaudRate   int
	DataBits   int
	StopBits   int
	Parity     string
	Timeout    time.Duration
}

type ModbusClient struct {
	Client  modbus.Client
	Handler interface{}
	Config  interface{}
}
type ModbusDev struct {
	Instance     mappercommon.DeviceInstance
	ModbusClient ModbusClient
}

var MqttClient mappercommon.MqttClient
