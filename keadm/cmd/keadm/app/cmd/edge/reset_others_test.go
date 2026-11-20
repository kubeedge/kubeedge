//go:build !windows

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
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
)

type fakeExtSystem struct {
	extsystem.ExtSystem
	hasService bool
	active     bool
	stuck      bool
	stopErr    error
	removed    bool
}

func (f *fakeExtSystem) ServiceExists(string) bool {
	return f.hasService
}

func (f *fakeExtSystem) ServiceIsActive(string) bool {
	return f.active
}

func (f *fakeExtSystem) ServiceStop(string) error {
	f.active = f.stuck
	return f.stopErr
}

func (*fakeExtSystem) ServiceIsEnabled(string) bool {
	return false
}

func (f *fakeExtSystem) ServiceRemove(string) error {
	f.removed = true
	return nil
}

func TestKillProcess(t *testing.T) {
	bash, err := exec.LookPath("bash")
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Symlink(bash, filepath.Join(dir, "bash")))
	pkill := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PKILL_ARGS\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pkill"), []byte(pkill), 0o755))
	args := filepath.Join(dir, "args")
	t.Setenv("PKILL_ARGS", args)
	t.Setenv("PATH", dir)

	require.NoError(t, killProcess("edgecore"))
	got, err := os.ReadFile(args)
	require.NoError(t, err)
	assert.Equal(t, "-x\nedgecore\n", string(got))
}

func TestTearDownEdgeCore(t *testing.T) {
	type testCase struct {
		name         string
		svc          *fakeExtSystem
		bootPath     string
		running      bool
		runErr       error
		killErr      error
		survivesKill bool
		postKillErr  error
		wantKill     bool
		wantErr      string
	}
	var (
		tc        testCase
		killed    bool
		lingering int
	)
	defer func(timeout, interval time.Duration, bootPath string) {
		stopTimeout, stopInterval, systemdBootPath = timeout, interval, bootPath
	}(stopTimeout, stopInterval, systemdBootPath)
	stopTimeout, stopInterval = 100*time.Millisecond, time.Millisecond
	bootDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(bootDir, "system"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(bootDir, "file"), nil, 0o600))

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(extsystem.GetExtSystem, func() (extsystem.ExtSystem, error) {
		if tc.svc == nil {
			return nil, errors.New("no supported init system detected")
		}
		return tc.svc, nil
	})
	patches.ApplyFunc(util.IsKubeEdgeProcessRunning, func(string) (bool, error) {
		if !killed {
			return tc.running, tc.runErr
		}
		if lingering > 0 {
			lingering--
			return true, nil
		}
		return tc.survivesKill, tc.postKillErr
	})
	patches.ApplyFunc(killProcess, func(string) error {
		killed = true
		return tc.killErr
	})

	pidofErr := errors.New("pidof failed")
	cases := []testCase{
		{name: "no init system, edgecore running", running: true, wantKill: true},
		{name: "no init system, edgecore not running"},
		{name: "no init system found on a systemd host", bootPath: "system", running: true, wantErr: "failed to get init system"},
		{name: "no init system found, systemd check fails", bootPath: "file/system", running: true, wantErr: "failed to get init system"},
		{name: "no edgecore service, edgecore running", svc: &fakeExtSystem{}, running: true, wantKill: true},
		{name: "no edgecore service, edgecore not running", svc: &fakeExtSystem{}},
		{name: "inactive edgecore service, edgecore running", svc: &fakeExtSystem{hasService: true}, running: true, wantKill: true},
		{name: "inactive edgecore service, edgecore not running", svc: &fakeExtSystem{hasService: true}},
		{name: "edgecore service stops", svc: &fakeExtSystem{hasService: true, active: true}},
		{name: "edgecore service stop fails but it stops", svc: &fakeExtSystem{hasService: true, active: true, stopErr: errors.New("stop failed")}},
		{name: "edgecore service does not stop", svc: &fakeExtSystem{hasService: true, active: true, stuck: true}, running: true, wantErr: "waiting for edgecore service to stop"},
		{name: "process check fails", svc: &fakeExtSystem{}, runErr: pidofErr, wantErr: pidofErr.Error()},
		{name: "process check fails after kill", svc: &fakeExtSystem{}, running: true, postKillErr: pidofErr, wantKill: true, wantErr: pidofErr.Error()},
		{name: "kill fails, edgecore exits", svc: &fakeExtSystem{}, running: true, killErr: errors.New("kill failed"), wantKill: true},
		{name: "edgecore survives kill", svc: &fakeExtSystem{}, running: true, survivesKill: true, wantKill: true, wantErr: "waiting for edgecore to stop"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tc, killed, lingering = c, false, 2
			systemdBootPath = filepath.Join(bootDir, "missing")
			if c.bootPath != "" {
				systemdBootPath = filepath.Join(bootDir, c.bootPath)
			}
			err := TearDownEdgeCore()
			if c.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, c.wantErr)
			}
			assert.Equal(t, c.wantKill, killed)
			if c.svc != nil {
				assert.Equal(t, c.svc.hasService && !c.svc.stuck, c.svc.removed)
			}
		})
	}
}
