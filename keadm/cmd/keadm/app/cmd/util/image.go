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

package util

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	oteltrace "go.opentelemetry.io/otel/trace"
	internalapi "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/image"
)

// mqttLabel is used to select MQTT containers
var mqttLabel = map[string]string{"io.kubeedge.edgecore/mqtt": image.EdgeMQTT}

type ContainerRuntime interface {
	PullImages(images []string) error
	PullImage(image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error
	CopyResources(edgeImage string, files map[string]string) error
	RunMQTT(mqttImage string) error
	RemoveMQTT() error
	GetImageDigest(image string) (string, error)
}

func NewContainerRuntime(endpoint, cgroupDriver string) (ContainerRuntime, error) {
	var runtime ContainerRuntime
	imageService, err := remote.NewRemoteImageService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())
	if err != nil {
		return runtime, err
	}
	runtimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())
	if err != nil {
		return runtime, err
	}
	runtime = &CRIRuntime{
		endpoint:            endpoint,
		cgroupDriver:        cgroupDriver,
		ImageManagerService: imageService,
		RuntimeService:      runtimeService,
		ctx:                 context.Background(),
	}

	return runtime, nil
}

type CRIRuntime struct {
	endpoint            string
	cgroupDriver        string
	ImageManagerService internalapi.ImageManagerService
	RuntimeService      internalapi.RuntimeService
	ctx                 context.Context
}

func convertCRIImage(image string) string {
	imageSeg := strings.Split(image, "/")
	if len(imageSeg) == 1 {
		return "docker.io/library/" + image
	} else if len(imageSeg) == 2 {
		return "docker.io/" + image
	}
	return image
}

func (runtime *CRIRuntime) PullImages(images []string) error {
	for _, image := range images {
		fmt.Printf("Pulling %s ...\n", image)
		err := runtime.PullImage(image, nil, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully pulled %s\n", image)
	}
	return nil
}

func (runtime *CRIRuntime) GetImageDigest(image string) (string, error) {
	image = convertCRIImage(image)
	imageSpec := &runtimeapi.ImageSpec{Image: image}
	imageStatus, err := runtime.ImageManagerService.ImageStatus(runtime.ctx, imageSpec, true)
	if err != nil {
		return "", err
	}
	imageDigest := imageStatus.Image.Spec.Image
	return imageDigest, nil
}

func (runtime *CRIRuntime) PullImage(image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error {
	image = convertCRIImage(image)
	imageSpec := &runtimeapi.ImageSpec{Image: image}
	status, err := runtime.ImageManagerService.ImageStatus(runtime.ctx, imageSpec, true)
	if err != nil {
		return err
	}
	if status == nil || status.Image == nil {
		if _, err := runtime.ImageManagerService.PullImage(runtime.ctx, imageSpec, authConfig, sandboxConfig); err != nil {
			return err
		}
	}
	return nil
}

// CopyResources copies binary and configuration file from the image to the host.
// The same way as func (runtime *DockerRuntime) CopyResources
func (runtime *CRIRuntime) CopyResources(edgeImage string, files map[string]string) error {
	psc := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      KubeEdgeBinaryName,
			Uid:       uuid.New().String(),
			Namespace: constants.SystemNamespace,
		},
		Linux: &runtimeapi.LinuxPodSandboxConfig{
			SecurityContext: &runtimeapi.LinuxSandboxSecurityContext{
				NamespaceOptions: &runtimeapi.NamespaceOption{
					Network: runtimeapi.NamespaceMode_NODE,
				},
				Privileged: true,
			},
		},
	}
	if runtime.cgroupDriver == v1alpha2.CGroupDriverSystemd {
		cgroupName := cm.NewCgroupName(cm.CgroupName{"kubeedge", "setup", "podcopyresource"})
		psc.Linux.CgroupParent = cgroupName.ToSystemd()
	}
	sandbox, err := runtime.RuntimeService.RunPodSandbox(runtime.ctx, psc, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.RuntimeService.RemovePodSandbox(runtime.ctx, sandbox); err != nil {
			klog.V(3).ErrorS(err, "Remove pod sandbox failed", "containerID", sandbox)
		}
	}()

	var mounts []*runtimeapi.Mount
	for _, hostPath := range files {
		mounts = append(mounts, &runtimeapi.Mount{
			HostPath:      filepath.Dir(hostPath),
			ContainerPath: filepath.Join("/tmp", filepath.Dir(hostPath)),
		})
	}
	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: "container",
		},
		Image: &runtimeapi.ImageSpec{
			Image: edgeImage,
		},
		// Keep the container running by passing in a command that never ends.
		// so that we can ExecSync in the following operations,
		// to ensure that we can copy files from container to host totally and correctly
		Command: []string{
			"/bin/sh",
			"-c",
			"sleep infinity",
		},
		Mounts: mounts,
		Linux: &runtimeapi.LinuxContainerConfig{
			SecurityContext: &runtimeapi.LinuxContainerSecurityContext{
				Privileged: true,
			},
		},
	}
	containerID, err := runtime.RuntimeService.CreateContainer(runtime.ctx, sandbox, containerConfig, psc)
	if err != nil {
		return fmt.Errorf("create container failed: %v", err)
	}
	defer func() {
		if err := runtime.RuntimeService.RemoveContainer(runtime.ctx, containerID); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", containerID)
		}
	}()

	err = runtime.RuntimeService.StartContainer(runtime.ctx, containerID)
	if err != nil {
		return fmt.Errorf("start container failed: %v", err)
	}

	copyCmd := copyResourcesCmd(files)
	cmd := []string{
		"/bin/sh",
		"-c",
		copyCmd,
	}
	stdout, stderr, err := runtime.RuntimeService.ExecSync(runtime.ctx, containerID, cmd, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to exec copy cmd, err: %v, stderr: %s, stdout: %s", err, string(stderr), string(stdout))
	}

	return nil
}

