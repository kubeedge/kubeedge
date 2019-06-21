package panel

import (
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
)

type FakePanel struct {

}

func (fp *FakePanel) GetCircuitBreaker(inv invocation.Invocation, serviceType string) (string, hystrix.CommandConfig) {
	return "", hystrix.CommandConfig{}
}

func (fp *FakePanel) GetLoadBalancing(inv invocation.Invocation) control.LoadBalancingConfig {
	return control.LoadBalancingConfig{}
}
func (fp *FakePanel) GetRateLimiting(inv invocation.Invocation, serviceType string) control.RateLimitingConfig {
	return control.RateLimitingConfig{}
}
func (fp *FakePanel) GetFaultInjection(inv invocation.Invocation) model.Fault {
	return model.Fault{}
}
func (fp *FakePanel) GetEgressRule() []control.EgressConfig {
	return []control.EgressConfig{}
}


func init() {
	control.InstallPlugin("fake", func(options control.Options) control.Panel {
		return &FakePanel{}
	})
}