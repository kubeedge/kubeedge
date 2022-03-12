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

package beta

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
"keadm beta manifest" command renders charts by using a list of set flags like helm.
`

	cloudManifestGenerateLongDescription = `
"keadm beta manifest generate" command renders charts by using a list of set flags like helm, and generates kubernetes resources.
`

	cloudManifestExample = `
keadm beta manifest

- This command will render Kubernetes resources

keadm manifest generate --advertise-address=127.0.0.1 --profile version=v1.9.0 --kube-config=/root/.kube/config

  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
`

	cloudManifestGenerateExample = `
keadm beta manifest generate

- This command will render and generate Kubernetes resources

keadm manifest generate --advertise-address=127.0.0.1 --profile version=v1.9.0 --kube-config=/root/.kube/config

  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
`
)

// NewManifestGenerateBeta represents the keadm beta manifest command for cloud component
func NewManifestGenerateBeta() *cobra.Command {
	BetaInit := newInitBetaOptions()
	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	var generateCmd = &cobra.Command{
		Use:     "generate",
		Short:   "Checks and generate the manifests.",
		Long:    cloudManifestGenerateLongDescription,
		Example: cloudManifestGenerateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)
			err := AddManifestsGenerate2ToolsList(tools, flagVals, BetaInit)
			if err != nil {
				return err
			}
			return ExecuteManifestsGenerate(tools)
		},
	}

	addManifestsGenerateJoinOtherFlags(generateCmd, BetaInit)

	var manifestCmd = &cobra.Command{
		Use:     "manifest",
		Short:   "Render the manifests by using a list of set flags like helm.",
		Long:    cloudManifestLongDescription,
		Example: cloudManifestExample,
	}
	manifestCmd.AddCommand(generateCmd)
	return manifestCmd
}

func addManifestsGenerateJoinOtherFlags(cmd *cobra.Command, initBetaOpts *types.InitBetaOptions) {
	addInitBetaJoinOtherFlags(cmd, initBetaOpts)
	addHelmValueOptionsFlags(cmd, initBetaOpts)

	cmd.Flags().BoolVar(&initBetaOpts.SkipCRDs, types.SkipCRDs, initBetaOpts.SkipCRDs,
		"Skip printing the contents of CRDs to stdout")
}

//AddManifestsGenerate2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddManifestsGenerate2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, initBetaOpts *types.InitBetaOptions) error {
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
		KubeConfig:  initBetaOpts.KubeConfig,
	}
	toolList["helm"] = &helm.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: initBetaOpts.AdvertiseAddress,
		KubeEdgeVersion:  initBetaOpts.KubeEdgeVersion,
		Manifests:        initBetaOpts.Manifests,
		Namespace:        constants.SystemNamespace,
		DryRun:           initBetaOpts.DryRun,
		Sets:             initBetaOpts.Sets,
		Profile:          initBetaOpts.Profile,
		SkipCRDs:         initBetaOpts.SkipCRDs,
		Action:           types.HelmManifestAction,
	}
	return nil
}

//ExecuteInitBeta the installation for each tool and start cloudcore
func ExecuteManifestsGenerate(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
