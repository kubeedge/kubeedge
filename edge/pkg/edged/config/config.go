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

func ConvertConfigEdgedFlagToConfigKubeletFlag(in *v1alpha2.TailoredKubeletFlag, out *kubeletoptions.KubeletFlags) {
	out.KubeConfig = in.KubeConfig
	out.HostnameOverride = in.HostnameOverride
	out.NodeIP = in.NodeIP
	out.RootDirectory = in.RootDirectory
	out.RemoteRuntimeEndpoint = in.RemoteRuntimeEndpoint
	out.RemoteImageEndpoint = in.RemoteImageEndpoint
	out.ExperimentalMounterPath = in.ExperimentalMounterPath
	out.ExperimentalCheckNodeCapabilitiesBeforeMount = in.ExperimentalCheckNodeCapabilitiesBeforeMount
	out.ExperimentalNodeAllocatableIgnoreEvictionThreshold = in.ExperimentalNodeAllocatableIgnoreEvictionThreshold
	out.NodeLabels = in.NodeLabels
	out.MinimumGCAge = in.MinimumGCAge
	out.MaxPerPodContainerCount = in.MaxPerPodContainerCount
	out.MaxContainerCount = in.MaxContainerCount
	out.MasterServiceNamespace = in.MasterServiceNamespace
	out.RegisterSchedulable = in.RegisterSchedulable
	out.NonMasqueradeCIDR = in.NonMasqueradeCIDR
	out.KeepTerminatedPodVolumes = in.KeepTerminatedPodVolumes
	out.SeccompDefault = in.SeccompDefault

	// container-runtime-specific options
	out.ContainerRuntime = in.ContainerRuntime
	out.RuntimeCgroups = in.RuntimeCgroups
	out.DockershimRootDirectory = in.DockershimRootDirectory
	out.PodSandboxImage = in.PodSandboxImage
	out.DockerEndpoint = in.DockerEndpoint
	out.ImagePullProgressDeadline = in.ImagePullProgressDeadline
	out.NetworkPluginName = in.NetworkPluginName
	out.NetworkPluginMTU = in.NetworkPluginMTU
	out.CNIConfDir = in.CNIConfDir
	out.CNIBinDir = in.CNIBinDir
	out.CNICacheDir = in.CNICacheDir
}
