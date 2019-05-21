package pkg

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

//EdgeMesh defines EdgeMesh object structure
type EdgeMesh struct {
	context *context.Context
}

func init() {
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
func (em *EdgeMesh) Start(c *context.Context) {
	em.context = c
	go server.Start()
	// we need watch message to update the cache of instances
	for {
		if msg, ok := em.context.Receive(constant.ModuleNameEdgeMesh); ok == nil {
			log.LOGGER.Infof("get message: %v", msg)
			continue
		}
	}
}

//Cleanup sets up context cleanup through EdgeMesh name
func (em *EdgeMesh) Cleanup() {
	em.context.Cleanup(em.Name())
}