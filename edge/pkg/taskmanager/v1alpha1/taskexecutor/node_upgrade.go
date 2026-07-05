/*
Copyright 2023 The KubeEdge Authors.

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/pkg/containers"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	TaskUpgrade             = "upgrade"
	keadmUpgradeLogName     = "keadm-upgrade.log"
	keadmUpgradeLogDirPerm  = 0o750
	keadmUpgradeLogFilePerm = 0o600
)

type Upgrade struct {
	*BaseExecutor
}

func (u *Upgrade) Name() string {
	return u.name
}

func NewUpgradeExecutor() Executor {
	methods := map[string]func(types.NodeTaskRequest) fsm.Event{
		string(api.TaskChecking):     preCheck,
		string(api.TaskInit):         initUpgrade,
		"":                           initUpgrade,
		string(api.BackingUpState):   backupNode,
		string(api.RollingBackState): rollbackNode,
		string(api.UpgradingState):   upgrade,
	}
	return &Upgrade{
		BaseExecutor: NewBaseExecutor(TaskUpgrade, methods),
	}
}

func initUpgrade(taskReq types.NodeTaskRequest) (event fsm.Event) {
	event = fsm.Event{
		Type:   "Init",
		Action: api.ActionSuccess,
	}
	var err error
	defer func() {
		if err != nil {
			event.Action = api.ActionFailure
			event.Msg = err.Error()
		}
	}()

	var upgradeReq *commontypes.NodeUpgradeJobRequest
	upgradeReq, err = getTaskRequest(taskReq)
	if err != nil {
		return
	}
	if upgradeReq.UpgradeID == "" {
		err = errors.New("upgradeID cannot be empty")
		return
	}

	upgradedao := dbclient.NewUpgradeV1alpha1()
	err = upgradedao.Save(&taskReq)
	if err != nil {
		return
	}
	if upgradeReq.RequireConfirmation {
		dbc := dbclient.NewMetaV2Service()
		var upgradeJobReqDB = commontypes.NodeUpgradeJobRequest{
			UpgradeID:           upgradeReq.UpgradeID,
			HistoryID:           upgradeReq.HistoryID,
			Version:             upgradeReq.Version,
			UpgradeTool:         upgradeReq.UpgradeTool,
			Image:               upgradeReq.Image,
			ImageDigest:         upgradeReq.ImageDigest,
			RequireConfirmation: upgradeReq.RequireConfirmation,
		}
		if err = dbc.SaveNodeUpgradeJobRequestToMetaV2(upgradeJobReqDB); err != nil {
			event.Action = api.ActionFailure
			event.Msg = err.Error()
		}
		e, _ := GetExecutor(TaskUpgrade)
		var taskReqDB = types.NodeTaskRequest{
			TaskID: e.Name(),
			Type:   "Confirm",
			State:  string(api.NodeUpgrading),
			Item:   "Wait for a confirm for upgrade request on the edge site.",
		}
		if err = dbc.SaveNodeTaskRequestToMetaV2(taskReqDB); err != nil {
			event.Action = api.ActionFailure
			event.Msg = err.Error()
		}
		return fsm.Event{
			Type:   "Confirm",
			Action: api.ActionConfirmation,
			Msg:    "Wait for a confirm for upgrade request on the edge site.",
		}
	}
	if upgradeReq.Version == version.Get().String() {
		return fsm.Event{
			Type:   "Upgrading",
			Action: api.ActionSuccess,
		}
	}

	if upgradeReq.RequireConfirmation {
		event.Type = "Confirm"
		event.Action = api.ActionConfirmation
		event.Msg = "Wait for a confirm for upgrade request on the edge site."
		return
	}

	if upgradeReq.Version == version.Get().String() {
		event.Type = "Upgrading"
		event.Action = api.ActionSuccess
		return
	}
	err = prepareKeadm(upgradeReq)
	if err != nil {
		return
	}
	return event
}

func getTaskRequest(taskReq commontypes.NodeTaskRequest) (*commontypes.NodeUpgradeJobRequest, error) {
	data, err := json.Marshal(taskReq.Item)
	if err != nil {
		return nil, err
	}
	var upgradeReq commontypes.NodeUpgradeJobRequest
	err = json.Unmarshal(data, &upgradeReq)
	if err != nil {
		return nil, err
	}
	return &upgradeReq, err
}

func upgrade(taskReq types.NodeTaskRequest) (event fsm.Event) {
	// The NodeTaskRequest of v1alpha1 upgrade node job only needs data when confirming.
	// The edgecore process will be interrupted when keadm executes the upgrade command,
	// so the data needs to be cleaned up in advance.
	upgradedao := dbclient.NewUpgradeV1alpha1()
	if err := upgradedao.Delete(); err != nil {
		return
	}

	opts := options.GetEdgeCoreOptions()
	event = fsm.Event{
		Type: "Upgrade",
	}
	upgradeReq, err := getTaskRequest(taskReq)
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = err.Error()
		return
	}
	err = keadmUpgrade(*upgradeReq, opts)
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = err.Error()
	}
	return
}

func keadmUpgrade(upgradeReq commontypes.NodeUpgradeJobRequest, opts *options.EdgeCoreOptions) error {
	klog.Infof("Begin to run upgrade command")
	launcher, err := newKeadmUpgradeLauncher(upgradeReq, opts)
	if err != nil {
		return err
	}
	if launcher.systemdManaged {
		output, err := launcher.cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("start systemd upgrade command %v failed: %w, %s", launcher.cmd.Args, err, output)
		}
		klog.Infof("Started transient systemd unit for upgrade command %v", launcher.cmd.Args)
	} else {
		if err := launcher.cmd.Start(); err != nil {
			if cerr := launcher.logFile.Close(); cerr != nil {
				klog.Warningf("failed to close upgrade log file %s: %v", launcher.logFile.Name(), cerr)
			}
			return fmt.Errorf("start upgrade command %v failed: %w", launcher.cmd.Args, err)
		}
		go waitForUpgradeCommand(launcher.cmd, launcher.logFile)
	}
	klog.Infof("!!! Finish upgrade from Version %s to %s ...", version.Get(), upgradeReq.Version)
	return nil
}

type keadmUpgradeLauncher struct {
	cmd            *exec.Cmd
	logFile        *os.File
	systemdManaged bool
}

func waitForUpgradeCommand(cmd *exec.Cmd, logFile *os.File) {
	defer func() {
		if cerr := logFile.Close(); cerr != nil {
			klog.Warningf("failed to close upgrade log file %s: %v", logFile.Name(), cerr)
		}
	}()
	if err := cmd.Wait(); err != nil {
		klog.Errorf("upgrade command %v failed: %v", cmd.Args, err)
	}
}

func newKeadmUpgradeLauncher(upgradeReq commontypes.NodeUpgradeJobRequest, opts *options.EdgeCoreOptions) (*keadmUpgradeLauncher, error) {
	logPath := keadmUpgradeLogPath()
	if canUseSystemdRun() {
		if err := prepareUpgradeLogFile(logPath); err != nil {
			return nil, err
		}
		cmd, err := newSystemdRunUpgradeCommand(upgradeReq, opts, logPath)
		if err != nil {
			return nil, err
		}
		return &keadmUpgradeLauncher{
			cmd:            cmd,
			systemdManaged: true,
		}, nil
	}

	cmd, logFile, err := newKeadmUpgradeCommand(upgradeReq, opts, logPath)
	if err != nil {
		return nil, err
	}
	return &keadmUpgradeLauncher{
		cmd:     cmd,
		logFile: logFile,
	}, nil
}

func newKeadmUpgradeCommand(upgradeReq commontypes.NodeUpgradeJobRequest, opts *options.EdgeCoreOptions, logPath string) (*exec.Cmd, *os.File, error) {
	logFile, err := openUpgradeLogFile(logPath)
	if err != nil {
		return nil, nil, err
	}

	keadmPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	cmd := exec.Command(keadmPath, buildKeadmUpgradeArgs(upgradeReq, opts)...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func newSystemdRunUpgradeCommand(upgradeReq commontypes.NodeUpgradeJobRequest, opts *options.EdgeCoreOptions, logPath string) (*exec.Cmd, error) {
	systemdRunPath, err := exec.LookPath("systemd-run")
	if err != nil {
		return nil, fmt.Errorf("find systemd-run failed: %w", err)
	}
	keadmPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	args := []string{
		"--unit", buildKeadmUpgradeUnitName(upgradeReq.UpgradeID),
		"--description", "KubeEdge node upgrade",
		"--collect",
		"--service-type=exec",
		"--property", "StandardOutput=append:" + logPath,
		"--property", "StandardError=append:" + logPath,
		keadmPath,
	}
	args = append(args, buildKeadmUpgradeArgs(upgradeReq, opts)...)
	return exec.Command(systemdRunPath, args...), nil
}

func buildKeadmUpgradeArgs(upgradeReq commontypes.NodeUpgradeJobRequest, opts *options.EdgeCoreOptions) []string {
	return []string{
		"upgrade",
		"edge",
		"--upgradeID", upgradeReq.UpgradeID,
		"--historyID", upgradeReq.HistoryID,
		"--fromVersion", version.Get().String(),
		"--toVersion", upgradeReq.Version,
		"--config", opts.ConfigFile,
		"--image", upgradeReq.Image,
	}
}

func keadmUpgradeLogPath() string {
	return filepath.Join(constants.KubeEdgeLogPath, keadmUpgradeLogName)
}

func canUseSystemdRun() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}
	_, err := exec.LookPath("systemd-run")
	return err == nil
}

func buildKeadmUpgradeUnitName(upgradeID string) string {
	var b strings.Builder
	b.WriteString("kubeedge-keadm-upgrade")
	if upgradeID == "" {
		return b.String()
	}
	b.WriteByte('-')
	for _, r := range upgradeID {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func prepareUpgradeLogFile(logPath string) error {
	logDir := filepath.Dir(logPath)
	if err := ensureSecureLogDir(logDir); err != nil {
		return err
	}
	logFile, err := openUpgradeLogFile(logPath)
	if err != nil {
		return err
	}
	return logFile.Close()
}

func ensureSecureLogDir(logDir string) error {
	if info, err := os.Lstat(logDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("upgrade log directory %s must not be a symlink", logDir)
		}
		if !info.IsDir() {
			return fmt.Errorf("upgrade log directory %s is not a directory", logDir)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, keadmUpgradeLogDirPerm); err != nil {
			return fmt.Errorf("create upgrade log directory %s failed: %w", logDir, err)
		}
	} else {
		return fmt.Errorf("inspect upgrade log directory %s failed: %w", logDir, err)
	}

	if err := os.Chmod(logDir, keadmUpgradeLogDirPerm); err != nil {
		return fmt.Errorf("set upgrade log directory permissions on %s failed: %w", logDir, err)
	}
	return nil
}

func openUpgradeLogFile(logPath string) (*os.File, error) {
	if info, err := os.Lstat(logPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("upgrade log file %s must not be a symlink", logPath)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("upgrade log file %s must be a regular file", logPath)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect upgrade log file %s failed: %w", logPath, err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, keadmUpgradeLogFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open upgrade log file %s failed: %w", logPath, err)
	}
	if err := logFile.Chmod(keadmUpgradeLogFilePerm); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("set upgrade log file permissions on %s failed: %w", logPath, err)
	}
	return logFile, nil
}

func prepareKeadm(upgradeReq *commontypes.NodeUpgradeJobRequest) error {
	ctx := context.Background()
	config := options.GetEdgeCoreConfig()

	// install the requested installer keadm from docker image
	klog.Infof("Begin to download version %s keadm", upgradeReq.Version)
	ctrcli, err := containers.NewContainerRuntime(config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint, config.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}
	image := upgradeReq.Image

	// TODO: do some verification 1.sha256(pass in using CRD) 2.image signature verification
	// TODO: release verification mechanism
	err = ctrcli.PullImages(ctx, []string{image}, nil)
	if err != nil {
		return fmt.Errorf("pull image failed: %v", err)
	}
	// Check installation-package image digest
	if upgradeReq.ImageDigest != "" {
		var local string
		local, err = ctrcli.GetImageDigest(ctx, image)
		if err != nil {
			return err
		}
		if upgradeReq.ImageDigest != local {
			return fmt.Errorf("invalid installation-package image digest value: %s", local)
		}
	}
	containerPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	hostPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	files := map[string]string{containerPath: hostPath}
	err = ctrcli.CopyResources(ctx, image, files)
	if err != nil {
		return fmt.Errorf("failed to cp file from image to host: %v", err)
	}
	return nil
}
