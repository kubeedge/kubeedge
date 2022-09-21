package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeCoreEdgeRelay
	relayID string
	// 本机ID
	nodeID string
}

func InitConfig(er *v1alpha1.EdgeCoreEdgeRelay, nodeID string) {
	once.Do(func() {
		Config = Configure{
			EdgeCoreEdgeRelay: *er,
			relayID:           "",
			nodeID:            nodeID,
		}
	})
}
