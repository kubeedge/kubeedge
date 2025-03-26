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
	"context"
	"fmt"
	"path/filepath"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func request(opt *common.JoinOptions, step *common.Step) error {
	ctx := context.Background()
	imageSet := util.EdgeSet(opt)
	images := imageSet.List()

	runtime, err := util.NewContainerRuntime(opt.RemoteRuntimeEndpoint, opt.CGroupDriver)
	if err != nil {
		return err
	}

	step.Printf("Pull Images")
	if err := runtime.PullImages(ctx, images, nil); err != nil {
		return fmt.Errorf("pull Images failed: %v", err)
	}

	step.Printf("Copy resources from the image to the management directory")
	containerPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName)
	hostPath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName)
	files := map[string]string{containerPath: hostPath}
	if err := runtime.CopyResources(ctx, imageSet.Get(util.EdgeCore), files); err != nil {
		return fmt.Errorf("copy resources failed: %v", err)
	}
	return nil
}
