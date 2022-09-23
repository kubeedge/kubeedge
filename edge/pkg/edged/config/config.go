package config

import (
	"sync"

	"k8s.io/component-base/featuregate"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/kubelet"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha2.Edged
	*kubelet.Dependencies
	featuregate.FeatureGate
}

func InitConfigure(e *v1alpha2.Edged) {
	once.Do(func() {
		Config = Configure{
			Edged: *e,
		}
	})
}

func Convert_Config_EdgedFlag_To_config_KubeletFlag(in *v1alpha2.TailoredKubeletFlag, out *kubeletoptions.KubeletFlags) {
	out.KubeConfig = in.KubeConfig
	out.HostnameOverride = in.HostnameOverride
	out.NodeIP = in.NodeIP
	out.ContainerRuntime = in.ContainerRuntime
	out.DockerEndpoint = in.DockerEndpoint
	out.PodSandboxImage = in.PodSandboxImage
	out.ImagePullProgressDeadline = in.ImagePullProgressDeadline
	out.CNIConfDir = in.CNIConfDir
	out.CNIBinDir = in.CNIBinDir
	out.CNICacheDir = in.CNICacheDir
	out.NetworkPluginMTU = in.NetworkPluginMTU
	out.RemoteRuntimeEndpoint = in.RemoteRuntimeEndpoint
	out.RemoteImageEndpoint = in.RemoteImageEndpoint
	out.RegisterNode = in.RegisterNode
	out.RegisterSchedulable = in.RegisterSchedulable
	out.RootDirectory = in.RootDirectory
}
