package framework

import (
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type CloseFunc func()

func RunCloud(cfgModifyFn func(config *v1alpha1.CloudCoreConfig)) CloseFunc {
	cfg := NewCloudCoreConfig("")
	if cfgModifyFn != nil {
		cfgModifyFn(cfg)
	}

	closeFn2 := RunCloudCore(cfg)

	cancel := func() {
		//closeFn()
		closeFn2()
	}
	return cancel
}

func RunCloudCore(cfg *v1alpha1.CloudCoreConfig) CloseFunc {
	client.InitKubeEdgeClient(cfg.KubeAPIConfig)
	gis := informers.GetInformersManager()
	registerModules(cfg)

	neverStop := make(chan struct{})
	gis.Start(neverStop)
	core.StartModules()

	closeFn := func() {
		beehiveContext.Cancel()
		for name := range core.GetModules() {
			beehiveContext.Cleanup(name)
		}
	}

	return closeFn
}

func NewCloudCoreConfig(MasterURL string) *v1alpha1.CloudCoreConfig {
	cfg := v1alpha1.NewDefaultCloudCoreConfig()
	cfg.KubeAPIConfig.KubeConfig = "kubeconfig"
	cfg.KubeAPIConfig.Master = MasterURL
	return cfg
}
