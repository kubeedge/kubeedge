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
	"os/exec"
	"path/filepath"

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
	TaskUpgrade = "upgrade"
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
	upgradeCmd := fmt.Sprintf("keadm upgrade edge --upgradeID %s --historyID %s --fromVersion %s --toVersion %s --config %s --image %s > /tmp/keadm.log 2>&1",
		upgradeReq.UpgradeID, upgradeReq.HistoryID, version.Get(), upgradeReq.Version, opts.ConfigFile, upgradeReq.Image)

	// run upgrade cmd to upgrade edge node
	// use nohup command to start a child progress
	command := fmt.Sprintf("nohup %s &", upgradeCmd)
	cmd := exec.Command("bash", "-c", command)
	s, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run upgrade command %s failed: %v, %s", command, err, s)
	}
	klog.Infof("!!! Finish upgrade from Version %s to %s ...", version.Get(), upgradeReq.Version)
	return nil
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
