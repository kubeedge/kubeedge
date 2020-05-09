/*
Copyright 2019 The KubeEdge Authors.

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

package bluetooth

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/helpers"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/scheduler"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/watcher"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var TokenClient Token
var ClientOpts *MQTT.ClientOptions
var Client MQTT.Client
var timesSpecified int
var timesExecuted int

type ScheduleResult struct {
	EventName   string
	TimeStamp   int64
	EventResult string
}

var scheduleResult ScheduleResult
var dataConverted bool
var readWrittenData bool

// DataConversion checks whether data is properly as expected by the data converter.
func DataConversion(client MQTT.Client, message MQTT.Message) {
	topic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
	expectedTemp := "32.375000"
	dataConverted = false
	if message.Topic() == topic {
		devicePayload := message.Payload()
		err := json.Unmarshal(devicePayload, &scheduleResult)
		if err != nil {
			utils.Fatalf("Unmarshall failed %s", err)
		} else {
			if reflect.DeepEqual(scheduleResult.EventResult, expectedTemp) {
				dataConverted = true
			}
		}
	}
}

// WriteDataReceived checks whether data is properly written to connected device.
func WriteDataReceived(client MQTT.Client, message MQTT.Message) {
	topic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
	readWrittenData = false
	if message.Topic() == topic {
		devicePayload := message.Payload()
		err := json.Unmarshal(devicePayload, &scheduleResult)
		data := []byte{1}
		if err != nil {
			utils.Fatalf("Unmarshall failed %s", err)
		} else {
			if reflect.DeepEqual(string(data), scheduleResult.EventResult) {
				readWrittenData = true
			}
		}
	}
}

// ScheculeExecute counts the number of times schedule is executed by the Scheduler.
func ScheduleExecute(client MQTT.Client, message MQTT.Message) {
	topic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
	expectedTemp := "36"
	if message.Topic() == topic {
		devicePayload := message.Payload()
		err := json.Unmarshal(devicePayload, &scheduleResult)
		if err != nil {
			utils.Fatalf("Unmarshall failed %s", err)
		} else {
			if reflect.DeepEqual(scheduleResult.EventResult, expectedTemp) {
				timesExecuted++
			}
		}
	}
}

// Checks whether mapper has written data correctly to connected device
var _ = Describe("Application deployment test in E2E scenario", func() {
	Context("Test write characteristic of mapper", func() {
		BeforeEach(func() {
			// Subscribing to topic where scheduler publishes the data
			ClientOpts = helpers.HubClientInit(ctx.Cfg.MqttEndpoint, "bluetoothmapper", "", "")
			Client = MQTT.NewClient(ClientOpts)
			if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("client.Connect() Error is %s", TokenClient.Error())
			} else {
				utils.Infof("Connection successful")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			scheduletopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
			Token := Client.Subscribe(scheduletopic, 0, WriteDataReceived)
			if Token.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Subscribe to Topic  Failed  %s, %s", TokenClient.Error(), scheduletopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			Client.Disconnect(1)
			common.PrintTestcaseNameandStatus()
		})
		It("E2E_WRITE: Ensure whether mapper is able to write data and read it back from connected device", func() {
			//creating deployment from deployment yaml of bluetooth mapper
			curpath := getpwd()
			file := path.Join(curpath, deployPath)
			body, err := ioutil.ReadFile(file)
			Expect(err).Should(BeNil())
			client := &http.Client{}
			BodyBuf := bytes.NewReader(body)
			req, err := http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+deploymentHandler, BodyBuf)
			Expect(err).Should(BeNil())
			req.Header.Set("Content-Type", "application/yaml")
			resp, err := client.Do(req)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+appHandler, nodeName)
			utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, podlist, 240*time.Second)
			Eventually(func() bool {
				return readWrittenData
			}, "120s", "0.5s").ShouldNot(Equal(false), "Message is not received in expected time !!")
		})
	})

	// Checking whether dataconverter works correctly
	Context("Test whether dataconverter works properly", func() {
		BeforeEach(func() {
			// Subscribing to topic where scheduler publishes the data
			ClientOpts = helpers.HubClientInit(ctx.Cfg.MqttEndpoint, "bluetoothmapper", "", "")
			Client = MQTT.NewClient(ClientOpts)
			if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("client.Connect() Error is %s", TokenClient.Error())
			} else {
				utils.Infof("Subscribe Connection Successful")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			scheduletopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
			Token := Client.Subscribe(scheduletopic, 0, DataConversion)
			if Token.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Subscribe to Topic  Failed  %s, %s", TokenClient.Error(), scheduletopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			var expectedSchedule []scheduler.Schedule
			// Create and publish run time data schedule for data conversion.
			schedule := scheduler.Schedule{Name: "temperatureconversion", Interval: 3000, OccurrenceLimit: 1, Actions: []string{"ConvertTemperatureData"}}
			expectedSchedule = []scheduler.Schedule{schedule}
			scheduleCreateTopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/create"
			scheduleCreate, err := json.Marshal(expectedSchedule)
			if err != nil {
				utils.Fatalf("Error in marshalling: %s", err)
			}
			TokenClient = Client.Publish(scheduleCreateTopic, 0, false, scheduleCreate)
			if TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Publish to Topic  Failed  %s, %s", TokenClient.Error(), scheduleCreateTopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
		})
		It("E2E_DATACONVERTER: Test whether dataconverter has performed conversion correctly", func() {
			Eventually(func() bool {
				return dataConverted
			}, "30s", "0.5s").Should(Equal(true))
		})
		AfterEach(func() {
			Client.Disconnect(1)
			common.PrintTestcaseNameandStatus()
		})
	})

	// Checking whether schedule executed the expected number of times using runtime update
	Context("Test whether schedule is executed properly", func() {
		BeforeEach(func() {
			// Subscribing to topic where scheduler publishes the data
			ClientOpts = helpers.HubClientInit(ctx.Cfg.MqttEndpoint, "bluetoothmapper", "", "")
			Client = MQTT.NewClient(ClientOpts)
			if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("client.Connect() Error is %s", TokenClient.Error())
			} else {
				utils.Infof("Subscribe Connection successful")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			scheduletopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/result"
			Token := Client.Subscribe(scheduletopic, 0, ScheduleExecute)
			if Token.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Subscribe to Topic  Failed  %s, %s", TokenClient.Error(), scheduletopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			var expectedSchedule []scheduler.Schedule
			// Create and publish run time data for scheduler
			schedule := scheduler.Schedule{Name: "temperature", Interval: 1000, OccurrenceLimit: 5, Actions: []string{"IRTemperatureData"}}
			expectedSchedule = []scheduler.Schedule{schedule}
			scheduleCreateTopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/scheduler/create"
			scheduleCreate, err := json.Marshal(expectedSchedule)
			if err != nil {
				utils.Fatalf("Error in marshalling: %s", err)
			}
			TokenClient = Client.Publish(scheduleCreateTopic, 0, false, scheduleCreate)
			if TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Publish to Topic  Failed  %s, %s", TokenClient.Error(), scheduleCreateTopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
		})
		It("E2E_SCHEDULER: Test whether the schedule is executed the specified number of times", func() {
			timesSpecified = 5
			time.Sleep(10 * time.Second)
			Expect(timesExecuted).To(Equal(timesSpecified))
		})
		AfterEach(func() {
			Client.Disconnect(1)
			common.PrintTestcaseNameandStatus()
		})
	})

	// check whether watcher has performed its operation successfully.
	Context("Test whether watcher has successfully updated the twin state as desired", func() {
		BeforeEach(func() {
			var expectedWatchAttribute watcher.Watcher
			// Create and publish scheduler run time data
			watchAttribute := watcher.Attribute{Name: "io-data", Actions: []string{"IOData"}}
			devTwinAtt := []watcher.Attribute{watchAttribute}
			expectedWatchAttribute = watcher.Watcher{DeviceTwinAttributes: devTwinAtt}
			// Subscribing to topic where scheduler publishes the data
			ClientOpts = helpers.HubClientInit(ctx.Cfg.MqttEndpoint, "bluetoothmapper", "", "")
			Client = MQTT.NewClient(ClientOpts)
			if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("client.Connect() Error is %s", TokenClient.Error())
			} else {
				utils.Infof("Publish Connection successful")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			watcherCreateTopic := "$ke/device/bluetooth-mapper/mock-temp-sensor-instance/watcher/create"
			scheduleCreate, err := json.Marshal(expectedWatchAttribute)
			if err != nil {
				utils.Fatalf("Error in marshalling: %s", err)
			}
			TokenClient = Client.Publish(watcherCreateTopic, 0, false, scheduleCreate)
			if TokenClient.Wait() && TokenClient.Error() != nil {
				utils.Fatalf("Publish to Topic  Failed  %s, %s", TokenClient.Error(), watcherCreateTopic)
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
		})
		It("E2E_WATCHER: Test whether the watcher performs it operation correctly", func() {
			var deviceList v1alpha1.DeviceList
			newLedDevice := utils.NewMockInstance(nodeName)
			time.Sleep(20 * time.Second)
			list, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+mockInstanceHandler, &newLedDevice)
			Expect(err).To(BeNil())
			Expect(list[0].Status.Twins[0].PropertyName).To(Equal("io-data"))
			Expect(list[0].Status.Twins[0].Reported.Value).To(Equal("Red"))
		})
		AfterEach(func() {
			Client.Disconnect(1)
			common.PrintTestcaseNameandStatus()
		})
	})
})
