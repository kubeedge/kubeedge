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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	cri "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/image"
)

func dockerRequest(opt *common.JoinOptions, step *common.Step, imageSet image.Set) error {
	ctx := context.Background()
	step.Printf("Init docker dockerclient")
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return fmt.Errorf("init docker dockerclient failed: %v", err)
	}

	step.Printf("Pull Images")
	if err := dockerPullImages(ctx, imageSet, cli); err != nil {
		return fmt.Errorf("pull Images failed: %v", err)
	}

	step.Printf("Copy resources from the image to the management directory")
	if err := dockerCopyResources(ctx, opt, imageSet, cli); err != nil {
		return fmt.Errorf("copy resources failed: %v", err)
	}

	if opt.WithMQTT {
		step.Printf("Start the default mqtt service")
		if err := createMQTTConfigFile(); err != nil {
			return fmt.Errorf("create MQTT config file failed: %v", err)
		}
		if err := dockerRunMQTT(ctx, imageSet, cli); err != nil {
			return fmt.Errorf("run MQTT failed: %v", err)
		}
	}
	return nil
}

func dockerPullImages(ctx context.Context, imageSet image.Set, cli *dockerclient.Client) error {
	for _, v := range imageSet {
		args := filters.NewArgs()
		args.Add("reference", v)
		list, err := cli.ImageList(ctx, dockertypes.ImageListOptions{Filters: args})
		if err != nil {
			return err
		}
		if len(list) > 0 {
			continue
		}

		rc, err := cli.ImagePull(ctx, v, dockertypes.ImagePullOptions{})
		if err != nil {
			return err
		}
		io.Copy(io.Discard, rc)
		rc.Close()
	}
	return nil
}

// dockerCopyResources copies binary and configuration file from the image to the host.
// The command it executes are as follows:
//
// docker run -v /etc/kubeedge:/tmp/kubeedge/data -v /usr/local/bin:/tmp/kubeedge/bin <IMAGE-NAME> \
// bash -c cp -r /etc/kubeedge:/tmp/kubeedge/data cp /usr/local/bin/edgecore:/tmp/kubeedge/bin/edgecore
func dockerCopyResources(ctx context.Context, opt *common.JoinOptions, imageSet image.Set, cli *dockerclient.Client) error {
	containerDataTmpPath := filepath.Join(util.KubeEdgeTmpPath, "data")
	containerBinTmpPath := filepath.Join(util.KubeEdgeTmpPath, "bin")
	config := &dockercontainer.Config{
		Image: imageSet.Get(image.EdgeCore),
		Cmd: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("cp -r %s %s && cp %s %s",
				util.KubeEdgePath, containerDataTmpPath,
				filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
				filepath.Join(containerBinTmpPath, util.KubeEdgeBinaryName),
			),
		},
	}
	hostConfig := &dockercontainer.HostConfig{
		Binds: []string{
			util.KubeEdgePath + ":" + containerDataTmpPath,
			util.KubeEdgeUsrBinPath + ":" + containerBinTmpPath,
		},
	}

	// Randomly generate container names to prevent duplicate names.
	container, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := cli.ContainerRemove(ctx, container.ID, dockertypes.ContainerRemoveOptions{}); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", container.ID)
		}
	}()
	return cli.ContainerStart(ctx, container.ID, dockertypes.ContainerStartOptions{})
}

func dockerRunMQTT(ctx context.Context, imageSet image.Set, cli *dockerclient.Client) error {
	_, portMap, err := nat.ParsePortSpecs([]string{
		"1883:1883",
		"9001:9001",
	})
	if err != nil {
		return err
	}

	hostConfig := &dockercontainer.HostConfig{
		PortBindings: portMap,
		RestartPolicy: dockercontainer.RestartPolicy{
			Name: "unless-stopped",
		},
		Binds: []string{
			filepath.Join(util.KubeEdgeSocketPath, image.EdgeMQTT) + ":/mosquitto",
		},
	}
	config := &dockercontainer.Config{Image: imageSet.Get(image.EdgeMQTT)}

	container, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, image.EdgeMQTT)
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, container.ID, dockertypes.ContainerStartOptions{})
}

