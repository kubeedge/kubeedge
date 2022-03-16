package util

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	internalapi "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"

	"github.com/kubeedge/kubeedge/pkg/image"
)

type ContainerRuntime interface {
	PullImages(images []string) error
	CopyResources(edgeImage string, dirs map[string]string, files map[string]string) error
	RunMQTT(mqttImage string) error
}

func NewContainerRuntime(runtimeType string, endpoint string) (ContainerRuntime, error) {
	var runtime ContainerRuntime
	switch runtimeType {
	case kubetypes.DockerContainerRuntime:
		cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
		if err != nil {
			return runtime, fmt.Errorf("init docker client failed: %v", err)
		}

		ctx := context.Background()
		cli.NegotiateAPIVersion(ctx)

		runtime = &DockerRuntime{
			Client: cli,
			ctx:    ctx,
		}
	case kubetypes.RemoteContainerRuntime:
		imageService, err := remote.NewRemoteImageService(endpoint, time.Second*10)
		if err != nil {
			return runtime, err
		}
		runtimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10)
		if err != nil {
			return runtime, err
		}
		runtime = &CRIRuntime{
			endpoint:            endpoint,
			ImageManagerService: imageService,
			RuntimeService:      runtimeService,
		}
	default:
		return runtime, fmt.Errorf("unsupport CRI runtime: %s", runtimeType)
	}

	return runtime, nil
}

type DockerRuntime struct {
	Client *dockerclient.Client
	ctx    context.Context
}

func (runtime *DockerRuntime) PullImages(images []string) error {
	for _, image := range images {
		fmt.Printf("Pulling %s ...\n", image)
		args := filters.NewArgs()
		args.Add("reference", image)
		list, err := runtime.Client.ImageList(runtime.ctx, dockertypes.ImageListOptions{Filters: args})
		if err != nil {
			return err
		}
		if len(list) > 0 {
			continue
		}

		rc, err := runtime.Client.ImagePull(runtime.ctx, image, dockertypes.ImagePullOptions{})
		if err != nil {
			return err
		}

		io.Copy(io.Discard, rc)
		rc.Close()
		fmt.Printf("Successfully pulled %s\n", image)
	}

	return nil
}

func (runtime *DockerRuntime) RunMQTT(mqttImage string) error {
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
			filepath.Join(KubeEdgeSocketPath, image.EdgeMQTT) + ":/mosquitto",
		},
	}
	config := &dockercontainer.Config{Image: mqttImage}

	container, err := runtime.Client.ContainerCreate(runtime.ctx, config, hostConfig, nil, nil, image.EdgeMQTT)
	if err != nil {
		return err
	}
	return runtime.Client.ContainerStart(runtime.ctx, container.ID, dockertypes.ContainerStartOptions{})
}

// CopyResources copies binary and configuration file from the image to the host.
// The command it executes are as follows:
//
// docker run -v /etc/kubeedge:/tmp/kubeedge/data -v /usr/local/bin:/tmp/kubeedge/bin <IMAGE-NAME> \
// bash -c cp -r /etc/kubeedge:/tmp/kubeedge/data cp /usr/local/bin/edgecore:/tmp/kubeedge/bin/edgecore
func (runtime *DockerRuntime) CopyResources(image string, dirs map[string]string, files map[string]string) error {
	if len(files) == 0 && len(dirs) == 0 {
		return fmt.Errorf("no resources need copying")
	}

	copyCmd := copyResourcesCmd(dirs, files)

	config := &dockercontainer.Config{
		Image: image,
		Cmd: []string{
			"/bin/sh",
			"-c",
			copyCmd,
		},
	}
	var binds []string
	for origin, bind := range dirs {
		binds = append(binds, origin+":"+bind)
	}
	for origin, bind := range files {
		binds = append(binds, filepath.Dir(origin)+":"+filepath.Dir(bind))
	}

	hostConfig := &dockercontainer.HostConfig{
		Binds: binds,
	}

	// Randomly generate container names to prevent duplicate names.
	container, err := runtime.Client.ContainerCreate(runtime.ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.Client.ContainerRemove(runtime.ctx, container.ID, dockertypes.ContainerRemoveOptions{}); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", container.ID)
		}
	}()
	return runtime.Client.ContainerStart(runtime.ctx, container.ID, dockertypes.ContainerStartOptions{})
}

