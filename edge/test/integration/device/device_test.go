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

package device_test

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	. "github.com/kubeedge/kubeedge/edge/test/integration/utils/helpers"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//Devicestate from subscribed MQTT topic
var DeviceState string

type DeviceUpdates struct {
	EventId     string `json:"event_id"`
	Timestamp   int    `json:"timestamp"`
	DeviceField `json:"device"`
}

type DeviceField struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	LastOnline string `json:"last_online"`
}

type MembershipUpdate struct {
	BaseMessage
	AddDevices    []Device `json:"added_devices"`
	RemoveDevices []Device `json:"removed_devices"`
}

type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

var MemDeviceUpdate MembershipUpdate

func SubMessageReceived(client MQTT.Client, message MQTT.Message) {
	var deviceState DeviceUpdates
	topic := dtcommon.DeviceETPrefix + DeviceIDN + dtcommon.DeviceETStateUpdateSuffix + "/result"
	if message.Topic() == topic {
		devicePayload := (message.Payload())
		err := json.Unmarshal(devicePayload, &deviceState)
		if err != nil {
			common.Failf("Unmarshall failed %s", err)
		}
	}
	DeviceState = deviceState.State
	common.InfoV2("device updated %+v", deviceState)
}
func DeviceSubscribed(client MQTT.Client, message MQTT.Message) {
	topic := dtcommon.MemETPrefix + ctx.Cfg.NodeId + dtcommon.MemETUpdateSuffix
	if message.Topic() == topic {
		devicePayload := (message.Payload())
		err := json.Unmarshal(devicePayload, &MemDeviceUpdate)
		if err != nil {
			common.Failf("Unmarshall failed %s", err)
		}
	}
	common.InfoV2("device list is %+v", MemDeviceUpdate)
}

// Deviceid from the DB and assigning to it
var DeviceIDN string
var DeviceN dttype.Device
var DeviceIDWithAttr string
var DeviceATT dttype.Device
var DeviceIDWithTwin string
var DeviceTW dttype.Device