func remoteRequest(opt *common.JoinOptions, step *common.Step, imageSet image.Set) error {
	step.Printf("Pull Images")
	if err := pullImages(opt, imageSet); err != nil {
		return fmt.Errorf("pull Images failed: %v", err)
	}

	runtimeService, err := remote.NewRemoteRuntimeService(opt.RemoteRuntimeEndpoint, time.Second*10)
	if err != nil {
		return err
	}

	step.Printf("Copy resources from the image to the management directory")
	if err := copyResources(opt, imageSet, runtimeService); err != nil {
		return fmt.Errorf("copy resources failed: %v", err)
	}

	if opt.WithMQTT {
		step.Printf("Start the default mqtt service")
		if err := createMQTTConfigFile(); err != nil {
			return fmt.Errorf("create MQTT config file failed: %v", err)
		}
		if err := runMQTT(imageSet, runtimeService); err != nil {
			return fmt.Errorf("run MQTT failed: %v", err)
		}
	}
	return nil
}

func pullImages(opt *common.JoinOptions, imageSet image.Set) error {
	imageService, err := remote.NewRemoteImageService(opt.RemoteRuntimeEndpoint, time.Second*10)
	if err != nil {
		return err
	}
	for _, v := range imageSet {
		imageSpec := &runtimeapi.ImageSpec{Image: v}
		status, err := imageService.ImageStatus(imageSpec)
		if err != nil {
			return err
		}
		if status == nil || status.Id == "" {
			if _, err := imageService.PullImage(imageSpec, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyResources(opt *common.JoinOptions, imageSet image.Set, runtimeService cri.RuntimeService) error {
	containerDataTmpPath := filepath.Join(util.KubeEdgeTmpPath, "data")
	containerBinTmpPath := filepath.Join(util.KubeEdgeTmpPath, "bin")
	psc := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{Name: util.KubeEdgeBinaryName},
	}
	sandbox, err := runtimeService.RunPodSandbox(psc, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := runtimeService.RemovePodSandbox(sandbox); err != nil {
			klog.V(3).ErrorS(err, "Remove pod sandbox failed", "containerID", sandbox)
		}
	}()

	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: "container",
		},
		Image: &runtimeapi.ImageSpec{
			Image: imageSet.Get(image.EdgeCore),
		},
		Command: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("cp -r %s %s && cp %s %s",
				util.KubeEdgePath, containerDataTmpPath,
				filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
				filepath.Join(containerBinTmpPath, util.KubeEdgeBinaryName),
			),
		},
		Mounts: []*runtimeapi.Mount{
			{
				HostPath:      util.KubeEdgePath,
				ContainerPath: containerDataTmpPath,
			},
			{
				HostPath:      util.KubeEdgeUsrBinPath,
				ContainerPath: containerBinTmpPath,
			},
		},
	}
	containerID, err := runtimeService.CreateContainer(sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	defer func() {
		if err := runtimeService.RemoveContainer(containerID); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", containerID)
		}
	}()

	return runtimeService.StartContainer(containerID)
}

func runMQTT(imageSet image.Set, runtimeService cri.RuntimeService) error {
	psc := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{Name: image.EdgeMQTT},
		PortMappings: []*runtimeapi.PortMapping{
			{
				ContainerPort: 1883,
				HostPort:      1883,
			},
			{
				ContainerPort: 9001,
				HostPort:      9001,
			},
		},
	}
	sandbox, err := runtimeService.RunPodSandbox(psc, "")
	if err != nil {
		return err
	}

	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{Name: image.EdgeMQTT},
		Image: &runtimeapi.ImageSpec{
			Image: imageSet.Get(image.EdgeMQTT),
		},
		Mounts: []*runtimeapi.Mount{
			{
				ContainerPath: "/mosquitto",
				HostPath:      filepath.Join(util.KubeEdgeSocketPath, image.EdgeMQTT),
			},
		},
	}
	containerID, err := runtimeService.CreateContainer(sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	return runtimeService.StartContainer(containerID)
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
