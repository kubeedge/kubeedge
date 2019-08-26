package config

import (
	"github.com/kubeedge/kubeedge/common/constants"
)

// KubeInfo contains Kubernetes related configuration
type KubeInfo struct {
	// KubeMaster is the url of edge master(kube api server)
	KubeMaster string

	// KubeConfig is the config used connect to edge master
	KubeConfig string

	// KubeNamespace is the namespace to watch(default is NamespaceAll)
	KubeNamespace string
}

// NewKubeInfo create KubeInfo struct with default values
func newKubeInfo() *KubeInfo {
	return &KubeInfo{
		KubeNamespace: constants.DefaultKubeNamespace,
	}
}
