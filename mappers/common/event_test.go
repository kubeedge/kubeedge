package mappercommon

import (
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	. "github.com/kubeedge/kubeedge/mappers/common"
)

func onMessage(client mqtt.Client, message mqtt.Message) {
	fmt.Println("Get topic", message.Topic())

}
func TestEvent() {
	var c MqttClient = MqttClient{IP: "tcp://127.0.0.1:1883", ClientName: "testevent"}
	err := c.Connect()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("Connect mqtt server success", c.IP)
	}
	err = c.Subscribe("$hw/events/device/#", onMessage)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("Subscribe topic success")
	}
	c.Publish("$hw/events/device/001/data/update", "test")
	for {
		time.Sleep(time.Second)
	}
}
