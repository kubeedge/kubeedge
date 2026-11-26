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

package common

import "testing"

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "float32", value: float32(0.1), want: "0.1"},
		{name: "float32 reading", value: float32(23.4), want: "23.4"},
		{name: "float32 integral", value: float32(25), want: "25"},
		{name: "float64", value: 0.1, want: "0.1"},
		{name: "int", value: 42, want: "42"},
		{name: "int64", value: int64(-7), want: "-7"},
		{name: "uint64", value: uint64(18446744073709551615), want: "18446744073709551615"},
		{name: "bool", value: true, want: "true"},
		{name: "string", value: "on", want: "on"},
		{name: "bytes", value: []byte("on"), want: "on"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToString(tt.value)
			if err != nil {
				t.Fatalf("ConvertToString(%v) returned error: %v", tt.value, err)
			}
			if got != tt.want {
				t.Errorf("ConvertToString(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
