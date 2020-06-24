/*
Copyright 2020 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
