/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
Changes done are
1. For ImageServiceServer interface is been implemented here.
2. Directly call docker client methods for container and image operations
*/

package dockertools

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/net/context"

	dockerref "github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	dockerapi "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	dockerdigest "github.com/opencontainers/go-digest"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/util/flowcontrol"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/dockershim"
	"k8s.io/kubernetes/pkg/kubelet/gpu"
	"k8s.io/kubernetes/pkg/kubelet/images"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/util/version"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis/runtime/cri"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/containers"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/util"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/util/record"
)

const (
	defaultRequestTimeOut = 2 * time.Minute

	secretAK = "AK"
	secretSK = "SK"

	statusRunningPrefix = "Up"
	statusCreatedPrefix = "Created"
	statusExitedPrefix  = "Exited"
	statusPausedSuffix  = "Paused"
	//ZeroTime defines base time
	ZeroTime = "0001-01-01T00:00:00Z"
)

//DockerDefaultAddress is default address for docker
const DockerDefaultAddress = "unix:///var/run/docker.sock"

//PROJECTNAME needs to be specified, temporarily written as "southchina"
const PROJECTNAME = "southchina"

var (
	// ErrImageNeverPull : Required Image is absent on host and PullPolicy is NeverPullImage
	ErrImageNeverPull = errors.New("ErrImageNeverPull")
	// DockerAddress is address for docker
	DockerAddress = DockerDefaultAddress
)

// InitDockerAddress inits docker address
func InitDockerAddress(dockerAddress string) {
	DockerAddress = dockerAddress
	if DockerAddress == "" {
		DockerAddress = DockerDefaultAddress
	}
}

//DockerManager defines object structure of docker manager
type DockerManager struct {
	containers.ContainerManager
	imgManager images.ImageManager
	client     dockerapi.CommonAPIClient
}

// NewDockerClient gets a *dockerapi.Client, either using the endpoint passed in, or using
// DOCKER_HOST, DOCKER_TLS_VERIFY, and DOCKER_CERT path per their spec
func NewDockerClient(dockerEndpoint string) (dockerapi.CommonAPIClient, error) {
	if len(dockerEndpoint) > 0 {
		log.LOGGER.Infof("Connecting to docker on %s", dockerEndpoint)
		return dockerapi.NewClient(dockerEndpoint, "", nil, nil)
	}
	return dockerapi.NewEnvClient()
}

//NewDockerManager returns a docker manager object
func NewDockerManager(livenessManager proberesults.Manager, qps float32, burst int, backOff *flowcontrol.Backoff, serializeImagePulls bool, devicePluginEnabled bool, gpuManager gpu.GPUManager, interfaceName string) (*DockerManager, error) {
	var err error
	dm := &DockerManager{}
	client, err := NewDockerClient(DockerAddress)
	if err != nil {
		return nil, fmt.Errorf("new docker client failed: %s", err)
	}
	dm.client = client
	dm.imgManager = images.NewImageManager(record.NewEventRecorder(), dm, backOff, serializeImagePulls, qps, burst)
	dm.ContainerManager, err = containers.NewContainerManager(dm, livenessManager, backOff, devicePluginEnabled, gpuManager, interfaceName)

	return dm, err
}

//PullImage pulls an image from network to local storage
func (dm *DockerManager) PullImage(image kubecontainer.ImageSpec, pullSecrets []v1.Secret) (string, error) {
	dockerConfigEntrys, err := getDockerConfigEntryFromSecret(pullSecrets)
	if err != nil {
		return "", err
	}

	if len(dockerConfigEntrys) == 0 {
		err = dm.pullImage(image, nil)
		if err != nil {
			log.LOGGER.Errorf("", fmt.Errorf("docker manager pull image %s failed, err: %v", image.Image, err))
			return "", err
		}
		return "", nil
	}

	for _, entry := range dockerConfigEntrys {
		err = dm.pullImage(image, &entry)
		if err != nil {
			log.LOGGER.Errorf("", fmt.Errorf("docker manager pull image %s failed, err: %v", image.Image, err))
			return "", err
		}
		//else
		break
	}
	return "", nil
}

// GetImageRef is part of interface.
func (dm *DockerManager) GetImageRef(image kubecontainer.ImageSpec) (string, error) {
	log.LOGGER.Infof("GetImageRef not implemented yet.")
	return "", nil
}

