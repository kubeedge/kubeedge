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

package mqtt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMQTTTopicName(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		wantErr bool
		errMsg  string
	}{
		// Valid topics
		{
			name:    "valid device twin topic",
			topic:   "$hw/events/device/test-device/twin/update",
			wantErr: false,
		},
		{
			name:    "valid node membership topic",
			topic:   "$hw/events/node/test-node/membership/get",
			wantErr: false,
		},
		{
			name:    "valid upload records topic",
			topic:   "SYS/dis/upload_records",
			wantErr: false,
		},
		{
			name:    "valid user topic",
			topic:   "namespace/user/custom/data",
			wantErr: false,
		},
		{
			name:    "valid topic with dollar sign prefix",
			topic:   "$hw/events/upload/data",
			wantErr: false,
		},
		{
			name:    "valid single-segment topic",
			topic:   "simple",
			wantErr: false,
		},
		{
			name:    "valid topic with leading slash",
			topic:   "/leading/slash/topic",
			wantErr: false,
		},
		{
			name:    "valid topic with trailing slash",
			topic:   "trailing/slash/",
			wantErr: false,
		},
		{
			name:    "valid topic with consecutive slashes",
			topic:   "some//topic",
			wantErr: false,
		},
		{
			name:    "valid topic with unicode characters",
			topic:   "$hw/events/设备/test",
			wantErr: false,
		},
		// Invalid topics
		{
			name:    "empty topic",
			topic:   "",
			wantErr: true,
			errMsg:  "must not be empty",
		},
		{
			name:    "topic with null character",
			topic:   "test/\x00/topic",
			wantErr: true,
			errMsg:  "null character",
		},
		{
			name:    "topic with null character at start",
			topic:   "\x00topic",
			wantErr: true,
			errMsg:  "null character",
		},
		{
			name:    "topic with plus wildcard",
			topic:   "$hw/events/device/+/twin/update",
			wantErr: true,
			errMsg:  "wildcard",
		},
		{
			name:    "topic with hash wildcard",
			topic:   "$hw/events/upload/#",
			wantErr: true,
			errMsg:  "wildcard",
		},
		{
			name:    "topic with both wildcards",
			topic:   "+/user/#",
			wantErr: true,
			errMsg:  "wildcard",
		},
		{
			name:    "topic exceeding max length",
			topic:   strings.Repeat("a", maxTopicLength+1),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "topic at max length boundary",
			topic:   strings.Repeat("a", maxTopicLength),
			wantErr: false,
		},
		{
			name:    "topic with control character SOH",
			topic:   "test/\x01/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with control character BEL",
			topic:   "test/\x07/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with control character TAB",
			topic:   "test/\t/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with control character newline",
			topic:   "test/\n/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with control character carriage return",
			topic:   "test/\r/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with DEL character",
			topic:   "test/\x7F/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with C1 control character",
			topic:   "test/\u0080/topic",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "topic with invalid UTF-8",
			topic:   string([]byte{0xff, 0xfe}),
			wantErr: true,
			errMsg:  "invalid UTF-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMQTTTopicName(tt.topic)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
