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
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
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
var mqttLabel = map[string]string{"io.kubeedge.edgecore/mqtt": EdgeMQTT}

type ContainerRuntime interface {
	CopyResources(ctx context.Context, edgeImage string, files map[string]string) error
	// RunMQTT create and start the mosquitto container service.
	// Deprecated: MQTT server is maintained by DaemonSet now.
	RunMQTT(ctx context.Context, mqttImage string) error
	// RemoveMQTT stop and remove the mosquitto container service.
	// Deprecated: MQTT server is maintained by DaemonSet now.
	RemoveMQTT(ctx context.Context) error

	image.Runtime
}

type containerRuntimeImpl struct {
	cgroupDriver string
	ctrsvc       internalapi.RuntimeService

	*image.RuntimeImpl
}

func NewContainerRuntime(endpoint, cgroupDriver string) (ContainerRuntime, error) {
	const timeout = 10 * time.Second
	imgrt, err := image.NewImageRuntime(endpoint, timeout)
	if err != nil {
		return nil, err
	}
	ctrsvc, err := remote.NewRemoteRuntimeService(endpoint, timeout, noop.NewTracerProvider())
	if err != nil {
		return nil, fmt.Errorf("failed to new remote runtime service, err: %v", err)
	}
	return &containerRuntimeImpl{
		RuntimeImpl:  imgrt,
		cgroupDriver: cgroupDriver,
		ctrsvc:       ctrsvc,
	}, nil
}

// CopyResources copies binary and configuration file from the image to the host.
// The same way as func (runtime *DockerRuntime) CopyResources
func (runtime *containerRuntimeImpl) CopyResources(
	ctx context.Context,
	edgeImage string,
	files map[string]string,
) error {
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
	sandbox, err := runtime.ctrsvc.RunPodSandbox(ctx, psc, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.ctrsvc.RemovePodSandbox(ctx, sandbox); err != nil {
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
	containerID, err := runtime.ctrsvc.CreateContainer(ctx, sandbox, containerConfig, psc)
	if err != nil {
		return fmt.Errorf("create container failed: %v", err)
	}
	defer func() {
		if err := runtime.ctrsvc.RemoveContainer(ctx, containerID); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", containerID)
		}
	}()

	err = runtime.ctrsvc.StartContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("start container failed: %v", err)
	}

	cmd := []string{"/bin/sh", "-c", copyResourcesCmd(files)}
	stdout, stderr, err := runtime.ctrsvc.ExecSync(ctx, containerID, cmd, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to exec copy cmd, err: %v, stderr: %s, stdout: %s", err, string(stderr), string(stdout))
	}

	return nil
}

func (runtime *containerRuntimeImpl) RunMQTT(ctx context.Context, mqttImage string) error {
	mqttImage = image.ConvToCRIImage(mqttImage)
	psc := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{Name: EdgeMQTT},
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
	sandbox, err := runtime.ctrsvc.RunPodSandbox(ctx, psc, "")
	if err != nil {
		return err
	}

	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{Name: EdgeMQTT},
		Image: &runtimeapi.ImageSpec{
			Image: mqttImage,
		},
		Mounts: []*runtimeapi.Mount{
			{
				ContainerPath: "/mosquitto",
				HostPath:      filepath.Join(KubeEdgeSocketPath, EdgeMQTT),
			},
		},
	}
	containerID, err := runtime.ctrsvc.CreateContainer(ctx, sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	return runtime.ctrsvc.StartContainer(ctx, containerID)
}

func (runtime *containerRuntimeImpl) RemoveMQTT(ctx context.Context) error {
	sandboxFilter := &runtimeapi.PodSandboxFilter{
		LabelSelector: mqttLabel,
	}

	sandbox, err := runtime.ctrsvc.ListPodSandbox(ctx, sandboxFilter)
	if err != nil {
		fmt.Printf("List MQTT containers failed: %v\n", err)
		return err
	}

	for _, c := range sandbox {
		// by reference doc
		// RemovePodSandbox removes the sandbox. If there are running containers in the
		// sandbox, they should be forcibly removed.
		// so we can remove mqtt containers totally.
		err = runtime.ctrsvc.RemovePodSandbox(ctx, c.Id)
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
