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

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	cloudInitLongDescription = `
"keadm init" command install KubeEdge's master node (on the cloud) component.
It checks if the Kubernetes Master are installed already,
If not installed, please install the Kubernetes first.
`
	cloudInitExample = `
keadm init

- This command will download and install the default version of KubeEdge cloud component

keadm init --kubeedge-version=%s  --kube-config=/root/.kube/config

  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
`
)

// NewCloudInit represents the keadm init command for cloud component
func NewCloudInit(out io.Writer, init *types.InitOptions) *cobra.Command {
	if init == nil {
		init = newInitOptions()
	}

	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: fmt.Sprintf(cloudInitExample, types.DefaultKubeEdgeVersion),
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)
			err := Add2ToolsList(tools, flagVals, init)
			if err != nil {
				return err
			}
			return Execute(tools)
		},
	}

	addJoinOtherFlags(cmd, init)
	return cmd
}

//newInitOptions will initialise new instance of options everytime
func newInitOptions() *types.InitOptions {
	opts := &types.InitOptions{}
	opts.KubeConfig = types.DefaultKubeConfig
	return opts
}

func addJoinOtherFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	cmd.Flags().StringVar(&initOpts.KubeEdgeVersion, types.KubeEdgeVersion, initOpts.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(types.KubeEdgeVersion).NoOptDefVal = initOpts.KubeEdgeVersion

	cmd.Flags().StringVar(&initOpts.KubeConfig, types.KubeConfig, initOpts.KubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")

	cmd.Flags().StringVar(&initOpts.Master, types.Master, initOpts.Master,
		"Use this key to set K8s master address, eg: http://127.0.0.1:8080")

	cmd.Flags().StringVar(&initOpts.AdvertiseAddress, types.AdvertiseAddress, initOpts.AdvertiseAddress,
		"Use this key to set IPs in cloudcore's certificate SubAltNames field. eg: 10.10.102.78,10.10.102.79")

	cmd.Flags().StringVar(&initOpts.DNS, types.DomainName, initOpts.DNS,
		"Use this key to set domain names in cloudcore's certificate SubAltNames field. eg: www.cloudcore.cn,www.kubeedge.cn")

	cmd.Flags().StringVar(&initOpts.TarballPath, types.TarballPath, initOpts.TarballPath,
		"Use this key to set the temp directory path for KubeEdge tarball, if not exist, download it")
}

//Add2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, initOptions *types.InitOptions) error {
	var kubeVer string
	flgData, ok := flagData[types.KubeEdgeVersion]
	if ok {
		kubeVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	}
	if kubeVer == "" {
		var latestVersion string
		for i := 0; i < util.RetryTimes; i++ {
			version, err := util.GetLatestVersion()
			if err != nil {
				fmt.Println("Failed to get the latest KubeEdge release version")
				continue
			}
			if len(version) > 0 {
				kubeVer = strings.TrimPrefix(version, "v")
				latestVersion = version
				break
			}
		}
		if len(latestVersion) == 0 {
			fmt.Println("Failed to get the latest KubeEdge release version, will use default version")
			kubeVer = types.DefaultKubeEdgeVersion
		}
	}
	common := util.Common{
		ToolVersion: semver.MustParse(kubeVer),
		KubeConfig:  initOptions.KubeConfig,
		Master:      initOptions.Master,
	}
	toolList["Cloud"] = &util.KubeCloudInstTool{
		Common:           common,
		AdvertiseAddress: initOptions.AdvertiseAddress,
		DNSName:          initOptions.DNS,
		TarballPath:      initOptions.TarballPath,
	}
	toolList["Kubernetes"] = &util.K8SInstTool{
		Common: common,
	}
	return nil
}

//Execute the installation for each tool and start cloudcore
func Execute(toolList map[string]types.ToolsInstaller) error {
	for name, tool := range toolList {
		if name != "Cloud" {
			err := tool.InstallTools()
			if err != nil {
				return err
			}
		}
	}

	return toolList["Cloud"].InstallTools()
}
