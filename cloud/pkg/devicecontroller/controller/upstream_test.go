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

package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

type MockMessageLayer struct {
	messages chan model.Message
}

func NewMockMessageLayer() *MockMessageLayer {
	return &MockMessageLayer{
		messages: make(chan model.Message, 10),
	}
}

func (ml *MockMessageLayer) Send(_ model.Message) error {
	// In a real implementation, this would send the message.  Here, we just simulate success.
	return nil
}

func (ml *MockMessageLayer) Receive() (model.Message, error) {
	msg := <-ml.messages
	return msg, nil
}

func (ml *MockMessageLayer) SendResponse(_ model.Message) error {
	//TODO: consider removing it if have same functionality
	return nil
}

func (ml *MockMessageLayer) Response(_ model.Message) error {
	//TODO: consider removing it if have same functionality
	return nil
}

func createMessage(resourceType string) model.Message {
	resource := "node/device/" + resourceType
	return model.Message{
		Header: model.MessageHeader{
			ID:              "test-id",
			ParentID:        "",
			Timestamp:       time.Now().UnixNano() / 1e6,
			ResourceVersion: "1",
		},
		Router: model.MessageRoute{
			Resource: resource,
		},
		Content: "test-content",
	}
}

func initTestConfig() {
	deviceController := &v1alpha1.DeviceController{
		Enable: true,
		Buffer: &v1alpha1.DeviceControllerBuffer{
			UpdateDeviceTwins:  100,
			UpdateDeviceStates: 100,
		},
		Load: &v1alpha1.DeviceControllerLoad{
			UpdateDeviceStatusWorkers: 2,
		},
	}
	config.InitConfigure(deviceController)
}

func TestMain(m *testing.M) {
	initTestConfig()
	m.Run()
}

func TestUpstreamControllerStart(t *testing.T) {
	mockML := NewMockMessageLayer()
	dc := &DownstreamController{}
	uc := &UpstreamController{
		messageLayer: mockML,
		dc:           dc,
	}
	err := uc.Start()
	assert.NoError(t, err)

	assert.NotNil(t, uc.deviceTwinsChan, "deviceTwinsChan should be initialized")
	assert.NotNil(t, uc.deviceStatesChan, "deviceStatesChan should be initialized")

	assert.Equal(t, int(config.Config.Buffer.UpdateDeviceTwins), cap(uc.deviceTwinsChan),
		"deviceTwinsChan should have correct buffer size")
	assert.Equal(t, int(config.Config.Buffer.UpdateDeviceStates), cap(uc.deviceStatesChan),
		"deviceStatesChan should have correct buffer size")
	closeChannels(uc)
}

func TestDispatchMessage(t *testing.T) {
	mockML := NewMockMessageLayer()

	uc := &UpstreamController{
		messageLayer:     mockML,
		deviceTwinsChan:  make(chan model.Message, 1),
		deviceStatesChan: make(chan model.Message, 1),
	}

	go uc.dispatchMessage()

	defer closeChannels(uc)

	tests := []struct {
		name         string
		resourceType string
		expectChan   chan model.Message
	}{
		{
			name:         "Twin Update Message",
			resourceType: constants.ResourceTypeTwinEdgeUpdated,
			expectChan:   uc.deviceTwinsChan,
		},
		{
			name:         "Device State Update Message",
			resourceType: constants.ResourceDeviceStateUpdated,
			expectChan:   uc.deviceStatesChan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := createMessage(tt.resourceType)
			mockML.messages <- msg

			select {
			case receivedMsg := <-tt.expectChan:
				assert.Equal(t, msg.GetID(), receivedMsg.GetID())
				assert.Equal(t, msg.GetResource(), receivedMsg.GetResource())
			case <-time.After(2 * time.Second):
				t.Error("Timeout waiting for message dispatch")
			}
		})
	}
}

func TestDispatchMessageInvalidResource(t *testing.T) {
	mockML := NewMockMessageLayer()

	uc := &UpstreamController{
		messageLayer:     mockML,
		deviceTwinsChan:  make(chan model.Message, 1),
		deviceStatesChan: make(chan model.Message, 1),
	}

	go uc.dispatchMessage()

	defer closeChannels(uc)

	msg := createMessage("invalid-resource-type")
	mockML.messages <- msg

	time.Sleep(1 * time.Second)

	select {
	case <-uc.deviceTwinsChan:
		t.Error("Message should not have been dispatched to deviceTwinsChan")
	case <-uc.deviceStatesChan:
		t.Error("Message should not have been dispatched to deviceStatesChan")
	default:
	}
}

func closeChannels(uc *UpstreamController) {
	if uc.deviceTwinsChan != nil {
		close(uc.deviceTwinsChan)
	}
	if uc.deviceStatesChan != nil {
		close(uc.deviceStatesChan)
	}
}
func TestNewUpstreamController(t *testing.T) {
	assert := assert.New(t)

	dc := &DownstreamController{}
	uc, err := NewUpstreamController(dc)
	assert.NoError(err)
	assert.NotNil(uc)

	assert.NotNil(uc.messageLayer)
	assert.NotNil(uc.dc)
	assert.Equal(dc, uc.dc)
	// Channels are not initialized (they should be initialized in Start())
	assert.Nil(uc.deviceTwinsChan)
	assert.Nil(uc.deviceStatesChan)
}

func TestFindOrCreateTwinByName(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name       string
		twinName   string
		properties []v1beta1.DeviceProperty
		status     *DeviceStatus
		expected   *v1beta1.Twin
	}{
		{
			name:     "finding existing twin",
			twinName: "temperature",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
					},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
				Reported: v1beta1.TwinProperty{
					Value: "25",
				},
			},
		},
		{
			name:     "creating new twin",
			twinName: "humidity",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "humidity",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "humidity",
			},
		},
		{
			name:     "property not found",
			twinName: "nonexistent",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{},
				},
			},
			expected: nil,
		},
		{
			name:     "multiple properties",
			twinName: "temperature",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "humidity",
				},
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "humidity",
							Reported: v1beta1.TwinProperty{
								Value: "60",
							},
						},
					},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findOrCreateTwinByName(tt.twinName, tt.properties, tt.status)
			if tt.expected == nil {
				assert.Nil(result)
			} else {
				assert.Equal(tt.expected.PropertyName, result.PropertyName)
				if tt.expected.Reported.Value != "" {
					assert.Equal(tt.expected.Reported, result.Reported)
				}
				// Verify twin was added to DeviceStatus if created
				if len(tt.status.Status.Twins) > 0 {
					found := false
					for _, twin := range tt.status.Status.Twins {
						if twin.PropertyName == tt.twinName {
							found = true
							break
						}
					}
					assert.True(found)
				}
			}
		})
	}
}
func TestFindTwinByName(t *testing.T) {
	tests := []struct {
		name         string
		twinName     string
		deviceStatus *DeviceStatus
		expected     *v1beta1.Twin
	}{
		{
			name:     "twin exists",
			twinName: "temperature",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
						{
							PropertyName: "humidity",
							Reported: v1beta1.TwinProperty{
								Value: "60",
							},
						},
					},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
				Reported: v1beta1.TwinProperty{
					Value: "25",
				},
			},
		},
		{
			name:     "twin doesn't exist",
			twinName: "pressure",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name:     "device status is nil",
			twinName: "temperature",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTwinByName(tt.twinName, tt.deviceStatus)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected.PropertyName, result.PropertyName)
				assert.Equal(t, tt.expected.Reported, result.Reported)
			}
		})
	}
}
