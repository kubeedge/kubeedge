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

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"

	cmdcommon "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/image"
)

// Configuration represent keadm config options
type Configuration struct {
	// eg. v1.9.0
	KubeEdgeVersion string
	// eg. kubeedge
	ImageRepository string
	// eg. cloud/edge
	Part string

	RuntimeType           string
	RemoteRuntimeEndpoint string
}

func newDefaultConfiguration() *Configuration {
	return &Configuration{
		ImageRepository: "kubeedge",
		Part:            "",
		RuntimeType:     kubetypes.RemoteContainerRuntime,
	}
}

func (cfg *Configuration) GetImageRepository() string {
	return cfg.ImageRepository
}

// newCmdConfig returns cobra.Command for "keadm config" command
func newCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Use this command to configure keadm",
		Long:  "Use this command to configure keadm",
	}

	cmd.AddCommand(newCmdConfigImages())
	return cmd
}

// newCmdConfigImages returns the "keadm config images" command
func newCmdConfigImages() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Interact with container images used by keadm",
		Long:  "Use this command to `list/pull` keadm required container images",
	}
	cmd.AddCommand(newCmdConfigImagesList())
	cmd.AddCommand(newCmdConfigImagesPull())
	return cmd
}

// newCmdConfigImagesList returns the "keadm config images list" command
func newCmdConfigImagesList() *cobra.Command {
	cfg := newDefaultConfiguration()

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print a list of images keadm will use.",
		RunE: func(_ *cobra.Command, _ []string) error {
			ver, err := util.GetCurrentVersion(cfg.KubeEdgeVersion)
			if err != nil {
				return err
			}
			cfg.KubeEdgeVersion = ver

			images := GetKubeEdgeImages(cfg)
			for _, v := range images {
				fmt.Println(v)
			}

			return nil
		},
		Args: cobra.NoArgs,
	}

	AddImagesCommonConfigFlags(cmd, cfg)
	return cmd
}

// newCmdConfigImagesPull returns the `keadm config images pull` command
func newCmdConfigImagesPull() *cobra.Command {
	cfg := newDefaultConfiguration()

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull images used by keadm",
		RunE: func(_ *cobra.Command, _ []string) error {
			ver, err := util.GetCurrentVersion(cfg.KubeEdgeVersion)
			if err != nil {
				return err
			}
			cfg.KubeEdgeVersion = ver

			images := GetKubeEdgeImages(cfg)
			return pullImages(cfg.RuntimeType, cfg.RemoteRuntimeEndpoint, images)
		},
		Args: cobra.NoArgs,
	}
	AddImagesCommonConfigFlags(cmd, cfg)

	return cmd
}

func pullImages(runtimeType string, endpoint string, images []string) error {
	runtime, err := util.NewContainerRuntime(runtimeType, endpoint)
	if err != nil {
		return err
	}

	return runtime.PullImages(images)
}

// AddImagesCommonConfigFlags adds the flags that configure keadm
func AddImagesCommonConfigFlags(cmd *cobra.Command, cfg *Configuration) {
	cmd.Flags().StringVar(&cfg.KubeEdgeVersion, cmdcommon.KubeEdgeVersion, cfg.KubeEdgeVersion,
		`Use this key to decide which a specific KubeEdge version to be used.`,
	)
	cmd.Flags().StringVar(&cfg.ImageRepository, cmdcommon.ImageRepository, cfg.ImageRepository,
		`Use this key to decide which image repository to pull images from.`,
	)
	cmd.Flags().StringVar(&cfg.Part, "part", cfg.Part,
		"Use this key to set which part keadm will install: cloud part or edge part. If not set, keadm will list/pull all images used by both cloud part and edge part.")

	cmd.Flags().StringVar(&cfg.RuntimeType, cmdcommon.RuntimeType, cfg.RuntimeType,
		"Container runtime type, default is remote")

	cmd.Flags().StringVar(&cfg.RemoteRuntimeEndpoint, cmdcommon.RemoteRuntimeEndpoint, cfg.RemoteRuntimeEndpoint,
		"The endpoint of remote runtime service in edge node")
}

// GetKubeEdgeImages returns a list of container images that related part expects to use
func GetKubeEdgeImages(cfg *Configuration) []string {
	var images []string
	switch strings.ToLower(cfg.Part) {
	case "cloud":
		images = image.CloudSet(cfg.ImageRepository, cfg.KubeEdgeVersion).List()
	case "edge":
		images = image.EdgeSet(cfg.ImageRepository, cfg.KubeEdgeVersion).List()
	default:
		// if not specified, will return all images used by both cloud part and edge part
		cloudSet := image.CloudSet(cfg.ImageRepository, cfg.KubeEdgeVersion)
		edgeSet := image.EdgeSet(cfg.ImageRepository, cfg.KubeEdgeVersion)
		images = cloudSet.Merge(edgeSet).List()
	}
	return images
}
