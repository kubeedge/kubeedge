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

	"github.com/kubeedge/kubeedge/kectl/app/cmd/util"

	"github.com/spf13/cobra"
)

var (
	cloudResetLongDescription = `
cloud reset command will tear down KubeEdge
cloud component and stop K8S master node
`
	cloudResetExample = `
kectl cloud reset
`
)

// NewCloudReset represents KubeEdge's cloud components reset command
func NewCloudReset(out io.Writer) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "reset",
		Short: "Teardowns cloud component",
		Long:  cloudResetLongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Work your own magic here
			fmt.Println("cloud reset called")
			TearDownCloud()
		},
		Example: cloudResetExample,
		Args:    cobra.NoArgs,
	}

	return cmd
}

//Tear Down nodes will do kubeadm reset and kill edgecontroller
func TearDownCloud() {
	cloud := &util.KubeCloudInstTool{Common: util.Common{}}
	cloud.KillEdgeController()
	cloud.ResetKubernetes()
	fmt.Println("Reset is sucessful")
}
