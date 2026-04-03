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

package edge

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func TestNewTaskEventReporter(t *testing.T) {
	config := &edgeconfig.EdgeCoreConfig{}
	reporter := NewTaskEventReporter("job1", "upgrade", config)

	r, ok := reporter.(*TaskEventReporter)
	assert.True(t, ok)
	assert.Equal(t, "job1", r.JobName)
	assert.Equal(t, "upgrade", r.EventType)
	assert.Equal(t, config, r.Config)
}

func TestReport(t *testing.T) {
	tests := []struct {
		name           string
		err            error
	}{
		{
			name:           "success case - nil error",
			err:            nil,
		},
		{
			name:           "failure case - non nil error",
			err:            errors.New("something went wrong"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &edgeconfig.EdgeCoreConfig{}
			config.Modules = &edgeconfig.Modules{
				EdgeHub: &edgeconfig.EdgeHub{
					TLSCAFile: "/nonexistent/ca.crt",
				},
				Edged: &edgeconfig.Edged{
					TailoredKubeletFlag: edgeconfig.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
			}
			r := &TaskEventReporter{
				JobName:   "test-job",
				EventType: "upgrade",
				Config:    config,
			}
			err := r.Report(tt.err)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read ca")
		})
	}
}

func TestReportTaskResult_MissingCAFile(t *testing.T) {
	config := &edgeconfig.EdgeCoreConfig{}
	config.Modules = &edgeconfig.Modules{
		EdgeHub: &edgeconfig.EdgeHub{
			TLSCAFile: "/nonexistent/ca.crt",
		},
		Edged: &edgeconfig.Edged{
			TailoredKubeletFlag: edgeconfig.TailoredKubeletFlag{
				HostnameOverride: "test-node",
			},
		},
	}

	err := ReportTaskResult(config, TaskTypeUpgrade, "job1", fsm.Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ca")
}
