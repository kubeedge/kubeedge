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

package services

import (
	"fmt"

	"github.com/paypal/gatt"
)

const (
	// serviceUUID is characteristic UUID for creating a new temperature service.
	serviceUUID = "09fc95c0-c111-11e3-9904-0002a5d5c51b"
	// readCharacteristicUUID is characteristic UUID for reading temperature.
	readCharacteristicUUID = "11fac9e0-c111-11e3-9246-0002a5d5c51b"
	// writeCharacteristicUUID is characteristic UUID for writing temperature.
	writeCharacteristicUUID = "16fe0d80-c111-11e3-b8c8-0002a5d5c51b"
	// readWrittenDataCharacteristicUUID is characteristic UUID for reading data written to connected device.
	readWrittenDataCharacteristicUUID = "1c927b50-c116-11e3-8a33-0800200c9a66"
	// dataConverterUUID is characteristic UUID for data conversion.
	dataConverterUUID = "2d816a41-d335-44f5-7b55-9000200c8a77"
	// twinStateUUID is characteristic UUID for changing twin state of device.
	twinStateUUID = "3d816a41-e336-55e5-7c66-8000100d8a44"
)

var dataWrite string
var write bool
var state string

func NewTemperatureService() *gatt.Service {
	temp := 36
	s := gatt.NewService(gatt.MustParseUUID(serviceUUID))

	s.AddCharacteristic(gatt.MustParseUUID(readCharacteristicUUID)).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			fmt.Fprintf(rsp, "%d", temp)
		})

	s.AddCharacteristic(gatt.MustParseUUID(writeCharacteristicUUID)).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			fmt.Println("Wrote:", string(data))
			write = true
			dataWrite = string(data)
			return gatt.StatusSuccess
		})

	s.AddCharacteristic(gatt.MustParseUUID(readWrittenDataCharacteristicUUID)).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			if write {
				fmt.Fprintf(rsp, "%s", dataWrite)
			}
		})

	s.AddCharacteristic(gatt.MustParseUUID(dataConverterUUID)).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			data := []uint8{32, 10, 248, 12}
			fmt.Fprintf(rsp, "%s", data)
		})

	s.AddCharacteristic(gatt.MustParseUUID(twinStateUUID)).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			state = "Red"
			return gatt.StatusSuccess
		})

	return s
}
