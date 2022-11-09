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

package deployment

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	off = "OFF"
)

var CRDTestTimerGroup = utils.NewTestTimerGroup()

// Run Test cases
var _ = Describe("Device Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport SpecReport
	var clientSet clientset.Interface
	var edgeClientSet edgeclientset.Interface

	BeforeEach(func() {
		clientSet = utils.NewKubeClient(ctx.Cfg.KubeConfigPath)
		edgeClientSet = utils.NewKubeEdgeClient(ctx.Cfg.KubeConfigPath)
	})

	Context("Test Device Model Creation, Updation and deletion", func() {
		BeforeEach(func() {
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device models created
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_DEVICE_MODEL_1: Create device model for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			newLedDeviceModel := utils.NewLedDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &newLedDeviceModel)).To(BeNil())
		})
		It("E2E_CREATE_DEVICE_MODEL_2: Create device model for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			Expect(err).To(BeNil())
			newBluetoothDeviceModel := utils.NewBluetoothDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &newBluetoothDeviceModel)).To(BeNil())
		})
		It("E2E_CREATE_DEVICE_MODEL_3: Create device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			Expect(err).To(BeNil())
			newModbusDeviceMode := utils.NewModbusDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &newModbusDeviceMode)).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_1: Update device model for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedLedDeviceModel().Name, "led")
			Expect(err).To(BeNil())
			updatedLedDeviceModel := utils.UpdatedLedDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedLedDeviceModel)).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_2: Update device model for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedBluetoothDeviceModel().Name, "bluetooth")
			Expect(err).To(BeNil())
			updatedBluetoothDeviceModel := utils.UpdatedBluetoothDeviceModel()

			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedBluetoothDeviceModel)).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_3: Update device model for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedModbusDeviceModel().Name, "modbus")
			Expect(err).To(BeNil())
			updatedModbusDeviceModel := utils.UpdatedModbusDeviceModel()
			deviceModelList, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())

			Expect(utils.CheckDeviceModelExists(deviceModelList, &updatedModbusDeviceModel)).To(BeNil())
		})
		It("E2E_UPDATE_DEVICE_MODEL_4: Update device model for incorrect device model", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceModel(edgeClientSet, http.MethodPatch, utils.UpdatedLedDeviceModel().Name, "incorrect-model")
			Expect(err).NotTo(BeNil())
		})
		It("E2E_DELETE_DEVICE_MODEL_1: Delete non existent device model(No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, utils.NewLedDeviceModel().Name, "")
			Expect(err).To(BeNil())
		})
	})
	Context("Test Device Instance Creation, Updation and Deletion", func() {
		BeforeEach(func() {
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, device.Name, "")
				Expect(err).To(BeNil())
			}
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			utils.TwinResult = utils.DeviceTwinResult{}
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, device.Name, "")
				Expect(err).To(BeNil())
			}
			// Delete the device models created
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_DEVICE_1: Create device instance for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())
			newLedDevice := utils.NewLedDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
			Expect(err).To(BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapLED(nodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewLedDeviceInstance(nodeName).Name)
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
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "bluetooth")
			Expect(err).To(BeNil())
			newBluetoothDevice := utils.NewBluetoothDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newBluetoothDevice)
			Expect(err).To(BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapBluetooth(nodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewBluetoothDeviceInstance(nodeName).Name)
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
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "modbus")
			Expect(err).To(BeNil())
			newModbusDevice := utils.NewModbusDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			Expect(err).To(BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapModbus(nodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewModbusDeviceInstance(nodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
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
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_CREATE_DEVICE_4: Create device instance for customized protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "customized")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "customized")
			Expect(err).To(BeNil())
			newCustomizedDevice := utils.NewCustomizedDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newCustomizedDevice)
			Expect(err).To(BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapCustomized(nodeName))
			Expect(isEqual).Should(Equal(true))
			go utils.TwinSubscribe(utils.NewCustomizedDeviceInstance(nodeName).Name)
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
		It("E2E_UPDATE_DEVICE_1: Update device instance for LED device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())

			newLedDevice := utils.NewLedDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)
			Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
				return err == nil
			}, "20s", "2s").Should(Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, nodeName, utils.UpdatedLedDeviceInstance(nodeName).Name, "led")
			Expect(err).To(BeNil())
			updatedLedDevice := utils.UpdatedLedDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedLedDevice)
			Expect(err).To(BeNil())

			go utils.TwinSubscribe(utils.UpdatedLedDeviceInstance(nodeName).Name)
			Eventually(func() bool {
				return utils.TwinResult.Twin != nil
			}, "20s", "2s").Should(Equal(true), "Device information not reaching edge!!")
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
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_UPDATE_DEVICE_2: Update device instance for bluetooth protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "bluetooth")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "bluetooth")
			Expect(err).To(BeNil())

			newBluetoothDevice := utils.NewBluetoothDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)
			Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newBluetoothDevice)
				return err == nil
			}, "20s", "2s").Should(Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, nodeName, utils.UpdatedBluetoothDeviceInstance(nodeName).Name, "bluetooth")
			Expect(err).To(BeNil())
			updatedBluetoothDevice := utils.UpdatedBluetoothDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedBluetoothDevice)
			Expect(err).To(BeNil())

			go utils.TwinSubscribe(utils.UpdatedBluetoothDeviceInstance(nodeName).Name)
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
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "modbus")
			Expect(err).To(BeNil())

			newModbusDevice := utils.NewModbusDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)
			Eventually(func() bool {
				deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
				if err != nil {
					return false
				}

				err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
				return err == nil
			}, "20s", "2s").Should(Equal(true), "Device creation is not finished!!")

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, nodeName, utils.UpdatedModbusDeviceInstance(nodeName).Name, "modbus")
			Expect(err).To(BeNil())
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(nodeName)
			time.Sleep(2 * time.Second)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedModbusDevice)
			Expect(err).To(BeNil())

			go utils.TwinSubscribe(utils.UpdatedModbusDeviceInstance(nodeName).Name)
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
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, nodeName, utils.UpdatedLedDeviceInstance(nodeName).Name, "incorrect-instance")
			Expect(err).NotTo(BeNil())
		})
		It("E2E_UPDATE_DEVICE_4: Update device instance data and twin for modbus protocol", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "modbus")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "modbus")
			Expect(err).To(BeNil())
			newModbusDevice := utils.NewModbusDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newModbusDevice)
			Expect(err).To(BeNil())

			time.Sleep(3 * time.Second)

			configMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			isEqual := utils.CompareConfigMaps(*configMap, utils.NewConfigMapModbus(nodeName))
			Expect(isEqual).Should(Equal(true))
			// update twins and data section should reflect on change on config map
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPatch, nodeName, utils.UpdatedModbusDeviceInstance(nodeName).Name, "modbus")
			Expect(err).To(BeNil())
			updatedModbusDevice := utils.UpdatedModbusDeviceInstance(nodeName)
			time.Sleep(3 * time.Second)

			deviceInstanceList, err = utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &updatedModbusDevice)
			Expect(err).To(BeNil())

			updatedConfigMap, err := clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			updatedConfigMap.TypeMeta = metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			}

			isEqual = utils.CompareDeviceProfileInConfigMaps(*updatedConfigMap, utils.UpdatedConfigMapModbusForDataAndTwins(nodeName))
			Expect(isEqual).Should(Equal(true))
		})
		It("E2E_DELETE_DEVICE_1: Delete device instance for an existing device (No Protocol)", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, utils.NewLedDeviceInstance(nodeName).Name, "")
			Expect(err).To(BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
		It("E2E_DELETE_DEVICE_2: Delete device instance for a non-existing device", func() {
			err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, utils.NewLedDeviceModel().Name, "")
			Expect(err).To(BeNil())
		})
		It("E2E_DELETE_DEVICE_3: Delete device instance without device model", func() {
			err := utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, utils.NewLedDeviceInstance(nodeName).Name, "")
			Expect(err).To(BeNil())
			time.Sleep(1 * time.Second)

			_, err = clientSet.CoreV1().ConfigMaps("default").Get(context.TODO(), "device-profile-config-"+nodeName, metav1.GetOptions{})
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})
	Context("Test Change in device twin", func() {
		BeforeEach(func() {
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, device.Name, "")
				Expect(err).To(BeNil())
			}
			// Delete any pre-existing device models
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			utils.TwinResult = utils.DeviceTwinResult{}
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the device instances created
			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, device := range deviceInstanceList {
				err := utils.HandleDeviceInstance(edgeClientSet, http.MethodDelete, nodeName, device.Name, "")
				Expect(err).To(BeNil())
			}
			// Delete the device models created
			list, err := utils.ListDeviceModel(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, model := range list {
				err := utils.HandleDeviceModel(edgeClientSet, http.MethodDelete, model.Name, "")
				Expect(err).To(BeNil())
			}
			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_TWIN_STATE_1: Change the twin state of an existing device", func() {
			err := utils.HandleDeviceModel(edgeClientSet, http.MethodPost, "", "led")
			Expect(err).To(BeNil())
			err = utils.HandleDeviceInstance(edgeClientSet, http.MethodPost, nodeName, "", "led")
			Expect(err).To(BeNil())
			newLedDevice := utils.NewLedDeviceInstance(nodeName)
			time.Sleep(3 * time.Second)
			var deviceTwinUpdateMessage utils.DeviceTwinUpdate
			reportedValue := off
			deviceTwinUpdateMessage.Twin = map[string]*utils.MsgTwin{
				"power-status": {Actual: &utils.TwinValue{Value: &reportedValue}, Metadata: &utils.TypeMetadata{Type: "string"}},
			}
			err = utils.ChangeTwinValue(deviceTwinUpdateMessage, utils.NewLedDeviceInstance(nodeName).Name)
			Expect(err).To(BeNil())
			time.Sleep(3 * time.Second)
			newLedDevice = utils.NewLedDeviceInstance(nodeName)

			deviceInstanceList, err := utils.ListDevice(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckDeviceExists(deviceInstanceList, &newLedDevice)
			Expect(err).To(BeNil())

			Expect(deviceInstanceList[0].Status.Twins[0].PropertyName).To(Equal("power-status"))
			Expect(deviceInstanceList[0].Status.Twins[0].Reported.Value).To(Equal(off))
		})
	})
})
