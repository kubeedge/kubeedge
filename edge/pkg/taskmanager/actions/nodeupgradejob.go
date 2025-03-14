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

package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	klog "k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	daov2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/containers"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	upgradeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

func newNodeUpgradeJobRunner() *ActionRunner {
	logger := klog.Background().WithName("node-upgrade-job-runner")
	config := options.GetEdgeCoreConfig()
	handler := nodeUpgradeJobActionHandler{
		logger: logger,
		backupFiles: []string{
			config.DataBase.DataSource,
			constants.DefaultConfigDir + "edgecore.yaml",
			filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
		},
	}
	runner := &ActionRunner{
		Flow:               actionflow.FlowNodeUpgradeJob,
		ReportActionStatus: handler.reportActionStatus,
		GetSpecSerializer:  handler.getSpecSerializer,
		PreRun:             handler.preRun,
		PostRun:            handler.postRun,
		Logger:             logger,
	}
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionCheck), handler.checkItems)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionWaitingConfirmation), handler.waitingConfirmation)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionBackUp), handler.backup)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionUpgrade), handler.upgrade)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionRollBack), handler.rollback)
	return runner
}

type nodeUpgradeJobActionResponse struct {
	FromVersion string
	ToVersion   string

	baseActionResponse
}

// nodeUpgradeJobActionHandler defines action-related functions
type nodeUpgradeJobActionHandler struct {
	backupFiles []string
	logger      logr.Logger
}

func (nodeUpgradeJobActionHandler) preRun(
	_ctx context.Context,
	jobname, nodename, _action string,
	specser SpecSerializer,
) error {
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		return fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
	}
	upgradeDao := daov2.NewUpgrade()
	if err := upgradeDao.Save(jobname, nodename, spec); err != nil {
		return err
	}
	return nil
}

func (nodeUpgradeJobActionHandler) postRun(
	_ctx context.Context,
	_jobname, _nodename, _action string,
	_specser SpecSerializer,
) error {
	// Since the keadm upgrade / rollback command is asynchronous, so this method can not be executed now.
	// When obtaining the task report in the taskmanager module, it is also necessary to determine whether
	// to delete the record in the meta_v2 table(hub_connected_hooker.go).
	upgradeDao := daov2.NewUpgrade()
	if err := upgradeDao.Delete(); err != nil {
		return err
	}
	return nil
}

func (nodeUpgradeJobActionHandler) checkItems(
	ctx context.Context,
	_jobname, _nodename string,
	specser SpecSerializer,
) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}

	// Check cpu/memory/disk.
	if len(spec.CheckItems) > 0 {
		if err := PreCheck(spec.CheckItems); err != nil {
			resp.err = err
			return resp
		}
	}

	// Pull installation-package image.
	cfg := options.GetEdgeCoreConfig()
	ctrcli, err := containers.NewContainerRuntime(
		cfg.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		cfg.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		resp.err = fmt.Errorf("failed to new container runtime, err: %v", err)
		return resp
	}
	image := spec.Image + ":" + spec.Version
	if err := ctrcli.PullImage(ctx, image, nil, nil); err != nil {
		resp.err = fmt.Errorf("failed to pull image %s, err: %v", image, err)
		return resp
	}

	// If the ImageDigestGetter is not empty, verify the image digest.
	if getter := spec.ImageDigestGetter; getter != nil {
		var expectedDigest string
		switch {
		case runtime.GOARCH == "arm64" && getter.ARM64 != "":
			expectedDigest = getter.ARM64
		case runtime.GOARCH == "amd64" && getter.AMD64 != "":
			expectedDigest = getter.AMD64
		default:
			resp.err = fmt.Errorf("unsupport the arch %s to verify the image digest", runtime.GOARCH)
			return resp
		}
		local, err := ctrcli.GetImageDigest(ctx, image)
		if err != nil {
			resp.err = fmt.Errorf("failed to get image digest of %s, err: %v", image, err)
		}
		if local != expectedDigest {
			resp.err = fmt.Errorf("image digest of %s is not correct, local: %s, expected: %s",
				image, local, expectedDigest)
			return resp
		}
	}

	// Copy new keadm bainnary from the image to /usr/local/bin.
	containerPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	hostPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KeadmBinaryName)
	files := map[string]string{containerPath: hostPath}
	if err := ctrcli.CopyResources(ctx, image, files); err != nil {
		resp.err = fmt.Errorf("failed to copy keadm from %s in the image %s to the host path %s, err: %v",
			containerPath, image, hostPath, err)
		return resp
	}
	return resp
}

