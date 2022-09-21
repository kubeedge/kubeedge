package edgerelay

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"sync"
)

type EdgeRelay struct {
	enable bool
	// 再来一个通信clientManager
	chClient   clients.Adapter
	keeperLock sync.RWMutex
}

var _ core.Module = (*EdgeRelay)(nil)

func newEdgeRelay(enable bool) *EdgeRelay {

	er := &EdgeRelay{
		enable: enable,
	}
	return er
}

func Register(er *v1alpha1.EdgeCoreEdgeRelay, nodeID string) {
	config.InitConfig(er, nodeID)
	core.Register(newEdgeRelay(er.Enable))
}

func (er *EdgeRelay) Name() string {
	return modules.EdgeRelayModuleName
}

func (er *EdgeRelay) Group() string {
	return modules.RelayGroup
}

func (er *EdgeRelay) Enable() bool {
	return er.enable
}

func (er *EdgeRelay) Start() {
	// 首先读取qlite里的信息，放到内存中（更新config）
	er.LoadRelayID()

	go er.MsgFromOtherEdge()

}
