package constants

import (
	"k8s.io/api/core/v1"
)

// Config
const (
	DefaultKubeContentType = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace   = v1.NamespaceAll
	DefaultKubeQPS         = 100.0
	DefaultKubeBurst       = 10

	DefaultMessageLayer = "context"
)