func (runtime *CRIRuntime) RunMQTT(mqttImage string) error {
	mqttImage = convertCRIImage(mqttImage)
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
		Labels: mqttLabel,
		Linux: &runtimeapi.LinuxPodSandboxConfig{
			SecurityContext: &runtimeapi.LinuxSandboxSecurityContext{
				NamespaceOptions: &runtimeapi.NamespaceOption{
					Network: runtimeapi.NamespaceMode_POD,
					Pid:     runtimeapi.NamespaceMode_CONTAINER,
					Ipc:     runtimeapi.NamespaceMode_POD,
				},
			},
		},
	}
	sandbox, err := runtime.RuntimeService.RunPodSandbox(runtime.ctx, psc, "")
	if err != nil {
		return err
	}

	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{Name: image.EdgeMQTT},
		Image: &runtimeapi.ImageSpec{
			Image: mqttImage,
		},
		Mounts: []*runtimeapi.Mount{
			{
				ContainerPath: "/mosquitto",
				HostPath:      filepath.Join(KubeEdgeSocketPath, image.EdgeMQTT),
			},
		},
	}
	containerID, err := runtime.RuntimeService.CreateContainer(runtime.ctx, sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	return runtime.RuntimeService.StartContainer(runtime.ctx, containerID)
}

func (runtime *CRIRuntime) RemoveMQTT() error {
	sandboxFilter := &runtimeapi.PodSandboxFilter{
		LabelSelector: mqttLabel,
	}

	sandbox, err := runtime.RuntimeService.ListPodSandbox(runtime.ctx, sandboxFilter)
	if err != nil {
		fmt.Printf("List MQTT containers failed: %v\n", err)
		return err
	}

	for _, c := range sandbox {
		// by reference doc
		// RemovePodSandbox removes the sandbox. If there are running containers in the
		// sandbox, they should be forcibly removed.
		// so we can remove mqtt containers totally.
		err = runtime.RuntimeService.RemovePodSandbox(runtime.ctx, c.Id)
		if err != nil {
			fmt.Printf("failed to remove MQTT container: %v\n", err)
		}
	}

	return nil
}

func copyResourcesCmd(files map[string]string) string {
	var copyCmd string
	first := true

	for containerPath, hostPath := range files {
		if first {
			copyCmd = copyCmd + fmt.Sprintf("cp %s %s", containerPath, filepath.Join("/tmp", hostPath))
		} else {
			copyCmd = copyCmd + fmt.Sprintf(" && cp %s %s", containerPath, filepath.Join("/tmp", hostPath))
		}
		first = false
	}
	return copyCmd
}
