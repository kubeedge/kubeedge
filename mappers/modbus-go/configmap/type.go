/*
Copyright 2020 The KubeEdge Authors.

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

package configmap

// ModbusVisitorConfig is the modbus register configuration.
type ModbusVisitorConfig struct {
	Register       string `json:"register"`
	Offset         uint16 `json:"offset"`
	Limit          int    `json:"limit"`
	Scale          int    `json:"scale,omitempty"`
	IsSwap         bool   `json:"isSwap,omitempty"`
	IsRegisterSwap bool   `json:"isRegisterSwap,omitempty"`
}

// ModbusProtocolCommonConfig is the modbus protocol configuration.
type ModbusProtocolCommonConfig struct {
	COM              COMStruct       `json:"com,omitempty"`
	TCP              TCPStruct       `json:"tcp,omitempty"`
	CustomizedValues CustomizedValue `json:"customizedValues,omitempty"`
}

// CustomizedValue is the customized part for modbus protocol.
type CustomizedValue map[string]interface{}

// COMStruct is the serial configuration.
type COMStruct struct {
	SerialPort string `json:"serialPort"`
	BaudRate   int64  `json:"baudRate"`
	DataBits   int64  `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   int64  `json:"stopBits"`
}

// TCPStruct is the TCP configuation.
type TCPStruct struct {
	IP   string `json:"ip"`
	Port int64  `json:"port"`
}
