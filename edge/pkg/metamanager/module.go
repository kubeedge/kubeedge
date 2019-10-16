package metamanager

import (
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metaconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

//constant metamanager module name
const (
	MetaManagerModuleName = "metaManager"
)

// Register register metamanager
func Register(m *edgecoreconfig.Metamanager) {
	metaconfig.InitMetamanagerConfig(m)
	dbm.RegisterModel(MetaManagerModuleName, new(dao.Meta))
	core.Register(&metaManager{})
}

type metaManager struct {
	context *context.Context
}

func (*metaManager) Name() string {
	return MetaManagerModuleName
}

func (*metaManager) Group() string {
	return modules.MetaGroup
}

func (m *metaManager) Start(c *context.Context) {
	m.context = c
	go func() {
		period := getSyncInterval()
		timer := time.NewTimer(period)
		for {
			select {
			case <-timer.C:
				timer.Reset(period)
				msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, model.ResourceTypePodStatus, OperationMetaSync)
				m.context.Send(MetaManagerModuleName, *msg)
			}
		}
	}()
	m.mainLoop()
}

func (m *metaManager) Cleanup() {
	m.context.Cleanup(m.Name())
}

func getSyncInterval() time.Duration {
	return time.Duration(DefaultSyncInterval) * time.Second
}