// ImageStats is part of interface.
func (dm *DockerManager) ImageStats() (*kubecontainer.ImageStats, error) {
	log.LOGGER.Infof("GetImageRef not implemented yet.")
	return nil, nil
}

func (dm *DockerManager) pullImage(image kubecontainer.ImageSpec, dockerConfig *DockerConfigEntry) error {
	log.LOGGER.Infof("docker manager start to pull image %s.", image.Image)
	ctx := context.Background()
	opt := types.ImagePullOptions{}

	authConfig := types.AuthConfig{}
	if dockerConfig != nil {
		authConfig.Username = dockerConfig.Username
		authConfig.Password = dockerConfig.Password
		authConfig.Email = dockerConfig.Email
	}

	edcodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.LOGGER.Errorf("", fmt.Errorf("PullImage failed: %s ", err))
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(edcodedJSON)
	opt.RegistryAuth = authStr

	resp, err := dm.client.ImagePull(ctx, image.Image, opt)
	if err != nil {
		log.LOGGER.Errorf("", fmt.Errorf("PullImage failed: %s ", err))
		return err
	}
	defer resp.Close()

	d := json.NewDecoder(resp)

	for {
		var event jsonmessage.JSONMessage
		err := d.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.LOGGER.Errorf("%v", fmt.Errorf("PullImage failed: %s ", err))
			return err
		}
		if event.Error != nil {
			log.LOGGER.Errorf("%v", fmt.Errorf("PullImage failed: %s ", event.Error.Error()))
			return event.Error
		}
		log.LOGGER.Infof("%v", event)
	}

	log.LOGGER.Infof("docker pull image %s successfully.", image.Image)
	return nil
}

// IsImagePresent checks whether the container image is already in the local storage
func (dm *DockerManager) IsImagePresent(image kubecontainer.ImageSpec) (bool, error) {
	respImages, err := dm.ListImages()
	if err != nil {
		return false, err
	}
	for _, respImage := range respImages {
		for _, imageName := range respImage.RepoTags {
			if image.Image == imageName {
				return true, nil
			}
		}
	}
	return false, nil
}

// ListImages gets all images currently on the machine
func (dm *DockerManager) ListImages() ([]kubecontainer.Image, error) {
	ctx := context.Background()
	opt := types.ImageListOptions{}
	respImages, err := dm.client.ImageList(ctx, opt)
	if err != nil {
		log.LOGGER.Errorf("%v", fmt.Errorf("ListImages failed: %s ", err))
		return nil, err
	}
	imageSlice := make([]kubecontainer.Image, 0, len(respImages))
	for _, respImage := range respImages {
		image := kubecontainer.Image{}
		image.ID = respImage.ID
		image.RepoDigests = respImage.RepoDigests
		image.RepoTags = respImage.RepoTags
		image.Size = respImage.Size
		imageSlice = append(imageSlice, image)
	}
	return imageSlice, nil
}

// RemoveImage removes the specified image
func (dm *DockerManager) RemoveImage(image kubecontainer.ImageSpec) error {
	log.LOGGER.Infof("docker manager start to remove image [%s]", image.Image)
	ctx := context.Background()
	opt := types.ImageRemoveOptions{}
	_, err := dm.client.ImageRemove(ctx, image.Image, opt)
	if err != nil {
		log.LOGGER.Errorf("%v", fmt.Errorf("RemoveImage failed: %s ", err))
		return err
	}
	log.LOGGER.Infof("docker manager remove image [%s] successfully", image.Image)
	return nil
}

// InspectImageByID checks if the inspected image matches what we are looking for
func (dm *DockerManager) InspectImageByID(imageID string) (*types.ImageInspect, error) {
	resp, err := dm.inspectImageRaw(imageID)
	if err != nil {
		return nil, err
	}

	if !matchImageIDOnly(*resp, imageID) {
		return nil, imageNotFoundError{ID: imageID}
	}
	return resp, nil
}

func (dm *DockerManager) inspectImageRaw(ref string) (*types.ImageInspect, error) {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	resp, _, err := dm.client.ImageInspectWithRaw(ctx, ref)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		if dockerapi.IsErrImageNotFound(err) {
			err = imageNotFoundError{ID: ref}
		}
		return nil, err
	}

	return &resp, nil
}

