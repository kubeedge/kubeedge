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

package dtmanager

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// TestDealMetaDeviceOperation tests dealMetaDeviceOperation for the input
// validation paths that do not require a DMI client or cache.
func TestDealMetaDeviceOperation(t *testing.T) {
	dw := &DMIWorker{}

	tests := []struct {
		name     string
		msg      interface{}
		wantErr  bool
		errValue string
	}{
		{
			name:     "MsgNotMessageType",
			msg:      "not a message",
			wantErr:  true,
			errValue: "msg not Message type",
		},
		{
			name: "WrongResourceSegmentCount",
			msg: &model.Message{
				Router: model.MessageRoute{Resource: "ns/device"},
			},
			wantErr:  true,
			errValue: "wrong resources ns/device",
		},
		{
			name: "UnsupportedResourceType",
			msg: &model.Message{
				Router: model.MessageRoute{Resource: "ns/unknowntype/name"},
			},
			wantErr: false,
		},
		{
			name: "DeviceWithInvalidContent",
			msg: &model.Message{
				Router:  model.MessageRoute{Resource: "ns/device/dev1"},
				Content: []byte("invalid-json"),
			},
			wantErr: true,
		},
		{
			name: "DeviceModelWithInvalidContent",
			msg: &model.Message{
				Router:  model.MessageRoute{Resource: "ns/devicemodel/dm1"},
				Content: []byte("invalid-json"),
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dw.dealMetaDeviceOperation(nil, "", test.msg)
			if (err != nil) != test.wantErr {
				t.Errorf("dealMetaDeviceOperation() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if test.errValue != "" {
				if err == nil {
					t.Errorf("dealMetaDeviceOperation() expected error %q, got nil", test.errValue)
				} else if err.Error() != test.errValue {
					t.Errorf("dealMetaDeviceOperation() error = %q, want %q", err.Error(), test.errValue)
				}
			}
		})
	}
}
