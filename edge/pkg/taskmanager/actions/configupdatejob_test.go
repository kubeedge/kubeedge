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

package actions

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

const (
	testConfigUpdateCommand = "config-update"
	testConfigUpdateSetFlag = "--set"
)

func TestBuildConfigUpdateArgsDoesNotUseShell(t *testing.T) {
	updateFields := map[string]string{
		"modules.edgehub.websocket.url":           "ws://127.0.0.1:10000/e632aba927ea4ac2b575ec1603d56f10/events",
		"modules.edgehub.websocket.writeDeadline": "30; touch /tmp/pwned",
	}

	args := buildConfigUpdateArgs(updateFields)

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != testConfigUpdateCommand {
		t.Fatalf("expected config-update subcommand, got %q", args[0])
	}
	if args[1] != testConfigUpdateSetFlag {
		t.Fatalf("expected --set flag, got %q", args[1])
	}
	if !strings.Contains(args[2], "30; touch /tmp/pwned") {
		t.Fatalf("expected update value to remain a single argv value, got %q", args[2])
	}

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "bash -c") || strings.Contains(joined, "sh -c") {
		t.Fatalf("args must not invoke a shell: %v", args)
	}
}

func TestBuildConfigUpdateArgsSortsFields(t *testing.T) {
	updateFields := map[string]string{
		"z.key": "z",
		"a.key": "a",
	}

	args := buildConfigUpdateArgs(updateFields)

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != testConfigUpdateCommand {
		t.Fatalf("expected config-update subcommand, got %q", args[0])
	}
	if args[1] != testConfigUpdateSetFlag {
		t.Fatalf("expected --set flag, got %q", args[1])
	}
	if args[2] != "a.key=a,z.key=z" {
		t.Fatalf("unexpected --set value: %q", args[2])
	}
}

func TestBuildConfigUpdateArgsEmptyFields(t *testing.T) {
	args := buildConfigUpdateArgs(map[string]string{})

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != testConfigUpdateCommand || args[1] != testConfigUpdateSetFlag || args[2] != "" {
		t.Fatalf("unexpected args for empty update fields: %v", args)
	}
}

func TestConfigUpdateJobUpdateConfig(t *testing.T) {
	var (
		ctx      = context.TODO()
		jobName  = "test-job"
		nodeName = "test-node"
		spec     = &operationsv1alpha2.ConfigUpdateJobSpec{
			UpdateFields: map[string]string{"modules.edgehub.heartbeat": "20"},
		}
		specser = &cachedSpecSerializer{spec: spec}
		h       = configUpdateJobActionHandler{logger: klog.Background()}
	)

	t.Run("failed to save config update record", func(t *testing.T) {
		var runCmdCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Save",
			func(string, string, *operationsv1alpha2.ConfigUpdateJobSpec) error {
				return errors.New("test error")
			})
		patches.ApplyMethod(reflect.TypeOf((*exec.Cmd)(nil)), "CombinedOutput",
			func(*exec.Cmd) ([]byte, error) {
				runCmdCalled = true
				return nil, nil
			})

		resp := h.updateConfig(ctx, jobName, nodeName, specser)
		require.ErrorContains(t, resp.Error(), "failed to save config update record")
		assert.False(t, runCmdCalled)
	})

	t.Run("edgecore is not restarted", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			cmdErr error
		}{
			{name: "config update successful"},
			{name: "config update failed", cmdErr: errors.New("exit status 1")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var (
					savedJob, savedNode string
					savedSpec           *operationsv1alpha2.ConfigUpdateJobSpec
					deleteCalled        bool
				)
				patches := gomonkey.NewPatches()
				defer patches.Reset()

				patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Save",
					func(jobname, nodename string, spec *operationsv1alpha2.ConfigUpdateJobSpec) error {
						savedJob, savedNode, savedSpec = jobname, nodename, spec
						return nil
					})
				patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Delete",
					func() error {
						deleteCalled = true
						return nil
					})
				patches.ApplyMethod(reflect.TypeOf((*exec.Cmd)(nil)), "CombinedOutput",
					func(*exec.Cmd) ([]byte, error) {
						// The record must be saved before keadm config-update restarts edgecore.
						assert.Equal(t, jobName, savedJob)
						return []byte("output"), tc.cmdErr
					})

				resp := h.updateConfig(ctx, jobName, nodeName, specser)
				if tc.cmdErr != nil {
					require.ErrorContains(t, resp.Error(), "update config failed")
				} else {
					require.NoError(t, resp.Error())
				}
				assert.Equal(t, jobName, savedJob)
				assert.Equal(t, nodeName, savedNode)
				assert.Equal(t, spec, savedSpec)
				// The action reports the result itself, so the record is no longer needed.
				assert.True(t, deleteCalled)
			})
		}
	})

	t.Run("failed to delete config update record", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Save",
			func(string, string, *operationsv1alpha2.ConfigUpdateJobSpec) error {
				return nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.ConfigUpdate)(nil)), "Delete",
			func() error {
				return errors.New("test error")
			})
		patches.ApplyMethod(reflect.TypeOf((*exec.Cmd)(nil)), "CombinedOutput",
			func(*exec.Cmd) ([]byte, error) {
				return nil, nil
			})

		// The record is only used after a restart, failing to delete it must not fail the action.
		resp := h.updateConfig(ctx, jobName, nodeName, specser)
		require.NoError(t, resp.Error())
	})
}
