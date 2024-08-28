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

package device

import (
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/kubernetes/test/e2e/framework"

	edgeclientset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	off = "OFF"
)

// Run Test cases
var _ = GroupDescribe("Device Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.SpecReport
	var edgeClientSet edgeclientset.Interface

	ginkgo.BeforeEach(func() {
		edgeClientSet = utils.NewKubeEdgeClient(framework.TestContext.KubeConfig)
	})

	ginkgo.Context("Test Device Model Creation, Updation and Deletion", func() {
		ginkgo.BeforeEach(func() {
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentSpecReport()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device models created
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			utils.PrintTestcaseNameandStatus()
		})
		framework.ConformanceIt("E2E_CREATE_DEVICE_MODEL_1: Create device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			newModbusDeviceMode := utils.NewModbusDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &newModbusDeviceMode)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_1: Update device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedModbusDeviceModel().Name, utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			updatedModbusDeviceModel := utils.UpdatedModbusDeviceModel()
			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedModbusDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_2: Update device model for incorrect device model", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedModbusDeviceModel().Name, utils.IncorrectModel)
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_DELETE_DEVICE_MODEL_1: Delete non existent device model(No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, utils.NewModbusDeviceModel().Name, "")
			gomega.Expect(err).To(gomega.BeNil())
		})
	})
	ginkgo.Context("Test Device Instance Creation, Updation and Deletion", func() {
		ginkgo.BeforeEach(func() {
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, device.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			utils.TwinResult = utils.DeviceTwinResult{}
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentSpecReport()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, device.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Delete the device models created
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			utils.PrintTestcaseNameandStatus()
		})
		ginkgo.It("E2E_CREATE_DEVICE_1: Create device instance for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			newModbusDevice := utils.NewModbusDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())
		})
		ginkgo.It("E2E_UPDATE_DEVICE_1: Update device instance for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())

			newModbusDevice := utils.NewModbusDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)
			gomega.Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
				return err == nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedModbusDeviceInstance(constants.NodeName).Name, utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())
		})
		ginkgo.It("E2E_UPDATE_DEVICE_2: Update device instance for incorrect device instance", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedModbusDeviceInstance(constants.NodeName).Name, utils.IncorrectInstance)
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
		ginkgo.It("E2E_DELETE_DEVICE_1: Delete device instance for an existing device", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", utils.ModBus)
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, utils.NewModbusDeviceInstance(constants.NodeName).Name, "")
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			newModbusDevice := utils.NewModbusDeviceInstance(constants.NodeName)
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
		ginkgo.It("E2E_DELETE_DEVICE_2: Delete device instance for a non-existing device", func() {
			err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, utils.NewModbusDeviceModel().Name, "")
			gomega.Expect(err).To(gomega.BeNil())
		})
	})
})
