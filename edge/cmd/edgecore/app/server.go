package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/mitchellh/go-ps"
	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus"
	"github.com/kubeedge/kubeedge/edge/test"
	edgemesh "github.com/kubeedge/kubeedge/edgemesh/pkg"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1/validation"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

// NewEdgeCoreCommand create edgecore cmd
func NewEdgeCoreCommand() *cobra.Command {
	opts := options.NewEdgeCoreOptions()
	cmd := &cobra.Command{
		Use: "edgecore",
		Long: `Edgecore is the core edge part of KubeEdge, which contains six modules: devicetwin, edged,
edgehub, eventbus, metamanager, and servicebus. DeviceTwin is responsible for storing device status
and syncing device status to the cloud. It also provides query interfaces for applications. Edged is an
agent that runs on edge nodes and manages containerized applications and devices. Edgehub is a web socket
client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge
Architecture). This includes syncing cloud-side resource updates to the edge, and reporting
edge-side host and device status changes to the cloud. EventBus is a MQTT client to interact with MQTT
servers (mosquito), offering publish and subscribe capabilities to other components. MetaManager
is the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata
to/from a lightweight database (SQLite).ServiceBus is a HTTP client to interact with HTTP servers (REST),
offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge. `,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintMinConfigAndExitIfRequested(v1alpha1.NewMinEdgeCoreConfig())
			flag.PrintDefaultConfigAndExitIfRequested(v1alpha1.NewDefaultEdgeCoreConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs))
			}

			config, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			if errs := validation.ValidateEdgeCoreConfiguration(config); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())

			// Check the running environment by default
			checkEnv := os.Getenv("CHECK_EDGECORE_ENVIRONMENT")
			if checkEnv != "false" {
				// Check running environment before run edge core
				if err := environmentCheck(); err != nil {
					klog.Fatal(fmt.Errorf("Failed to check the running environment: %v", err))
				}
			}

			// get edge node local ip
			if config.Modules.Edged.NodeIP == "" {
				hostnameOverride, err := os.Hostname()
				if err != nil {
					hostnameOverride = constants.DefaultHostnameOverride
				}
				localIP, _ := util.GetLocalIP(hostnameOverride)
				config.Modules.Edged.NodeIP = localIP
			}

			registerModules(config)
			// start all modules
			core.Run()
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	flag.AddFlags(namedFs.FlagSet("global"))
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFs, cols)
	})

	return cmd
}

// findProcess find a running process by name
func findProcess(name string) (bool, error) {
	processes, err := ps.Processes()
	if err != nil {
		return false, err
	}

	for _, process := range processes {
		if process.Executable() == name {
			return true, nil
		}
	}

	return false, nil
}

// environmentCheck check the environment before edgecore start
// if Check failed,  return errors
func environmentCheck() error {
	// if kubelet is running, return error
	if find, err := findProcess("kubelet"); err != nil {
		return err
	} else if find {
		return errors.New("Kubelet should not running on edge node when running edgecore")
	}

	// if kube-proxy is running, return error
	if find, err := findProcess("kube-proxy"); err != nil {
		return err
	} else if find {
		return errors.New("Kube-proxy should not running on edge node when running edgecore")
	}

	return nil
}

// registerModules register all the modules started in edgecore
func registerModules(c *v1alpha1.EdgeCoreConfig) {
	devicetwin.Register(c.Modules.DeviceTwin, c.Modules.Edged.HostnameOverride)
	edged.Register(c.Modules.Edged)
	edgehub.Register(c.Modules.EdgeHub, c.Modules.Edged.HostnameOverride)
	eventbus.Register(c.Modules.EventBus, c.Modules.Edged.HostnameOverride)
	edgemesh.Register(c.Modules.EdgeMesh)
	metamanager.Register(c.Modules.MetaManager)
	servicebus.Register(c.Modules.ServiceBus)
	edgestream.Register(c.Modules.EdgeStream, c.Modules.Edged.HostnameOverride, c.Modules.Edged.NodeIP)
	test.Register(c.Modules.DBTest)
	// Note: Need to put it to the end, and wait for all models to register before executing
	dbm.InitDBConfig(c.DataBase.DriverName, c.DataBase.AliasName, c.DataBase.DataSource)
}
