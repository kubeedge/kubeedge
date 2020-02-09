package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
)

const (
	//DockerEndpoint gives the default endpoint for docker engine
	DockerEndpoint = "unix:///var/run/docker.sock"

	//RemoteRuntimeEndpoint gives the default endpoint for CRI runtime
	RemoteRuntimeEndpoint = "unix:///var/run/dockershim.sock"

	//RemoteContainerRuntime give Remote container runtime name
	RemoteContainerRuntime = "remote"

	//MinimumEdgedMemoryCapacity gives the minimum default memory (2G) of edge
	MinimumEdgedMemoryCapacity = 2147483647

	//PodSandboxImage gives the default pause container image
	PodSandboxImage = "k8s.gcr.io/pause"

	// ImagePullProgressDeadlineDefault gives the default image pull progress deadline
	ImagePullProgressDeadlineDefault = 60
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Edged
}

func InitConfigure(e *v1alpha1.Edged) {
	once.Do(func() {
		Config = Configure{
			Edged: *e,
		}
	})
}
