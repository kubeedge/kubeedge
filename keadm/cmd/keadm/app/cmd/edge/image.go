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

package edge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/image"
)

func request(opt *common.JoinOptions, step *common.Step) error {
	imageSet := image.EdgeSet(opt.ImageRepository, opt.KubeEdgeVersion)
	images := imageSet.List()

	runtime, err := util.NewContainerRuntime(opt.RuntimeType, opt.RemoteRuntimeEndpoint)
	if err != nil {
		return err
	}

	step.Printf("Pull Images")
	if err := runtime.PullImages(images); err != nil {
		return fmt.Errorf("pull Images failed: %v", err)
	}

	step.Printf("Copy resources from the image to the management directory")
	files := map[string]string{
		filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName): filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
	}
	if err := runtime.CopyResources(imageSet.Get(image.EdgeCore), files); err != nil {
		return fmt.Errorf("copy resources failed: %v", err)
	}

	if opt.WithMQTT {
		step.Printf("Start the default mqtt service")
		if err := createMQTTConfigFile(); err != nil {
			return fmt.Errorf("create MQTT config file failed: %v", err)
		}
		if err := runtime.RunMQTT(imageSet.Get(image.EdgeMQTT)); err != nil {
			return fmt.Errorf("run MQTT failed: %v", err)
		}
	}
	return nil
}

func createMQTTConfigFile() error {
	dir := filepath.Join(util.KubeEdgeSocketPath, image.EdgeMQTT, "config")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	data := `persistence true
persistence_location /mosquitto/data
log_dest file /mosquitto/log/mosquitto.log
`
	currentPath := filepath.Join(dir, "mosquitto.conf")
	return os.WriteFile(currentPath, []byte(data), os.ModePerm)
}
