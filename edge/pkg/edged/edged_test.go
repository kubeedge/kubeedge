/*
Copyright 2016 The Kubernetes Authors.

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

package edged

import (
	"testing"
	"time"
)

func TestKubeletHealthCheckURL(t *testing.T) {
	testCases := []struct {
		name    string
		address string
		port    int32
		want    string
	}{
		{
			name:    "normal IPv4 address",
			address: "192.168.1.10",
			port:    10255,
			want:    "http://192.168.1.10:10255/healthz/syncloop",
		},
		{
			name:    "empty address maps to IPv4 loopback",
			address: "",
			port:    10255,
			want:    "http://127.0.0.1:10255/healthz/syncloop",
		},
		{
			name:    "wildcard IPv4 maps to loopback",
			address: "0.0.0.0",
			port:    10255,
			want:    "http://127.0.0.1:10255/healthz/syncloop",
		},
		{
			name:    "wildcard IPv6 maps to loopback",
			address: "::",
			port:    10255,
			want:    "http://[::1]:10255/healthz/syncloop",
		},
		{
			name:    "IPv6 address is bracketed",
			address: "::1",
			port:    10350,
			want:    "http://[::1]:10350/healthz/syncloop",
		},
		{
			name:    "full IPv6 address is bracketed",
			address: "2001:db8::1",
			port:    10350,
			want:    "http://[2001:db8::1]:10350/healthz/syncloop",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := kubeletHealthCheckURL(tc.address, tc.port)
			if got != tc.want {
				t.Errorf("kubeletHealthCheckURL(%q, %d) = %q, want %q", tc.address, tc.port, got, tc.want)
			}
		})
	}
}

func TestKubeletHealthCheckReadOnlyPortDisabled(t *testing.T) {
	kubeletReadyChan := make(chan struct{}, 1)
	// When ReadOnlyPort is 0, the health check must signal readiness and
	// return immediately without polling.
	kubeletHealthCheck("127.0.0.1", 0, kubeletReadyChan)

	select {
	case <-kubeletReadyChan:
		// expected: readiness signaled
	case <-time.After(2 * time.Second):
		t.Fatal("kubeletHealthCheck did not signal readiness when ReadOnlyPort is 0")
	}
}
