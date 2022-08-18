/*
Copyright 2022 The KubeEdge Authors.

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

package dmi

import (
	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1alpha1"
)

// DeviceManagerService defines the public APIS for remote device management.
// The server is implemented by the module of device manager in edgecore
// and the client is implemented by the device mapper for upstreaming.
// The mapper should register itself to the device manager when it is online
// to get the list of devices. And then the mapper can report the device status to the device manager.
type DeviceManagerService interface {
	// MapperRegister registers the information of the mapper to device manager
	// when the mapper is online. Device manager returns the list of devices and device models which
	// this mapper should manage.
	MapperRegister(*dmiapi.MapperRegisterRequest) (*dmiapi.MapperRegisterResponse, error)
	// ReportDeviceStatus reports the status of devices to device manager.
	// When the mapper collects some properties of a device, it can make them a map of device twins
	// and report it to the device manager through the interface of ReportDeviceStatus.
	ReportDeviceStatus(*dmiapi.ReportDeviceStatusRequest) (*dmiapi.ReportDeviceStatusResponse, error)
}

// DeviceMapperService defines the public APIS for remote device management.
// The server is implemented by the device mapper
// and the client is implemented by the module of device manager in edgecore for downstreaming.
// The device manager can manage the device life cycle through these interfaces provided by DeviceMapperService.
// When device manager gets a message of device management from cloudcore, it should call the corresponding grpc interface
// to make the mapper maintain the list of device information.
type DeviceMapperService interface {
	// RegisterDevice registers a device to the device mapper.
	// Device manager registers a device instance with the information of device
	// to the mapper through the interface of RegisterDevice.
	// When the mapper gets the request of register with device information,
	// it should add the device to the device list and connect to the real physical device via the specific protocol.
	RegisterDevice(*dmiapi.RegisterDeviceRequest) (*dmiapi.RegisterDeviceResponse, error)
	// RemoveDevice unregisters a device to the device mapper.
	// Device manager unregisters a device instance with the name of device
	// to the mapper through the interface of RemoveDevice.
	// When the mapper gets the request of unregister with device name,
	// it should remove the device from the device list and disconnect to the real physical device.
	RemoveDevice(*dmiapi.RemoveDeviceRequest) (*dmiapi.RemoveDeviceResponse, error)
	// UpdateDevice updates a device to the device mapper
	// Device manager updates the information of a device used by the mapper
	// through the interface of UpdateDevice.
	// The information of a device includes the meta data and the status data of a device.
	// When the mapper gets the request of updating with the information of a device,
	// it should update the device of the device list and connect to the real physical device via the updated information.
	UpdateDevice(*dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error)
	// UpdateDeviceStatus update a device status to the device mapper
	// Device manager sends the new device status to the mapper
	// through the interface of UpdateDeviceStatus.
	// The device status represents the properties of device twins.
	// When the mapper gets the request of updating with the new device status,
	// it should update the device status of the device list and the real device status of the physical device via the updated information.
	UpdateDeviceStatus(*dmiapi.UpdateDeviceStatusRequest) (*dmiapi.UpdateDeviceStatusResponse, error)
	// GetDevice get the information of a device from the device mapper.
	// Device sends the request of querying device information with the device name to the mapper
	// through the interface of GetDevice.
	// When the mapper gets the request of querying with the device name,
	// it should return the device information.
	GetDevice(*dmiapi.GetDeviceRequest) (*dmiapi.GetDeviceResponse, error)

	// CreateDeviceModel creates a device model to the device mapper.
	// Device manager sends the information of device model to the mapper
	// through the interface of CreateDeviceModel.
	// When the mapper gets the request of creating with the information of device model,
	// it should create a new device model to the list of device models.
	CreateDeviceModel(request *dmiapi.CreateDeviceModelRequest) (*dmiapi.CreateDeviceModelResponse, error)
	// RemoveDeviceModel remove a device model to the device mapper.
	// Device manager sends the name of device model to the mapper
	// through the interface of RemoveDeviceModel.
	// When the mapper gets the request of removing with the name of device model,
	// it should remove the device model to the list of device models.
	RemoveDeviceModel(*dmiapi.RemoveDeviceModelRequest) (*dmiapi.RemoveDeviceModelResponse, error)
	// UpdateDeviceModel update a device model to the device mapper.
	// Device manager sends the information of device model to the mapper
	// through the interface of UpdateDeviceModel.
	// When the mapper gets the request of updating with the information of device model,
	// it should update the device model to the list of device models.
	UpdateDeviceModel(*dmiapi.UpdateDeviceModelRequest) (*dmiapi.UpdateDeviceModelResponse, error)
}
