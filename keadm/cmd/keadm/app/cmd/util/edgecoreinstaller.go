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

	v1 "k8s.io/api/core/v1"

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
	EdgeNodeIP            string
	EdgedClusterDNS       string
	EdgedClusterDomain    string
	RuntimeType           string
	RemoteRuntimeEndpoint string
	Token                 string
	CertPort              string
	QuicPort              string
	TunnelPort            string
	CGroupDriver          string
	TarballPath           string
	Labels                []string
	EnableDefaultTaint    bool
	EnableMetaServer      bool
	EnableServiceBus      bool
	DownloadRegion        string
	KubeEdgeDownloadURL   string
	ServiceFileURLFormat  string
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

	if ku.DownloadRegion == "zh" && ku.KubeEdgeDownloadURL != "" && ku.ServiceFileURLFormat != "" {
		KubeEdgeDownloadURL = ku.KubeEdgeDownloadURL
		ServiceFileURLFormat = ku.ServiceFileURLFormat
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

	if ku.EdgeNodeIP != "" {
		edgeCoreConfig.Modules.Edged.NodeIP = ku.EdgeNodeIP
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

	// Edge nodes will connect these exposed servers
	cloudCoreIP := strings.Split(ku.CloudCoreIP, ":")[0]
	if ku.CertPort != "" {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + net.JoinHostPort(cloudCoreIP, ku.CertPort)
	} else {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + net.JoinHostPort(cloudCoreIP, "10002")
	}
	if ku.QuicPort != "" {
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = net.JoinHostPort(cloudCoreIP, ku.QuicPort)
	} else {
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = net.JoinHostPort(cloudCoreIP, "10001")
	}
	if ku.TunnelPort != "" {
		edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(cloudCoreIP, ku.TunnelPort)
	} else {
		edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(cloudCoreIP, strconv.Itoa(constants.DefaultTunnelPort))
	}

	// Determine whether to add the default NoSchedule taints or not
	if ku.EnableDefaultTaint {
		taint := v1.Taint{
			Key:    "node-role.kubernetes.io/edge",
			Effect: "NoSchedule",
		}
		edgeCoreConfig.Modules.Edged.Taints = append(edgeCoreConfig.Modules.Edged.Taints, taint)
	}

	if len(ku.Labels) >= 1 {
		labelsMap := make(map[string]string)
		for _, label := range ku.Labels {
			key := strings.Split(label, "=")[0]
			value := strings.Split(label, "=")[1]
			labelsMap[key] = value
		}
		edgeCoreConfig.Modules.Edged.Labels = labelsMap
	}

	// Determin whether to enable service bus
	edgeCoreConfig.Modules.ServiceBus.Enable = ku.EnableServiceBus

	// For edgemesh configuration
	edgeCoreConfig.Modules.MetaManager.MetaServer.Enable = ku.EnableMetaServer
	edgeCoreConfig.Modules.Edged.ClusterDNS = ku.EdgedClusterDNS
	edgeCoreConfig.Modules.Edged.ClusterDomain = ku.EdgedClusterDomain

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
