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

package e2e

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/kubeedge/kubeedge/test/Edge-SDV/utils/common"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//DeviceUpdate device update
type DeviceUpdate struct {
	State string `json:"state,omitempty"`
}

func checkDeviceState(deviceId string) string {

	var id []uint8
	var name string
	var Description string
	var State string
	var LastOnline []uint8

	pwd, err := os.Getwd()
	if err != nil {
		common.InfoV6("Failed to get PWD :%v", err)
		os.Exit(1)
	}

	destpath := filepath.Join(pwd, "../../edge.db")

	db, err := sql.Open("sqlite3", destpath)
	if err != nil {
		common.InfoV6("Open Sqlite DB failed :%v", err)
	}
	defer db.Close()

	state, _ := db.Query("SELECT * FROM device", 0)
	if err != nil {
		common.InfoV6("Query Sqlite DB failed :%v", err)
	}
	defer state.Close()

	for state.Next() {
		err = state.Scan(&id, &name, &Description, &State, &LastOnline)
		if err != nil {
			common.InfoV6("Failed to scan DB rows :%v", err)
		}

		if string(id) == deviceId {
			break
		}
	}

	return State
}

var _ = Describe("Event BUS Testing", func() {
	var Token_client Token
	var ClientOpts *MQTT.ClientOptions
	var Client MQTT.Client
	Context("Individual Create", func() {

		BeforeEach(func() {
			ClientOpts = HubclientInit(ctx.Cfg.MqttEndpoint, clientID, "", "")

			Client = MQTT.NewClient(ClientOpts)
			if Token_client = Client.Connect(); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Connect() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Connect success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())

		})
		AfterEach(func() {
			common.PrintTestcaseNameandStatus()
		})
		It("TC_TEST_EBUS_1 :Sending Messge on SYS/dis/upload_records Topic", func() {

			if Token_client = Client.Publish(Upload_record_to_cloud, 0, false, "messagetoUpload_record_to_cloud"); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())

			Client.Disconnect(1)
		})
		It("TC_TEST_EBUS_2 :Sending Messge on $hw/events/device/+/state/update Topic", func() {

			if Token_client = Client.Publish(Device_status_update, 0, false, "messagetoDevice_status_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}

			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})
		It("TC_TEST_EBUS_3 :Sending Messge on $hw/events/device/+/twin/+ Topic", func() {

			if Token_client = Client.Publish(Device_Twin_update, 0, false, "messagetoDevice_Twin_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}

			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})
		It("TC_TEST_EBUS_4 :Sending Messge on $hw/events/node/+/membership/get Topic", func() {

			if Token_client = Client.Publish(Device_Membership_update, 0, false, "messagetoDevice_Membership_update"); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}

			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})
		It("TC_TEST_EBUS_5 :Sending Messge on $hw/events/upload/# Topic", func() {

			if Token_client = Client.Publish(Device_upload, 0, false, "messagetoDevice_upload"); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}

			Expect(Token_client.Error()).NotTo(HaveOccurred())
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_6 :Sending Messge on $hw/events/upload/# Topic", func() {
			var message DeviceUpdate
			var deviceId string
			message.State = "offline"
			body, _ := json.Marshal(message)

			deviceId = ctx.Cfg.DeviceId

			if Token_client = Client.Publish("$hw/events/device/"+deviceId+"/state/update", 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			time.Sleep(1 * time.Second)
			DeviceState := checkDeviceState(deviceId)
			Expect(DeviceState).Should(Equal("offline"))
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_7 :Sending Messge on $hw/events/upload/# Topic", func() {
			var message DeviceUpdate
			message.State = "online"
			body, _ := json.Marshal(message)
			deviceId := ctx.Cfg.DeviceId

			if Token_client = Client.Publish("$hw/events/device/"+deviceId+"/state/update", 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			time.Sleep(1 * time.Second)
			DeviceState := checkDeviceState(deviceId)
			Expect(DeviceState).Should(Equal("online"))
			Client.Disconnect(1)
		})

		It("TC_TEST_EBUS_8 :Sending Messge on $hw/events/upload/# Topic", func() {
			var message DeviceUpdate
			message.State = "unknown"
			body, _ := json.Marshal(message)
			deviceId := ctx.Cfg.DeviceId

			if Token_client = Client.Publish("$hw/events/device/"+deviceId+"/state/update", 0, false, body); Token_client.Wait() && Token_client.Error() != nil {
				common.InfoV6("client.Publish() Error is %s", Token_client.Error())
			} else {
				common.InfoV6("client.Publish Success !!")
			}
			Expect(Token_client.Error()).NotTo(HaveOccurred())
			time.Sleep(1 * time.Second)
			DeviceState := checkDeviceState(deviceId)
			Expect(DeviceState).Should(Equal("unknown"))
			Client.Disconnect(1)
		})

	})
})
