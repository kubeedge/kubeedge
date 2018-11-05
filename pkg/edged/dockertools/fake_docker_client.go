package dockertools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerapi "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"golang.org/x/net/context"
)

const EMPTYVALUE byte = 0

var IMAGEPULL_ERROR error

type IFakeDockerClien interface {
	ServerVersion(ctx context.Context) (types.Version, error)
}

type FakeDockerClient struct {
	dockerapi.CommonAPIClient
	images     map[string]types.ImageSummary
	containers map[string]byte
}

func NewFakeDockerClient(imageList map[string]types.ImageSummary) *FakeDockerClient {
	client, _ := NewDockerClient("")
	containers := map[string]byte{
		"34554324045172153046": 0,
		"56345328674564353046": 0,
		"45632654791321864641": 0,
		"21873546854796345495": 0,
		"35497463546546494984": 0,
		"54245527252262555242": 0,
		"56867354989678324676": 0,
	}

	return &FakeDockerClient{
		images:          imageList,
		CommonAPIClient: client,
		containers:      containers,
	}
}

func (cli *FakeDockerClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := strconv.Itoa(rnd.Int())
	newImage := types.ImageSummary{ID: id, RepoTags: []string{ref}, RepoDigests: []string{}}
	if IMAGEPULL_ERROR != nil {
		return nil, IMAGEPULL_ERROR
	}
	cli.images[ref] = newImage
	event := jsonmessage.JSONMessage{ID: id}
	byteEvent, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(bytes.NewReader(byteEvent)), IMAGEPULL_ERROR
}

func (cli *FakeDockerClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	images := make([]types.ImageSummary, 0, 10)
	for _, image := range cli.images {
		images = append(images, image)
	}
	return images, nil
}

func (cli *FakeDockerClient) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	for k, image := range cli.images {
		if image.ID == imageID {
			delete(cli.images, k)
			return nil, nil
		}
	}
	return nil, fmt.Errorf("image %v not found", imageID)
}

func (cli *FakeDockerClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{
		Version:       "1.1",
		APIVersion:    "12.3.4",
		GitCommit:     "1234567890",
		GoVersion:     "1.9",
		Os:            "linux",
		KernelVersion: "14.04",
	}, nil
}

func (cli *FakeDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	if _, ok := cli.images[config.Image]; !ok {
		return container.ContainerCreateCreatedBody{}, fmt.Errorf("Error: No such image: %s", config.Image)
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := strconv.Itoa(rnd.Int())
	cli.containers[id] = EMPTYVALUE
	return container.ContainerCreateCreatedBody{ID: id}, nil
}

func (cli *FakeDockerClient) ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error {
	if _, ok := cli.containers[containerID]; !ok {
		return fmt.Errorf("Error response from daemon: {\"message\":\"No such container: %+v\"}", containerID)
	}
	return nil
}

func (cli *FakeDockerClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	if _, ok := cli.containers[containerID]; !ok {
		return fmt.Errorf("Error response from daemon: {\"message\":\"No such container: %+v\"}", containerID)
	}
	return nil
}

func (cli *FakeDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	if _, ok := cli.containers[containerID]; !ok {
		return fmt.Errorf("Error response from daemon: {\"message\":\"No such container: %+v\"}", containerID)
	}
	delete(cli.containers, containerID)
	return nil
}

func (cli *FakeDockerClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {

	status := []string{
		"Up 5 weeks",
		"Exited (143) 7 weeks ago",
		"Created",
		"Up 5 weeks (Paused)",
	}

	created := []int64{
		1520284303,
		1520384333,
		1523254333,
		1521284333,
		1520264437,
		1522288383,
		1522246336,
		1521284333,
		1523204333,
		1521254333,
		1524214333,
		1522289033,
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	containers := make([]types.Container, 0)
	for k := range cli.containers {
		containers = append(containers, types.Container{
			ID:      k,
			Created: created[rand.Intn(len(created))],
			Status:  status[rand.Intn(len(status))],
		})
	}
	return containers, nil
}

func (cli *FakeDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}
