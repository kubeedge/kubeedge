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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetErrMessage(t *testing.T) {
	cases := []struct {
		name      string
		bodyBytes []byte
		stdResult string
	}{
		{
			name:      "Valid JSON with error message",
			bodyBytes: []byte(`{"status": "Failure", "message": "Resource not found"}`),
			stdResult: "Resource not found",
		},
		{
			name:      "Valid JSON with error message",
			bodyBytes: []byte(`{"status": "Failure", "message": "Invalid request"}`),
			stdResult: "Invalid request",
		},
		{
			name:      "Invalid JSON",
			bodyBytes: []byte(`invalid json`),
			stdResult: "parsing response's body failed with err: invalid character 'i' looking for beginning of value",
		},
		{
			name:      "Empty JSON",
			bodyBytes: []byte(`{}`),
			stdResult: "",
		},
	}

	assert := assert.New(t)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := GetErrMessage(test.bodyBytes)
			assert.Error(err)
			assert.EqualError(err, test.stdResult)
		})
	}
}
