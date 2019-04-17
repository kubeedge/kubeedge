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

package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/kubeedgeinst/app/cmd/options"
)

var (
	cloudInitLongDescription = `
cloud init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
`
	cloudInitExample = `
kubeedge cloud init  
`
)

// NewCloudInit represents the kubeedge cloud init command
func NewCloudInit(out io.Writer, init *options.InitOptions) *cobra.Command {
	if init == nil {
		init = newInitOptions()
	}

	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: cloudInitExample,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Work your own magic here
			fmt.Println("cloud init called")
		},
	}

	addJoinOtherFlags(cmd, init)
	return cmd
}

//newInitOptions will initialise new instance of options everytime
func newInitOptions() *options.InitOptions {
	return &options.InitOptions{}
}

func addJoinOtherFlags(cmd *cobra.Command, initOpts *options.InitOptions) {

	//add logic
	// add logs
	// --kubeedge-version   string   use this key to download and use the required KubeEdge version (Optional, default will be Latest)
	//--kubernetes-version string   use this key to download and use the required Kubernetes version (Optional, default will be Latest)
	//--docker-version     string   use this key to download and use the required Docker version (Optional, default will be Latest)
	// Add flags to the command
	cmd.Flags().StringVar(&initOpts.KubeedgeVersion, options.KubeedgeVersion, initOpts.KubeedgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().StringVar(&initOpts.DockerVersion, options.DockerVersion, initOpts.DockerVersion,
		"Use this key to download and use the required Docker version")
	cmd.Flags().StringVar(&initOpts.Kubernetesversion, options.Kubernetesversion, initOpts.Kubernetesversion,
		"Use this key to download and use the required Kubernetes version")
}

// func init() {

// }