func (nodeUpgradeJobActionHandler) waitingConfirmation(
	_ctx context.Context,
	jobname, nodename string,
	specser SpecSerializer,
) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	// If confirmation is required, return false to block the action flow.
	resp.interrupt = spec.RequireConfirmation
	return resp
}

func (h *nodeUpgradeJobActionHandler) backup(
	_ctx context.Context,
	_jobname, _nodename string,
	_specser SpecSerializer,
) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	cmdline := "keadm backup edge"
	cmd := execs.NewCommand(cmdline)
	h.logger.V(2).Info("run backup cmd", "cmd", cmdline)
	if err := cmd.Exec(); err != nil {
		resp.err = err
		return resp
	}

	// The backup command is synchronous, and the execution result file can be obtained after the command ends.
	info, err := upgradeedge.ParseJSONReporterInfo()
	if err != nil {
		resp.err = err
		return resp
	}
	if err := upgradeedge.RemoveJSONReporterInfo(); err != nil {
		klog.Errorf("failed to remove upgrade report file, err: %v", err)
	}
	resp.FromVersion = info.FromVersion
	resp.ToVersion = info.ToVersion
	if !info.Success {
		resp.err = fmt.Errorf("keadm backup failed, err: %v", info.ErrorMessage)
	}

	return resp
}

func (h *nodeUpgradeJobActionHandler) upgrade(
	_ctx context.Context,
	_jobname, _nodename string,
	specser SpecSerializer,
) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		resp.interrupt = true // No upgrade yet, no need to roll back.
		return resp
	}
	var cmdline strings.Builder
	cmdline.WriteString("keadm upgrade edge --force --toVersion " + spec.Version)
	if spec.Image != "" {
		cmdline.WriteString(" --image " + spec.Image)
	}
	cmdline.WriteString(" >> /tmp/keadm.log 2>&1")
	cmd := execs.NewCommand(cmdline.String())
	h.logger.V(2).Info("run upgrade cmd", "cmd", cmdline.String())
	resp.err = cmd.Exec()
	return resp
}

func (h *nodeUpgradeJobActionHandler) rollback(
	_ctx context.Context,
	_jobname, _nodename string,
	specser SpecSerializer,
) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	// Roll back to the previous version
	cmdline := "keadm rollback edge >> /tmp/keadm.log 2>&1"
	cmd := execs.NewCommand(cmdline)
	h.logger.V(2).Info("run rollback cmd", "cmd", cmdline)
	resp.err = cmd.Exec()
	return resp
}

func (nodeUpgradeJobActionHandler) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.NodeUpgradeJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (nodeUpgradeJobActionHandler) reportActionStatus(jobname, nodename, action string, resp ActionResponse) {
	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	var extend string
	if resp, ok := resp.(*nodeUpgradeJobActionResponse); ok &&
		action == string(operationsv1alpha2.NodeUpgradeJobActionBackUp) ||
		action == string(operationsv1alpha2.NodeUpgradeJobActionUpgrade) ||
		action == string(operationsv1alpha2.NodeUpgradeJobActionRollBack) {
		extend = taskmsg.FormatNodeUpgradeJobExtend(resp.FromVersion, resp.ToVersion)
	}
	body := taskmsg.UpstreamMessage{
		Action:     action,
		FinishTime: time.Now().UTC().Format(time.RFC3339),
		Extend:     extend,
	}
	if err := resp.Error(); err != nil {
		body.Succ = false
		body.Reason = err.Error()
	} else {
		body.Succ = true
	}
	message.ReportNodeTaskStatus(res, body)
}
