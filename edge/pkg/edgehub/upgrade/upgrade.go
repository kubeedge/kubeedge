package upgrade

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/image"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func init() {
	handler := &upgradeHandler{}
	msghandler.RegisterHandler(handler)
}

type upgradeHandler struct {
}

func (uh *upgradeHandler) Filter(message *model.Message) bool {
	name := message.GetGroup()
	return name == modules.UpgradeControllerModuleGroup
}

func (uh *upgradeHandler) Process(message *model.Message, clientHub clients.Adapter) error {
	upgradeReq := &commontypes.UpgradeRequest{}
	data, err := message.GetContentData()
	if err != nil {
		return fmt.Errorf("failed to get content data: %v", err)
	}
	err = json.Unmarshal(data, upgradeReq)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %v", err)
	}

	// get edgecore start options and config
	opts := options.GetEdgeCoreOptions()
	config := options.GetEdgeCoreConfig()

	// If UpgradeInstaller is customized, use it.
	// or use the default way: install the requested installer keadm from docker image
	if upgradeReq.UpgradeInstallerCmd != "" {
		cmd := exec.Command("bash", "-c", upgradeReq.UpgradeInstallerCmd)
		s, err := cmd.CombinedOutput()
		if err != nil {
			klog.Errorf("run install upgrader command %s failed: %v, %s", upgradeReq.UpgradeInstallerCmd, err, s)
			return fmt.Errorf("run install upgrader command %s failed: %v, %s", upgradeReq.UpgradeInstallerCmd, err, s)
		}
	} else {
		container, err := util.NewContainerRuntime(config.Modules.Edged.RuntimeType, config.Modules.Edged.RemoteRuntimeEndpoint)
		if err != nil {
			return fmt.Errorf("failed to new container runtime: %v", err)
		}
		image := image.EdgeSet("kubeedge", upgradeReq.Version).Get(image.EdgeCore)
		err = container.PullImages([]string{image})
		if err != nil {
			return fmt.Errorf("pull image failed: %v", err)
		}
		files := map[string]string{
			filepath.Join(util.KubeEdgeUsrBinPath, util.KeadmBinaryName): filepath.Join(util.KubeEdgeTmpPath, "bin", util.KeadmBinaryName),
		}
		err = container.CopyResources(image, nil, files)
		if err != nil {
			return fmt.Errorf("failed to cp file from image to host: %v", err)
		}
	}

	var upgradeCmd string
	// If upgradeCmd is customized, use it, or use the default upgrade cmd
	if upgradeReq.UpgradeCmd != "" {
		upgradeCmd = upgradeReq.UpgradeCmd
	} else {
		upgradeCmd = fmt.Sprintf("keadm upgrade --upgradeID %s --fromVersion %s --toVersion %s --config %s > /var/log/kubeedge/keadm.log 2>&1",
			upgradeReq.UpgradeID, version.Get(), upgradeReq.Version, opts.ConfigFile)
	}

	// run upgrade cmd to upgrade edge node
	// use setsid command and nohup command to start a separate progress
	command := fmt.Sprintf("setsid nohup %s &", upgradeCmd)
	cmd := exec.Command("bash", "-c", command)
	s, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("run upgrade command %s failed: %v, %s", command, err, s)
		return fmt.Errorf("run upgrade command %s failed: %v, %s", command, err, s)
	}

	klog.Infof("!!! Begin to upgrade...")
	return nil
}
