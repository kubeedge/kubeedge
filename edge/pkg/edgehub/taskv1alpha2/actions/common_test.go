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

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type fakeRunner struct {
	stepcount    int
	triggerError bool
	baseActionRunner
}

func (fr *fakeRunner) step1(_ context.Context, _ *nodetaskmsg.TaskDownstreamMessage,
) (bool, error) {
	fr.stepcount++
	return true, nil
}

func (fr *fakeRunner) step2(_ context.Context, _ *nodetaskmsg.TaskDownstreamMessage,
) (bool, error) {
	fr.stepcount++
	return false, nil
}

func (fr *fakeRunner) step3(_ context.Context, _ *nodetaskmsg.TaskDownstreamMessage,
) (bool, error) {
	fr.stepcount++
	return false, errors.New("test error")
}

func (fr *fakeRunner) step3fail(_ context.Context, _ *nodetaskmsg.TaskDownstreamMessage,
) (bool, error) {
	fr.stepcount++
	return true, nil
}

func (fr *fakeRunner) reportFun(_, _, _ string, err error) {
	if err != nil {
		fr.triggerError = true
	}
}

func newFakeRunner() *fakeRunner {
	fr := &fakeRunner{}
	fr.baseActionRunner = baseActionRunner{
		actions: map[string]ActionFun{
			"step1":     fr.step1,
			"step2":     fr.step2,
			"step3":     fr.step3,
			"step3fail": fr.step3fail,
		},
		flow: actionflow.Flow{
			First: actionflow.Action{
				Name: "step1",
				NextSuccessful: &actionflow.Action{
					Name: "step2",
					NextSuccessful: &actionflow.Action{
						Name: "step3",
						NextFailure: &actionflow.Action{
							Name: "step3fail",
						},
					},
				},
			},
		},
		reportFun: fr.reportFun,
	}
	return fr
}

func TestRunAction(t *testing.T) {
	msg := &nodetaskmsg.TaskDownstreamMessage{}
	fr := newFakeRunner()
	fr.RunAction("step1", msg)
	require.Equal(t, 2, fr.stepcount)
	require.False(t, fr.triggerError)
	fr.RunAction("step3", msg)
	require.Equal(t, 3, fr.stepcount)
	require.True(t, fr.triggerError)
}
