package taskexecutor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/common/constants"
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

	require.Equal(t, filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName), cmd.Path)
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
	require.Contains(t, err.Error(), "must not be a symlink")
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

func TestPrepareUpgradeLogFileCreatesProtectedDirectoryAndFile(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "var", "log", "kubeedge")
	logPath := filepath.Join(logDir, keadmUpgradeLogName)

	err := prepareUpgradeLogFile(logPath)
	require.NoError(t, err)

	dirInfo, err := os.Stat(logDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o750), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(logPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestNewSystemdRunUpgradeCommandBuildsTransientUnitInvocation(t *testing.T) {
	req := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade id/42",
		HistoryID: "history-id",
		Version:   "v1.17.0",
		Image:     "example.com/installation-package:v1.17.0",
	}
	opts := &options.EdgeCoreOptions{ConfigFile: "/etc/kubeedge/edgecore.yaml"}
	logPath := "/var/log/kubeedge/keadm-upgrade.log"

	executablePath, err := os.Executable()
	require.NoError(t, err)
	binDir := t.TempDir()
	systemdRunPath := filepath.Join(binDir, "systemd-run")
	require.NoError(t, os.Symlink(executablePath, systemdRunPath))
	t.Setenv("PATH", binDir)

	cmd, err := newSystemdRunUpgradeCommand(req, opts, logPath)
	require.NoError(t, err)
	require.Equal(t, systemdRunPath, cmd.Path)
	require.Contains(t, cmd.Args, "--collect")
	require.Contains(t, cmd.Args, "--service-type=exec")
	require.Contains(t, cmd.Args, "StandardOutput=append:"+logPath)
	require.Contains(t, cmd.Args, "StandardError=append:"+logPath)
	require.Contains(t, cmd.Args, filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName))
	require.Contains(t, cmd.Args, "--upgradeID")
	require.Contains(t, cmd.Args, req.UpgradeID)
	require.Contains(t, cmd.Args, "--image")
	require.Contains(t, cmd.Args, req.Image)
	require.Contains(t, cmd.Args, "kubeedge-keadm-upgrade-upgrade-id-42")
}
