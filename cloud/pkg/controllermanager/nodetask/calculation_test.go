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

package nodetask

import (
	"testing"

	"github.com/stretchr/testify/assert"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestCalculateStatusWithCounts(t *testing.T) {
	cases := []struct {
		name                string
		total               int64
		proc                int64
		fail                int64
		failureTolerateSpec string
		wantStatus          operationsv1alpha2.JobPhase
	}{
		{
			name:       "failed total value",
			total:      0,
			wantStatus: operationsv1alpha2.JobPhaseFailure,
		},
		{
			name:       "processing",
			total:      1,
			proc:       1,
			wantStatus: operationsv1alpha2.JobPhaseInProgress,
		},
		{
			name:       "default failureTolerate",
			total:      10,
			fail:       1,
			wantStatus: operationsv1alpha2.JobPhaseFailure,
		},
		{
			name:                "partial error, within the ratio",
			total:               10,
			fail:                5,
			failureTolerateSpec: "0.5",
			wantStatus:          operationsv1alpha2.JobPhaseComplated,
		},
		{
			name:                "partial error, outside the ratio",
			total:               10,
			fail:                6,
			failureTolerateSpec: "0.5",
			wantStatus:          operationsv1alpha2.JobPhaseFailure,
		},
		{
			name:                "all success",
			total:               10,
			fail:                0,
			failureTolerateSpec: "0.5",
			wantStatus:          operationsv1alpha2.JobPhaseComplated,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := CalculatePhaseWithCounts(c.total, c.proc, c.fail, c.failureTolerateSpec)
			assert.Equal(t, c.wantStatus, actual)
		})
	}
}
