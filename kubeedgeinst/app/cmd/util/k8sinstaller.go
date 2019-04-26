/*
Copyright 2019 The Kubeedge Authors.

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

import "fmt"

type K8SInstTool struct {
	Common
	IsEdgeNode     bool //True - Edgenode False - Cloudnode
	DefaultToolVer string
}

func (ks *K8SInstTool) InstallTools() error {
	ks.SetOSInterface(GetOSInterface())
	ks.SetK8SVersionAndIsNodeFlag(ks.ToolVersion, ks.IsEdgeNode)

	component := "kubeadm"
	if ks.IsEdgeNode == true {
		component = "kubectl"
	}
	action, err := ks.IsK8SComponentInstalled(component, ks.DefaultToolVer)
	if err != nil {
		return err
	}
	switch action {
	case VersionNAInRepo:
		return fmt.Errorf("Expected %s version is not available in OS repo", component)
	case AlreadySameVersionExist:
		return fmt.Errorf("Same version of %s already installed in this host", component)
	case DefVerInstallRequired:
		ks.SetK8SVersionAndIsNodeFlag(ks.DefaultToolVer, ks.IsEdgeNode)
		fallthrough
	case NewInstallRequired:
		err := ks.InstallK8S()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error in getting the %s version from host", component)
	}
	return nil
}

func (ks *K8SInstTool) TearDown() error {
	return nil
}
