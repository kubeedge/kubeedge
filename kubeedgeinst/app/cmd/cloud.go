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
	"io"

	"github.com/spf13/cobra"

	cloud "github.com/kubeedge/kubeedge/kubeedgeinst/app/cmd/cloud"
)

var (
	cloudLongDescription = `
cloud commands help in operating with KubeEdge's cloud component.
`
	cloudExample = `
kubeedge cloud init <arguments> 
kubeedge cloud reset 
`
)

// NewCmdCloud represents the cloud commands
func NewCmdCloud(out io.Writer) *cobra.Command {

	var cmd = &cobra.Command{
		Use:     "cloud",
		Short:   "Cloud component command option for KubeEdge",
		Long:    cloudLongDescription,
		Args:    cobra.MinimumNArgs(1),
		Example: cloudExample,
	}

	cmd.AddCommand(cloud.NewCloudInit(out, nil))
	cmd.AddCommand(cloud.NewCloudReset(out))
	return cmd
}

func init() {

}
