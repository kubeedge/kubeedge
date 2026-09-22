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

func TestDealMetaDeviceOperation(t *testing.T) {
	tests := []struct {
		name    string
		msg     interface{}
		wantErr bool
	}{
		{
			name:    "dealMetaDeviceOperationTest-WrongMessageType",
			msg:     "",
			wantErr: true,
		},
		{
			name:    "dealMetaDeviceOperationTest-WrongResourceCount",
			msg:     &model.Message{Router: model.MessageRoute{Resource: "device"}},
			wantErr: true,
		},
		{
			name:    "dealMetaDeviceOperationTest-InvalidContentType",
			msg:     &model.Message{Router: model.MessageRoute{Resource: "ns/device/name"}, Content: "not-bytes"},
			wantErr: true,
		},
	}

	dw := &DMIWorker{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dw.dealMetaDeviceOperation(nil, "", test.msg)
			if (err != nil) != test.wantErr {
				t.Errorf("dealMetaDeviceOperation() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
