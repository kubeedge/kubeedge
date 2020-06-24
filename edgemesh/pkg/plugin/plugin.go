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

package plugin

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	"github.com/go-chassis/go-chassis/core/registry"

	meshConfig "github.com/kubeedge/kubeedge/edgemesh/pkg/config"
	// Register panel to aviod panic error
	_ "github.com/kubeedge/kubeedge/edgemesh/pkg/plugin/panel"
	meshRegistry "github.com/kubeedge/kubeedge/edgemesh/pkg/plugin/registry"
)

// Install installs go-chassis plugins
func Install() {
	// service discovery
	opt := registry.Options{}
	registry.DefaultServiceDiscoveryService = meshRegistry.NewEdgeServiceDiscovery(opt)
	// load balance
	loadbalancer.InstallStrategy(meshConfig.Config.LBStrategy, func() loadbalancer.Strategy {
		switch meshConfig.Config.LBStrategy {
		case loadbalancer.StrategyRoundRobin:
			return &loadbalancer.RoundRobinStrategy{}
		case loadbalancer.StrategyRandom:
			return &loadbalancer.RandomStrategy{}
		case loadbalancer.StrategySessionStickiness:
			return &loadbalancer.SessionStickinessStrategy{}
		default:
			return &loadbalancer.RoundRobinStrategy{}
		}
	})
	// control panel
	config.GlobalDefinition = &model.GlobalCfg{
		Panel: model.ControlPanel{
			Infra: "edge",
		},
		Ssl: make(map[string]string),
	}
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	control.Init(opts)
	// init archaius
	archaius.Init()
}
