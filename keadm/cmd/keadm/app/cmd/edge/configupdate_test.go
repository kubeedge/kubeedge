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

package edge

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func writeTestEdgeCoreConfig(t *testing.T) string {
	t.Helper()

	configFile := filepath.Join(t.TempDir(), "edgecore.yaml")
	if err := v1alpha2.NewDefaultEdgeCoreConfig().WriteTo(configFile); err != nil {
		t.Fatalf("failed to write test edgecore config: %v", err)
	}
	return configFile
}

func stubRestartEdgeCore(t *testing.T, fn func() error) *int {
	t.Helper()

	orig := restartEdgeCore
	calls := 0
	restartEdgeCore = func() error {
		calls++
		return fn()
	}
	t.Cleanup(func() { restartEdgeCore = orig })
	return &calls
}

func TestConfigUpdate(t *testing.T) {
	t.Run("non hot reloadable field restarts edgecore", func(t *testing.T) {
		configFile := writeTestEdgeCoreConfig(t)
		restarts := stubRestartEdgeCore(t, func() error { return nil })

		executor := newConfigUpdateExecutor()
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{Config: configFile},
			Sets:        "modules.edgeHub.websocket.server=example.com:10000",
		}

		if err := executor.configUpdate(opts); err != nil {
			t.Fatalf("configUpdate() = %v, want nil", err)
		}
		if *restarts != 1 {
			t.Errorf("restartEdgeCore called %d times, want 1", *restarts)
		}
	})

	t.Run("restart failure is surfaced", func(t *testing.T) {
		configFile := writeTestEdgeCoreConfig(t)
		stubRestartEdgeCore(t, func() error { return errors.New("systemctl: unit not found") })

		executor := newConfigUpdateExecutor()
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{Config: configFile},
			Sets:        "modules.edgeHub.websocket.server=example.com:10000",
		}

		if err := executor.configUpdate(opts); err == nil {
			t.Fatal("configUpdate() = nil, want an error when the restart fails")
		}
	})

	t.Run("hot reloadable field skips the restart", func(t *testing.T) {
		configFile := writeTestEdgeCoreConfig(t)
		restarts := stubRestartEdgeCore(t, func() error { return nil })

		origSignal, origState := signalEdgeCoreReload, edgeCoreServiceState
		origWindow, origPoll := hotReloadHealthWindow, hotReloadHealthPoll
		t.Cleanup(func() {
			signalEdgeCoreReload, edgeCoreServiceState = origSignal, origState
			hotReloadHealthWindow, hotReloadHealthPoll = origWindow, origPoll
		})
		signalEdgeCoreReload = func() error { return nil }
		edgeCoreServiceState = func() (string, error) { return "active", nil }
		hotReloadHealthWindow = 3 * time.Millisecond
		hotReloadHealthPoll = time.Millisecond

		executor := newConfigUpdateExecutor()
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{Config: configFile},
			Sets:        "modules.edgeHub.heartbeat=45",
		}

		if err := executor.configUpdate(opts); err != nil {
			t.Fatalf("configUpdate() = %v, want nil", err)
		}
		if *restarts != 0 {
			t.Errorf("restartEdgeCore called %d times, want 0 when the hot reload stays healthy", *restarts)
		}
	})

	t.Run("unhealthy hot reload rolls back and restarts", func(t *testing.T) {
		configFile := writeTestEdgeCoreConfig(t)
		before, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("failed to read the config file before the update: %v", err)
		}

		restarts := stubRestartEdgeCore(t, func() error { return nil })

		origSignal, origState := signalEdgeCoreReload, edgeCoreServiceState
		origWindow, origPoll := hotReloadHealthWindow, hotReloadHealthPoll
		t.Cleanup(func() {
			signalEdgeCoreReload, edgeCoreServiceState = origSignal, origState
			hotReloadHealthWindow, hotReloadHealthPoll = origWindow, origPoll
		})
		signalEdgeCoreReload = func() error { return nil }
		edgeCoreServiceState = func() (string, error) { return "failed", nil }
		hotReloadHealthWindow = 3 * time.Millisecond
		hotReloadHealthPoll = time.Millisecond

		executor := newConfigUpdateExecutor()
		opts := ConfigUpdateOptions{
			BaseOptions: BaseOptions{Config: configFile},
			Sets:        "modules.edgeHub.heartbeat=45",
		}

		if err := executor.configUpdate(opts); err == nil {
			t.Fatal("configUpdate() = nil, want an error when edgecore is unhealthy after the hot reload")
		}
		if *restarts != 1 {
			t.Errorf("restartEdgeCore called %d times, want 1 during rollback", *restarts)
		}

		after, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("failed to read the config file after the rollback: %v", err)
		}
		if string(after) != string(before) {
			t.Error("config file was not restored to its previous contents after a failed hot reload")
		}
	})
}
