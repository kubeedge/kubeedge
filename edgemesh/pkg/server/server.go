package server

import (
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	"github.com/go-chassis/go-chassis/core/registry"

	meshconfig "github.com/kubeedge/kubeedge/edgemesh/pkg/config"
	_ "github.com/kubeedge/kubeedge/edgemesh/pkg/panel"
	edgeregistry "github.com/kubeedge/kubeedge/edgemesh/pkg/registry"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
)

func Start() {
	//Initialize the resolvers
	r := &resolver.MyResolver{"http"}
	resolver.RegisterResolver(r)
	//Initialize the handlers
	//
	//
	config.GlobalDefinition = &model.GlobalCfg{}
	config.GlobalDefinition.Panel.Infra = "fake"
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	config.GlobalDefinition.Ssl = make(map[string]string)

	control.Init(opts)
	opt := registry.Options{}
	registry.DefaultServiceDiscoveryService = edgeregistry.NewServiceDiscovery(opt)
	myStrategy := meshconfig.Get().LBStrategy
	loadbalancer.InstallStrategy(myStrategy, func() loadbalancer.Strategy {
		switch myStrategy {
		case "RoundRobin":
			return &loadbalancer.RoundRobinStrategy{}
		case "Random":
			return &loadbalancer.RandomStrategy{}
		default:
			return &loadbalancer.RoundRobinStrategy{}
		}
	})
	//Start dns server
	go DnsStart()
	//Start server
	StartTCP()
}
