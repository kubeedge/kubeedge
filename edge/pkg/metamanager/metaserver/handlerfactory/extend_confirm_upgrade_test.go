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

package handlerfactory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	commonmsg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	daov2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1/taskexecutor"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func TestConfirmUpgrade(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeName = "test-node"
	)
	req := httptest.NewRequest("POST", "/confirm-upgrade", nil)

	t.Run("get upgrade spec failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return "", "", nil, errors.New("get upgrade spec failed")
			})

		factory := NewFactory()
		w := httptest.NewRecorder()
		factory.ConfirmUpgrade().ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to get upgrade spec")
	})

	t.Run("do v1alpha2+ upgrade", func(t *testing.T) {
		var wg sync.WaitGroup
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return jobName, nodeName, &operationsv1alpha2.NodeUpgradeJobSpec{}, nil
			})
		patches.ApplyFunc(doUpgrade, func(_jobname, _nodename string, _spec *operationsv1alpha2.NodeUpgradeJobSpec) error {
			wg.Done()
			return nil
		})

		wg.Add(1)
		factory := NewFactory()
		w := httptest.NewRecorder()
		factory.ConfirmUpgrade().ServeHTTP(w, req)
		wg.Wait()
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get v1alpha1 upgrade request failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return jobName, nodeName, nil, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.UpgradeV1alpha1)(nil)), "Get",
			func() (*types.NodeTaskRequest, error) {
				return nil, errors.New("get upgrade request failed")
			})

		factory := NewFactory()
		w := httptest.NewRecorder()
		factory.ConfirmUpgrade().ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to get upgrade request")
	})

	t.Run("no valid upgrade tasks to execute", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return jobName, nodeName, nil, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.UpgradeV1alpha1)(nil)), "Get",
			func() (*types.NodeTaskRequest, error) {
				return nil, nil
			})

		factory := NewFactory()
		w := httptest.NewRecorder()
		factory.ConfirmUpgrade().ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "there are no valid upgrade tasks to execute")
	})

	t.Run("do v1alpha1 upgrade", func(t *testing.T) {
		var wg sync.WaitGroup
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return jobName, nodeName, nil, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*daov2.UpgradeV1alpha1)(nil)), "Get",
			func() (*types.NodeTaskRequest, error) {
				return &types.NodeTaskRequest{}, nil
			})
		patches.ApplyFunc(doV1alpha1Upgrade, func(_req *types.NodeTaskRequest) error {
			wg.Done()
			return nil
		})

		wg.Add(1)
		factory := NewFactory()
		w := httptest.NewRecorder()
		factory.ConfirmUpgrade().ServeHTTP(w, req)
		wg.Wait()
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestConfirmUpgradeDoUpgrade(t *testing.T) {
	var (
		jobName         = "test-job"
		nodeName        = "test-node"
		runActionCalled = false
	)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
		return &actions.ActionRunner{}
	})
	patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
		func(ctx context.Context, jobname, nodename, action string, specData []byte) {
			runActionCalled = true
			assert.Equal(t, jobName, jobname)
			assert.Equal(t, nodeName, nodename)
			assert.Equal(t, string(operationsv1alpha2.NodeUpgradeJobActionUpgrade), action)
		})
	err := doUpgrade(jobName, nodeName, &operationsv1alpha2.NodeUpgradeJobSpec{})
	require.NoError(t, err)
	assert.True(t, runActionCalled)
}

func TestConfirmUpgradeDoV1alpha1Upgrade(t *testing.T) {
	var (
		jobName   = "test-job"
		eventType = "Upgrade"
		action    = fsmv1alpha1.ActionSuccess
	)
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(taskexecutor.GetExecutor, func(_name string) (taskexecutor.Executor, error) {
		return &fakeTaskExecutor{
			event: fsm.Event{
				Type:   eventType,
				Action: action,
			},
		}, nil
	})
	patches.ApplyFunc(options.GetEdgeCoreConfig, func() *configv1alpha2.EdgeCoreConfig {
		return configv1alpha2.NewDefaultEdgeCoreConfig()
	})
	patches.ApplyFunc(commonmsg.ReportTaskResult, func(taskType, taskID string, resp types.NodeTaskResponse) {
		assert.Equal(t, operationsv1alpha2.ResourceNodeUpgradeJob, taskType)
		assert.Equal(t, jobName, taskID)
		assert.Equal(t, eventType, resp.Event)
		assert.Equal(t, action, resp.Action)
	})
	err := doV1alpha1Upgrade(&types.NodeTaskRequest{
		TaskID: jobName,
		Type:   operationsv1alpha2.ResourceNodeUpgradeJob,
	})
	require.NoError(t, err)
}

type fakeTaskExecutor struct {
	event fsm.Event
}

func (f fakeTaskExecutor) Name() string {
	return "fake"
}

func (f fakeTaskExecutor) Do(_req types.NodeTaskRequest) (fsm.Event, error) {
	return f.event, nil
}
