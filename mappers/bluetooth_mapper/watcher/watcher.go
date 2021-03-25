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

package watcher

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	actionmanager "github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/action_manager"
	dataconverter "github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/data_converter"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/helper"
)

var DeviceConnected = make(chan bool)
var done = make(chan struct{})
var deviceName string
var deviceID string
var actionManager []actionmanager.Action // TODO: 这个到底是干嘛用的
var dataConverter dataconverter.Converter

//Watch structure contains the watcher specific configurations
type Watcher struct {
	DeviceTwinPropertyNames []Attribute `yaml:"device-twin-attributes" json:"device-twin-attributes"` // TODO: 这玩意是干嘛的
}

//Attribute structure contains the name of the attribute along with the actions to be performed for this attribute
type Attribute struct {
	Name    string   `yaml:"device-property-name" json:"device-property-name"`
	Actions []string `yaml:"actions" json:"actions"`
}

//Initiate initiates the watcher module
func (w *Watcher) Initiate(device gatt.Device, nameOfDevice, idOfDevice string, actions []actionmanager.Action, converter dataconverter.Converter) {
	deviceID = idOfDevice
	deviceName = nameOfDevice
	actionManager = actions
	dataConverter = converter
	// Register optional handlers.
	device.Handle(
		gatt.PeripheralConnected(w.onPeripheralConnected),
		gatt.PeripheralDisconnected(onPeripheralDisconnected),
		gatt.PeripheralDiscovered(onPeripheralDiscovered),
	)
	if err := device.Init(onStateChanged); err != nil {
		klog.Errorf("Init device failed with error: %v", err)
	}
	<-done
	klog.Infof("Watcher Done")
}

//onStateChanged contains the operations to be performed when the state of the peripheral device changes
func onStateChanged(device gatt.Device, s gatt.State) {
	switch s {
	case gatt.StatePoweredOn:
		klog.Infof("Scanning for BLE device Broadcasts...")
		device.Scan([]gatt.UUID{}, true)
		return
	default:
		device.StopScanning()
	}
}

//onPeripheralDiscovered contains the operations to be performed as soon as the peripheral device is discovered
func onPeripheralDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if strings.EqualFold(a.LocalName, strings.Replace(deviceName, "-", " ", -1)) {
		klog.Infof("Device: %s found !!!! Stop Scanning for devices", deviceName)
		// Stop scanning once we've got the peripheral we're looking for.
		p.Device().StopScanning()
		klog.Infof("Connecting to %s", deviceName)
		p.Device().Connect(p)
	}
}

//onPeripheralDisconnected contains the operations to be performed as soon as the peripheral device is disconnected
func onPeripheralDisconnected(p gatt.Peripheral, err error) {
	klog.Infof("Disconnecting  from bluetooth device....")
	DeviceConnected <- false
	close(done)
	p.Device().CancelConnection(p)
}

//onPeripheralConnected contains the operations to be performed as soon as the peripheral device is connected
func (w *Watcher) onPeripheralConnected(p gatt.Peripheral, err error) {
	actionmanager.GattPeripheral = p
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		klog.Errorf("Failed to discover services, err: %s\n", err)
		os.Exit(1)
	}
	for _, s := range ss {
		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
		if err != nil {
			klog.Errorf("Failed to discover characteristics for service %s, err: %v\n", s.Name(), err)
			continue
		}
		actionmanager.CharacteristicsList = append(actionmanager.CharacteristicsList, cs...)
	}
	DeviceConnected <- true
	for {
		newWatcher := &Watcher{}
		if !reflect.DeepEqual(w, newWatcher) {
			err := w.EquateTwinValue(deviceID)
			if err != nil {
				klog.Errorf("Error in watcher functionality: %s", err)
			}
		}
	}
}

func getTwin(propertyName string) (*v1alpha2.Twin, error) {
	for _, twin := range helper.TwinResult.Status.Twins {
		if twin.PropertyName == propertyName {
			result := twin
			return &result, nil
		}
	}
	return nil, errors.New("twin " + propertyName + " is not exist")
}

//EquateTwinValue is responsible for equating the actual state of the device to the expected state that has been set and syncing back the result to the cloud
func (w *Watcher) EquateTwinValue(deviceID string) error {
	var updateMessage []v1alpha2.Twin
	updatedActualValues := make(map[string]string)
	helper.Wg.Add(1)
	klog.Infof("Watching on the device twin values for device with deviceID: %s", deviceID)
	go helper.TwinSubscribe(deviceID)
	helper.GetTwin(updateMessage, deviceID)
	helper.Wg.Wait()
	twinUpdated := false
	for _, twinAttribute := range w.DeviceTwinPropertyNames {
		twin, err := getTwin(twinAttribute.Name)
		if err != nil {
			return err
		}

		nilTwin := v1alpha2.Twin{}
		if !reflect.DeepEqual(twin.Desired, nilTwin) && ((reflect.DeepEqual(twin.Reported, nilTwin) && twin.Desired.Value != "") || (twin.Desired.Value != twin.Reported.Value)) {
			klog.Infof("%s Expected Value : %s", twinAttribute.Name, twin.Desired.Value)
			if reflect.DeepEqual(twin.Reported, nilTwin) {
				klog.Infof("%s  Actual Value: %v", twinAttribute.Name, twin.Reported)
			} else {
				klog.Infof("%s Actual Value: %s", twinAttribute.Name, twin.Reported.Value)
			}
			klog.Infof("Equating the actual value to expected value for: %s", twinAttribute.Name)
			for _, watcherAction := range twinAttribute.Actions {
				actionExists := false
				for _, action := range actionManager {
					if strings.EqualFold(action.Name, watcherAction) {
						actionExists = true
						for _, converterAttribute := range dataConverter.DataWrite.Attributes {
							if strings.EqualFold(converterAttribute.Name, twinAttribute.Name) {
								for operationName, dataMap := range converterAttribute.Operations {
									if action.Name == operationName {
										expectedValue := twin.Desired.Value
										// TODO: DataMappping 有啥用
										if _, ok := dataMap.DataMapping[expectedValue]; ok {
											action.Operation.Value = dataMap.DataMapping[expectedValue]
										}
									}
									action.PerformOperation()
								}
							}
						}
					}
				}
				if !actionExists {
					return errors.New("The action: " + watcherAction + " does not exist for this device")
				}
			}
			// TODO: 这个地方应该是reported的吧
			updatedActualValues[twinAttribute.Name] = twin.Reported.Value
			twinUpdated = true
		}
	}
	if twinUpdated {
		updateMessage = helper.CreateActualUpdateMessage(updatedActualValues)
		helper.ChangeTwinValue(updateMessage, deviceID)
		time.Sleep(2 * time.Second)
		// TODO：感觉功能跟上面的功能重复，需要删除
		klog.Infof("Syncing to cloud.....")
		helper.SyncToCloud(updateMessage, deviceID)
	} else {
		klog.Infof("Actual values are in sync with Expected value")
	}
	return nil
}
