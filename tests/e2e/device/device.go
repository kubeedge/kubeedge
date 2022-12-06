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
	"context"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	off = "OFF"
)

// Run Test cases
var _ = GroupDescribe("Device Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.GinkgoTestDescription
	var clientSet clientset.Interface
	var edgeClientSet edgeclientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
		edgeClientSet = utils.NewKubeEdgeClient(framework.TestContext.KubeConfig)
	})

	ginkgo.Context("Test Device Model Creation, Updation and deletion", func() {
		ginkgo.BeforeEach(func() {
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.TestText)
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
		framework.ConformanceIt("E2E_CREATE_DEVICE_MODEL_1: Create device model for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			newLedDeviceModel := utils.NewLedDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &newLedDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_CREATE_DEVICE_MODEL_2: Create device model for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			newBluetoothDeviceModel := utils.NewBluetoothDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &newBluetoothDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_CREATE_DEVICE_MODEL_3: Create device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			newModbusDeviceMode := utils.NewModbusDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &newModbusDeviceMode)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_1: Update device model for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedLedDeviceModel().Name, "led")
			gomega.Expect(err).To(gomega.BeNil())
			updatedLedDeviceModel := utils.UpdatedLedDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedLedDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_2: Update device model for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedBluetoothDeviceModel().Name, "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			updatedBluetoothDeviceModel := utils.UpdatedBluetoothDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedBluetoothDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_3: Update device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedModbusDeviceModel().Name, "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			updatedModbusDeviceModel := utils.UpdatedModbusDeviceModel()
			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedModbusDeviceModel)).To(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_UPDATE_DEVICE_MODEL_4: Update device model for incorrect device model", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedLedDeviceModel().Name, "incorrect-model")
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
		framework.ConformanceIt("E2E_DELETE_DEVICE_MODEL_1: Delete non existent device model(No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, utils.NewLedDeviceModel().Name, "")
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
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.TestText)
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
		ginkgo.It("E2E_CREATE_DEVICE_1: Create device instance for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			newLedDevice := utils.NewLedDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapLED(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
			go utils.TwinSubscribe(utils.NewLedDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_CREATE_DEVICE_2: Create device instance for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			newBluetoothDevice := utils.NewBluetoothDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newBluetoothDevice)
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapBluetooth(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
			go utils.TwinSubscribe(utils.NewBluetoothDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_CREATE_DEVICE_3: Create device instance for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			newModbusDevice := utils.NewModbusDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapModbus(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
			go utils.TwinSubscribe(utils.NewModbusDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
			stringValue := off
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_CREATE_DEVICE_4: Create device instance for customized protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "customized")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "customized")
			gomega.Expect(err).To(gomega.BeNil())
			newCustomizedDevice := utils.NewCustomizedDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newCustomizedDevice)
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapCustomized(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
			go utils.TwinSubscribe(utils.NewCustomizedDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_UPDATE_DEVICE_1: Update device instance for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())

			newLedDevice := utils.NewLedDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)
			gomega.Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
				return err == nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedLedDeviceInstance(constants.NodeName).Name, "led")
			gomega.Expect(err).To(gomega.BeNil())
			updatedLedDevice := utils.UpdatedLedDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedLedDevice)
			gomega.Expect(err).To(gomega.BeNil())

			go utils.TwinSubscribe(utils.UpdatedLedDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
			stringValue := off
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_UPDATE_DEVICE_2: Update device instance for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())

			newBluetoothDevice := utils.NewBluetoothDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)
			gomega.Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newBluetoothDevice)
				return err == nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedBluetoothDeviceInstance(constants.NodeName).Name, "bluetooth")
			gomega.Expect(err).To(gomega.BeNil())
			updatedBluetoothDevice := utils.UpdatedBluetoothDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedBluetoothDevice)
			gomega.Expect(err).To(gomega.BeNil())

			go utils.TwinSubscribe(utils.UpdatedBluetoothDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_UPDATE_DEVICE_3: Update device instance for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "modbus")
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

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedModbusDeviceInstance(constants.NodeName).Name, "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(constants.NodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())

			go utils.TwinSubscribe(utils.UpdatedModbusDeviceInstance(constants.NodeName).Name)
			gomega.Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(gomega.Equal(true), "Device information not reaching edge!!")
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
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_UPDATE_DEVICE_4: Update device instance for incorrect device instance", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedLedDeviceInstance(constants.NodeName).Name, "incorrect-instance")
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
		ginkgo.It("E2E_UPDATE_DEVICE_4: Update device instance data and twin for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			newModbusDevice := utils.NewModbusDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapModbus(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
			// update twins and data section should reflect on change on config map
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, constants.NodeName, utils.UpdatedModbusDeviceInstance(constants.NodeName).Name, "modbus")
			gomega.Expect(err).To(gomega.BeNil())
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(constants.NodeName)
			time.Sleep(3 * time.Second)

			deviceInstanceList, err = utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedModbusDevice)
			gomega.Expect(err).To(gomega.BeNil())

			updatedConfigMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			updatedConfigMap.TypeMeta = metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			}

			isEqual = utils.CompareDeviceProfileInConfigMaps(*updatedConfigMap, utils.UpdatedConfigMapModbusForDataAndTwins(constants.NodeName))
			gomega.Expect(isEqual).Should(gomega.Equal(true))
		})
		ginkgo.It("E2E_DELETE_DEVICE_1: Delete device instance for an existing device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, utils.NewLedDeviceInstance(constants.NodeName).Name, "")
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue())
		})
		ginkgo.It("E2E_DELETE_DEVICE_2: Delete device instance for a non-existing device", func() {
			err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, utils.NewLedDeviceModel().Name, "")
			gomega.Expect(err).To(gomega.BeNil())
		})
		ginkgo.It("E2E_DELETE_DEVICE_3: Delete device instance without device model", func() {
			err := utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, constants.NodeName, utils.NewLedDeviceInstance(constants.NodeName).Name, "")
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+constants.NodeName, metav1.GetOptions{})
			gomega.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue())
		})
	})
	ginkgo.Context("Test Change in device twin", func() {
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
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.TestText)
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
		ginkgo.It("E2E_TWIN_STATE_1: Change the twin state of an existing device", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, constants.NodeName, "", "led")
			gomega.Expect(err).To(gomega.BeNil())
			newLedDevice := utils.NewLedDeviceInstance(constants.NodeName)
			time.Sleep(3 * time.Second)
			var deviceTwinUpdateMessage utils.DeviceTwinUpdate
			reportedValue := off
			deviceTwinUpdateMessage.Twin = map[string]*utils.MsgTwin{
				"power-status": {Actual: &utils.TwinValue{Value: &reportedValue}, Metadata: &utils.TypeMetadata{Type: "string"}},
			}
			err = utils.ChangeTwinValue(deviceTwinUpdateMessage, utils.NewLedDeviceInstance(constants.NodeName).Name)
			gomega.Expect(err).To(gomega.BeNil())
			time.Sleep(3 * time.Second)
			newLedDevice = utils.NewLedDeviceInstance(constants.NodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(deviceInstanceList[0].Status.Twins[0].PropertyName).To(gomega.Equal("power-status"))
			gomega.Expect(deviceInstanceList[0].Status.Twins[0].Reported.Value).To(gomega.Equal(off))
		})
	})
})
