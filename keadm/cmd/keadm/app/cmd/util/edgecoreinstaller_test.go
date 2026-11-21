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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNodeLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   map[string]string
	}{
		{
			name:   "key and value",
			labels: []string{"region=east", "zone=a"},
			want:   map[string]string{"region": "east", "zone": "a"},
		},
		{
			name:   "key without value",
			labels: []string{"gpu"},
			want:   map[string]string{"gpu": ""},
		},
		{
			name:   "empty value",
			labels: []string{"gpu="},
			want:   map[string]string{"gpu": ""},
		},
		{
			name:   "value containing equals sign",
			labels: []string{"selector=app=web"},
			want:   map[string]string{"selector": "app=web"},
		},
		{
			name:   "missing key is skipped",
			labels: []string{"=east", "", "zone=a"},
			want:   map[string]string{"zone": "a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseNodeLabels(tt.labels))
		})
	}
}
