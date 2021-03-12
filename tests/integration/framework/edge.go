package framework

import (
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

func RunEdge(cfgModifyFn func(cfg *v1alpha1.EdgeCoreConfig)) CloseFunc {
	cfg := NewEdgeCoreConfig()
	if cfgModifyFn != nil {
		cfgModifyFn(cfg)
	}

	closeFn := RunEdgeCore(cfg)

	return closeFn
}

func RunEdgeCore(config *v1alpha1.EdgeCoreConfig) CloseFunc {
	registerModules(config)
	core.StartModules()

	closeFn := func() {
		beehiveContext.Cancel()
		for name := range core.GetModules() {
			beehiveContext.Cleanup(name)
		}
	}
	return closeFn
}

func NewEdgeCoreConfig() *v1alpha1.EdgeCoreConfig {
	cfg := v1alpha1.NewDefaultEdgeCoreConfig()
	return cfg
}
