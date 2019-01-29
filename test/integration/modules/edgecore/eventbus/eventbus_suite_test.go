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

package integration

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/test/integration/utils/edge"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx *edge.TestContext
	cfg edge.Config
)

//Interface to validate the MQTT connection.
type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

var (
	//deviceupload topic
	DeviceUpload = "$hw/events/upload/#"
	//device status update topic "$hw/events/device/+/state/update"
	DevicestatusUpdate = dtcommon.DeviceETPrefix + "+" + dtcommon.DeviceETStateUpdateSuffix
	//device twin update topic "$hw/events/device/+/twin/+"
	DeviceTwinUpdate = dtcommon.DeviceETPrefix + "+" + dtcommon.DeviceTwinModule + "/+"
	//device membership update topic "$hw/events/node/+/membership/get"
	DeviceMembershipUpdate = dtcommon.MemETPrefix + "+" + dtcommon.MemETGetSuffix
	//upload record to cloud topic
	UploadRecordToCloud = "SYS/dis/upload_records"
	//client id used in MQTT connection
	ClientID = "eventbus"
)

// HubclientInit create mqtt client config
func HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

//function to handle device addition and deletion.
func HandleAddAndDeleteDevice(operation string) bool {
	var req *http.Request
	var err error

	client := &http.Client{}
	switch operation {
	case "PUT":
		payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{
			{
				ID:          DeviceID,
				Name:        "edgedevice",
				Description: "integrationtest",
				State:       "unknown",
			}}}
		respbytes, err := json.Marshal(payload)
		if err != nil {
			common.Failf("Add device to edge_core DB is failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPut, ctx.Cfg.TestManager+"/devices", bytes.NewBuffer(respbytes))
	case "DELETE":
		payload := dttype.MembershipUpdate{RemoveDevices: []dttype.Device{
			{
				ID:          DeviceID,
				Name:        "edgedevice",
				Description: "integrationtest",
				State:       "unknown",
			}}}
		respbytes, err := json.Marshal(payload)
		if err != nil {
			common.Failf("Remove device from edge_core DB failed: %v", err)
			return false
		}
		req, err = http.NewRequest(http.MethodDelete, ctx.Cfg.TestManager+"/devices", bytes.NewBuffer(respbytes))
	}
	if err != nil {
		// handle error
		common.Failf("Open Sqlite DB failed :%v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	t := time.Now()
	resp, err := client.Do(req)
	common.InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	if err != nil {
		// handle error
		common.Failf("HTTP request is failed :%v", err)
		return false
	}
	return true
}

//Function to run the Ginkgo Test
func TestEdgecoreEventBus(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		common.InfoV6("Before Suite execution")

		cfg = edge.LoadConfig()
		ctx = edge.NewTestContext(cfg)
		common.InfoV2("Adding Mock device to edgenode !!")
		//Generate the random string and assign as a DeviceID
		DeviceID = "kubeedge-device-" + edge.GetRandomString(10)
		IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut)
		Expect(IsDeviceAdded).Should(BeTrue())
	})
	AfterSuite(func() {
		By("After Suite Executing....!")
		common.InfoV2("Remove Mock device from edgenode !!")
		IsDeviceDeleted := HandleAddAndDeleteDevice(http.MethodDelete)
		Expect(IsDeviceDeleted).Should(BeTrue())
	})

	RunSpecs(t, "edgecore Suite")
}
