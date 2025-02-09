/*
Copyright 2025 The KubeEdge Authors.

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

package certs

import (
	"reflect"
	"testing"
)

func TestGetCAHandler(t *testing.T) {
	tests := []struct {
		name        string
		handlerType CAHandlerType
		want        CAHandler
	}{
		{
			name:        "x509 CA handler",
			handlerType: CAHandlerTypeX509,
			want:        &x509CAHandler{},
		},
		{
			name:        "unknown handler type",
			handlerType: "unknown",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCAHandler(tt.handlerType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCAHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHandler(t *testing.T) {
	tests := []struct {
		name        string
		handlerType HanndlerType
		want        Handler
	}{
		{
			name:        "x509 certs handler",
			handlerType: HandlerTypeX509,
			want:        &x509CertsHandler{},
		},
		{
			name:        "unknown handler type",
			handlerType: "unknown",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHandler(tt.handlerType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
