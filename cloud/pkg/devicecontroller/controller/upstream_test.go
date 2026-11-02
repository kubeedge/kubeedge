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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

type responseRecorder struct {
	responses chan model.Message
}

func (r *responseRecorder) Send(model.Message) error { return nil }

func (r *responseRecorder) Receive() (model.Message, error) { return model.Message{}, nil }

func (r *responseRecorder) Response(message model.Message) error {
	r.responses <- message
	return nil
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{
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
				Status: v1beta1.DeviceStatusStatus{},
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

func TestUpdateDeviceStatusRejectsOtherNode(t *testing.T) {
	assert := assert.New(t)

	var patches int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			atomic.AddInt32(&patches, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	crdClient, err := crdClientset.NewForConfig(&rest.Config{Host: server.URL})
	assert.NoError(err)

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: "sensor", Namespace: "default"},
		Spec:       v1beta1.DeviceSpec{NodeName: "edge-a"},
	}
	deviceStatus := &v1beta1.DeviceStatus{
		ObjectMeta: metav1.ObjectMeta{Name: "sensor", Namespace: "default"},
		Status:     v1beta1.DeviceStatusStatus{State: "online"},
	}
	dc := &DownstreamController{
		crdClient:           crdClient,
		deviceManager:       &manager.DeviceManager{},
		deviceStatusManager: &manager.DeviceStatusManager{},
	}
	dc.deviceManager.Device.Store("default/sensor", device)
	dc.deviceStatusManager.DeviceStatus.Store("default/sensor", deviceStatus)

	recorder := &responseRecorder{responses: make(chan model.Message, 4)}
	uc := &UpstreamController{
		crdClient:        crdClient,
		messageLayer:     recorder,
		deviceTwinsChan:  make(chan model.Message, 4),
		deviceStatesChan: make(chan model.Message, 4),
		dc:               dc,
	}
	go uc.updateDeviceStatus()

	stateMsg := func(nodeID, state string) model.Message {
		return *model.NewMessage("").
			BuildRouter("edgehub", "twin", "node/"+nodeID+"/device/default/sensor/state/update", model.UpdateOperation).
			FillBody(types.DeviceStateUpdate{Device: types.Device{State: state}})
	}
	twinMsg := func(nodeID string) model.Message {
		value := "42"
		return *model.NewMessage("").
			BuildRouter("edgehub", "twin", "node/"+nodeID+"/device/default/sensor/twin/edge_updated", model.UpdateOperation).
			FillBody(types.DeviceTwinUpdate{Twin: map[string]*types.MsgTwin{
				"temperature": {Actual: &types.TwinValue{Value: &value}},
			}})
	}

	// Updates from a node the device is not bound to must be dropped.
	uc.deviceStatesChan <- stateMsg("edge-b", "offline")
	uc.deviceTwinsChan <- twinMsg("edge-b")
	time.Sleep(200 * time.Millisecond)
	assert.Len(recorder.responses, 0)
	assert.Equal(int32(0), atomic.LoadInt32(&patches))
	assert.Equal("online", deviceStatus.Status.State)
	assert.Empty(deviceStatus.Status.Twins)

	// Updates from the owning node are still applied.
	uc.deviceStatesChan <- stateMsg("edge-a", "offline")
	select {
	case resp := <-recorder.responses:
		assert.Equal("node/edge-a/twin", resp.GetResource())
	case <-time.After(5 * time.Second):
		t.Fatal("no response for update from owning node")
	}
	assert.Equal(int32(1), atomic.LoadInt32(&patches))
	assert.Equal("offline", deviceStatus.Status.State)
}
