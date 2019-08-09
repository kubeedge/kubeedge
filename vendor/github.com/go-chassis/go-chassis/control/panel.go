package control

import (
	"fmt"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
	"strings"
)

var panelPlugin = make(map[string]func(options Options) Panel)

//DefaultPanel get fetch config
var DefaultPanel Panel

const (
	//ScopeAPI is config const
	ScopeAPI = "api"

	//ScopeInstance is config const
	ScopeInstance = "instance"

	//ScopeInstanceAPI is config const
	ScopeInstanceAPI = "instance-api"
)

//Panel is a abstraction of pulling configurations from various of systems, and transfer different configuration into standardized model
//you can use different panel implementation to pull different of configs from Istio or Archaius
//TODO able to set configs
type Panel interface {
	GetCircuitBreaker(inv invocation.Invocation, serviceType string) (string, hystrix.CommandConfig)
	GetLoadBalancing(inv invocation.Invocation) LoadBalancingConfig
	GetRateLimiting(inv invocation.Invocation, serviceType string) RateLimitingConfig
	GetFaultInjection(inv invocation.Invocation) model.Fault
	GetEgressRule() []EgressConfig
}

//InstallPlugin install implementation
func InstallPlugin(name string, f func(options Options) Panel) {
	panelPlugin[name] = f
}

//Init initialize DefaultPanel
func Init(opts Options) error {
	infra := opts.Infra
	if infra == "" {
		infra = "archaius"
	}
	f, ok := panelPlugin[infra]
	if !ok {
		return fmt.Errorf("do not support [%s] panel", infra)
	}

	DefaultPanel = f(opts)
	return nil
}

//NewCircuitName create circuit command string
//scope means has two choices, service and api
//if you set it to api, a api level command string will be created. like "Consumer.mall.rest./test"
//set to service, a service level command will be created, like "Consumer.mall"
func NewCircuitName(serviceType, scope string, inv invocation.Invocation) string {
	var cmd = serviceType
	if inv.MicroServiceName != "" {
		cmd = strings.Join([]string{cmd, inv.MicroServiceName}, ".")
	}
	if scope == "" {
		scope = ScopeAPI
	}

	if scope == ScopeAPI {
		if inv.SchemaID != "" {
			cmd = strings.Join([]string{cmd, inv.SchemaID}, ".")
		}
		if inv.OperationID != "" {
			cmd = strings.Join([]string{cmd, inv.OperationID}, ".")
		}
		return cmd
	}
	if scope == ScopeInstance {
		if inv.Endpoint != "" {
			cmd = strings.Join([]string{cmd, inv.Endpoint}, ".")
		}
		return cmd
	}
	if scope == ScopeInstanceAPI {
		if inv.Endpoint != "" {
			cmd = strings.Join([]string{cmd, inv.Endpoint}, ".")
		}
		if inv.SchemaID != "" {
			cmd = strings.Join([]string{cmd, inv.SchemaID}, ".")
		}
		if inv.OperationID != "" {
			cmd = strings.Join([]string{cmd, inv.OperationID}, ".")
		}
		return cmd
	}

	return cmd

}
