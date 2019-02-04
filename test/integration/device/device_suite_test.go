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
	"net/http"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/test/integration/utils/edge"
	. "github.com/kubeedge/kubeedge/test/integration/utils/helpers"

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

const
(
	Devicehandler = "/devices"
)

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
		IsDeviceAdded := HandleAddAndDeleteDevice(http.MethodPut, DeviceID, ctx.Cfg.TestManager+Devicehandler)
		Expect(IsDeviceAdded).Should(BeTrue())
	})
	AfterSuite(func() {
		By("After Suite Executing....!")
		common.InfoV2("Remove Mock device from edgenode !!")
		IsDeviceDeleted := HandleAddAndDeleteDevice(http.MethodDelete, DeviceID, ctx.Cfg.TestManager+Devicehandler)
		Expect(IsDeviceDeleted).Should(BeTrue())
	})

	RunSpecs(t, "edgecore Suite")
}