type CRIRuntime struct {
	endpoint            string
	ImageManagerService internalapi.ImageManagerService
	RuntimeService      internalapi.RuntimeService
}

func (runtime *CRIRuntime) PullImages(images []string) error {
	for _, image := range images {
		fmt.Printf("Pulling %s ...\n", image)
		imageSpec := &runtimeapi.ImageSpec{Image: image}
		status, err := runtime.ImageManagerService.ImageStatus(imageSpec)
		if err != nil {
			return err
		}
		if status == nil || status.Id == "" {
			if _, err := runtime.ImageManagerService.PullImage(imageSpec, nil, nil); err != nil {
				return err
			}
		}
		fmt.Printf("Successfully pulled %s\n", image)
	}

	return nil
}

// CopyResources copies binary and configuration file from the image to the host.
// The same way as func (runtime *DockerRuntime) CopyResources
func (runtime *CRIRuntime) CopyResources(edgeImage string, dirs map[string]string, files map[string]string) error {

	psc := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{Name: KubeEdgeBinaryName},
	}
	sandbox, err := runtime.RuntimeService.RunPodSandbox(psc, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.RuntimeService.RemovePodSandbox(sandbox); err != nil {
			klog.V(3).ErrorS(err, "Remove pod sandbox failed", "containerID", sandbox)
		}
	}()

	copyCmd := copyResourcesCmd(dirs, files)
	var mounts []*runtimeapi.Mount
	for origin, bind := range dirs {
		mounts = append(mounts, &runtimeapi.Mount{
			HostPath:      origin,
			ContainerPath: bind,
		})
	}
	for origin, bind := range files {
		mounts = append(mounts, &runtimeapi.Mount{
			HostPath:      filepath.Dir(origin),
			ContainerPath: filepath.Dir(bind),
		})
	}
	containerConfig := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: "container",
		},
		Image: &runtimeapi.ImageSpec{
			Image: edgeImage,
		},
		Command: []string{
			"/bin/sh",
			"-c",
			copyCmd,
		},
		Mounts: mounts,
	}
	containerID, err := runtime.RuntimeService.CreateContainer(sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.RuntimeService.RemoveContainer(containerID); err != nil {
			klog.V(3).ErrorS(err, "Remove container failed", "containerID", containerID)
		}
	}()

	return runtime.RuntimeService.StartContainer(containerID)
}

func (runtime *CRIRuntime) RunMQTT(mqttImage string) error {
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
	sandbox, err := runtime.RuntimeService.RunPodSandbox(psc, "")
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
	containerID, err := runtime.RuntimeService.CreateContainer(sandbox, containerConfig, psc)
	if err != nil {
		return err
	}
	return runtime.RuntimeService.StartContainer(containerID)
}

func copyResourcesCmd(dirs map[string]string, files map[string]string) string {
	var copyCmd string
	first := true
	for origin, bind := range dirs {
		if first {
			copyCmd = copyCmd + fmt.Sprintf("cp -r %s %s", origin, bind)
		} else {
			copyCmd = copyCmd + fmt.Sprintf(" && cp -r %s %s", origin, bind)
		}
		first = false
	}
	for origin, bind := range files {
		if first {
			copyCmd = copyCmd + fmt.Sprintf("cp %s %s", origin, bind)
		} else {
			copyCmd = copyCmd + fmt.Sprintf(" && cp %s %s", origin, bind)
		}
		first = false
	}
	return copyCmd
}
