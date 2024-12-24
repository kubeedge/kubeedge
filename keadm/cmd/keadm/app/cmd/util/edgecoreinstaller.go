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

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2/validation"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// KubeEdgeInstTool embeds Common struct and contains cloud node ip:port information
// It implements ToolsInstaller interface
type KubeEdgeInstTool struct {
	Common
	CertPath              string
	CloudCoreIP           string
	EdgeNodeName          string
	RemoteRuntimeEndpoint string
	Token                 string
	CertPort              string
	CGroupDriver          string
	TarballPath           string
	Labels                []string
	HubProtocol           string
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

	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	if ku.EdgeNodeName != "" {
		edgeCoreConfig.Modules.Edged.HostnameOverride = ku.EdgeNodeName
	}
	if ku.CGroupDriver != "" {
		switch ku.CGroupDriver {
		case v1alpha2.CGroupDriverSystemd:
			edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverSystemd
		case v1alpha2.CGroupDriverCGroupFS:
			edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverCGroupFS
		default:
			return fmt.Errorf("unsupported CGroupDriver: %s", ku.CGroupDriver)
		}
	}

	if ku.RemoteRuntimeEndpoint != "" {
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint = ku.RemoteRuntimeEndpoint
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ImageServiceEndpoint = ku.RemoteRuntimeEndpoint
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

	switch ku.HubProtocol {
	case api.ProtocolTypeQuic:
		edgeCoreConfig.Modules.EdgeHub.Quic.Enable = true
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Enable = false
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = ku.CloudCoreIP
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = net.JoinHostPort(cloudCoreIP, strconv.Itoa(constants.DefaultWebSocketPort))
	case api.ProtocolTypeWS:
		edgeCoreConfig.Modules.EdgeHub.Quic.Enable = false
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Enable = true
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = net.JoinHostPort(cloudCoreIP, strconv.Itoa(constants.DefaultQuicPort))
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = ku.CloudCoreIP
	default:
		return fmt.Errorf("unsupported hub of protocol: %s", ku.HubProtocol)
	}
	edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(cloudCoreIP, strconv.Itoa(constants.DefaultTunnelPort))

	if len(ku.Labels) >= 1 {
		labelsMap := make(map[string]string)
		for _, label := range ku.Labels {
			key := strings.Split(label, "=")[0]
			value := strings.Split(label, "=")[1]
			labelsMap[key] = value
		}
		edgeCoreConfig.Modules.Edged.NodeLabels = labelsMap
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
	return ku.KillKubeEdgeBinary(KubeEdgeBinaryName)
}
