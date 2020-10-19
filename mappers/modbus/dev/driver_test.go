package dev

import (
	"fmt"
	"os"
)

func test_driver() {
	var modbusrtu ModbusRtu

	modbusrtu.SerialName = "/dev/ttyS0"
	modbusrtu.BaudRate = 9600
	modbusrtu.DataBits = 8
	modbusrtu.StopBits = 1
	modbusrtu.SlaveId = 1
	modbusrtu.Parity = "N"

	client := NewClient(modbusrtu)
	if client == nil {
		fmt.Println("New client error")
		os.Exit(1)
	}
	fmt.Println("status: ", client.GetStatus())

	results, err := client.Client.ReadCoils(0, 1)
	if err != nil {
		fmt.Println("Read error: ", err)
		os.Exit(1)
	}
	fmt.Println("result: ", results)

	os.Exit(0)

}

func main() {
	var modbustcp ModbusTcp

	modbustcp.DeviceIp = "192.168.56.1"
	modbustcp.TcpPort = "502"
	modbustcp.SlaveId = 0x1
	client := NewClient(modbustcp)
	if client == nil {
		fmt.Println("New client error")
		os.Exit(1)
	}
	fmt.Println("status: ", client.GetStatus())

	results, err := client.Client.ReadDiscreteInputs(0, 1)
	if err != nil {
		fmt.Println("Read error: ", err)
		os.Exit(1)
	}
	fmt.Println("result: ", results)

	os.Exit(0)
}
