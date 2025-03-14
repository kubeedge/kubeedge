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

package wrap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestWithEventObj(t *testing.T) {
	cases := []struct {
		name     string
		input    any
		wantType any
		hasError bool
	}{
		{
			name:     "input NodeUpgradeJob",
			input:    &operationsv1alpha2.NodeUpgradeJob{},
			wantType: &NodeUpgradeJob{},
		},
		{
			name:     "input ImagePrePullJob",
			input:    &operationsv1alpha2.ImagePrePullJob{},
			wantType: &ImagePrePullJob{},
		},
		{
			name:     "invalid input type",
			input:    "",
			hasError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			job, err := WithEventObj(c.input)
			if c.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.IsType(t, c.wantType, job)
		})
	}
}
