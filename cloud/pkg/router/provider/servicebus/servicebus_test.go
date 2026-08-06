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

package servicebus

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoToTarget(t *testing.T) {
	sb := &ServiceBus{
		targetPath:  "/api",
		servicePort: "8080",
	}

	// notInitialized is the error the valid cases hit: they pass every type
	// assertion and reach the session manager, which is not initialised in a
	// unit test.
	const notInitialized = "cloudhub not initialized"

	testCases := []struct {
		name        string
		data        map[string]interface{}
		expectedErr string
	}{
		{
			name: "valid data with param",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      []byte("test-data"),
				"param":     "test-param",
			},
			expectedErr: notInitialized,
		},
		{
			name: "valid data without param",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: notInitialized,
		},
		{
			name: "valid data without optional header",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"data":      []byte("test-data"),
			},
			expectedErr: notInitialized,
		},
		{
			name: "valid data without optional body",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    http.Header{},
			},
			expectedErr: notInitialized,
		},
		{
			name: "missing messageID",
			data: map[string]interface{}{
				"nodeName": "test-node",
				"method":   http.MethodGet,
				"header":   http.Header{},
				"data":     []byte("test-data"),
			},
			expectedErr: "messageID",
		},
		{
			name: "missing nodeName",
			data: map[string]interface{}{
				"messageID": "test-id",
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: "nodeName",
		},
		{
			name: "missing method",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: "method",
		},
		{
			name: "invalid header type",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    "not-a-header",
				"data":      []byte("test-data"),
			},
			expectedErr: "header",
		},
		{
			name: "invalid body type",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      "not-bytes",
			},
			expectedErr: "data body",
		},
		{
			name: "mistyped messageID (not a string)",
			data: map[string]interface{}{
				"messageID": 42,
				"nodeName":  "test-node",
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: "messageID",
		},
		{
			name: "mistyped nodeName (not a string)",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  42,
				"method":    http.MethodGet,
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: "nodeName",
		},
		{
			name: "mistyped method (not a string)",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"method":    42,
				"header":    http.Header{},
				"data":      []byte("test-data"),
			},
			expectedErr: "method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The invalid cases fail their individual type assertion; the
			// valid cases pass every assertion and error only at the
			// session manager lookup. Asserting the substring keeps a bug
			// in one assertion from being masked by a later error.
			_, err := sb.GoToTarget(tc.data, nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
