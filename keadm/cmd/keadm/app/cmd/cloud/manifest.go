/*
Copyright 2022 The KubeEdge Authors.

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

package cloud

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	cloudManifestLongDescription = `
"keadm manifest" command renders charts by using a list of set flags like helm.
`

	cloudManifestGenerateLongDescription = `
"keadm manifest generate" command renders charts by using a list of set flags like helm, and generates kubernetes resources.
`

	cloudManifestExample = `
keadm manifest
- This command will render Kubernetes resources

keadm generate --advertise-address=127.0.0.1 --profile version=v%s --kube-config=/root/.kube/config
  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
	- a list of helm style set flags like "--set key=value" can be implemented, ref: https://github.com/kubeedge/kubeedge/tree/master/manifests/charts/cloudcore/README.md
`

	cloudManifestGenerateExample = `
keadm manifest generate
- This command will render and generate Kubernetes resources

keadm manifest generate --advertise-address=127.0.0.1 --profile version=v%s --kube-config=/root/.kube/config
  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
	- a list of helm style set flags like "--set key=value" can be implemented, ref: https://github.com/kubeedge/kubeedge/tree/master/manifests/charts/cloudcore/README.md
`
)

// NewManifestGenerate represents the keadm manifest command for cloud component
func NewManifestGenerate() *cobra.Command {
	opts := newInitOptions()
	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	var generateCmd = &cobra.Command{
		Use:     "generate",
		Short:   "Checks and generate the manifests.",
		Long:    cloudManifestGenerateLongDescription,
		Example: fmt.Sprintf(cloudManifestGenerateExample, types.DefaultKubeEdgeVersion),
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)
			if err := AddManifestsGenerate2ToolsList(tools, flagVals, opts); err != nil {
				return err
			}
			return ExecuteManifestsGenerate(tools)
		},
	}

	addManifestsGenerateJoinOtherFlags(generateCmd, opts)

	var manifestCmd = &cobra.Command{
		Use:     "manifest",
		Short:   "Render the manifests by using a list of set flags like helm.",
		Long:    cloudManifestLongDescription,
		Example: fmt.Sprintf(cloudManifestExample, types.DefaultKubeEdgeVersion),
	}
	manifestCmd.AddCommand(generateCmd)
	return manifestCmd
}

func addManifestsGenerateJoinOtherFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	addInitOtherFlags(cmd, initOpts)
	addHelmValueOptionsFlags(cmd, initOpts)

	cmd.Flags().BoolVar(&initOpts.SkipCRDs, types.SkipCRDs, initOpts.SkipCRDs,
		"Skip printing the contents of CRDs to stdout")
}

//AddManifestsGenerate2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddManifestsGenerate2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, initOpts *types.InitOptions) error {
	var latestVersion string
	var kubeedgeVersion string
	for i := 0; i < util.RetryTimes; i++ {
		version, err := util.GetLatestVersion()
		if err != nil {
			fmt.Println("Failed to get the latest KubeEdge release version, error: ", err)
			continue
		}
		if len(version) > 0 {
			kubeedgeVersion = strings.TrimPrefix(version, "v")
			latestVersion = version
			break
		}
	}
	if len(latestVersion) == 0 {
		kubeedgeVersion = types.DefaultKubeEdgeVersion
		fmt.Println("Failed to get the latest KubeEdge release version, will use default version: ", kubeedgeVersion)
	}

	common := util.Common{
		ToolVersion: semver.MustParse(kubeedgeVersion),
		KubeConfig:  initOpts.KubeConfig,
	}
	toolList["helm"] = &helm.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: initOpts.AdvertiseAddress,
		KubeEdgeVersion:  initOpts.KubeEdgeVersion,
		Manifests:        initOpts.Manifests,
		Namespace:        constants.SystemNamespace,
		DryRun:           initOpts.DryRun,
		Sets:             initOpts.Sets,
		Profile:          initOpts.Profile,
		SkipCRDs:         initOpts.SkipCRDs,
		Action:           types.HelmManifestAction,
	}
	return nil
}

//ExecuteManifestsGenerate executes the installation for helm
func ExecuteManifestsGenerate(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
