/*
Copyright 2019 The KubeEdge Authors.

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

package util

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1/validation"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// KubeEdgeInstTool embeds Common struct and contains cloud node ip:port information
// It implements ToolsInstaller interface
type KubeEdgeInstTool struct {
	Common
	CertPath              string
	CloudCoreIP           string
	EdgeNodeName          string
	RuntimeType           string
	RemoteRuntimeEndpoint string
	Token                 string
	CertPort              string
	CGroupDriver          string
	TarballPath           string
	Labels                []string
}

// InstallTools downloads KubeEdge for the specified version
// and makes the required configuration changes and initiates edgecore.
func (ku *KubeEdgeInstTool) InstallTools() error {
	ku.SetOSInterface(GetOSInterface())

	edgeCoreRunning, err := ku.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		return err
	}
	if edgeCoreRunning {
		return fmt.Errorf("EdgeCore is already running on this node, please run reset to clean up first")
	}

	ku.SetKubeEdgeVersion(ku.ToolVersion)

	opts := &types.InstallOptions{
		TarballPath:   ku.TarballPath,
		ComponentType: types.EdgeCore,
	}

	err = ku.InstallKubeEdge(*opts)
	if err != nil {
		return err
	}

	err = ku.createEdgeConfigFiles()
	if err != nil {
		return err
	}

	err = ku.RunEdgeCore()
	if err != nil {
		return err
	}
	return nil
}

func (ku *KubeEdgeInstTool) createEdgeConfigFiles() error {
	//This makes sure the path is created, if it already exists also it is fine
	err := os.MkdirAll(KubeEdgeConfigDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeConfigDir)
	}

	edgeCoreConfig := v1alpha1.NewDefaultEdgeCoreConfig()
	edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = ku.CloudCoreIP

	if ku.EdgeNodeName != "" {
		edgeCoreConfig.Modules.Edged.HostnameOverride = ku.EdgeNodeName
	}
	if ku.RuntimeType != "" {
		edgeCoreConfig.Modules.Edged.RuntimeType = ku.RuntimeType
	}
	if ku.CGroupDriver != "" {
		switch ku.CGroupDriver {
		case v1alpha1.CGroupDriverSystemd:
			edgeCoreConfig.Modules.Edged.CGroupDriver = v1alpha1.CGroupDriverSystemd
		case v1alpha1.CGroupDriverCGroupFS:
			edgeCoreConfig.Modules.Edged.CGroupDriver = v1alpha1.CGroupDriverCGroupFS
		default:
			return fmt.Errorf("unsupported CGroupDriver: %s", ku.CGroupDriver)
		}
	}

	if ku.RemoteRuntimeEndpoint != "" {
		edgeCoreConfig.Modules.Edged.RemoteRuntimeEndpoint = ku.RemoteRuntimeEndpoint
		edgeCoreConfig.Modules.Edged.RemoteImageEndpoint = ku.RemoteRuntimeEndpoint
	}
	if ku.Token != "" {
		edgeCoreConfig.Modules.EdgeHub.Token = ku.Token
	}
	cloudCoreIP := strings.Split(ku.CloudCoreIP, ":")[0]
	if ku.CertPort != "" {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + cloudCoreIP + ":" + ku.CertPort
	} else {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + cloudCoreIP + ":10002"
	}
	edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(cloudCoreIP, strconv.Itoa(constants.DefaultTunnelPort))

	if len(ku.Labels) >= 1 {
		labelsMap := make(map[string]string)
		for _, label := range ku.Labels {
			key := strings.Split(label, "=")[0]
			value := strings.Split(label, "=")[1]
			labelsMap[key] = value
		}
		edgeCoreConfig.Modules.Edged.Labels = labelsMap
	}

	if errs := validation.ValidateEdgeCoreConfiguration(edgeCoreConfig); len(errs) > 0 {
		return errors.New(util.SpliceErrors(errs.ToAggregate().Errors()))
	}
	return types.Write2File(KubeEdgeEdgeCoreNewYaml, edgeCoreConfig)
}

// TearDown method will remove the edge node from api-server and stop edgecore process
func (ku *KubeEdgeInstTool) TearDown() error {
	ku.SetOSInterface(GetOSInterface())
	ku.SetKubeEdgeVersion(ku.ToolVersion)

	//Kill edge core process
	if err := ku.KillKubeEdgeBinary(KubeEdgeBinaryName); err != nil {
		return err
	}

	return nil
}
