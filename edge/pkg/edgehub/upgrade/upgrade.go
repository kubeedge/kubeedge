/*
Copyright 2022 The KubeEdge Authors.

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

package upgrade

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func init() {
	handler := &upgradeHandler{}
	msghandler.RegisterHandler(handler)

	// register upgrade tool: keadm
	// if not specify it, also use default upgrade tool: keadm
	RegisterUpgradeProvider("", &keadmUpgrade{})
	RegisterUpgradeProvider("keadm", &keadmUpgrade{})
}

type upgradeHandler struct{}

func (uh *upgradeHandler) Filter(message *model.Message) bool {
	name := message.GetGroup()
	return name == modules.NodeUpgradeJobControllerModuleGroup
}

func (uh *upgradeHandler) Process(message *model.Message, clientHub clients.Adapter) error {
	upgradeReq := &commontypes.NodeUpgradeJobRequest{}
	data, err := message.GetContentData()
	if err != nil {
		return fmt.Errorf("failed to get content data: %v", err)
	}
	err = json.Unmarshal(data, upgradeReq)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %v", err)
	}

	err = validateUpgrade(upgradeReq)
	if err != nil {
		return fmt.Errorf("upgrade request is not valid: %v", err)
	}

	tool := strings.ToLower(upgradeReq.UpgradeTool)
	if _, ok := upgradeToolProviders[tool]; !ok {
		return fmt.Errorf("not supported upgrade tool type: %v", upgradeReq.UpgradeTool)
	}

	return upgradeToolProviders[tool].Upgrade(upgradeReq)
}

func validateUpgrade(upgrade *commontypes.NodeUpgradeJobRequest) error {
	if upgrade.UpgradeID == "" {
		return fmt.Errorf("upgradeID cannot be empty")
	}
	if upgrade.Version == version.Get().String() {
		return fmt.Errorf("edge node already on specific version, no need to upgrade")
	}

	return nil
}

type Provider interface {
	Upgrade(upgrade *commontypes.NodeUpgradeJobRequest) error
}

var (
	upgradeToolProviders = make(map[string]Provider)
	mutex                sync.Mutex
)

func RegisterUpgradeProvider(upgradeToolType string, provider Provider) {
	mutex.Lock()
	defer mutex.Unlock()
	upgradeToolProviders[upgradeToolType] = provider
}

type keadmUpgrade struct{}

func (*keadmUpgrade) Upgrade(upgradeReq *commontypes.NodeUpgradeJobRequest) error {
	// get edgecore start options and config
	opts := options.GetEdgeCoreOptions()
	config := options.GetEdgeCoreConfig()

	// install the requested installer keadm from docker image
	klog.Infof("Begin to download version %s keadm", upgradeReq.Version)
	container, err := util.NewContainerRuntime(config.Modules.Edged.ContainerRuntime, config.Modules.Edged.RemoteRuntimeEndpoint)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	image := upgradeReq.Image

	// TODO: do some verification 1.sha256(pass in using CRD) 2.image signature verification
	// TODO: release verification mechanism
	err = container.PullImages([]string{image})
	if err != nil {
		return fmt.Errorf("pull image failed: %v", err)
	}
	files := map[string]string{
		filepath.Join(util.KubeEdgeUsrBinPath, util.KeadmBinaryName): filepath.Join(util.KubeEdgeUsrBinPath, util.KeadmBinaryName),
	}
	err = container.CopyResources(image, files)
	if err != nil {
		return fmt.Errorf("failed to cp file from image to host: %v", err)
	}

	klog.Infof("Begin to run upgrade command")
	upgradeCmd := fmt.Sprintf("keadm upgrade --upgradeID %s --historyID %s --fromVersion %s --toVersion %s --config %s --image %s > /tmp/keadm.log 2>&1",
		upgradeReq.UpgradeID, upgradeReq.HistoryID, version.Get(), upgradeReq.Version, opts.ConfigFile, image)

	// run upgrade cmd to upgrade edge node
	// use setsid command and nohup command to start a separate progress
	command := fmt.Sprintf("setsid nohup %s &", upgradeCmd)
	cmd := exec.Command("bash", "-c", command)
	s, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("run upgrade command %s failed: %v, %s", command, err, s)
		return fmt.Errorf("run upgrade command %s failed: %v, %s", command, err, s)
	}

	klog.Infof("!!! Begin to upgrade from Version %s to %s ...", version.Get(), upgradeReq.Version)

	return nil
}