//Run Test cases
var _ = Describe("Event Bus Testing", func() {
	var TokenClient Token
	var ClientOpts *MQTT.ClientOptions
	var Client MQTT.Client
	Context("Publish on eventbus topics throgh MQTT internal broker", func() {
		BeforeEach(func() {
			ClientOpts = HubClientInit(ctx.Cfg.MqttEndpoint, ClientID, "", "")
			Client = MQTT.NewClient(ClientOpts)
			if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Connect() Error is %s", TokenClient.Error())
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			devicetopic := dtcommon.MemETPrefix + ctx.Cfg.NodeId + dtcommon.MemETUpdateSuffix
			topic := dtcommon.DeviceETPrefix + DeviceIDN + dtcommon.DeviceETStateUpdateSuffix + "/result"
			Client.Subscribe(devicetopic, 0, DeviceSubscribed)
			Client.Subscribe(topic, 0, SubMessageReceived)
		})
		AfterEach(func() {
			common.PrintTestcaseNameandStatus()
		})

		It("TC_TEST_EBUS_1: Sending data to Cloud", func() {

			if TokenClient = Client.Publish(UploadRecordToCloud, 0, false, "messagetoUpload_record_to_cloud"); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Publish() Error is %s", TokenClient.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_2: Sending data to device module", func() {
			if TokenClient = Client.Publish(DevicestatusUpdate, 0, false, "messagetoDevice_status_update"); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Publish() Error is %s", TokenClient.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_3: Sending data to device twin module", func() {
			if TokenClient = Client.Publish(DeviceTwinUpdate, 0, false, "messagetoDevice_Twin_update"); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Publish() Error is %s", TokenClient.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_4: Sending data to membership module", func() {
			if TokenClient = Client.Publish(DeviceMembershipUpdate, 0, false, "messagetoDevice_Membership_update"); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Publish() Error is %s", TokenClient.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_5: Sending data to device module", func() {
			if TokenClient = Client.Publish(DeviceUpload, 0, false, "messagetoDevice_upload"); TokenClient.Wait() && TokenClient.Error() != nil {
				common.Failf("client.Publish() Error is %s", TokenClient.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_6: change the device status to online from eventbus", func() {
			var message DeviceUpdate
			message.State = "online"
			topic := dtcommon.DeviceETPrefix + DeviceIDN + dtcommon.DeviceETStateUpdateSuffix + "/result"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}
			Eventually(func() string {
				var deviceEvent Device
				for _, deviceEvent = range MemDeviceUpdate.AddDevices {
					if strings.Compare(deviceEvent.ID, DeviceIDN) == 0 {
						if TokenClient = Client.Publish(dtcommon.DeviceETPrefix+DeviceIDN+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); TokenClient.Wait() && TokenClient.Error() != nil {
							common.Failf("client.Publish() Error is %s", TokenClient.Error())
						} else {
							common.InfoV6("client.Publish Success !!")
						}
					}
				}
				return deviceEvent.ID
			}, "10s", "2s").Should(Equal(DeviceIDN), "Device state is not online within specified time")
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				common.InfoV2("subscribed to the topic %v", topic)
				return DeviceState
			}, "10s", "2s").Should(Equal("online"), "Device state is not online within specified time")
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_7: change the device status to unknown from eventbus", func() {
			var message DeviceUpdate
			message.State = "unknown"
			topic := dtcommon.DeviceETPrefix + DeviceIDN + dtcommon.DeviceETStateUpdateSuffix + "/result"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}
			Eventually(func() string {
				var deviceEvent Device
				for _, deviceEvent = range MemDeviceUpdate.AddDevices {
					if strings.Compare(deviceEvent.ID, DeviceIDN) == 0 {
						if TokenClient = Client.Publish(dtcommon.DeviceETPrefix+DeviceIDN+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); TokenClient.Wait() && TokenClient.Error() != nil {
							common.Failf("client.Publish() Error is %s", TokenClient.Error())
						} else {
							common.InfoV6("client.Publish Success !!")
						}
					}
				}
				return deviceEvent.ID
			}, "10s", "2s").Should(Equal(DeviceIDN), "Device state is not online within specified time")
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				common.InfoV2("subscribed to the topic %v", topic)
				return DeviceState
			}, "10s", "2s").Should(Equal("unknown"), "Device state is not unknown within specified time")
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_8: change the device status to offline from eventbus", func() {
			var message DeviceUpdate
			message.State = "offline"
			topic := dtcommon.DeviceETPrefix + DeviceIDN + dtcommon.DeviceETStateUpdateSuffix + "/result"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}
			Eventually(func() string {
				var deviceEvent Device
				for _, deviceEvent = range MemDeviceUpdate.AddDevices {
					if strings.Compare(deviceEvent.ID, DeviceIDN) == 0 {
						if TokenClient = Client.Publish(dtcommon.DeviceETPrefix+DeviceIDN+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); TokenClient.Wait() && TokenClient.Error() != nil {
							common.Failf("client.Publish() Error is %s", TokenClient.Error())
						} else {
							common.InfoV6("client.Publish Success !!")
						}
					}
				}
				return deviceEvent.ID
			}, "10s", "2s").Should(Equal(DeviceIDN), "Device state is not online within specified time")
			Expect(TokenClient.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				common.InfoV2("subscribed to the topic %v", topic)
				return DeviceState
			}, "10s", "2s").Should(Equal("offline"), "Device state is not offline within specified time")
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_9: Add a sample device with device attributes to kubeedge node", func() {
			//Generating Device ID
			DeviceIDWithAttr = GenerateDeviceID("kubeedge-device-WithDeviceAttributes")
			//Generate a Device
			DeviceATT = CreateDevice(DeviceIDWithAttr, "DeviceATT", "unknown")
			//Add Attribute to device
			AddDeviceAttribute(DeviceATT, "Temperature", "25.25", "float")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, DeviceATT)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetDeviceAttributesFromDB(DeviceIDWithAttr, "Temperature")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Value)
				return attributeDB.Value
			}, "60s", "2s").Should(Equal("25.25"), "Device is not added within specified time")
			Client.Disconnect(1)

		})

		It("TC_TEST_EBUS_10: Add a sample device with Twin attributes to kubeedge node", func() {
			//Generating Device ID
			DeviceIDWithTwin = GenerateDeviceID("kubeedge-device-WithTwinAttributes")
			//Generate a Device
			DeviceTW = CreateDevice(DeviceIDWithTwin, "DeviceTW", "unknown")
			//Add twin attribute
			AddTwinAttribute(DeviceTW, "Temperature", "25.25", "float")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, DeviceTW)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetTwinAttributesFromDB(DeviceIDWithTwin, "Temperature")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Expected)
				return attributeDB.Expected
			}, "60s", "2s").Should(Equal("25.25"), "Device is not added within specified time")
			Client.Disconnect(1)

		})

		It("TC_TEST_EBUS_11: Update existing device with new attributes", func() {

			//Generate a Device
			device := CreateDevice(DeviceIDWithAttr, "DeviceATT", "unknown")
			//Add Attribute to device
			AddDeviceAttribute(device, "Temperature", "50.50", "float")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, device)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetDeviceAttributesFromDB(DeviceIDWithAttr, "Temperature")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Value)
				return attributeDB.Value
			}, "60s", "2s").Should(Equal("50.50"), "Device Attributes are not updated within specified time")
			Client.Disconnect(1)

		})

		It("TC_TEST_EBUS_12: Update existing device with new Twin attributes", func() {

			//Generate a Device
			device := CreateDevice(DeviceIDWithTwin, "DeviceTW", "unknown")
			//Add twin attribute
			AddTwinAttribute(device, "Temperature", "50.50", "float")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, device)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetTwinAttributesFromDB(DeviceIDWithTwin, "Temperature")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Expected)
				return attributeDB.Expected
			}, "60s", "2s").Should(Equal("50.50"), "Device Twin Attributes are not updated within specified time")
			Client.Disconnect(1)

		})

		It("TC_TEST_EBUS_13: Add a new Device attribute to existing device", func() {
			//Adding a new attribute to a device
			AddDeviceAttribute(DeviceATT, "Humidity", "30", "Int")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, DeviceATT)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetDeviceAttributesFromDB(DeviceIDWithAttr, "Humidity")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Value)
				return attributeDB.Value
			}, "60s", "2s").Should(Equal("30"), "Device Attributes are not Added within specified time")
			Client.Disconnect(1)

		})

		It("TC_TEST_EBUS_14: Add a new Twin attribute to existing device", func() {
			//Preparing temporary Twin Attributes
			AddTwinAttribute(DeviceTW, "Humidity", "100.100", "float")

			IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, ctx.Cfg.TestManager+Devicehandler, DeviceTW)
			Expect(IsDeviceAdded).Should(BeTrue())

			Eventually(func() string {
				attributeDB := GetTwinAttributesFromDB(DeviceIDWithTwin, "Humidity")
				common.InfoV2("DeviceID= %s, Value= %s", attributeDB.DeviceID, attributeDB.Expected)
				return attributeDB.Expected
			}, "60s", "2s").Should(Equal("100.100"), "Device Twin Attributes are not Added within specified time")
			Client.Disconnect(1)

		})

	})
})