// matchImageIDOnly checks that the given image specifier is a digest-only
// reference, and that it matches the given image.
func matchImageIDOnly(inspected types.ImageInspect, image string) bool {
	// If the image ref is literally equal to the inspected image's ID,
	// just return true here (this might be the case for Docker 1.9,
	// where we won't have a digest for the ID)
	if inspected.ID == image {
		return true
	}

	// Otherwise, we should try actual parsing to be more correct
	ref, err := dockerref.Parse(image)
	if err != nil {
		log.LOGGER.Infof("couldn't parse image reference %q: %v", image, err)
		return false
	}

	digest, isDigested := ref.(dockerref.Digested)
	if !isDigested {
		log.LOGGER.Infof("the image reference %q was not a digest reference")
		return false
	}

	id := dockerdigest.Digest(inspected.ID)
	err = id.Validate()
	if err != nil {
		log.LOGGER.Infof("couldn't parse image ID reference %q: %v", id, err)
		return false
	}

	if digest.Digest().Algorithm().String() == id.Algorithm().String() && digest.Digest().Hex() == id.Hex() {
		return true
	}

	log.LOGGER.Infof("The reference %s does not directly refer to the given image's ID (%q)", image, inspected.ID)
	return false
}

//Version returns kubecontainer version
func (dm *DockerManager) Version() (kubecontainer.Version, error) {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	resp, err := dm.client.ServerVersion(ctx)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	runtimeVersion := resp.KernelVersion
	if ver, err := version.ParseSemantic(runtimeVersion); err == nil {
		return ver, err
	}
	return version.ParseGeneric(runtimeVersion)
}

//CreateContainer creates container and returns ID
func (dm *DockerManager) CreateContainer(config *cri.ContainerConfig) (string, error) {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	dockerNetwork := &network.NetworkingConfig{}
	createResp, err := dm.client.ContainerCreate(ctx, config.Config, config.HostConfig, dockerNetwork, config.Name)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return "", ctxErr
	}
	if err != nil {
		return "", err
	}
	return createResp.ID, nil
}

//StartContainer is for ContainerStart
func (dm *DockerManager) StartContainer(containerID string) error {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	err := dm.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	return err
}

//StopContainer is to stop the container given its ID
func (dm *DockerManager) StopContainer(containerID string, timeout uint32) error {
	t := time.Duration(timeout) * time.Second
	ctx, cancel := dm.getCustomTimeoutContext(t)
	defer cancel()
	err := dm.client.ContainerStop(ctx, containerID, &t)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	return err
}

//DeleteContainer deletes container given ID
func (dm *DockerManager) DeleteContainer(containerID kubecontainer.ContainerID) error {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()

	opts := types.ContainerRemoveOptions{}
	err := dm.client.ContainerRemove(ctx, containerID.ID, opts)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	return err
}

//ListContainers returns all containers list
func (dm *DockerManager) ListContainers() ([]*cri.Container, error) {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	dcontainers, err := dm.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if ctxErr := contextError(ctx); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	containers := make([]*cri.Container, 0)
	for _, v := range dcontainers {
		container := &cri.Container{
			ID:      v.ID,
			StartAt: time.Unix(v.Created, 0),
			Status:  toEdgedStatus(v.Status),
		}
		containers = append(containers, container)
	}

	return containers, nil
}

//InspectContainer is to inspect the container returning ContainerInspect object
func (dm *DockerManager) InspectContainer(containerID string) (*cri.ContainerInspect, error) {
	ctx, cancel := dm.getTimeoutContext()
	defer cancel()
	containerJSON, err := dm.client.ContainerInspect(ctx, containerID)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		if dockerapi.IsErrContainerNotFound(err) {
			return nil, containerNotFoundError{ID: containerID}
		}
		return nil, err
	}
	status := cri.ContainerStatus{
		ContainerStatus: kubecontainer.ContainerStatus{
			ID: kubecontainer.ContainerID{
				ID: containerJSON.ID,
			},
			Name:     containerJSON.Name,
			ExitCode: containerJSON.State.ExitCode,
			Image:    containerJSON.Image,
			Reason:   containerJSON.State.Error,
			Message:  containerJSON.State.Error,
		},
		Labels:       containerJSON.Config.Labels,
		LogPath:      containerJSON.LogPath,
		RestartCount: int32(containerJSON.RestartCount),
	}

	if cname, ok := status.Labels[containers.KubernetesContainerNameLabel]; ok {
		status.Name = cname
	}

	status.CreatedAt = convertTime(containerJSON.Created)
	status.StartedAt = convertTime(containerJSON.State.StartedAt)
	status.FinishedAt = convertTime(containerJSON.State.FinishedAt)
	if containerJSON.State.StartedAt == ZeroTime {
		status.StartedAt = time.Unix(0, 0)
	}
	if containerJSON.State.FinishedAt == ZeroTime {
		status.FinishedAt = time.Unix(0, 0)
	}
	status.State = kubecontainer.ContainerState(containerJSON.State.Status)
	setStatusReason(&status)
	return &cri.ContainerInspect{Status: status}, nil
}

