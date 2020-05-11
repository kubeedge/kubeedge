package panel

import (
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
)

type EdgePanel struct {
}

func (ep *EdgePanel) GetCircuitBreaker(inv invocation.Invocation, serviceType string) (string, hystrix.CommandConfig) {
	return "", hystrix.CommandConfig{}
}

func (ep *EdgePanel) GetLoadBalancing(inv invocation.Invocation) control.LoadBalancingConfig {
	return control.LoadBalancingConfig{}
}
func (ep *EdgePanel) GetRateLimiting(inv invocation.Invocation, serviceType string) control.RateLimitingConfig {
	return control.RateLimitingConfig{}
}
func (ep *EdgePanel) GetFaultInjection(inv invocation.Invocation) model.Fault {
	return model.Fault{}
}
func (ep *EdgePanel) GetEgressRule() []control.EgressConfig {
	return []control.EgressConfig{}
}

// TODO Remove the init method, because it will cause invalid logs to be printed when the program is running @kadisi
// init install Plugin
func init() {
	control.InstallPlugin("edge", func(options control.Options) control.Panel {
		return &EdgePanel{}
	})
}
