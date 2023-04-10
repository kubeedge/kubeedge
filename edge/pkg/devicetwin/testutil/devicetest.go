/*
Copyright 2023 The KubeEdge Authors.

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

package testutil

import (
	"encoding/json"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

// GenerateAddDevicePalyloadMsg generates content for adding device payload message
func GenerateAddDevicePalyloadMsg(t *testing.T) []byte {
	//Creating content for model.message type
	payload := dttype.MembershipUpdate{
		AddDevices: []dttype.Device{
			{
				ID:    "DeviceA",
				Name:  "Router",
				State: "unknown",
			},
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Got error on marshalling: %+v", err)
		return nil
	}

	return content
}
