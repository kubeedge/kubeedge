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
	"testing"
	"time"
)

func withHotReloadStubs(t *testing.T, signal func() error, state func() (string, error)) {
	t.Helper()

	origSignal, origState := signalEdgeCoreReload, edgeCoreServiceState
	origWindow, origPoll := hotReloadHealthWindow, hotReloadHealthPoll
	t.Cleanup(func() {
		signalEdgeCoreReload, edgeCoreServiceState = origSignal, origState
		hotReloadHealthWindow, hotReloadHealthPoll = origWindow, origPoll
	})

	signalEdgeCoreReload = signal
	edgeCoreServiceState = state
	hotReloadHealthWindow = 3 * time.Millisecond
	hotReloadHealthPoll = time.Millisecond
}

func TestHotReload(t *testing.T) {
	t.Run("signal failure is returned without polling", func(t *testing.T) {
		withHotReloadStubs(t,
			func() error { return errors.New("kill: no such process") },
			func() (string, error) { return "", errors.New("edgeCoreServiceState should not be called") },
		)

		if err := hotReload(); err == nil {
			t.Fatal("hotReload() = nil, want an error when signaling edgecore fails")
		}
	})

	t.Run("service stays active for the whole window", func(t *testing.T) {
		withHotReloadStubs(t,
			func() error { return nil },
			func() (string, error) { return systemctlActiveState, nil },
		)

		if err := hotReload(); err != nil {
			t.Fatalf("hotReload() = %v, want nil when edgecore stays active", err)
		}
	})

	t.Run("service becomes inactive during the window", func(t *testing.T) {
		withHotReloadStubs(t,
			func() error { return nil },
			func() (string, error) { return "failed", nil },
		)

		if err := hotReload(); err == nil {
			t.Fatal("hotReload() = nil, want an error when edgecore stops being active")
		}
	})
}

func TestEdgeCoreActive(t *testing.T) {
	cases := []struct {
		name  string
		state func() (string, error)
		want  bool
	}{
		{
			name:  systemctlActiveState,
			state: func() (string, error) { return systemctlActiveState, nil },
			want:  true,
		},
		{
			name:  "inactive",
			state: func() (string, error) { return "inactive", nil },
			want:  false,
		},
		{
			name:  "query error",
			state: func() (string, error) { return "", errors.New("systemctl not found") },
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := edgeCoreServiceState
			defer func() { edgeCoreServiceState = orig }()
			edgeCoreServiceState = tc.state

			if got := edgeCoreActive(); got != tc.want {
				t.Errorf("edgeCoreActive() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsHotReloadable(t *testing.T) {
	cases := []struct {
		name string
		sets string
		want bool
	}{
		{
			name: "single hot reloadable field",
			sets: "modules.edgeHub.heartbeat=30",
			want: true,
		},
		{
			name: "multiple hot reloadable fields are case insensitive",
			sets: "Modules.EdgeHub.Heartbeat=30,modules.metamanager.remotequerytimeout=90",
			want: true,
		},
		{
			name: "field requiring a restart falls back",
			sets: "modules.edgeHub.websocket.server=example.com",
			want: false,
		},
		{
			name: "mixing a safe and unsafe field falls back",
			sets: "modules.edgeHub.heartbeat=30,modules.edged.address=0.0.0.0",
			want: false,
		},
		{
			name: "empty sets is not hot reloadable",
			sets: "",
			want: false,
		},
		{
			name: "malformed field falls back",
			sets: "modules.edgeHub.heartbeat",
			want: false,
		},
		{
			name: "whitespace around fields is ignored",
			sets: " modules.edgeHub.heartbeat = 30 , modules.metaManager.remoteQueryTimeout = 90 ",
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHotReloadable(tc.sets); got != tc.want {
				t.Errorf("isHotReloadable(%q) = %v, want %v", tc.sets, got, tc.want)
			}
		})
	}
}
