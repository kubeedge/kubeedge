package dockertools

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/util/flowcontrol"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis/runtime/cri"
)

const (
	backOffPeriod       = 10 * time.Second
	MaxContainerBackOff = 300 * time.Second
)

var imageList = map[string]types.ImageSummary{
	"registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest":  {ID: "1234567890", RepoTags: []string{"registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest"}, RepoDigests: []string{}, Size: 100},
	"registry.southchina.huaweicloud.com/dgh/edge-demo-app2:latest": {ID: "1234567891", RepoTags: []string{"registry.southchina.huaweicloud.com/dgh/edge-demo-app2:latest"}, RepoDigests: []string{}, Size: 100},
}

func getDockerManager() (*DockerManager, error) {
	backoff := flowcontrol.NewBackOff(backOffPeriod, MaxContainerBackOff)
	dm, err := NewDockerManager(nil, 0, 0, backoff, true, false, nil, "")
	if err != nil {
		return nil, err
	}
	dm.client = NewFakeDockerClient(imageList)
	return dm, nil
}

func TestPullImage(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v ", err)
	}

	image := kubecontainer.ImageSpec{"testPullImage:latest"}

	ImagePullError = fmt.Errorf("test error")
	_, err = dm.PullImage(image, []v1.Secret{})
	if !reflect.DeepEqual(err, ImagePullError) {
		t.Errorf("TestPullImage failed, test error failed, err(%v) != IMAGEPULL_ERROR(%v)", err, ImagePullError)
	}

	ImagePullError = nil
	_, err = dm.PullImage(image, []v1.Secret{})
	if err != nil {
		t.Errorf("TestPullImage failed, pull image failed err: %v", err)
	}
	present, err := dm.IsImagePresent(image)
	if !present || err != nil {
		t.Errorf("Check if the image exists,present: %v err: %v", present, err)
	}

	id := getImageIDByName(dm, image.Image)
	dm.RemoveImage(kubecontainer.ImageSpec{id})
}

func TestListImages(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v ", err)
	}

	images, err := dm.ListImages()
	if err != nil {
		t.Errorf("list images failed, err: %v", err)
	}

	if len(imageList) != len(images) {
		t.Errorf("TestListImages failed, len(imageList) != len(images)")
	}

	for _, initImage := range imageList {
		exist := false
		for _, image := range images {
			if initImage.ID == image.ID {
				exist = true
			}
		}
		if !exist {
			t.Errorf("TestListImages failed, image(%v) not found", initImage)
		}
	}
}

func TestIsImagePresent(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v ", err)
	}

	notExistImage := kubecontainer.ImageSpec{Image: "notExist:latest"}
	exist, err := dm.IsImagePresent(notExistImage)
	if exist || err != nil {
		t.Errorf("TestIsImagePresent failed, image: %v, exist: %v, err: %v", notExistImage, exist, err)
	}

	existImage := kubecontainer.ImageSpec{Image: "registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest"}
	exist, err = dm.IsImagePresent(existImage)
	if !exist || err != nil {
		t.Errorf("TestIsImagePresent failed, image: %v, exist: %v, err: %v", existImage, exist, err)
	}

}

func getImageIDByName(dm *DockerManager, name string) string {
	images, err := dm.ListImages()
	if err != nil {
		return ""
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if name == tag {
				return image.ID
			}
		}
	}
	return ""
}

func TestRemoveImage(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v ", err)
	}

	newImage := kubecontainer.ImageSpec{"willRemove:latest"}
	_, err = dm.PullImage(newImage, []v1.Secret{})
	if err != nil {
		t.Errorf("TestRemoveImage failed, pull Image(%v) failed", newImage)
	}

	id := getImageIDByName(dm, newImage.Image)
	if id == "" {
		t.Errorf("TestRemoveImage failed, getImageIdByName failed")
	}

	err = dm.RemoveImage(kubecontainer.ImageSpec{id})
	if err != nil {
		t.Errorf("TestRemoveImage failed, image: %v, err: %v", newImage, err)
	}

	notExistImage := kubecontainer.ImageSpec{Image: "1234"}
	err = dm.RemoveImage(notExistImage)
	if err == nil {
		t.Errorf("TestRemoveImage failed, image: %v, err: %v", notExistImage, err)
	}
}

func TestCreateContainer(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	config := cri.ContainerConfig{
		Name: "qcjtest",
		Config: &container.Config{
			Hostname: "127.0.0.1",
			Image:    "registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest",
		},
	}
	_, err = dm.CreateContainer(&config)
	if err != nil {
		t.Errorf("test create container failed, err [%v]", err)
	}
	config2 := cri.ContainerConfig{
		Name: "qcjtest",
		Config: &container.Config{
			Hostname: "127.0.0.1",
			Image:    "registry.southchina.huaweicloud.com/dgh/edge-demo-app3:latest",
		},
	}
	_, err = dm.CreateContainer(&config2)
	if err == nil {
		t.Errorf("test create container failed, err [%v]", err)
	}
}

