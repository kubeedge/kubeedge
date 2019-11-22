package pkg

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/proxy"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

//EdgeMesh defines EdgeMesh object structure
type EdgeMesh struct {
}

// Register register edgemesh
func Register() {
	core.Register(&EdgeMesh{})
}

//Name returns the name of EdgeMesh module
func (em *EdgeMesh) Name() string {
	return constants.EdgeMeshModuleName
}

//Group returns EdgeMesh group
func (em *EdgeMesh) Group() string {
	return constants.MeshGroup
}

//Start sets context and starts the controller
func (em *EdgeMesh) Start() {
	proxy.Init()
	go server.Start()
	// we need watch message to update the cache of instances
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeMesh Stop")
			return
		default:
		}
		msg, err := beehiveContext.Receive(constants.EdgeMeshModuleName)
		if err != nil {
			klog.Warningf("edgemesh receive msg error %v", err)
			continue
		}
		klog.V(4).Infof("edgemesh get message: %v", msg)
		proxy.MsgProcess(msg)
	}
}
