/*
Copyright 2024 The KubeEdge Authors.

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

package dmiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
)

const testProtocol = "testProtocol"

func TestCreateDeviceModelRequest(t *testing.T) {
	assert := assert.New(t)

	model := &v1beta1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testModel",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceModelSpec{
			Properties: []v1beta1.ModelProperty{
				{
					Name:       "temperature",
					Type:       v1beta1.INT,
					AccessMode: v1beta1.ReadOnly,
				},
			},
			Protocol: "modbus",
		},
	}

	request, err := createDeviceModelRequest(model)

	assert.NoError(err)
	assert.NotNil(request)
	assert.IsType(&dmiapi.CreateDeviceModelRequest{}, request)
	assert.Equal(model.Name, request.Model.Name)
	assert.Equal(model.Namespace, request.Model.Namespace)
}

func TestUpdateDeviceModelRequest(t *testing.T) {
	assert := assert.New(t)

	model := &v1beta1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testModel",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceModelSpec{
			Properties: []v1beta1.ModelProperty{
				{
					Name:       "temperature",
					Type:       v1beta1.INT,
					AccessMode: v1beta1.ReadOnly,
				},
				{
					Name:       "humidity",
					Type:       v1beta1.INT,
					AccessMode: v1beta1.ReadOnly,
				},
			},
			Protocol: "modbus",
		},
	}

	request, err := updateDeviceModelRequest(model)

	assert.NoError(err)
	assert.NotNil(request)
	assert.IsType(&dmiapi.UpdateDeviceModelRequest{}, request)

	assert.Equal(model.Name, request.Model.Name)
	assert.Equal(model.Namespace, request.Model.Namespace)
}

func TestRemoveDeviceModelRequest(t *testing.T) {
	assert := assert.New(t)

	deviceModel := &v1beta1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testModel",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceModelSpec{
			Protocol: "modbus",
			Properties: []v1beta1.ModelProperty{
				{
					Name: "temperature",
					Type: v1beta1.INT,
				},
			},
		},
	}

	request, err := removeDeviceModelRequest(deviceModel)

	assert.NoError(err)
	assert.NotNil(request)
	assert.IsType(&dmiapi.RemoveDeviceModelRequest{}, request)
	assert.Equal(deviceModel.Name, request.ModelName)
	assert.Equal(deviceModel.Namespace, request.ModelNamespace)
}

func TestGetDMIClientByProtocol(t *testing.T) {
	assert := assert.New(t)

	dmiClients := &DMIClients{
		clients: make(map[string]*DMIClient),
	}

	testClient := &DMIClient{
		protocol: testProtocol,
		socket:   "/test/socket",
	}
	dmiClients.clients[testProtocol] = testClient

	client, err := dmiClients.getDMIClientByProtocol(testProtocol)
	assert.NoError(err)
	assert.NotNil(client)
	assert.Equal(testClient, client)

	// Test retrieval of non-existent protocol
	nonExistentProtocol := "nonExistentProtocol"
	client, err = dmiClients.getDMIClientByProtocol(nonExistentProtocol)
	assert.Error(err)
	assert.Nil(client)
	assert.Contains(err.Error(), "fail to get dmi client of protocol")
	assert.Contains(err.Error(), nonExistentProtocol)
}

func TestCreateDMIClient(t *testing.T) {
	assert := assert.New(t)

	dmiClients := &DMIClients{
		clients: make(map[string]*DMIClient),
	}

	socketPath := "/test/socket"
	dmiClients.CreateDMIClient(testProtocol, socketPath)

	client, err := dmiClients.getDMIClientByProtocol(testProtocol)
	assert.NoError(err)
	assert.NotNil(client)
	assert.Equal(testProtocol, client.protocol)
	assert.Equal(socketPath, client.socket)

	// Updating an existing client
	newSocketPath := "/new/test/socket"
	dmiClients.CreateDMIClient(testProtocol, newSocketPath)

	updatedClient, err := dmiClients.getDMIClientByProtocol(testProtocol)
	assert.NoError(err)
	assert.NotNil(updatedClient)
	assert.Equal(testProtocol, updatedClient.protocol)
	assert.Equal(newSocketPath, updatedClient.socket)
	assert.Equal(client, updatedClient)

	// Creating a new client and testing that the original client still exists and wasn't changed
	anotherProtocol := "anotherProtocol"
	anotherSocketPath := "/another/test/socket"
	dmiClients.CreateDMIClient(anotherProtocol, anotherSocketPath)

	anotherClient, err := dmiClients.getDMIClientByProtocol(anotherProtocol)
	assert.NoError(err)
	assert.NotNil(anotherClient)
	assert.Equal(anotherProtocol, anotherClient.protocol)
	assert.Equal(anotherSocketPath, anotherClient.socket)

	originalClient, err := dmiClients.getDMIClientByProtocol(testProtocol)
	assert.NoError(err)
	assert.NotNil(originalClient)
	assert.Equal(testProtocol, originalClient.protocol)
	assert.Equal(newSocketPath, originalClient.socket)
}

func TestGetDMIClientConn(t *testing.T) {
	assert := assert.New(t)

	dmiClients := &DMIClients{
		clients: make(map[string]*DMIClient),
	}

	// Test case 1: Client doesn't exist
	_, err := dmiClients.getDMIClientConn("nonexistent")
	assert.Error(err)
	assert.Contains(err.Error(), "fail to get dmi client of protocol nonexistent")

	// Test case 2: Client exists
	socket := "/test/socket"
	dmiClients.CreateDMIClient(testProtocol, socket)

	client, _ := dmiClients.getDMIClientConn(testProtocol)
	assert.NotNil(client)
	assert.Equal(testProtocol, client.protocol)
	assert.Equal(socket, client.socket)
}
