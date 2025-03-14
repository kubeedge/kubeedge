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

package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type fakeFuncs struct {
	stepcount    int
	triggerError bool
	step3ToFail  bool
}

func (fr *fakeFuncs) step1(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	return &baseActionResponse{}
}

func (fr *fakeFuncs) step2(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	return &baseActionResponse{interrupt: true}
}

func (fr *fakeFuncs) step3(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	resp := new(baseActionResponse)
	if fr.step3ToFail {
		resp.err = errors.New("test error")
		return resp
	}
	return resp
}

func (fr *fakeFuncs) step3fail(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	return &baseActionResponse{}
}

func (fr *fakeFuncs) step4(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	return &baseActionResponse{err: errors.New("test error"), interrupt: true}
}

func (fr *fakeFuncs) step4fail(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	fr.stepcount++
	return &baseActionResponse{}
}

func (fr *fakeFuncs) reportActionStatus(_jobname, _nodename, _action string, resp ActionResponse) {
	if err := resp.Error(); err != nil {
		fr.triggerError = true
	}
}

func (fr *fakeFuncs) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(_data []byte) (any, error) {
		return nil, nil
	})
}

func newFakeRunner() (*ActionRunner, *fakeFuncs) {
	funcs := &fakeFuncs{}
	runner := &ActionRunner{
		Actions: map[string]ActionFun{
			"step1":     funcs.step1,
			"step2":     funcs.step2,
			"step3":     funcs.step3,
			"step3fail": funcs.step3fail,
			"step4":     funcs.step4,
			"step4fail": funcs.step4fail,
		},
		Flow: &actionflow.Flow{
			First: &actionflow.Action{
				Name: "step1",
				NextSuccessful: &actionflow.Action{
					Name: "step2",
					NextSuccessful: &actionflow.Action{
						Name: "step3",
						NextSuccessful: &actionflow.Action{
							Name: "step4",
							NextFailure: &actionflow.Action{
								Name: "step4fail",
							},
						},
						NextFailure: &actionflow.Action{
							Name: "step3fail",
						},
					},
				},
			},
		},
		ReportActionStatus: funcs.reportActionStatus,
		GetSpecSerializer:  funcs.getSpecSerializer,
	}
	return runner, funcs
}

func TestRunAction(t *testing.T) {
	ctx := context.TODO()
	jobname, nodename := "test", "node1"
	r, funcs := newFakeRunner()

	// step1 -> step2(continue)
	r.RunAction(ctx, jobname, nodename, "step1", nil)
	assert.Equal(t, 2, funcs.stepcount)
	assert.False(t, funcs.triggerError)

	// step3 -- [error] --> step3fail
	funcs.stepcount = 0
	funcs.step3ToFail = true
	r.RunAction(ctx, jobname, nodename, "step3", nil)
	assert.Equal(t, 2, funcs.stepcount)
	assert.True(t, funcs.triggerError)

	// step4 -- [error] -- x
	funcs.stepcount = 0
	r.RunAction(ctx, jobname, nodename, "step4", nil)
	assert.Equal(t, 1, funcs.stepcount)
	assert.True(t, funcs.triggerError)
}