func setStatusReason(status *cri.ContainerStatus) {
	if status.State == cri.StatusCREATED &&
		status.ExitCode != 0 && status.FinishedAt.IsZero() {
		status.Reason = "ContainerCannotRun"
	}
}

func toEdgedStatus(state string) kubecontainer.ContainerState {
	switch {
	case strings.Contains(state, statusPausedSuffix):
		return cri.StatusPAUSED
	case strings.HasPrefix(state, statusRunningPrefix):
		return cri.StatusRUNNING
	case strings.HasPrefix(state, statusExitedPrefix):
		return cri.StatusEXITED
	case strings.HasPrefix(state, statusCreatedPrefix):
		return cri.StatusCREATED
	default:
		return cri.StatusUNKNOWN
	}
}

//ContainerStatus gives container status
func (dm *DockerManager) ContainerStatus(containerID string) (*cri.ContainerStatus, error) {

	containerInspect, err := dm.InspectContainer(containerID)
	if err != nil {
		return nil, err
	}
	image := containerInspect.Status.Image
	ir, err := dm.InspectImageByID(image)
	if err != nil {
		return nil, fmt.Errorf("unable to inspect docker image %q while inspecting docker container %q: %v", image, containerID, err)
	}
	containerInspect.Status.ImageRef = toPullableImageID(image, ir)
	containerInspect.Status.Image = ir.ContainerConfig.Image
	return &containerInspect.Status, nil
}

//EnsureImageExists checks existence of image in container
func (dm *DockerManager) EnsureImageExists(pod *v1.Pod, secrets []v1.Secret) error {
	log.LOGGER.Infof("start to pull image for pod %s", pod.Name)

	for _, container := range pod.Spec.Containers {
		_, msg, err := dm.imgManager.EnsureImageExists(pod, &container, secrets)
		if err != nil {
			log.LOGGER.Errorf("DockerManager EnsureImageExists failed, msg:%s, %v", msg, err)
			return err
		}
	}
	return nil
}

// operationTimeout is the error returned when the docker operations are timeout.
type operationTimeout struct {
	err error
}

func (e operationTimeout) Error() string {
	return fmt.Sprintf("operation timeout: %v", e.err)
}

func contextError(ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return operationTimeout{err: ctx.Err()}
	}
	return ctx.Err()
}

// getTimeoutContext returns a new context with default request timeout
func (dm *DockerManager) getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultRequestTimeOut)
}

// getCustomTimeoutContext returns a new context with a specific request timeout
func (dm *DockerManager) getCustomTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	// Pick the larger of the two
	if defaultRequestTimeOut > timeout {
		timeout = defaultRequestTimeOut
	}
	return context.WithTimeout(context.Background(), timeout)
}

type containerNotFoundError struct {
	ID string
}

func (e containerNotFoundError) Error() string {
	return fmt.Sprintf("no such container: %q", e.ID)
}

func toPullableImageID(id string, image *types.ImageInspect) string {
	// Default to the image ID, but if RepoDigests is not empty, use
	// the first digest instead.
	imageID := cri.DockerImageIDPrefix + id
	if len(image.RepoDigests) > 0 {
		imageID = cri.DockerPullableImageIDPrefix + image.RepoDigests[0]
	}
	return imageID
}

type imageNotFoundError struct {
	ID string
}

func (e imageNotFoundError) Error() string {
	return fmt.Sprintf("no such image: %q", e.ID)
}

//IsImageNotFoundError checks error to be of type imageNotFoundError
func IsImageNotFoundError(err error) bool {
	_, ok := err.(imageNotFoundError)
	return ok
}

func convertTime(stringTime string) time.Time {
	numTime, _ := util.ParseTimestampStr2Int64(stringTime)
	metav1Time := util.ParseTimestampInt64(numTime)
	return metav1Time.Time
}

//NewDockerConfig configures docker client
// TODO: fillup other field in this struct
func NewDockerConfig() *dockershim.ClientConfig {
	return &dockershim.ClientConfig{
		DockerEndpoint: DockerAddress,
	}
}
