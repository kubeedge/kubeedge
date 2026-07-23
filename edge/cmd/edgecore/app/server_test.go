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

package app

import (
	"testing"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus"
)

func TestBuildServiceBusTLSOptions(t *testing.T) {
	tests := []struct {
		name string
		sb   *v1alpha2.ServiceBus
		want servicebus.TLSOptions
	}{
		{
			name: "nil ServiceBus config",
			sb:   nil,
			want: servicebus.TLSOptions{},
		},
		{
			name: "TLSCertFile empty, plain HTTP default",
			sb:   &v1alpha2.ServiceBus{TLSCertFile: "", TLSPrivateKeyFile: ""},
			want: servicebus.TLSOptions{},
		},
		{
			name: "TLSCertFile set, TLS enabled",
			sb: &v1alpha2.ServiceBus{
				TLSCertFile:       "/etc/kubeedge/certs/servicebus-server.crt",
				TLSPrivateKeyFile: "/etc/kubeedge/certs/servicebus-server.key",
			},
			want: servicebus.TLSOptions{
				TLSEnabled: true,
				CertFile:   "/etc/kubeedge/certs/servicebus-server.crt",
				KeyFile:    "/etc/kubeedge/certs/servicebus-server.key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildServiceBusTLSOptions(tt.sb)
			if got != tt.want {
				t.Errorf("buildServiceBusTLSOptions(%+v): got %+v, want %+v", tt.sb, got, tt.want)
			}
		})
	}
}
