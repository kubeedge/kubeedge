package metamanager

import (
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

//constant metamanager module name
const (
	MetaManagerModuleName = "metaManager"
)

type metaManager struct {
}

func newMetaManager() *metaManager {
	return &metaManager{}
}

// Register register metamanager
func Register() {
	dbm.RegisterModel(MetaManagerModuleName, new(dao.Meta))
	core.Register(newMetaManager())
}

func (*metaManager) Name() string {
	return MetaManagerModuleName
}

func (*metaManager) Group() string {
	return modules.MetaGroup
}

func (m *metaManager) Start() {
	InitMetaManagerConfig()

	go func() {
		period := getSyncInterval()
		timer := time.NewTimer(period)
		for {
			select {
			case <-beehiveContext.Done():
				klog.Warning("MetaManager stop")
				return
			case <-timer.C:
				timer.Reset(period)
				msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, model.ResourceTypePodStatus, OperationMetaSync)
				beehiveContext.Send(MetaManagerModuleName, *msg)
			}
		}
	}()

	m.runMetaManager()
}

func getSyncInterval() time.Duration {
	syncInterval, _ := config.CONFIG.GetValue("meta.sync.podstatus.interval").ToInt()
	if syncInterval < DefaultSyncInterval {
		syncInterval = DefaultSyncInterval
	}
	return time.Duration(syncInterval) * time.Second
}
