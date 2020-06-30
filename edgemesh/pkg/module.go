package pkg

import (
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/proxier"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	meshconfig "github.com/kubeedge/kubeedge/edgemesh/pkg/config"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/dns"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/listener"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/plugin"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/proxy"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

//EdgeMesh defines EdgeMesh object structure
type EdgeMesh struct {
	enable bool
}

// Register register edgemesh
func Register(m *v1alpha1.EdgeMesh) {
	meshconfig.InitConfigure(m)
	core.Register(&EdgeMesh{enable: m.Enable})
}

// Name returns the name of EdgeMesh module
func (em *EdgeMesh) Name() string {
	return constant.ModuleNameEdgeMesh
}

// Group returns EdgeMesh group
func (em *EdgeMesh) Group() string {
	return modules.MeshGroup
}

// Enable indicates whether this module is enabled
func (em *EdgeMesh) Enable() bool {
	return em.enable
}

//Start sets context and starts the controller
func (em *EdgeMesh) Start() {
	// install go-chassis plugins
	plugin.Install()
	// init tcp listener
	listener.Init()
	// init iptables
	proxy.Init()
	// start proxy listener
	go listener.Start()
	// start dns server
	go dns.Start()

	opts := proxier.NewOptions()
	proxier, err := proxier.NewProxyServer(opts)
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		if err := proxier.Run(); err != nil {
			klog.Errorf("[EdgeMesh] failed to start proxier, err: %v", err)
		}
	}()

	// we need watch message to update the cache of instances
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeMesh Stop")
			proxy.Clean()
			if err := proxier.CleanupAndExit(); err != nil {
				klog.Errorf("[EdgeMesh] proxier failed to cleanup, err: %v", err)
			}
			return
		default:
		}
		msg, err := beehiveContext.Receive(constant.ModuleNameEdgeMesh)
		if err != nil {
			klog.Warningf("[EdgeMesh] receive msg error %v", err)
			continue
		}
		klog.V(4).Infof("[EdgeMesh] get message: %v", msg)
		listener.MsgProcess(msg)
		klog.Warning("[EdgeMesh] proxier process msg")
		proxier.MsgProcess(msg)
	}
}
