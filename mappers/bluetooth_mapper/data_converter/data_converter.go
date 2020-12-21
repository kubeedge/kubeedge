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

package dataconverter

import (
	"strconv"
	"strings"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

//Converter is the structure that contains data conversion specific configuration
type Converter struct {
	DataWrite DataWrite `yaml:"write"`
	DataRead  DataRead  `yaml:"read"`
}

//dataWrite structure contains configuration information specific to data-writes
type DataWrite struct {
	Attributes []WriteAttribute `yaml:"attributes"`
}

//WriteAttribute structure contains the name of the attribute as well as a data-map of values to be written
type WriteAttribute struct {
	Name       string             `yaml:"name"`
	Operations map[string]DataMap `yaml:"operations"`
}

//DataMap structure contains a mapping between the value that arrives from the platform (expected value) and
// the byte value to be written into the device
type DataMap struct {
	DataMapping map[string][]byte `yaml:"data-map"`
}

//dataRead structure contains configuration information specific to data-read
type DataRead struct {
	Actions []ReadAction `yaml:"actions"`
}

//ReadAction specifies the name of the action along with the conversion operations to be performed in case of data-read
type ReadAction struct {
	ActionName          string        `yaml:"action-name"`
	ConversionOperation ReadOperation `yaml:"conversion-operation"`
}

//ReadOperation specifies how to convert the data received from the device into meaningful data
type ReadOperation struct {
	StartIndex       int      `yaml:"start-index"`
	EndIndex         int      `yaml:"end-index"`
	ShiftLeft        uint     `yaml:"shift-left"`
	ShiftRight       uint     `yaml:"shift-right"`
	Multiply         float64  `yaml:"multiply"`
	Divide           float64  `yaml:"divide"`
	Add              float64  `yaml:"add"`
	Subtract         float64  `yaml:"subtract"`
	OrderOfExecution []string `yaml:"order-of-execution"`
}

//ConvertReadData is the function responsible to convert the data read from the device into meaningful data
func (operation *ReadOperation) ConvertReadData(data []byte) float64 {
	var intermediateResult uint64
	var initialValue []byte
	var initialStringValue = ""
	if operation.StartIndex <= operation.EndIndex {
		for index := operation.StartIndex; index <= operation.EndIndex; index++ {
			initialValue = append(initialValue, data[index])
		}
	} else {
		for index := operation.StartIndex; index >= operation.EndIndex; index-- {
			initialValue = append(initialValue, data[index])
		}
	}
	for _, value := range initialValue {
		initialStringValue = initialStringValue + strconv.Itoa(int(value))
	}
	initialByteValue, _ := strconv.ParseUint(initialStringValue, 16, 16)

	if operation.ShiftLeft != 0 {
		intermediateResult = initialByteValue << operation.ShiftLeft
	} else if operation.ShiftRight != 0 {
		intermediateResult = initialByteValue >> operation.ShiftRight
	}
	finalResult := float64(intermediateResult)
	for _, executeOperation := range operation.OrderOfExecution {
		switch strings.ToUpper(executeOperation) {
		case strings.ToUpper(string(v1alpha2.BluetoothAdd)):
			finalResult = finalResult + operation.Add
		case strings.ToUpper(string(v1alpha2.BluetoothSubtract)):
			finalResult = finalResult - operation.Subtract
		case strings.ToUpper(string(v1alpha2.BluetoothMultiply)):
			finalResult = finalResult * operation.Multiply
		case strings.ToUpper(string(v1alpha2.BluetoothDivide)):
			finalResult = finalResult / operation.Divide
		}
	}
	return finalResult
}
