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

package actionmanager

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/paypal/gatt"

	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/data_converter"
)

const (
	ActionWrite = "WRITE"
	ActionRead  = "READ"
)

// GattPeripheral  represents the remote gatt peripheral device
var GattPeripheral gatt.Peripheral

// Operation is structure to define device operation
type Operation struct {
	// Action can be one of read/write corresponding to get/set respectively
	Action string `yaml:"action" json:"action"`
	// Characteristic refers to the characteristic on which the operation needs to be performed
	CharacteristicUUID string `yaml:"characteristic-uuid" json:"characteristic-uuid"`
	// Value is the value to be written in case of write action and value read from the device in case of read action
	Value []byte `yaml:"value" json:"value"`
}

// Action is structure to define a device action
type Action struct {
	// PerformImmediately indicates whether the action is to be performed immediately or not
	PerformImmediately bool `yaml:"perform-immediately" json:"perform-immediately"`
	// Name is the name of the Action
	Name string `yaml:"name" json:"name"`
	// Operation specifies the operation to be performed for this action
	Operation Operation `yaml:"operation" json:"operation"`
}

//ActionManager is a structure that contains a list of actions
type ActionManager struct {
	Actions []Action `yaml:"actions"`
}

//PerformOperation executes the operation
func (action *Action) PerformOperation(readConverter ...dataconverter.DataRead) {
	glog.Infof("Performing operations associated with action:  %s", action.Name)
	characteristic, err := FindCharacteristic(GattPeripheral, action.Operation.CharacteristicUUID)
	if err != nil {
		glog.Errorf("Error in finding characteristics: %s", err)
	}
	if strings.ToUpper(action.Operation.Action) == ActionRead {
		readValue, err := ReadCharacteristic(GattPeripheral, characteristic)
		if err != nil {
			glog.Errorf("Error in reading  characteristic: %s", err)
			return
		}
		converted := false
		for _, conversionAction := range readConverter[0].Actions {
			if strings.ToUpper(conversionAction.ActionName) == strings.ToUpper(action.Name) {
				convertedValue := fmt.Sprintf("%f", conversionAction.ConversionOperation.ConvertReadData(readValue))
				action.Operation.Value = []byte(convertedValue)
				converted = true
			}
		}
		if !converted {
			action.Operation.Value = readValue
		}
		glog.Infof("Read Successful")
	} else if strings.ToUpper(action.Operation.Action) == ActionWrite {
		if action.Operation.Value == nil {
			glog.Errorf("Please provide a value to be written")
			return
		}
		err := WriteCharacteristic(GattPeripheral, characteristic, action.Operation.Value)
		if err != nil {
			glog.Errorf("Error in writing characteristic: %s", err)
			return
		}
		glog.Infof("Write Successful")
	}
}

//FindCharacteristic is used to find the bluetooth characteristic
func FindCharacteristic(p gatt.Peripheral, characteristicUUID string) (*gatt.Characteristic, error) {
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		glog.Errorf("Failed to discover services, err: %s\n", err)
		return nil, err
	}
	for _, s := range ss {
		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
		if err != nil {
			glog.Errorf("Failed to discover characteristics, err: %s\n", err)
			continue
		}
		for _, c := range cs {
			if c.UUID().String() == characteristicUUID {
				return c, nil
			}
		}
	}
	return nil, errors.New("unable to find the specified characteristic: " + characteristicUUID)
}

//ReadCharacteristic is used to read the value of the characteristic
func ReadCharacteristic(p gatt.Peripheral, c *gatt.Characteristic) ([]byte, error) {
	value, err := p.ReadCharacteristic(c)
	if err != nil {
		glog.Errorf("Error in reading characteristic, err: %s\n", err)
		return nil, err
	}
	return value, nil
}

//WriteCharacteristic is used to write some value into the characteristic
func WriteCharacteristic(p gatt.Peripheral, c *gatt.Characteristic, b []byte) error {
	err := p.WriteCharacteristic(c, b, false)
	if err != nil {
		glog.Errorf("Error in writing characteristic, err: %s\n", err)
		return err
	}
	return nil
}
