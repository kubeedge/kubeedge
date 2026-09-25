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

package taskmanager

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	upgradeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
)

func TestReportUpgradeStatus(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeName = "test-node"
		ctx      = context.Background()
	)

	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(upgradeedge.JSONReporterInfoExists, func() bool {
		return true
	})

	globpatches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
		return nil
	})
	globpatches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Get",
		func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
			return jobName, nodeName, nil, nil
		})
	globpatches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Delete",
		func() error {
			return nil
		})

	t.Run("reporter info not exists", func(t *testing.T) {
		var parseReporterInfoCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.JSONReporterInfoExists, func() bool {
			return false
		})
		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			parseReporterInfoCalled = true
			return upgradeedge.JSONReporterInfo{}, nil
		})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.False(t, parseReporterInfoCalled)
	})

	t.Run("no upgrade record found", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
				return "", "", nil, nil
			})

		err := ReportUpgradeStatus(ctx)
		require.ErrorContains(t, err, "no upgrade record found or invalid info from meta data")
	})

	t.Run("unsupportd event type", func(t *testing.T) {
		var reportStatusCalled bool
		reporterInfo := upgradeedge.JSONReporterInfo{
			EventType:   "Backup",
			Success:     true,
			FromVersion: "v1.20.0",
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return reporterInfo, nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
		})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.False(t, reportStatusCalled)
	})

	t.Run("upgrade failed and run rollback", func(t *testing.T) {
		var (
			reportStatusCalled bool
			runActionCalled    bool
		)
		reporterInfo := upgradeedge.JSONReporterInfo{
			EventType:   upgradeedge.EventTypeUpgrade,
			Success:     false,
			FromVersion: "v1.20.0",
			ToVersion:   "v1.21.0",
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return reporterInfo, nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
			assert.Equal(t, string(operationsv1alpha2.NodeUpgradeJobActionUpgrade), msgbody.Action)
			assert.False(t, msgbody.Succ)
			assert.Equal(t, taskmsg.FormatNodeUpgradeJobExtend(reporterInfo.FromVersion, reporterInfo.ToVersion), msgbody.Extend)
		})
		patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
			func(ctx context.Context, jobname, nodename, action string, specData []byte) {
				runActionCalled = true
				assert.Equal(t, jobName, jobname)
				assert.Equal(t, nodeName, nodename)
				assert.Equal(t, string(operationsv1alpha2.NodeUpgradeJobActionRollBack), action)
			})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.True(t, reportStatusCalled)
		assert.True(t, runActionCalled)
	})

	t.Run("rollback failed", func(t *testing.T) {
		var (
			reportStatusCalled bool
			runActionCalled    bool
		)
		reporterInfo := upgradeedge.JSONReporterInfo{
			EventType:   upgradeedge.EventTypeRollback,
			Success:     false,
			FromVersion: "v1.21.0",
			ToVersion:   "v1.20.0",
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return reporterInfo, nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
			assert.Equal(t, string(operationsv1alpha2.NodeUpgradeJobActionRollBack), msgbody.Action)
			assert.False(t, msgbody.Succ)
		})
		patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
			func(ctx context.Context, jobname, nodename, action string, specData []byte) {
				runActionCalled = true
			})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.True(t, reportStatusCalled)
		assert.False(t, runActionCalled)
	})
}

