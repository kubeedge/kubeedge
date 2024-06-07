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
"keadm manifest" command renders charts by using a list of set flags like helm, and generates kubernetes resources.
`

	cloudManifestGenerateExample = `
keadm manifest
- This command will render and generate Kubernetes resources

keadm manifest --advertise-address=127.0.0.1 --profile version=v%s --kube-config=/root/.kube/config
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
		Use:     "manifest",
		Short:   "Checks and generate the manifests.",
		Long:    cloudManifestLongDescription,
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
	return generateCmd
}

func addManifestsGenerateJoinOtherFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	addInitOtherFlags(cmd, initOpts)
	addHelmValueOptionsFlags(cmd, initOpts)

	cmd.Flags().BoolVar(&initOpts.SkipCRDs, types.FlagNameSkipCRDs, initOpts.SkipCRDs,
		"Skip printing the contents of CRDs to stdout")
}

// AddManifestsGenerate2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddManifestsGenerate2ToolsList(toolList map[string]types.ToolsInstaller, _ map[string]types.FlagData, initOpts *types.InitOptions) error {
	common := util.Common{
		ToolVersion: semver.MustParse(util.GetHelmVersion(initOpts.KubeEdgeVersion, util.RetryTimes)),
		KubeConfig:  initOpts.KubeConfig,
	}
	toolList["helm"] = &helm.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: initOpts.AdvertiseAddress,
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

// ExecuteManifestsGenerate executes the installation for helm
func ExecuteManifestsGenerate(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
