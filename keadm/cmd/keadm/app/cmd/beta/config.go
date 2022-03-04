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

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/common/constants"
	cmdcommon "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

// PullImageRetry represent pulling images retry times
const PullImageRetry = 5

// Configuration represent keadm config options
type Configuration struct {
	// eg. v1.9.0
	KubeEdgeVersion string
	// eg. kubeedge
	ImageRepository string
	// eg. cloud/edge
	Part string
}

func newDefaultConfiguration() *Configuration {
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
		kubeedgeVersion = cmdcommon.DefaultKubeEdgeVersion
		fmt.Println("Failed to get the latest KubeEdge release version, will use default version: ", kubeedgeVersion)
	}

	return &Configuration{
		KubeEdgeVersion: "v" + kubeedgeVersion,
		ImageRepository: "kubeedge",
		Part:            "",
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
			images := GetKubeEdgeImages(cfg)
			for _, image := range images {
				fmt.Println(image)
			}

			return nil
		},
		Args: cobra.NoArgs,
	}

	AddImagesCommonConfigFlags(cmd, cfg)
	return cmd
}

// newCmdConfigImagesPull returns the `keadm config images pull` command
// TODO: Now we only support docker images. If can, we will need to support more CRIs.
func newCmdConfigImagesPull() *cobra.Command {
	cfg := newDefaultConfiguration()

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull images used by keadm",
		RunE: func(_ *cobra.Command, _ []string) error {
			images := GetKubeEdgeImages(cfg)

			return PullControlPlaneImages(images)
		},
		Args: cobra.NoArgs,
	}
	AddImagesCommonConfigFlags(cmd, cfg)

	return cmd
}

// PullControlPlaneImages pulls all images
func PullControlPlaneImages(images []string) error {
	for _, image := range images {
		fmt.Printf("Pulling %s ...\n", image)
		if err := dockerPullImage(image); err != nil {
			fmt.Printf("Failed to pull image %q: %v", image, err)
			return fmt.Errorf("failed to pull image %q: %v", image, err)
		}
		fmt.Printf("Successfully pulled %s\n", image)
	}
	return nil
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
}

// GetKubeEdgeImages returns a list of container images that related part expects to use
func GetKubeEdgeImages(cfg *Configuration) []string {
	var images []string

	part := strings.ToLower(cfg.Part)

	if part == "cloud" {
		images = append(images, GetKubeEdgeCloudPartImages(cfg)...)
	} else if part == "edge" {
		images = append(images, GetKubeEdgeEdgePartImages(cfg)...)
	} else {
		// if not specified, will return all images used by both cloud part and edge part
		images = append(images, GetKubeEdgeCloudPartImages(cfg)...)
		images = append(images, GetKubeEdgeEdgePartImages(cfg)...)
	}

	return images
}

// GetKubeEdgeCloudPartImages returns a list of container images that Cloud part expects to use
func GetKubeEdgeCloudPartImages(cfg *Configuration) []string {
	images := []string{}

	// Cloud part images
	images = append(images, GetImage("admission", cfg))
	images = append(images, GetImage("cloudcore", cfg))
	images = append(images, GetImage("iptables-manager", cfg))
	images = append(images, GetImage("installation-package", cfg))

	return images
}

// GetKubeEdgeEdgePartImages returns a list of container images that Edge expects to use
func GetKubeEdgeEdgePartImages(cfg *Configuration) []string {
	images := []string{}

	// Edge part images
	images = append(images, GetPauseImage())
	images = append(images, GetImage("installation-package", cfg))
	images = append(images, "eclipse-mosquitto:1.6.15")

	return images
}

// GetImage generates the image required for the KubeEdge
func GetImage(image string, cfg *Configuration) string {
	repoPrefix := cfg.GetImageRepository()
	imageTag := cfg.KubeEdgeVersion

	return fmt.Sprintf("%s/%s:%s", repoPrefix, image, imageTag)
}

func GetPauseImage() string {
	return constants.DefaultPodSandboxImage
}

// dockerPullImage uses Docker to pull the image
func dockerPullImage(image string) error {
	var err error
	for i := 0; i < PullImageRetry; i++ {
		pullCommand := fmt.Sprintf("docker pull %s", image)
		err = util.NewCommand(pullCommand).Exec()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("dcoker pull image failed: %v", err)
}
