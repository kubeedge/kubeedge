package pkg

import (
	"context"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/proxy"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

//EdgeMesh defines EdgeMesh object structure
type EdgeMesh struct {
	context *beehiveContext.Context
	cancel  context.CancelFunc
}

// Register register edgemesh
func Register() {
	core.Register(&EdgeMesh{})
}

//Name returns the name of EdgeMesh module
func (em *EdgeMesh) Name() string {
	return constant.ModuleNameEdgeMesh
}

//Group returns EdgeMesh group
func (em *EdgeMesh) Group() string {
	return modules.MeshGroup
}

//Start sets context and starts the controller
func (em *EdgeMesh) Start(c *beehiveContext.Context) {
	em.context = c
	var ctx context.Context
	ctx, em.cancel = context.WithCancel(context.Background())
	proxy.Init()
	go server.Start()
	// we need watch message to update the cache of instances
	for {
		select {
		case <-ctx.Done():
			klog.Warning("EdgeMesh Stop")
			return
		default:
		}
		msg, err := em.context.Receive(constant.ModuleNameEdgeMesh)
		if err != nil {
			klog.Warningf("edgemesh receive msg error %v", err)
			continue
		}
		klog.V(4).Infof("edgemesh get message: %v", msg)
		proxy.MsgProcess(msg)
	}
}

//Cleanup sets up context cleanup through EdgeMesh name
func (em *EdgeMesh) Cleanup() {
	em.cancel()
	em.context.Cleanup(em.Name())
}
