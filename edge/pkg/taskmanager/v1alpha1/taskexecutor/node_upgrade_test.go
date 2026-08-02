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

package taskexecutor

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/pkg/nodetask/tasklog"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func TestBuildKeadmUpgradeArgsDoesNotUseShell(t *testing.T) {
	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-1; touch /tmp/pwned",
		HistoryID: "history-1$(touch /tmp/pwned)",
		Version:   "v1.23.1; rm -rf /",
		Image:     "kubeedge/installation-package; touch /tmp/pwned",
	}
	opts := &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml; touch /tmp/pwned",
	}

	args := buildKeadmUpgradeArgs(upgradeReq, opts)

	want := []string{
		"upgrade", "edge",
		"--upgradeID", upgradeReq.UpgradeID,
		"--historyID", upgradeReq.HistoryID,
		"--fromVersion", version.Get().String(),
		"--toVersion", upgradeReq.Version,
		"--config", opts.ConfigFile,
		"--image", upgradeReq.Image,
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got %v, want %v", args, want)
	}
}

func TestKeadmUpgradeReturnsErrorWhenCommandMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	logDir := t.TempDir()
	oldOpenLog := openKeadmTaskLog
	t.Cleanup(func() {
		openKeadmTaskLog = oldOpenLog
	})
	openKeadmTaskLog = func(name string, flag int) (*os.File, error) {
		return tasklog.OpenKeadmLogAt(logDir, name, flag)
	}

	err := keadmUpgrade(commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-1",
		HistoryID: "history-1",
		Version:   "v1.23.1",
		Image:     "kubeedge/installation-package:v1.23.1",
	}, &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
	})

	if err == nil {
		t.Fatal("expected error when keadm command cannot be started")
	}
	if !strings.Contains(err.Error(), "failed to start keadm upgrade command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV1alpha1KeadmLogName(t *testing.T) {
	got := v1alpha1KeadmLogName("upgrade", "upgrade 1", "history/1")
	want := "keadm-upgrade-upgrade_1-history_1.log"
	if got != want {
		t.Fatalf("v1alpha1KeadmLogName() = %q, want %q", got, want)
	}
}

func TestKeadmUpgradeWritesPrivateTaskLog(t *testing.T) {
	binDir := t.TempDir()
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "nohup"), []byte("#!/bin/sh\n\"$@\"\n"), 0755); err != nil {
		t.Fatalf("WriteFile(nohup) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "keadm"), []byte("#!/bin/sh\necho \"$@\"\n"), 0755); err != nil {
		t.Fatalf("WriteFile(keadm) error = %v", err)
	}
	t.Setenv("PATH", binDir)

	oldOpenLog := openKeadmTaskLog
	t.Cleanup(func() {
		openKeadmTaskLog = oldOpenLog
	})
	openKeadmTaskLog = func(name string, flag int) (*os.File, error) {
		return tasklog.OpenKeadmLogAt(logDir, name, flag)
	}

	err := keadmUpgrade(commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-1",
		HistoryID: "history-1",
		Version:   "v1.23.1",
		Image:     "kubeedge/installation-package:v1.23.1",
	}, &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
	})
	if err != nil {
		t.Fatalf("keadmUpgrade() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(logDir, "keadm-upgrade-upgrade-1-history-1.log"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("log mode = %o, want 0600", got)
	}
}
