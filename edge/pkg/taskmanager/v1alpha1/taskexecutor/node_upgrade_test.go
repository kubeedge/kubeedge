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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func TestBuildKeadmUpgradeArgsTreatsShellMetacharactersAsLiteralArgs(t *testing.T) {
	req := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade;touch /tmp/pwned",
		HistoryID: "history && whoami",
		Version:   "v1.17.0 $(uname -a)",
		Image:     "example.com/keadm:latest;echo owned",
	}
	opts := &options.EdgeCoreOptions{ConfigFile: "/etc/kubeedge/edgecore.yaml"}

	args := buildKeadmUpgradeArgs(req, opts)

	require.Equal(t, []string{
		"upgrade",
		"edge",
		"--upgradeID", req.UpgradeID,
		"--historyID", req.HistoryID,
		"--fromVersion", version.Get().String(),
		"--toVersion", req.Version,
		"--config", opts.ConfigFile,
		"--image", req.Image,
	}, args)
}

func TestNewKeadmUpgradeCommandBuildsDirectExecCommand(t *testing.T) {
	req := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-id",
		HistoryID: "history-id",
		Version:   "v1.17.0",
		Image:     "example.com/installation-package:v1.17.0",
	}
	opts := &options.EdgeCoreOptions{ConfigFile: "/etc/kubeedge/edgecore.yaml"}
	logPath := filepath.Join(t.TempDir(), "keadm.log")

	cmd, logFile, err := newKeadmUpgradeCommand(req, opts, logPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = logFile.Close()
	})

	require.Equal(t, keadmExecutablePath, cmd.Path)
	require.Equal(t, append([]string{cmd.Path}, buildKeadmUpgradeArgs(req, opts)...), cmd.Args)
	require.Same(t, logFile, cmd.Stdout)
	require.Same(t, logFile, cmd.Stderr)

	info, err := os.Stat(logPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestOpenUpgradeLogFileRejectsSymlink(t *testing.T) {
	logDir := t.TempDir()
	target := filepath.Join(logDir, "target.log")
	logPath := filepath.Join(logDir, "keadm-upgrade.log")

	require.NoError(t, os.WriteFile(target, []byte("target"), 0o600))
	require.NoError(t, os.Symlink(target, logPath))

	_, err := openUpgradeLogFile(logPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many levels of symbolic links")
}

func TestOpenUpgradeLogFileTruncatesExistingContent(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "keadm-upgrade.log")
	require.NoError(t, os.WriteFile(logPath, []byte("stale-output"), 0o644))

	logFile, err := openUpgradeLogFile(logPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = logFile.Close()
	})

	_, err = logFile.WriteString("fresh-output")
	require.NoError(t, err)
	require.NoError(t, logFile.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Equal(t, "fresh-output", string(data))

	info, err := os.Stat(logPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestOpenUpgradeLogFileCreatesProtectedDirectoryAndFile(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "var", "log", "kubeedge")
	logPath := filepath.Join(logDir, keadmUpgradeLogName)

	logFile, err := openUpgradeLogFile(logPath)
	require.NoError(t, err)
	require.NoError(t, logFile.Close())

	dirInfo, err := os.Stat(logDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o750), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(logPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestNewSystemdRunUpgradeCommandPreservesLiteralArgs(t *testing.T) {
	req := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade ${KUBEEDGE_TEST_VALUE}",
		HistoryID: "history ${KUBEEDGE_TEST_VALUE}",
		Version:   "v1.17.0 ${KUBEEDGE_TEST_VALUE}",
		Image:     "example.com/package:${KUBEEDGE_TEST_VALUE}",
	}
	opts := &options.EdgeCoreOptions{ConfigFile: "/etc/kubeedge/${KUBEEDGE_TEST_VALUE}.yaml"}

	binDir := t.TempDir()
	capturedArgsPath := filepath.Join(t.TempDir(), "argv.txt")
	helperPath := filepath.Join(binDir, "keadm-helper.sh")
	systemdRunPath := filepath.Join(binDir, "systemd-run")

	require.NoError(t, os.WriteFile(helperPath, []byte(fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$@" > %q
`, capturedArgsPath)), 0o755))
	require.NoError(t, os.WriteFile(systemdRunPath, []byte(`#!/bin/sh
if [ "$1" = "--help" ]; then
	printf '%s\n' '--setenv'
	exit 0
fi
while [ "$#" -gt 0 ]; do
	case "$1" in
		--unit|--description)
			shift 2
			;;
		--setenv=*)
			export "${1#--setenv=}"
			shift
			;;
		*)
			break
			;;
	esac
done
exec "$@"
`), 0o755))

	originalKeadmPath := keadmExecutablePath
	keadmExecutablePath = helperPath
	t.Cleanup(func() {
		keadmExecutablePath = originalKeadmPath
	})

	cmd, err := newSystemdRunUpgradeCommand(systemdRunPath, req, opts)
	require.NoError(t, err)
	require.NoError(t, cmd.Run())

	data, err := os.ReadFile(capturedArgsPath)
	require.NoError(t, err)
	require.Equal(t, strings.Join(buildKeadmUpgradeArgs(req, opts), "\n")+"\n", string(data))
}

func TestBuildKeadmUpgradeUnitNameEmptyIDUsesPrefix(t *testing.T) {
	require.Equal(t, keadmUpgradeUnitPrefix, buildKeadmUpgradeUnitName(""))
}

func TestBuildKeadmUpgradeUnitNameBoundsLengthAndHashes(t *testing.T) {
	longID := strings.Repeat("very-long-upgrade-id-", 32)
	unitName := buildKeadmUpgradeUnitName(longID)

	require.LessOrEqual(t, len(unitName), keadmUpgradeUnitMaxNameLen)
	require.True(t, strings.HasPrefix(unitName, keadmUpgradeUnitPrefix+"-"))
}

func TestBuildKeadmUpgradeUnitNameKeepsSamePrefixIDsDistinct(t *testing.T) {
	id1 := strings.Repeat("same-prefix-", 20) + "one"
	id2 := strings.Repeat("same-prefix-", 20) + "two"

	require.NotEqual(t, buildKeadmUpgradeUnitName(id1), buildKeadmUpgradeUnitName(id2))
}
