package driver

import (
	"fmt"
	"os"
	"time"

	. "github.com/kubeedge/kubeedge/mappers/modbus/dev"
	. "github.com/kubeedge/kubeedge/mappers/modbus/globals"
)

func test_driver() {
	var modbusrtu ModbusRtu

	modbusrtu.SerialName = "/dev/ttyS0"
	modbusrtu.BaudRate = 9600
	modbusrtu.DataBits = 8
	modbusrtu.StopBits = 1
	modbusrtu.SlaveId = 1
	modbusrtu.Parity = "N"
	modbusrtu.Timeout = 2 * time.Second

	client, err := NewClient(modbusrtu)
	if err != nil {
		fmt.Println("New client error")
		os.Exit(1)
	}

	results, err := Set(client, "DiscreteInputRegister", 2, 1)
	if err != nil {
		fmt.Println("Read error: ", err)
	}
	fmt.Println("result: ", results)
	for {
		results, err = Set(client, "CoilRegister", 2, 1)
		if err != nil {
			fmt.Println("Read error: ", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	os.Exit(0)

}

func main() {
	/*
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
	*/
	test_driver()
	os.Exit(0)
}
