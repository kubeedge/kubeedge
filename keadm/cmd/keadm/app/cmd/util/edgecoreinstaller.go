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
	"fmt"
	"os"

	"github.com/google/uuid"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

// KubeEdgeInstTool embedes Common struct and contains cloud node ip:port information
// It implements ToolsInstaller interface
type KubeEdgeInstTool struct {
	Common
	CertPath              string
	CloudCoreIP           string
	EdgeNodeName          string
	RuntimeType           string
	InterfaceName         string
	RemoteRuntimeEndpoint string
}

// InstallTools downloads KubeEdge for the specified verssion
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

	err = ku.InstallKubeEdge(types.EdgeCore)
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
	if ku.ToolVersion >= "1.2.0" {
		//This makes sure the path is created, if it already exists also it is fine
		err := os.MkdirAll(KubeEdgeNewConfigDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", KubeEdgeNewConfigDir)
		}

		edgeCoreConfig := v1alpha1.NewDefaultEdgeCoreConfig()
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = ku.CloudCoreIP

		if ku.EdgeNodeName != "" {
			edgeCoreConfig.Modules.Edged.HostnameOverride = ku.EdgeNodeName
		}
		if ku.RuntimeType != "" {
			edgeCoreConfig.Modules.Edged.RuntimeType = ku.RuntimeType
		}
		if ku.InterfaceName != "" {
			edgeCoreConfig.Modules.Edged.InterfaceName = ku.InterfaceName
		}
		if ku.RemoteRuntimeEndpoint != "" {
			edgeCoreConfig.Modules.Edged.RemoteRuntimeEndpoint = ku.RemoteRuntimeEndpoint
			edgeCoreConfig.Modules.Edged.RemoteImageEndpoint = ku.RemoteRuntimeEndpoint
		}

		if err := types.Write2File(KubeEdgeEdgeCoreNewYaml, edgeCoreConfig); err != nil {
			return err
		}
	} else {
		//This makes sure the path is created, if it already exists also it is fine
		err := os.MkdirAll(KubeEdgeConfPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", KubeEdgeConfPath)
		}

		// //Create edge.yaml
		//Update edge.yaml with a unique id against node id
		//If the user doesn't provide any edge ID on the command line, then it generates unique id and assigns it.
		edgeID := uuid.New().String()
		if "" != ku.EdgeNodeName {
			edgeID = ku.EdgeNodeName
		}

		serverIPAddr := "0.0.0.0"
		if "" != ku.CloudCoreIP {
			serverIPAddr = ku.CloudCoreIP
		}

		url := fmt.Sprintf("wss://%s:10000/%s/%s/events", serverIPAddr, types.DefaultProjectID, edgeID)
		edgeYaml := &types.EdgeYamlSt{EdgeHub: types.EdgeHubSt{WebSocket: types.WebSocketSt{URL: url}},
			EdgeD: types.EdgeDSt{RuntimeType: ku.RuntimeType, InterfaceName: ku.InterfaceName}}

		if err = types.WriteEdgeYamlFile(KubeEdgeConfigEdgeYaml, edgeYaml); err != nil {
			return err
		}

		//Create modules.yaml
		if err = types.WriteEdgeModulesYamlFile(KubeEdgeConfigModulesYaml); err != nil {
			return err
		}
	}
	return nil
}

//TearDown method will remove the edge node from api-server and stop edgecore process
func (ku *KubeEdgeInstTool) TearDown() error {
	ku.SetOSInterface(GetOSInterface())
	ku.SetKubeEdgeVersion(ku.ToolVersion)

	//Kill edge core process
	ku.KillKubeEdgeBinary(KubeEdgeBinaryName)

	return nil
}
