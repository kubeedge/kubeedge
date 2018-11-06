package metamanager

import (
	"time"

	"kubeedge/beehive/pkg/common/config"
	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
	"kubeedge/beehive/pkg/core/model"

	"kubeedge/pkg/common/dbm"
	"kubeedge/pkg/metamanager/dao"
)

const (
	MetaManagerModuleName = "metaManager"
)

func init() {
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
	return core.MetaGroup
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
	syncInterval, _ := config.CONFIG.GetValue("meta.sync.podstatus.interval").ToInt()
	if syncInterval < DefaultSyncInterval {
		syncInterval = DefaultSyncInterval
	}
	return time.Duration(syncInterval) * time.Second
}
