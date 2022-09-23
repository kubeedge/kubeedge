/*
Copyright 2022 The KubeEdge Authors.

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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliflag "k8s.io/component-base/cli/flag"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
	fakeremote "k8s.io/kubernetes/pkg/kubelet/cri/remote/fake"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

type hollowEdgeNodeConfig struct {
	Token           string
	NodeName        string
	HTTPServer      string
	WebsocketServer string
	NodeLabels      map[string]string
}

func main() {
	command := newHollowEdgeNodeCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

// newHollowEdgeNodeCommand creates a *cobra.Command object with default parameters
func newHollowEdgeNodeCommand() *cobra.Command {
	s := &hollowEdgeNodeConfig{
		NodeLabels: make(map[string]string),
	}

	cmd := &cobra.Command{
		Use:  "edgemark",
		Long: "edgemark",
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			run(s)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.AddGoFlagSet(flag.CommandLine) // for flags like --docker-only
	s.addFlags(fs)

	return cmd
}

func run(config *hollowEdgeNodeConfig) {
	c := EdgeCoreConfig(config)

	// TODO: fake runtime service
	edged.Register(c.Modules.Edged)
	edgehub.Register(c.Modules.EdgeHub, c.Modules.Edged.HostnameOverride)
	metamanager.Register(c.Modules.MetaManager)

	dbm.InitDBConfig(c.DataBase.DriverName, c.DataBase.AliasName, c.DataBase.DataSource)

	// start all modules
	core.Run()
}

func (c *hollowEdgeNodeConfig) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Token, "token", "", "Token indicates the priority of joining the cluster for the edge.")
	fs.StringVar(&c.NodeName, "name", "fake-node", "Name of this Hollow Node.")
	fs.StringVar(&c.WebsocketServer, "websocket-server", "", "Server indicates websocket server address.")
	fs.StringVar(&c.HTTPServer, "http-server", "", "HTTPServer indicates the server for edge to apply for the certificate.")
	bindableNodeLabels := cliflag.ConfigurationMap(c.NodeLabels)
	fs.Var(&bindableNodeLabels, "node-labels", "Additional node labels")
}

func EdgeCoreConfig(config *hollowEdgeNodeConfig) *v1alpha2.EdgeCoreConfig {
	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()

	// overWrite config
	edgeCoreConfig.DataBase.DataSource = "/edgecore.db"
	edgeCoreConfig.Modules.EdgeHub.Token = config.Token
	edgeCoreConfig.Modules.EdgeHub.HTTPServer = config.HTTPServer
	edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = config.WebsocketServer

	// use fake runtime for test
	edgeCoreConfig.Modules.Edged.ContainerRuntime = "fake"
	edgeCoreConfig.Modules.Edged.RemoteRuntimeEndpoint = "/run/fake/fake.sock"
	edgeCoreConfig.Modules.Edged.RemoteImageEndpoint = "/run/fake/fake.sock"

	edgeCoreConfig.Modules.Edged.HostnameOverride = config.NodeName
	edgeCoreConfig.Modules.Edged.NodeLabels = config.NodeLabels

	return edgeCoreConfig
}

func GetFakeRuntimeAndImageServices(
	remoteRuntimeEndpoint,
	remoteImageEndpoint string,
	runtimeRequestTimeout metav1.Duration) (internalapi.RuntimeService, internalapi.ImageManagerService, error) {
	endpoint, err := fakeremote.GenerateEndpoint()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate fake endpoint %v", err)
	}

	fakeRemoteRuntime := fakeremote.NewFakeRemoteRuntime()
	if err = fakeRemoteRuntime.Start(endpoint); err != nil {
		return nil, nil, fmt.Errorf("failed to start fake runtime %v", err)
	}

	runtimeService, err := remote.NewRemoteRuntimeService(endpoint, 15*time.Second)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init runtime service %v", err)
	}

	return runtimeService, fakeRemoteRuntime.ImageService, err
}
