/*
Copyright 2024 The KubeEdge Authors.
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

package edge

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func NewBatchProcessGenConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-config",
		Short: "Generate a YAML config file for batch process",
		Long:  `This command generates a template YAML configuration file for batch process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := []byte(getConfigTemplate())

			// Write to file
			fileName := "config.yaml"
			if err := os.WriteFile(fileName, data, 0644); err != nil {
				klog.Errorf("Error writing config file: %v", err)
				return err
			}

			klog.Infof("Config template generated: %s", fileName)
			return nil
		},
	}
	return cmd
}

func getConfigTemplate() string {
	return `# detailed design: https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/batch-node-process.md
keadm:
  download:
    enable: true              # <Optional> Whether to download the keadm package, which can be left unconfigured, default is true. if it is false, the 'offlinePackageDir' will be used.
    url: ""                   # <Optional> The download address of the keadm package, which can be left unconfigured. If this parameter is not configured, the official github repository will be used by default.
  keadmVersion: ""            # <Required> The version of keadm to be installed. for example: v1.19.0
  archGroup:                  # <Required> This parameter can configure one or more of amd64/arm64/arm.
    - amd64
  offlinePackageDir: ""       # <Optional> The path of the offline package. When download.enable is true, this parameter can be left unconfigured.
  cmdTplArgs:                 # <Optional> This parameter is the execution command template, which can be optionally configured and used in conjunction with nodes[x].keadmCmd.
    cmd: ""                   # This is an example parameter, which can be used in conjunction with nodes[x].keadmCmd.
    token: ""                 # This is an example parameter, which can be used in conjunction with nodes[x].keadmCmd.
nodes:
  - nodeName: edge-node       # <Required> Unique name, used to identify the node
    arch: amd64               # <Required> The architecture of the node, which can be configured as amd64/arm64/arm
    keadmCmd: ""              # <Required> The command to be executed on the node, can used in conjunction with keadm.cmdTplArgs. for example: "{{.cmd}} --edgenode-name=containerd-node1 --token={{.token}}"
    copyFrom: ""              # <Optional> The path of the file to be copied from the local machine to the node, which can be left unconfigured.
    ssh:
      ip: ""                  # <Required> The IP address of the node.
      username: root          # <Required> The username of the node, need administrator permissions.
      port: 22                # <Optional> The port number of the node, the default is 22.
      auth:                   # Log in to the node with a private key or password, only one of them can be configured.
        type: password        # <Required> The value can be configured as 'password' or 'privateKey'.
        passwordAuth:         # It can be configured as 'passwordAuth' or 'privateKeyAuth'.
          password: ""        # <Required> The key can be configured as 'password' or 'privateKeyPath'.
maxRunNum: 5                  # <Optional> The maximum number of concurrent executions, which can be left unconfigured. The default is 5.`
}
