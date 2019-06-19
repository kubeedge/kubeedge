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

package device_crd

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	DeviceInstanceHandler = "/apis/devices.kubeedge.io/v1alpha1/namespaces/default/devices"
	DeviceModelHandler    = "/apis/devices.kubeedge.io/v1alpha1/namespaces/default/devicemodels"
	ConfigmapHandler      = "/api/v1/namespaces/default/configmaps"
)

var CRDTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Device Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	Context("Test Device Model Creation, Updation and deletion", func() {
		BeforeEach(func() {
			// Delete any pre-exisitng device models
			var deviceModelList v1alpha1.DeviceModelList
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device models created
			var deviceModelList v1alpha1.DeviceModelList
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_DEVICE_MODEL_1: Create device model for LED device (No Protocol)", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newLedDeviceModel := utils.NewLedDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &newLedDeviceModel)
			Expect(err).To(BeNil())
		})
		It("E2E_CREATE_DEVICE_MODEL_2: Create device model for bluetooth protocol", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "bluetooth")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newBluetoothDeviceModel := utils.NewBluetoothDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &newBluetoothDeviceModel)
			Expect(err).To(BeNil())
		})
		It("E2E_CREATE_DEVICE_MODEL_3: Create device model for modbus protocol", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "modbus")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newModbusDeviceMode := utils.NewModbusDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &newModbusDeviceMode)
			Expect(err).To(BeNil())
		})
		It("E2E_CREATE_DEVICE_MODEL_4: Create device model for incorrect device model", func() {
			_ , statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "incorrect-model")
			Expect(statusCode).Should(Equal(http.StatusUnprocessableEntity))
		})
		It("E2E_UPDATE_DEVICE_MODEL_1: Update device model for LED device (No Protocol)", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceModelUpdated, statusCode := utils.HandleDeviceModel(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+utils.UpdatedLedDeviceModel().Name, "led")
			Expect(IsDeviceModelUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedLedDeviceModel := utils.UpdatedLedDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &updatedLedDeviceModel)
			Expect(err).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_2: Update device model for bluetooth protocol", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "bluetooth")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceModelUpdated, statusCode := utils.HandleDeviceModel(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+utils.UpdatedBluetoothDeviceModel().Name, "bluetooth")
			Expect(IsDeviceModelUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedBluetoothDeviceModel := utils.UpdatedBluetoothDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &updatedBluetoothDeviceModel)
			Expect(err).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_3: Update device model for modbus protocol", func() {
			var deviceModelList v1alpha1.DeviceModelList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "modbus")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceModelUpdated, statusCode := utils.HandleDeviceModel(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+utils.UpdatedModbusDeviceModel().Name, "modbus")
			Expect(IsDeviceModelUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedModbusDeviceModel := utils.UpdatedModbusDeviceModel()
			_, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, &updatedModbusDeviceModel)
			Expect(err).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_4: Update device model for incorrect device model", func() {
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceModelUpdated, statusCode := utils.HandleDeviceModel(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+utils.UpdatedLedDeviceModel().Name, "incorrect-model")
			Expect(IsDeviceModelUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusUnprocessableEntity))
		})
		It("E2E_DELETE_DEVICE_MODEL_1: Delete non existent device model(No Protocol)", func() {
			IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+utils.NewLedDeviceModel().Name, "")
			Expect(IsDeviceModelDeleted).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusNotFound))
		})
	})
	Context("Test Device Instance Creation, Updation and Deletion", func() {
		BeforeEach(func() {
			var deviceModelList v1alpha1.DeviceModelList
			var deviceList v1alpha1.DeviceList
			// Delete the device instances created
			deviceInstanceList, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, nil)
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+device.Name, "")
				Expect(IsDeviceDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete any pre-exisitng device models
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			utils.TwinResult = utils.DeviceTwinResult{}
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var deviceModelList v1alpha1.DeviceModelList
			var deviceList v1alpha1.DeviceList
			// Delete the device instances created
			deviceInstanceList, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, nil)
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+device.Name, "")
				Expect(IsDeviceDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete the device models created
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_DEVICE_1: Create device instance for LED device (No Protocol)", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "led")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newLedDevice := utils.NewLedDeviceInstance(NodeName)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &newLedDevice)
			Expect(err).To(BeNil())
			time.Sleep(3 * time.Second)
			statusCode, body := utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusOK))
			var configMap v1.ConfigMap
			err = json.Unmarshal(body, &configMap)
			Expect(err).To(BeNil())
			isEqual := utils.CompareConfigMaps(configMap, utils.NewConfigMapLED(NodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewLedDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			stringValue := "ON"
			expectedTwin := map[string]*utils.MsgTwin{
				"power-status": {
					Expected: &utils.TwinValue{
						Value: &stringValue,
					},
					Metadata: &utils.TypeMetadata{
						Type: "string",
					},
				},
			}
			isEqual = utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_CREATE_DEVICE_2: Create device instance for bluetooth protocol", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "bluetooth")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "bluetooth")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newBluetoothDevice := utils.NewBluetoothDeviceInstance(NodeName)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &newBluetoothDevice)
			Expect(err).To(BeNil())
			time.Sleep(3 * time.Second)
			statusCode, body := utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusOK))
			var configMap v1.ConfigMap
			err = json.Unmarshal(body, &configMap)
			Expect(err).To(BeNil())
			isEqual := utils.CompareConfigMaps(configMap, utils.NewConfigMapBluetooth(NodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewBluetoothDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			ioData := "1"
			expectedTwin := map[string]*utils.MsgTwin{
				"io-data": {
					Expected: &utils.TwinValue{
						Value: &ioData,
					},
					Metadata: &utils.TypeMetadata{
						Type: "int",
					},
				},
			}
			isEqual = utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_CREATE_DEVICE_3: Create device instance for modbus protocol", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "modbus")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "modbus")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newModbusDevice := utils.NewModbusDeviceInstance(NodeName)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &newModbusDevice)
			Expect(err).To(BeNil())
			time.Sleep(3 * time.Second)
			statusCode, body := utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusOK))
			var configMap v1.ConfigMap
			err = json.Unmarshal(body, &configMap)
			Expect(err).To(BeNil())
			isEqual := utils.CompareConfigMaps(configMap, utils.NewConfigMapModbus(NodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewModbusDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			stringValue := "OFF"
			expectedTwin := map[string]*utils.MsgTwin{
				"temperature-enable": {
					Expected: &utils.TwinValue{
						Value: &stringValue,
					},
					Metadata: &utils.TypeMetadata{
						Type: "string",
					},
				},
			}
			isEqual = utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_CREATE_DEVICE_4: Create device instance for incorrect device instance", func() {
			statusCode := utils.DeleteConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode == http.StatusOK || statusCode == http.StatusNotFound).Should(Equal(true))
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "incorrect-instance")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusUnprocessableEntity))
			statusCode, _ = utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusNotFound))
		})
		It("E2E_UPDATE_DEVICE_1: Update device instance for LED device (No Protocol)", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "led")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceUpdated, statusCode := utils.HandleDeviceInstance(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.UpdatedLedDeviceInstance(NodeName).Name, "led")
			Expect(IsDeviceUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedLedDevice := utils.UpdatedLedDeviceInstance(NodeName)
			time.Sleep(2 * time.Second)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &updatedLedDevice)
			Expect(err).To(BeNil())
			go utils.TwinSubscribe(utils.UpdatedLedDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			stringValue := "OFF"
			expectedTwin := map[string]*utils.MsgTwin{
				"power-status": {
					Expected: &utils.TwinValue{
						Value: &stringValue,
					},
					Metadata: &utils.TypeMetadata{
						Type: "string",
					},
				},
			}
			isEqual := utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_UPDATE_DEVICE_2: Update device instance for bluetooth protocol", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "bluetooth")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "bluetooth")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceUpdated, statusCode := utils.HandleDeviceInstance(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.UpdatedBluetoothDeviceInstance(NodeName).Name, "bluetooth")
			Expect(IsDeviceUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedBluetoothDevice := utils.UpdatedBluetoothDeviceInstance(NodeName)
			time.Sleep(2 * time.Second)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &updatedBluetoothDevice)
			Expect(err).To(BeNil())
			go utils.TwinSubscribe(utils.UpdatedBluetoothDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			ioData := "1"
			expectedTwin := map[string]*utils.MsgTwin{
				"io-data": {
					Expected: &utils.TwinValue{
						Value: &ioData,
					},
					Metadata: &utils.TypeMetadata{
						Type: "int",
					},
				},
			}
			isEqual := utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_UPDATE_DEVICE_3: Update device instance for modbus protocol", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "modbus")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "modbus")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceUpdated, statusCode := utils.HandleDeviceInstance(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.UpdatedModbusDeviceInstance(NodeName).Name, "modbus")
			Expect(IsDeviceUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(NodeName)
			time.Sleep(2 * time.Second)
			_, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &updatedModbusDevice)
			Expect(err).To(BeNil())
			go utils.TwinSubscribe(utils.UpdatedModbusDeviceInstance(NodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
			stringValue := "ON"
			expectedTwin := map[string]*utils.MsgTwin{
				"temperature-enable": {
					Expected: &utils.TwinValue{
						Value: &stringValue,
					},
					Metadata: &utils.TypeMetadata{
						Type: "string",
					},
				},
			}
			isEqual := utils.CompareTwin(utils.TwinResult.Twin, expectedTwin)
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_UPDATE_DEVICE_4: Update device instance for incorrect device instance", func() {
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "led")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceUpdated, statusCode := utils.HandleDeviceInstance(http.MethodPatch, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.UpdatedLedDeviceInstance(NodeName).Name, "incorrect-instance")
			Expect(IsDeviceUpdated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusUnprocessableEntity))
		})
		It("E2E_DELETE_DEVICE_1: Delete device instance for an existing device (No Protocol)", func() {
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "led")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			time.Sleep(1 * time.Second)
			statusCode, _ = utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusOK))
			IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.NewLedDeviceInstance(NodeName).Name, "")
			Expect(IsDeviceDeleted).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			time.Sleep(1 * time.Second)
			statusCode, _ = utils.GetConfigmap(ctx.Cfg.K8SMasterForKubeEdge + ConfigmapHandler + "/" + "device-profile-config-" + NodeName)
			Expect(statusCode).Should(Equal(http.StatusNotFound))
		})
		It("E2E_DELETE_DEVICE_2: Delete device instance for a non-existing device", func() {
			IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+utils.NewLedDeviceModel().Name, "")
			Expect(IsDeviceDeleted).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusNotFound))
		})
	})
	Context("Test Change in device twin", func() {
		BeforeEach(func() {
			var deviceModelList v1alpha1.DeviceModelList
			var deviceList v1alpha1.DeviceList
			// Delete the device instances created
			deviceInstanceList, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, nil)
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+device.Name, "")
				Expect(IsDeviceDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete any pre-exisitng device models
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			utils.TwinResult = utils.DeviceTwinResult{}
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var deviceModelList v1alpha1.DeviceModelList
			var deviceList v1alpha1.DeviceList
			// Delete the device instances created
			deviceInstanceList, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, nil)
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "/"+device.Name, "")
				Expect(IsDeviceDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete the device models created
			list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, nil)
			Expect(err).To(BeNil())
			for _, model := range list {
				IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "/"+model.Name, "")
				Expect(IsDeviceModelDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_TWIN_STATE_1: Change the twin state of an existing device", func() {
			var deviceList v1alpha1.DeviceList
			IsDeviceModelCreated, statusCode := utils.HandleDeviceModel(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceModelHandler, "", "led")
			Expect(IsDeviceModelCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			IsDeviceCreated, statusCode := utils.HandleDeviceInstance(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, NodeName, "", "led")
			Expect(IsDeviceCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newLedDevice := utils.NewLedDeviceInstance(NodeName)
			time.Sleep(3 * time.Second)
			var deviceTwinUpdateMessage utils.DeviceTwinUpdate
			reportedValue := "OFF"
			deviceTwinUpdateMessage.Twin = map[string]*utils.MsgTwin{
				"power-status": {Actual: &utils.TwinValue{Value: &reportedValue}, Metadata: &utils.TypeMetadata{Type: "string"}},
			}
			err := utils.ChangeTwinValue(deviceTwinUpdateMessage, utils.NewLedDeviceInstance(NodeName).Name)
			Expect(err).To(BeNil())
			time.Sleep(3 * time.Second)
			newLedDevice = utils.NewLedDeviceInstance(NodeName)
			list, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+DeviceInstanceHandler, &newLedDevice)
			Expect(err).To(BeNil())
			Expect(list[0].Status.Twins[0].PropertyName).To(Equal("power-status"))
			Expect(list[0].Status.Twins[0].Reported.Value).To(Equal("OFF"))
		})
	})
})
