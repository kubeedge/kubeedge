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

import (
	"fmt"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
)

//DockerInstTool embedes Common struct and contains the default docker version
//It implements ToolsInstaller interface
type DockerInstTool struct {
	Common
	DefaultToolVer string
}

//InstallTools sets the OS interface, checks if docker installation is required or not.
//If required then install the said version.
func (d *DockerInstTool) InstallTools() error {
	d.SetOSInterface(GetOSInterface())
	d.SetDockerVersion(d.ToolVersion)

	action, err := d.IsDockerInstalled(d.DefaultToolVer)
	if err != nil {
		return err
	}
	switch action {
	case types.VersionNAInRepo:
		return fmt.Errorf("Expected Docker version is not available in OS repo")
	case types.AlreadySameVersionExist:
		return fmt.Errorf("Same version of docker already installed in this host")
	case types.DefVerInstallRequired:
		d.SetDockerVersion(d.DefaultToolVer)
		fallthrough
	case types.NewInstallRequired:
		err := d.InstallDocker()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error in getting the docker version from host")
	}

	return nil
}

//TearDown shoud uninstall docker, but it is not required either for cloud or edge node.
//It is defined so that DockerInstTool implements ToolsInstaller interface
func (d *DockerInstTool) TearDown() error {
	return nil
}
