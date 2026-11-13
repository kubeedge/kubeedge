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

package v1beta1

import (
	"testing"

	_ "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func TestProtobufNamespaceAndGRPCCompatibility(t *testing.T) {
	if got, want := string((&Device{}).ProtoReflect().Descriptor().FullName()), "kubeedge.dmi.v1beta1.Device"; got != want {
		t.Fatalf("unexpected protobuf name: got %q, want %q", got, want)
	}
	if got, want := DeviceManagerService_MapperRegister_FullMethodName, "/v1beta1.DeviceManagerService/MapperRegister"; got != want {
		t.Fatalf("unexpected gRPC method name: got %q, want %q", got, want)
	}
}
