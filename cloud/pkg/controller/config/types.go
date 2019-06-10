package config

import (
	"time"

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

	// KubeContentType is the content type communicate with edge master(default is "application/vnd.kubernetes.protobuf")
	KubeContentType string

	// KubeQPS is the QPS communicate with edge master(default is 1024)
	KubeQPS float32

	// KubeBurst default is 10
	KubeBurst int

	// NodeID for the current node
	KubeNodeID string

	// NodeName for the current node
	KubeNodeName string

	// KubeUpdateNodeFrequency is the time duration for update node status(default is 20s)
	KubeUpdateNodeFrequency time.Duration
}

// NewKubeInfo create KubeInfo struct with default values
func NewKubeInfo() *KubeInfo {
	return &KubeInfo{
		KubeNamespace:           constants.DefaultKubeNamespace,
		KubeContentType:         constants.DefaultKubeContentType,
		KubeQPS:                 constants.DefaultKubeQPS,
		KubeBurst:               constants.DefaultKubeBurst,
		KubeUpdateNodeFrequency: constants.DefaultKubeUpdateNodeFrequency * time.Second,
	}
}
