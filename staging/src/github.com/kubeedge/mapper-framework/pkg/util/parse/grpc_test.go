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

package parse

import (
	"slices"
	"testing"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func TestGetDeviceFromGrpcTwins(t *testing.T) {
	tests := []struct {
		name            string
		properties      []string
		modelProperties []string
		wantTwins       []string
	}{
		{"configData omitted", []string{"b", "a"}, []string{"a", "b"}, []string{"b", "a"}},
		{"property not in model", []string{"typo", "a"}, []string{"a"}, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &dmiapi.Device{Spec: &dmiapi.DeviceSpec{Protocol: &dmiapi.ProtocolConfig{}}}
			for _, name := range tt.properties {
				device.Spec.Properties = append(device.Spec.Properties, &dmiapi.DeviceProperty{
					Name:     name,
					Desired:  &dmiapi.TwinProperty{},
					Visitors: &dmiapi.VisitorConfig{}, // configData is optional in the CRD
				})
			}
			model := &common.DeviceModel{}
			for _, name := range tt.modelProperties {
				model.Properties = append(model.Properties, common.ModelProperty{Name: name})
			}

			instance, err := GetDeviceFromGrpc(device, model)
			if err != nil {
				t.Fatalf("GetDeviceFromGrpc() error = %v", err)
			}

			var twins []string
			for _, twin := range instance.Twins {
				twins = append(twins, twin.PropertyName)
				if twin.Property == nil {
					t.Fatalf("twin %s has no property", twin.PropertyName)
				}
				if twin.Property.PProperty.Name != twin.PropertyName {
					t.Errorf("twin %s got model property %s", twin.PropertyName, twin.Property.PProperty.Name)
				}
			}
			if !slices.Equal(twins, tt.wantTwins) {
				t.Errorf("twins = %v, want %v", twins, tt.wantTwins)
			}
		})
	}
}