func TestStartContainer(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	config := cri.ContainerConfig{
		Name: "qcjtest",
		Config: &container.Config{
			Hostname: "127.0.0.1",
			Image:    "registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest",
		},
	}
	id, err := dm.CreateContainer(&config)
	if err != nil {
		t.Errorf("test create container failed, err [%v]", err)
	}
	err = dm.StartContainer(id)
	if err != nil {
		t.Errorf("test start container failed, err [%v]", err)
	}
	err = dm.StartContainer("123456")
	if err == nil {
		t.Errorf("test start container failed, err [%v]", err)
	}
	err = dm.StopContainer(id, 30)
	if err != nil {
		t.Errorf("test stop container failed, err [%v]", err)
	}
	err = dm.DeleteContainer(kubecontainer.ContainerID{ID: id})
	if err != nil {
		t.Errorf("test remove container failed, err [%v]", err)
	}
}

func TestStopContainer(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	config := cri.ContainerConfig{
		Name: "qcjtest",
		Config: &container.Config{
			Hostname: "127.0.0.1",
			Image:    "registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest",
		},
	}
	id, err := dm.CreateContainer(&config)
	if err != nil {
		t.Errorf("test create container failed, err [%v]", err)
	}
	err = dm.StartContainer(id)
	if err != nil {
		t.Errorf("test start container failed, err [%v]", err)
	}
	err = dm.StopContainer(id, 30)
	if err != nil {
		t.Errorf("test stop container failed, err [%v]", err)
	}
	err = dm.StopContainer("123456", 30)
	if err == nil {
		t.Errorf("test stop container failed, err [%v]", err)
	}
	err = dm.DeleteContainer(kubecontainer.ContainerID{ID: id})
	if err != nil {
		t.Errorf("test remove container failed, err [%v]", err)
	}
}

func TestRemoveContainer(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	config := cri.ContainerConfig{
		Name: "qcjtest",
		Config: &container.Config{
			Hostname: "127.0.0.1",
			Image:    "registry.southchina.huaweicloud.com/dgh/edge-demo-app:latest",
		},
	}
	id, err := dm.CreateContainer(&config)
	if err != nil {
		t.Errorf("test create container failed, err [%v]", err)
	}
	err = dm.StartContainer(id)
	if err != nil {
		t.Errorf("test start container failed, err [%v]", err)
	}
	err = dm.StopContainer(id, 30)
	if err != nil {
		t.Errorf("test stop container failed, err [%v]", err)
	}
	err = dm.DeleteContainer(kubecontainer.ContainerID{ID: id})
	if err != nil {
		t.Errorf("test remove container failed, err [%v]", err)
	}
	err = dm.DeleteContainer(kubecontainer.ContainerID{ID: "123456"})
	if err == nil {
		t.Errorf("test remove container failed, err [%v]", err)
	}
}

func TestVersion(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	version, err := dm.Version()
	if err != nil {
		t.Errorf("test Version failed, err [%v]", err)
	}
	t.Logf("test version: %+v\n", version)
}

func TestListContainers(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	containers, err := dm.ListContainers()
	if err != nil {
		t.Errorf("test TestListContainers failed, err [%v]", err)
	}

	for _, v := range containers {
		t.Logf("test TestListContainers: %+v\n", v)
	}
}

func TestEnsureImageExists(t *testing.T) {
	dm, err := getDockerManager()
	if err != nil {
		t.Errorf("test new docker manager failed. %v", err)
	}
	data := make(map[string][]byte)
	//data[".dockerconfigjson"] = []byte("eyJhdXRocyI6eyJyZWdpc3RyeS5zb3V0aGNoaW5hLmh1YXdlaWNsb3VkLmNvbSI6eyJhdXRoIjoiYzI5MWRHaGphR2x1WVVBNVkyaEpiMWg2Ym5CMVdtVlRNV0YwWm5JeGN6cGtNR0V3TkRSak5UUTJNREF6Wm1Wa05UUTBOMlUwWmpjNFpETXpORGRpWmpRM05XVTJZV000TVdVME1HVmlZMlZsWWpVeU56TmlZakJpTm1WaVlXSTUifX19")
	data[".dockerconfigjson"] = []byte(`{"auths":{"registry.southchina.huaweicloud.com":{"auth":"c291dGhjaGluYUA5Y2hJb1h6bnB1WmVTMWF0ZnIxczpkMGEwNDRjNTQ2MDAzZmVkNTQ0N2U0Zjc4ZDMzNDdiZjQ3NWU2YWM4MWU0MGViY2VlYjUyNzNiYjBiNmViYWI5"}}}`)
	secret := v1.Secret{
		Data: data,
		Type: "kubernetes.io/dockerconfigjson",
	}
	secrets := []v1.Secret{secret}
	container := v1.Container{
		ImagePullPolicy: v1.PullAlways,
		Image:           "nginx:latest",
	}
	containers := []v1.Container{container}
	podSpec := v1.PodSpec{
		Containers: containers,
	}

	pod := v1.Pod{
		Spec: podSpec,
	}
	dm.EnsureImageExists(&pod, secrets)
}
