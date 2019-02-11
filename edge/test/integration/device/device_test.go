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

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	. "github.com/kubeedge/kubeedge/edge/test/integration/utils/helpers"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//getting deviceid from the DB and assigning to it
var DeviceID string

//Run Test cases
var _ = Describe("Event Bus Testing", func() {
	var Token_client Token
	var ClientOpts *MQTT.ClientOptions
	var Client MQTT.Client
	Context("Publish on eventbus topics throgh MQTT internal broker", func() {
		BeforeEach(func() {
			ClientOpts = HubClientInit(ctx.Cfg.MqttEndpoint, ClientID, "", "")
			Client = MQTT.NewClient(ClientOpts)
			if Token_client = Client.Connect(); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Connect() Error is %s", Token_client.Error())
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			common.PrintTestcaseNameandStatus()
		})

		It("TC_TEST_EBUS_1: Sending data to Cloud", func() {
			if Token_client = Client.Publish(UploadRecordToCloud, 0, false, "messagetoUpload_record_to_cloud"); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_2: Sending data to device module", func() {
			if Token_client = Client.Publish(DevicestatusUpdate, 0, false, "messagetoDevice_status_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_3: Sending data to device twin module", func() {
			if Token_client = Client.Publish(DeviceTwinUpdate, 0, false, "messagetoDevice_Twin_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_4: Sending data to membership module", func() {
			if Token_client = Client.Publish(DeviceMembershipUpdate, 0, false, "messagetoDevice_Membership_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_5: Sending data to device module", func() {
			if Token_client = Client.Publish(DeviceUpload, 0, false, "messagetoDevice_upload"); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_6: change the device status to online from eventbus", func() {
			var message DeviceUpdate
			message.State = "online"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}

			if Token_client = Client.Publish(dtcommon.DeviceETPrefix+DeviceID+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				deviceState := GetDeviceStateFromDB(DeviceID)
				common.InfoV2("DeviceID= %s, DeviceState= %s", DeviceID, deviceState)
				return deviceState
			}, "60s", "2s").Should(Equal("online"), "Device state is not online within specified time")

			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_7: change the device status to unknown from eventbus", func() {
			var message DeviceUpdate
			message.State = "unknown"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}
			if Token_client = Client.Publish(dtcommon.DeviceETPrefix+DeviceID+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				deviceState := GetDeviceStateFromDB(DeviceID)
				common.InfoV2("DeviceID= %s, DeviceState= %s", DeviceID, deviceState)
				return deviceState
			}, "60s", "2s").Should(Equal("unknown"), "Device state is not unknown within specified time")
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_8: change the device status to offline from eventbus", func() {
			var message DeviceUpdate
			message.State = "offline"
			body, err := json.Marshal(message)
			if err != nil {
				common.Failf("Marshal failed %v", err)
			}
			if Token_client = Client.Publish(dtcommon.DeviceETPrefix+DeviceID+dtcommon.DeviceETStateUpdateSuffix, 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.Failf("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Eventually(func() string {
				deviceState := GetDeviceStateFromDB(DeviceID)
				common.InfoV2("DeviceID= %s, DeviceState= %s", DeviceID, deviceState)
				return deviceState
			}, "60s", "2s").Should(Equal("offline"), "Device state is not offline within specified time")
			Client.Disconnect(1)
		})
	})
})
