package relaycontroller

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/relaycontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/relaycontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"k8s.io/klog/v2"
	"time"
)

type RelayController struct {
	// kubeClient     kubernetes.Interface
	messageLayer   messagelayer.MessageLayer
	relayrcManager *manager.RelayRCManager
	enable         bool
}

var _ core.Module = (*RelayController)(nil)

func Register(rc *v1alpha1.RelayController) {
	config.InitConfigure(rc)
	core.Register(newRelayController(rc.Enable))
}
func (rc *RelayController) Name() string {
	//TODO implement me
	return modules.RelayControllerModuleName
}

func (rc *RelayController) Group() string {
	//TODO implement me
	return modules.RelayControllerModuleGroup
}

func (rc *RelayController) Start() {
	//TODO implement me
	klog.Info("Start relay devicecontroller")
	go rc.checkRelay()

	time.Sleep(1 * time.Second)
}

func (rc *RelayController) Enable() bool {
	//TODO implement me
	return rc.enable
}

func newRelayController(enable bool) *RelayController {
	if !enable {
		return &RelayController{enable: enable}
	}
	crdInformerFactory := informers.GetInformersManager().GetCRDInformerFactory()
	relayrcManager, err := manager.NewRelayRCManager(crdInformerFactory.Relays().V1().Relayrcs().Informer())
	if err != nil {
		klog.Warningf("Create relayrc manager failed with error: %s", err)
		return nil
	}
	return &RelayController{
		enable:         enable,
		messageLayer:   messagelayer.RelayRCControllerMessageLayer(),
		relayrcManager: relayrcManager,
	}
}
