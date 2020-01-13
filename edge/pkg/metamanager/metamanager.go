package metamanager

import (
	"time"

	"github.com/astaxie/beego/orm"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	metamanagerconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

type metaManager struct {
}

func newMetaManager() *metaManager {
	return &metaManager{}
}

// Register register metamanager
func Register() {
	InitDBTable()
	metamanagerconfig.InitConfigure()
	core.Register(newMetaManager())
}

//InitDBTable create table
func InitDBTable() {
	if !core.IsModuleEnabled(constants.MetaManagerModuleName) {
		klog.Infof("module %s has not been registered, so can not init db table", constants.MetaManagerModuleName)
		return
	}
	orm.RegisterModel(new(dao.Meta))
}

func (*metaManager) Name() string {
	return constants.MetaManagerModuleName
}

func (*metaManager) Group() string {
	return constants.MetaGroup
}

func (m *metaManager) Start() {

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
				msg := model.NewMessage("").BuildRouter(constants.MetaManagerModuleName, GroupResource, model.ResourceTypePodStatus, OperationMetaSync)
				beehiveContext.Send(constants.MetaManagerModuleName, *msg)
			}
		}
	}()

	m.runMetaManager()
}

func getSyncInterval() time.Duration {
	return time.Duration(metamanagerconfig.Get().SyncInterval) * time.Second
}
