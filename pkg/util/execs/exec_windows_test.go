//go:build windows

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

package execs

import (
	"testing"
)

func TestConvertByte2String(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		charset  string
		expected string
	}{
		{
			name:     "UTF8",
			input:    []byte("hello"),
			charset:  "UTF8",
			expected: "hello",
		},
		{
			name:     "default fallthrough",
			input:    []byte("hello"),
			charset:  "UNKNOWN",
			expected: "hello",
		},
		{
			name:     "GB18030 ASCII",
			input:    []byte("hello"),
			charset:  "GB18030",
			expected: "hello",
		},
		{
			name:     "GB18030 Chinese",
			input:    []byte{0xC4, 0xE3, 0xBA, 0xC3}, // "你好" in GB18030
			charset:  "GB18030",
			expected: "你好",
		},
		{
			name:     "empty input",
			input:    []byte{},
			charset:  "UTF8",
			expected: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ConvertByte2String(c.input, c.charset)
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
