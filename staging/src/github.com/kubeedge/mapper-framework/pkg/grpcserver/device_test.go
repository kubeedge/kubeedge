/*
Copyright 2026 The KubeEdge Authors.

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

package grpcserver

import (
	"context"
	"testing"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/global"
)

type fakeDevPanel struct {
	global.DevPanel
	device interface{}
}

func (f fakeDevPanel) GetDevice(string) (interface{}, error) {
	return f.device, nil
}

type fakeDevice struct {
	Instance common.DeviceInstance
}

func TestGetDevice(t *testing.T) {
	dev := fakeDevice{Instance: common.DeviceInstance{Twins: []common.Twin{{PropertyName: "temperature"}}}}
	tests := []struct {
		name   string
		device interface{}
	}{
		{name: "pointer", device: &dev},
		{name: "value", device: dev},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{devPanel: fakeDevPanel{device: tt.device}}
			res, err := s.GetDevice(context.Background(), &dmiapi.GetDeviceRequest{DeviceName: "dev"})
			if err != nil {
				t.Fatalf("GetDevice() error = %v", err)
			}
			twins := res.GetDevice().GetStatus().GetTwins()
			if len(twins) != 1 || twins[0].GetPropertyName() != "temperature" {
				t.Errorf("GetDevice() twins = %v, want one twin named temperature", twins)
			}
		})
	}
}

func TestCreateDeviceModelNilModel(t *testing.T) {
	s := &Server{}
	if _, err := s.CreateDeviceModel(context.Background(), &dmiapi.CreateDeviceModelRequest{}); err == nil {
		t.Error("CreateDeviceModel() without a model: want error, got nil")
	}
}
