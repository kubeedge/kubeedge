package main

import (
	"flag"

	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	"github.com/go-chassis/go-chassis/core/registry"
	"github.com/kubeedge/kubeedge/edgemesh/pkg"
	"github.com/spf13/pflag"
	"k8s.io/klog"

	_ "github.com/kubeedge/kubeedge/edgemesh/pkg/panel"
	edgeregistry "github.com/kubeedge/kubeedge/edgemesh/pkg/registry"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/resolver"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	//Initialize the resolvers
	r := &resolver.MyResolver{"http"}
	resolver.RegisterResolver(r)

	pkg.Register()

	//Initialize the handlers
	//
	config.Init()
	config.GlobalDefinition.Panel.Infra = "fake"
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	config.GlobalDefinition.Ssl = make(map[string]string)

	control.Init(opts)
	opt := registry.Options{}
	registry.DefaultServiceDiscoveryService = edgeregistry.NewServiceDiscovery(opt)
	loadbalancer.InstallStrategy(loadbalancer.StrategyRandom, func() loadbalancer.Strategy {
		return &loadbalancer.RandomStrategy{}
	})
	//Start server
	server.StartTCP()
}