func TestReportConfigUpdateStatus(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeName = "test-node"
		ctx      = context.Background()
		spec     = &operationsv1alpha2.ConfigUpdateJobSpec{
			UpdateFields: map[string]string{"modules.edgehub.heartbeat": "20"},
		}
	)

	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(upgradeedge.JSONReporterInfoExists, func() bool {
		return true
	})
	globpatches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Get",
		func() (string, string, *operationsv1alpha2.NodeUpgradeJobSpec, error) {
			t.Error("node upgrade record should not be used for config update report")
			return "", "", nil, nil
		})

	t.Run("no config update record found", func(t *testing.T) {
		var (
			removeReporterInfoCalled bool
			reportStatusCalled       bool
		)
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{EventType: upgradeedge.EventTypeConfigUpdate, Success: true}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return "", "", nil, nil
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			removeReporterInfoCalled = true
			return nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, _msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
		})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.False(t, reportStatusCalled)
		// The stale report file must be removed, otherwise it is parsed on every connection.
		assert.True(t, removeReporterInfoCalled)
	})

	t.Run("failed to get config update record", func(t *testing.T) {
		var removeReporterInfoCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{EventType: upgradeedge.EventTypeConfigUpdate, Success: true}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return "", "", nil, errors.New("test error")
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			removeReporterInfoCalled = true
			return nil
		})

		err := ReportUpgradeStatus(ctx)
		require.ErrorContains(t, err, "failed to get config update record")
		// Keep the report file so that the result can be reported on the next connection.
		assert.False(t, removeReporterInfoCalled)
	})

	t.Run("config update successful", func(t *testing.T) {
		var (
			reportStatusCalled       bool
			removeReporterInfoCalled bool
			deleteRecordCalled       bool
			runActionCalled          bool
		)
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{EventType: upgradeedge.EventTypeConfigUpdate, Success: true}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return jobName, nodeName, spec, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Delete",
			func() error {
				deleteRecordCalled = true
				return nil
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			removeReporterInfoCalled = true
			return nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
			assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, res.ResourceType)
			assert.Equal(t, jobName, res.JobName)
			assert.Equal(t, nodeName, res.NodeName)
			assert.Equal(t, string(operationsv1alpha2.ConfigUpdateJobActionUpdate), msgbody.Action)
			assert.True(t, msgbody.Succ)
			assert.NotEmpty(t, msgbody.FinishTime)
		})
		patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
			func(ctx context.Context, jobname, nodename, action string, specData []byte) {
				runActionCalled = true
			})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.True(t, reportStatusCalled)
		assert.True(t, removeReporterInfoCalled)
		assert.True(t, deleteRecordCalled)
		assert.False(t, runActionCalled)
	})

	t.Run("config update failed and run rollback", func(t *testing.T) {
		var (
			reportStatusCalled bool
			deleteRecordCalled bool
			runActionCalled    bool
		)
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{
				EventType:    upgradeedge.EventTypeConfigUpdate,
				Success:      false,
				ErrorMessage: "failed restart edgecore",
			}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return jobName, nodeName, spec, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Delete",
			func() error {
				deleteRecordCalled = true
				return nil
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			return nil
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
			assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, res.ResourceType)
			assert.Equal(t, string(operationsv1alpha2.ConfigUpdateJobActionUpdate), msgbody.Action)
			assert.False(t, msgbody.Succ)
			assert.Equal(t, "failed restart edgecore", msgbody.Reason)
		})
		patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
			assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, name)
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
			func(ctx context.Context, jobname, nodename, action string, specData []byte) {
				runActionCalled = true
				assert.Equal(t, jobName, jobname)
				assert.Equal(t, nodeName, nodename)
				assert.Equal(t, string(operationsv1alpha2.ConfigUpdateJobActionRollBack), action)
				var gotSpec operationsv1alpha2.ConfigUpdateJobSpec
				require.NoError(t, json.Unmarshal(specData, &gotSpec))
				assert.Equal(t, *spec, gotSpec)
			})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.True(t, reportStatusCalled)
		assert.True(t, deleteRecordCalled)
		assert.True(t, runActionCalled)
	})

	t.Run("no config update record found and failed to remove report file", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{EventType: upgradeedge.EventTypeConfigUpdate, Success: true}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return "", "", nil, nil
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			return errors.New("test error")
		})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
	})

	t.Run("failed to clean up does not stop report and rollback", func(t *testing.T) {
		var (
			reportStatusCalled bool
			runActionCalled    bool
		)
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{EventType: upgradeedge.EventTypeConfigUpdate, Success: false}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Get",
			func() (string, string, *operationsv1alpha2.ConfigUpdateJobSpec, error) {
				return jobName, nodeName, spec, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Delete",
			func() error {
				return errors.New("test error")
			})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			return errors.New("test error")
		})
		patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, _msgbody taskmsg.UpstreamMessage) {
			reportStatusCalled = true
		})
		patches.ApplyFunc(actions.GetRunner, func(name string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(reflect.TypeOf((*actions.ActionRunner)(nil)), "RunAction",
			func(ctx context.Context, jobname, nodename, action string, specData []byte) {
				runActionCalled = true
			})

		err := ReportUpgradeStatus(ctx)
		require.NoError(t, err)
		assert.True(t, reportStatusCalled)
		assert.True(t, runActionCalled)
	})
}
