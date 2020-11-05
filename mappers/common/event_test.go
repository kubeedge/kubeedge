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

package mappercommon

import (
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func onMessage(client mqtt.Client, message mqtt.Message) {
	fmt.Println("Get topic", message.Topic())
}

func main() {
	var c MqttClient = MqttClient{IP: "tcp://127.0.0.1:1883"}
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
	err = c.Publish("$hw/events/device/001/data/update", "test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for {
		time.Sleep(time.Second)
	}
}
