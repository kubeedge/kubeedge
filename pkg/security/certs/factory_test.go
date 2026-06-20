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

package certs

import (
	"testing"
)

func TestGetCAHandler(t *testing.T) {
	tests := []struct {
		name        string
		handlerType CAHandlerType
		wantNil     bool
	}{
		{
			name:        "Valid x509 CA Handler",
			handlerType: CAHandlerTypeX509,
			wantNil:     false,
		},
		{
			name:        "Invalid CA Handler",
			handlerType: CAHandlerType("invalid-type"),
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCAHandler(tt.handlerType)
			if (got == nil) != tt.wantNil {
				t.Errorf("GetCAHandler() returned nil = %v, wantNil = %v", got == nil, tt.wantNil)
			}
		})
	}
}

func TestGetHandler(t *testing.T) {
	tests := []struct {
		name        string
		handlerType HandlerType
		wantNil     bool
	}{
		{
			name:        "Valid x509 Handler",
			handlerType: HandlerTypeX509,
			wantNil:     false,
		},
		{
			name:        "Invalid Handler",
			handlerType: HandlerType("invalid-type"),
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHandler(tt.handlerType)
			if (got == nil) != tt.wantNil {
				t.Errorf("GetHandler() returned nil = %v, wantNil = %v", got == nil, tt.wantNil)
			}
		})
	}
}
