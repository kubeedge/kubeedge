/*
Copyright 2021 The Kubernetes Authors.

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

// Package app makes it easy to create a kubelet server for various contexts.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"

	cadvisorapi "github.com/google/cadvisor/info/v1"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelsdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapiserver "k8s.io/apiserver/pkg/server"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/connrotation"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/configz"
	"k8s.io/component-base/featuregate"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/tracing"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
	nodeutil "k8s.io/component-helpers/node/util"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/capabilities"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/kubelet"
	kubeletconfiginternal "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubeletscheme "k8s.io/kubernetes/pkg/kubelet/apis/config/scheme"
	kubeletconfigvalidation "k8s.io/kubernetes/pkg/kubelet/apis/config/validation"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/config"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/eviction"
	evictionapi "k8s.io/kubernetes/pkg/kubelet/eviction/api"
	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/configfiles"
	"k8s.io/kubernetes/pkg/kubelet/stats/pidlimit"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
	kubeletutil "k8s.io/kubernetes/pkg/kubelet/util"
	utilfs "k8s.io/kubernetes/pkg/util/filesystem"
	"k8s.io/kubernetes/pkg/util/oom"
	"k8s.io/kubernetes/pkg/util/rlimit"
	"k8s.io/kubernetes/pkg/volume/util/hostutil"
	"k8s.io/kubernetes/pkg/volume/util/subpath"
	"k8s.io/utils/cpuset"
	"k8s.io/utils/exec"
)

func init() {
	utilruntime.Must(logsapi.AddFeatureGates(utilfeature.DefaultMutableFeatureGate))
	// Prevent memory leak from OTel metrics, which we don't use:
	// https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5190
	otel.SetMeterProvider(noop.NewMeterProvider())
}

const (
	// Kubelet component name
	componentKubelet = "kubelet"
)

// NewKubeletCommand creates a *cobra.Command object with default parameters
func NewKubeletCommand() *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet(componentKubelet, pflag.ContinueOnError)
	cleanFlagSet.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	kubeletFlags := options.NewKubeletFlags()

	kubeletConfig, err := options.NewKubeletConfiguration()
	// programmer error
	if err != nil {
		klog.ErrorS(err, "Failed to create a new kubelet configuration")
		os.Exit(1)
	}

	cmd := &cobra.Command{
		Use: componentKubelet,
		Long: `The kubelet is the primary "node agent" that runs on each
node. It can register the node with the apiserver using one of: the hostname; a flag to
override the hostname; or specific logic for a cloud provider.

The kubelet works in terms of a PodSpec. A PodSpec is a YAML or JSON object
that describes a pod. The kubelet takes a set of PodSpecs that are provided through
various mechanisms (primarily through the apiserver) and ensures that the containers
described in those PodSpecs are running and healthy. The kubelet doesn't manage
containers which were not created by Kubernetes.

Other than from an PodSpec from the apiserver, there are two ways that a container
manifest can be provided to the Kubelet.

File: Path passed as a flag on the command line. Files under this path will be monitored
periodically for updates. The monitoring period is 20s by default and is configurable
via a flag.

HTTP endpoint: HTTP endpoint passed as a parameter on the command line. This endpoint
is checked every 20 seconds (also configurable with a flag).`,
		// The Kubelet has special flag parsing requirements to enforce flag precedence rules,
		// so we do all our parsing manually in Run, below.
		// DisableFlagParsing=true provides the full set of flags passed to the kubelet in the
		// `args` arg to Run, without Cobra's interference.
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initial flag parse, since we disable cobra's flag parsing
			if err := cleanFlagSet.Parse(args); err != nil {
				return fmt.Errorf("failed to parse kubelet flag: %w", err)
			}

			// check if there are non-flag arguments in the command line
			cmds := cleanFlagSet.Args()
			if len(cmds) > 0 {
				return fmt.Errorf("unknown command %+s", cmds[0])
			}

			// short-circuit on help
			help, err := cleanFlagSet.GetBool("help")
			if err != nil {
				return errors.New(`"help" flag is non-bool, programmer error, please correct`)
			}
			if help {
				return cmd.Help()
			}

			// short-circuit on verflag
			verflag.PrintAndExitIfRequested()

			// set feature gates from initial flags-based config
			if err := utilfeature.DefaultMutableFeatureGate.SetFromMap(kubeletConfig.FeatureGates); err != nil {
				return fmt.Errorf("failed to set feature gates from initial flags-based config: %w", err)
			}

			// validate the initial KubeletFlags
			if err := options.ValidateKubeletFlags(kubeletFlags); err != nil {
				return fmt.Errorf("failed to validate kubelet flags: %w", err)
			}

			if cleanFlagSet.Changed("pod-infra-container-image") {
				klog.InfoS("--pod-infra-container-image will not be pruned by the image garbage collector in kubelet and should also be set in the remote runtime")
				_ = cmd.Flags().MarkDeprecated("pod-infra-container-image", "--pod-infra-container-image will be removed in 1.30. Image garbage collector will get sandbox image information from CRI.")
			}

			// Config and flags parsed, now we can initialize logging.
			logs.InitLogs()
			if err := logsapi.ValidateAndApplyAsField(&kubeletConfig.Logging, utilfeature.DefaultFeatureGate, field.NewPath("logging")); err != nil {
				return fmt.Errorf("initialize logging: %v", err)
			}
			cliflag.PrintFlags(cleanFlagSet)

			// We always validate the local configuration (command line + config file).
			// This is the default "last-known-good" config for dynamic config, and must always remain valid.
			if err := kubeletconfigvalidation.ValidateKubeletConfiguration(kubeletConfig, utilfeature.DefaultFeatureGate); err != nil {
				return fmt.Errorf("failed to validate kubelet configuration, error: %w, path: %s", err, kubeletConfig)
			}

			if (kubeletConfig.KubeletCgroups != "" && kubeletConfig.KubeReservedCgroup != "") && (strings.Index(kubeletConfig.KubeletCgroups, kubeletConfig.KubeReservedCgroup) != 0) {
				klog.InfoS("unsupported configuration:KubeletCgroups is not within KubeReservedCgroup")
			}

			// construct a KubeletServer from kubeletFlags and kubeletConfig
			kubeletServer := &options.KubeletServer{
				KubeletFlags:         *kubeletFlags,
				KubeletConfiguration: *kubeletConfig,
			}

			// use kubeletServer to construct the default KubeletDeps
			kubeletDeps, err := UnsecuredDependencies(kubeletServer, utilfeature.DefaultFeatureGate)
			if err != nil {
				return fmt.Errorf("failed to construct kubelet dependencies: %w", err)
			}

			if err := checkPermissions(); err != nil {
				klog.ErrorS(err, "kubelet running with insufficient permissions")
			}

			// make the kubelet's config safe for logging
			config := kubeletServer.KubeletConfiguration.DeepCopy()
			for k := range config.StaticPodURLHeader {
				config.StaticPodURLHeader[k] = []string{"<masked>"}
			}
			// log the kubelet's config for inspection
			klog.V(5).InfoS("KubeletConfiguration", "configuration", klog.Format(config))

			// set up signal context for kubelet shutdown
			ctx := genericapiserver.SetupSignalContext()

			utilfeature.DefaultMutableFeatureGate.AddMetrics()
			// run the kubelet
			return Run(ctx, kubeletServer, kubeletDeps, utilfeature.DefaultFeatureGate)
		},
	}

	// keep cleanFlagSet separate, so Cobra doesn't pollute it with the global flags
	kubeletFlags.AddFlags(cleanFlagSet)
	options.AddKubeletConfigFlags(cleanFlagSet, kubeletConfig)
	options.AddGlobalFlags(cleanFlagSet)
	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("help for %s", cmd.Name()))

	// ugly, but necessary, because Cobra's default UsageFunc and HelpFunc pollute the flagset with global flags
	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine(), cleanFlagSet.FlagUsagesWrapped(2))
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine(), cleanFlagSet.FlagUsagesWrapped(2))
	})

	return cmd
}

// mergeKubeletConfigurations merges the provided drop-in configurations with the base kubelet configuration.
// The drop-in configurations are processed in lexical order based on the file names. This means that the
// configurations in files with lower numeric prefixes are applied first, followed by higher numeric prefixes.
// For example, if the drop-in directory contains files named "10-config.conf" and "20-config.conf",
// the configurations in "10-config.conf" will be applied first, and then the configurations in "20-config.conf" will be applied,
// potentially overriding the previous values.
func mergeKubeletConfigurations(kubeletConfig *kubeletconfiginternal.KubeletConfiguration, kubeletDropInConfigDir string) error {
	const dropinFileExtension = ".conf"
	baseKubeletConfigJSON, err := json.Marshal(kubeletConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal base config: %w", err)
	}
	// Walk through the drop-in directory and update the configuration for each file
	if err := filepath.WalkDir(kubeletDropInConfigDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == dropinFileExtension {
			dropinConfigJSON, err := loadDropinConfigFileIntoJSON(path)
			if err != nil {
				return fmt.Errorf("failed to load kubelet dropin file, path: %s, error: %w", path, err)
			}
			mergedConfigJSON, err := jsonpatch.MergePatch(baseKubeletConfigJSON, dropinConfigJSON)
			if err != nil {
				return fmt.Errorf("failed to merge drop-in and current config: %w", err)
			}
			baseKubeletConfigJSON = mergedConfigJSON
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk through kubelet dropin directory %q: %w", kubeletDropInConfigDir, err)
	}

	if err := json.Unmarshal(baseKubeletConfigJSON, kubeletConfig); err != nil {
		return fmt.Errorf("failed to unmarshal merged JSON into kubelet configuration: %w", err)
	}
	return nil
}

// newFlagSetWithGlobals constructs a new pflag.FlagSet with global flags registered
// on it.
func newFlagSetWithGlobals() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	// set the normalize func, similar to k8s.io/component-base/cli//flags.go:InitFlags
	fs.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	// explicitly add flags from libs that register global flags
	options.AddGlobalFlags(fs)
	return fs
}

// newFakeFlagSet constructs a pflag.FlagSet with the same flags as fs, but where
// all values have noop Set implementations
func newFakeFlagSet(fs *pflag.FlagSet) *pflag.FlagSet {
	ret := pflag.NewFlagSet("", pflag.ExitOnError)
	ret.SetNormalizeFunc(fs.GetNormalizeFunc())
	fs.VisitAll(func(f *pflag.Flag) {
		ret.VarP(cliflag.NoOp{}, f.Name, f.Shorthand, f.Usage)
	})
	return ret
}

// kubeletConfigFlagPrecedence re-parses flags over the KubeletConfiguration object.
// We must enforce flag precedence by re-parsing the command line into the new object.
// This is necessary to preserve backwards-compatibility across binary upgrades.
// See issue #56171 for more details.
func kubeletConfigFlagPrecedence(kc *kubeletconfiginternal.KubeletConfiguration, args []string) error {
	// We use a throwaway kubeletFlags and a fake global flagset to avoid double-parses,
	// as some Set implementations accumulate values from multiple flag invocations.
	fs := newFakeFlagSet(newFlagSetWithGlobals())
	// register throwaway KubeletFlags
	options.NewKubeletFlags().AddFlags(fs)
	// register new KubeletConfiguration
	options.AddKubeletConfigFlags(fs, kc)
	// Remember original feature gates, so we can merge with flag gates later
	original := kc.FeatureGates
	// avoid duplicate printing the flag deprecation warnings during re-parsing
	fs.SetOutput(io.Discard)
	// re-parse flags
	if err := fs.Parse(args); err != nil {
		return err
	}
	// Add back feature gates that were set in the original kc, but not in flags
	for k, v := range original {
		if _, ok := kc.FeatureGates[k]; !ok {
			kc.FeatureGates[k] = v
		}
	}
	return nil
}

func loadConfigFile(name string) (*kubeletconfiginternal.KubeletConfiguration, error) {
	const errFmt = "failed to load Kubelet config file %s, error %v"
	// compute absolute path based on current working dir
	kubeletConfigFile, err := filepath.Abs(name)
	if err != nil {
		return nil, fmt.Errorf(errFmt, name, err)
	}
	loader, err := configfiles.NewFsLoader(&utilfs.DefaultFs{}, kubeletConfigFile)
	if err != nil {
		return nil, fmt.Errorf(errFmt, name, err)
	}
	kc, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf(errFmt, name, err)
	}

	// EvictionHard may be nil if it was not set in kubelet's config file.
	// EvictionHard can have OS-specific fields, which is why there's no default value for it.
	// See: https://github.com/kubernetes/kubernetes/pull/110263
	if kc.EvictionHard == nil {
		kc.EvictionHard = eviction.DefaultEvictionHard
	}
	return kc, err
}

func loadDropinConfigFileIntoJSON(name string) ([]byte, error) {
	const errFmt = "failed to load drop-in kubelet config file %s, error %v"
	// compute absolute path based on current working dir
	kubeletConfigFile, err := filepath.Abs(name)
	if err != nil {
		return nil, fmt.Errorf(errFmt, name, err)
	}
	loader, err := configfiles.NewFsLoader(&utilfs.DefaultFs{}, kubeletConfigFile)
	if err != nil {
		return nil, fmt.Errorf(errFmt, name, err)
	}
	return loader.LoadIntoJSON()
}

// UnsecuredDependencies returns a Dependencies suitable for being run, or an error if the server setup
// is not valid.  It will not start any background processes, and does not include authentication/authorization
func UnsecuredDependencies(s *options.KubeletServer, featureGate featuregate.FeatureGate) (*kubelet.Dependencies, error) {
	mounter := mount.New(s.ExperimentalMounterPath)
	subpather := subpath.New(mounter)
	hu := hostutil.NewHostUtil()
	var pluginRunner = exec.New()

	plugins, err := ProbeVolumePlugins(featureGate)
	if err != nil {
		return nil, err
	}
	tp := oteltrace.NewNoopTracerProvider()
	if utilfeature.DefaultFeatureGate.Enabled(features.KubeletTracing) {
		tp, err = newTracerProvider(s)
		if err != nil {
			return nil, err
		}
	}
	return &kubelet.Dependencies{
		Auth:                nil, // default does not enforce auth[nz]
		CAdvisorInterface:   nil, // cadvisor.New launches background processes (bg http.ListenAndServe, and some bg cleaners), not set here
		ContainerManager:    nil,
		KubeClient:          nil,
		HeartbeatClient:     nil,
		EventClient:         nil,
		TracerProvider:      tp,
		HostUtil:            hu,
		Mounter:             mounter,
		Subpather:           subpather,
		OOMAdjuster:         oom.NewOOMAdjuster(),
		OSInterface:         kubecontainer.RealOS{},
		VolumePlugins:       plugins,
		DynamicPluginProber: GetDynamicPluginProber(s.VolumePluginDir, pluginRunner),
		TLSOptions:          nil}, nil
}

// Run runs the specified KubeletServer with the given Dependencies. This should never exit.
// The kubeDeps argument may be nil - if so, it is initialized from the settings on KubeletServer.
// Otherwise, the caller is assumed to have set up the Dependencies object and a default one will
// not be generated.
func Run(ctx context.Context, s *options.KubeletServer, kubeDeps *kubelet.Dependencies, featureGate featuregate.FeatureGate) error {
	// To help debugging, immediately log version
	klog.InfoS("Kubelet version", "kubeletVersion", version.Get())

	klog.InfoS("Golang settings", "GOGC", os.Getenv("GOGC"), "GOMAXPROCS", os.Getenv("GOMAXPROCS"), "GOTRACEBACK", os.Getenv("GOTRACEBACK"))

	if err := initForOS(s.KubeletFlags.WindowsService, s.KubeletFlags.WindowsPriorityClass); err != nil {
		return fmt.Errorf("failed OS init: %w", err)
	}
	if err := run(ctx, s, kubeDeps, featureGate); err != nil {
		return fmt.Errorf("failed to run Kubelet: %w", err)
	}
	return nil
}

func setConfigz(cz *configz.Config, kc *kubeletconfiginternal.KubeletConfiguration) error {
	scheme, _, err := kubeletscheme.NewSchemeAndCodecs()
	if err != nil {
		return err
	}
	versioned := kubeletconfigv1beta1.KubeletConfiguration{}
	if err := scheme.Convert(kc, &versioned, nil); err != nil {
		return err
	}
	cz.Set(versioned)
	return nil
}

func initConfigz(kc *kubeletconfiginternal.KubeletConfiguration) error {
	cz, err := configz.New("kubeletconfig")
	if err != nil {
		klog.ErrorS(err, "Failed to register configz")
		return err
	}
	if err := setConfigz(cz, kc); err != nil {
		klog.ErrorS(err, "Failed to register config")
		return err
	}
	return nil
}

// makeEventRecorder sets up kubeDeps.Recorder if it's nil. It's a no-op otherwise.
func makeEventRecorder(ctx context.Context, kubeDeps *kubelet.Dependencies, nodeName types.NodeName) {
	if kubeDeps.Recorder != nil {
		return
	}
	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	kubeDeps.Recorder = eventBroadcaster.NewRecorder(legacyscheme.Scheme, v1.EventSource{Component: componentKubelet, Host: string(nodeName)})
	eventBroadcaster.StartStructuredLogging(3)
	if kubeDeps.EventClient != nil {
		klog.V(4).InfoS("Sending events to api server")
		eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeDeps.EventClient.Events("")})
	} else {
		klog.InfoS("No api server defined - no events will be sent to API server")
	}
}

func getReservedCPUs(machineInfo *cadvisorapi.MachineInfo, cpus string) (cpuset.CPUSet, error) {
	emptyCPUSet := cpuset.New()

	if cpus == "" {
		return emptyCPUSet, nil
	}

	topo, err := topology.Discover(machineInfo)
	if err != nil {
		return emptyCPUSet, fmt.Errorf("unable to discover CPU topology info: %s", err)
	}
	reservedCPUSet, err := cpuset.Parse(cpus)
	if err != nil {
		return emptyCPUSet, fmt.Errorf("unable to parse reserved-cpus list: %s", err)
	}
	allCPUSet := topo.CPUDetails.CPUs()
	if !reservedCPUSet.IsSubsetOf(allCPUSet) {
		return emptyCPUSet, fmt.Errorf("reserved-cpus: %s is not a subset of online-cpus: %s", cpus, allCPUSet.String())
	}
	return reservedCPUSet, nil
}

func run(ctx context.Context, s *options.KubeletServer, kubeDeps *kubelet.Dependencies, featureGate featuregate.FeatureGate) (err error) {
	// Set global feature gates based on the value on the initial KubeletServer
	err = utilfeature.DefaultMutableFeatureGate.SetFromMap(s.KubeletConfiguration.FeatureGates)
	if err != nil {
		return err
	}
	// validate the initial KubeletServer (we set feature gates first, because this validation depends on feature gates)
	if err := options.ValidateKubeletServer(s); err != nil {
		return err
	}

	// Warn if MemoryQoS enabled with cgroups v1
	if utilfeature.DefaultFeatureGate.Enabled(features.MemoryQoS) &&
		!isCgroup2UnifiedMode() {
		klog.InfoS("Warning: MemoryQoS feature only works with cgroups v2 on Linux, but enabled with cgroups v1")
	}

	// Register current configuration with /configz endpoint
	err = initConfigz(&s.KubeletConfiguration)
	if err != nil {
		klog.ErrorS(err, "Failed to register kubelet configuration with configz")
	}

	if len(s.ShowHiddenMetricsForVersion) > 0 {
		metrics.SetShowHidden()
	}

	if kubeDeps == nil {
		kubeDeps, err = UnsecuredDependencies(s, featureGate)
		if err != nil {
			return err
		}
	}

	hostName, err := nodeutil.GetHostname(s.HostnameOverride)
	if err != nil {
		return err
	}
	nodeName, err := getNodeName(hostName)
	if err != nil {
		return err
	}

	if err := kubelet.PreInitRuntimeService(&s.KubeletConfiguration, kubeDeps); err != nil {
		return err
	}

	// Get cgroup driver setting from CRI
	if utilfeature.DefaultFeatureGate.Enabled(features.KubeletCgroupDriverFromCRI) {
		if err := getCgroupDriverFromCRI(ctx, s, kubeDeps); err != nil {
			return err
		}
	}

	var cgroupRoots []string
	nodeAllocatableRoot := cm.NodeAllocatableRoot(s.CgroupRoot, s.CgroupsPerQOS, s.CgroupDriver)
	cgroupRoots = append(cgroupRoots, nodeAllocatableRoot)
	kubeletCgroup, err := cm.GetKubeletContainer(s.KubeletCgroups)
	if err != nil {
		klog.InfoS("Failed to get the kubelet's cgroup. Kubelet system container metrics may be missing.", "err", err)
	} else if kubeletCgroup != "" {
		cgroupRoots = append(cgroupRoots, kubeletCgroup)
	}

	if s.RuntimeCgroups != "" {
		// RuntimeCgroups is optional, so ignore if it isn't specified
		cgroupRoots = append(cgroupRoots, s.RuntimeCgroups)
	}

	if s.SystemCgroups != "" {
		// SystemCgroups is optional, so ignore if it isn't specified
		cgroupRoots = append(cgroupRoots, s.SystemCgroups)
	}

	if kubeDeps.CAdvisorInterface == nil {
		imageFsInfoProvider := cadvisor.NewImageFsInfoProvider(s.ContainerRuntimeEndpoint)
		kubeDeps.CAdvisorInterface, err = cadvisor.New(imageFsInfoProvider, s.RootDirectory, cgroupRoots, cadvisor.UsingLegacyCadvisorStats(s.ContainerRuntimeEndpoint), s.LocalStorageCapacityIsolation)
		if err != nil {
			return err
		}
	}

	// Setup event recorder if required.
	makeEventRecorder(ctx, kubeDeps, nodeName)

	if kubeDeps.ContainerManager == nil {
		if s.CgroupsPerQOS && s.CgroupRoot == "" {
			klog.InfoS("--cgroups-per-qos enabled, but --cgroup-root was not specified.  defaulting to /")
			s.CgroupRoot = "/"
		}

		machineInfo, err := kubeDeps.CAdvisorInterface.MachineInfo()
		if err != nil {
			return err
		}
		reservedSystemCPUs, err := getReservedCPUs(machineInfo, s.ReservedSystemCPUs)
		if err != nil {
			return err
		}
		if reservedSystemCPUs.Size() > 0 {
			// at cmd option validation phase it is tested either --system-reserved-cgroup or --kube-reserved-cgroup is specified, so overwrite should be ok
			klog.InfoS("Option --reserved-cpus is specified, it will overwrite the cpu setting in KubeReserved and SystemReserved", "kubeReserved", s.KubeReserved, "systemReserved", s.SystemReserved)
			if s.KubeReserved != nil {
				delete(s.KubeReserved, "cpu")
			}
			if s.SystemReserved == nil {
				s.SystemReserved = make(map[string]string)
			}
			s.SystemReserved["cpu"] = strconv.Itoa(reservedSystemCPUs.Size())
			klog.InfoS("After cpu setting is overwritten", "kubeReserved", s.KubeReserved, "systemReserved", s.SystemReserved)
		}

		kubeReserved, err := parseResourceList(s.KubeReserved)
		if err != nil {
			return fmt.Errorf("--kube-reserved value failed to parse: %w", err)
		}
		systemReserved, err := parseResourceList(s.SystemReserved)
		if err != nil {
			return fmt.Errorf("--system-reserved value failed to parse: %w", err)
		}
		var hardEvictionThresholds []evictionapi.Threshold
		// If the user requested to ignore eviction thresholds, then do not set valid values for hardEvictionThresholds here.
		if !s.ExperimentalNodeAllocatableIgnoreEvictionThreshold {
			hardEvictionThresholds, err = eviction.ParseThresholdConfig([]string{}, s.EvictionHard, nil, nil, nil)
			if err != nil {
				return err
			}
		}
		experimentalQOSReserved, err := cm.ParseQOSReserved(s.QOSReserved)
		if err != nil {
			return fmt.Errorf("--qos-reserved value failed to parse: %w", err)
		}

		var cpuManagerPolicyOptions map[string]string
		if utilfeature.DefaultFeatureGate.Enabled(features.CPUManagerPolicyOptions) {
			cpuManagerPolicyOptions = s.CPUManagerPolicyOptions
		} else if s.CPUManagerPolicyOptions != nil {
			return fmt.Errorf("CPU Manager policy options %v require feature gates %q, %q enabled",
				s.CPUManagerPolicyOptions, features.CPUManager, features.CPUManagerPolicyOptions)
		}

		var topologyManagerPolicyOptions map[string]string
		if utilfeature.DefaultFeatureGate.Enabled(features.TopologyManagerPolicyOptions) {
			topologyManagerPolicyOptions = s.TopologyManagerPolicyOptions
		} else if s.TopologyManagerPolicyOptions != nil {
			return fmt.Errorf("topology manager policy options %v require feature gates %q enabled",
				s.TopologyManagerPolicyOptions, features.TopologyManagerPolicyOptions)
		}
		if utilfeature.DefaultFeatureGate.Enabled(features.NodeSwap) {
			if !isCgroup2UnifiedMode() && s.MemorySwap.SwapBehavior == kubelettypes.LimitedSwap {
				// This feature is not supported for cgroupv1 so we are failing early.
				return fmt.Errorf("swap feature is enabled and LimitedSwap but it is only supported with cgroupv2")
			}
			if !s.FailSwapOn && s.MemorySwap.SwapBehavior == "" {
				// This is just a log because we are using a default of NoSwap.
				klog.InfoS("NoSwap is set due to memorySwapBehavior not specified", "memorySwapBehavior", s.MemorySwap.SwapBehavior, "FailSwapOn", s.FailSwapOn)
			}
		}

		kubeDeps.ContainerManager, err = cm.NewContainerManager(
			kubeDeps.Mounter,
			kubeDeps.CAdvisorInterface,
			cm.NodeConfig{
				NodeName:              nodeName,
				RuntimeCgroupsName:    s.RuntimeCgroups,
				SystemCgroupsName:     s.SystemCgroups,
				KubeletCgroupsName:    s.KubeletCgroups,
				KubeletOOMScoreAdj:    s.OOMScoreAdj,
				CgroupsPerQOS:         s.CgroupsPerQOS,
				CgroupRoot:            s.CgroupRoot,
				CgroupDriver:          s.CgroupDriver,
				KubeletRootDir:        s.RootDirectory,
				ProtectKernelDefaults: s.ProtectKernelDefaults,
				NodeAllocatableConfig: cm.NodeAllocatableConfig{
					KubeReservedCgroupName:   s.KubeReservedCgroup,
					SystemReservedCgroupName: s.SystemReservedCgroup,
					EnforceNodeAllocatable:   sets.New(s.EnforceNodeAllocatable...),
					KubeReserved:             kubeReserved,
					SystemReserved:           systemReserved,
					ReservedSystemCPUs:       reservedSystemCPUs,
					HardEvictionThresholds:   hardEvictionThresholds,
				},
				QOSReserved:                             *experimentalQOSReserved,
				CPUManagerPolicy:                        s.CPUManagerPolicy,
				CPUManagerPolicyOptions:                 cpuManagerPolicyOptions,
				CPUManagerReconcilePeriod:               s.CPUManagerReconcilePeriod.Duration,
				ExperimentalMemoryManagerPolicy:         s.MemoryManagerPolicy,
				ExperimentalMemoryManagerReservedMemory: s.ReservedMemory,
				PodPidsLimit:                            s.PodPidsLimit,
				EnforceCPULimits:                        s.CPUCFSQuota,
				CPUCFSQuotaPeriod:                       s.CPUCFSQuotaPeriod.Duration,
				TopologyManagerPolicy:                   s.TopologyManagerPolicy,
				TopologyManagerScope:                    s.TopologyManagerScope,
				TopologyManagerPolicyOptions:            topologyManagerPolicyOptions,
			},
			s.FailSwapOn,
			kubeDeps.Recorder,
			kubeDeps.KubeClient,
		)

		if err != nil {
			return err
		}
	}

	if kubeDeps.PodStartupLatencyTracker == nil {
		kubeDeps.PodStartupLatencyTracker = kubeletutil.NewPodStartupLatencyTracker()
	}

	if kubeDeps.NodeStartupLatencyTracker == nil {
		kubeDeps.NodeStartupLatencyTracker = kubeletutil.NewNodeStartupLatencyTracker()
	}

	// TODO(vmarmol): Do this through container config.
	oomAdjuster := kubeDeps.OOMAdjuster
	if err := oomAdjuster.ApplyOOMScoreAdj(0, int(s.OOMScoreAdj)); err != nil {
		klog.InfoS("Failed to ApplyOOMScoreAdj", "err", err)
	}

	if err := RunKubelet(s, kubeDeps, s.RunOnce); err != nil {
		return err
	}

	if s.RunOnce {
		return nil
	}

	// If systemd is used, notify it that we have started
	go daemon.SdNotify(false, "READY=1")

	select {
	case <-ctx.Done():
		break
	}

	return nil
}

// buildKubeletClientConfig constructs the appropriate client config for the kubelet depending on whether
// bootstrapping is enabled or client certificate rotation is enabled.
func buildKubeletClientConfig(ctx context.Context, s *options.KubeletServer, tp oteltrace.TracerProvider, nodeName types.NodeName) (*restclient.Config, func(), error) {
	return nil, nil, nil
}

// updateDialer instruments a restconfig with a dial. the returned function allows forcefully closing all active connections.
func updateDialer(clientConfig *restclient.Config) (func(), error) {
	if clientConfig.Transport != nil || clientConfig.Dial != nil {
		return nil, fmt.Errorf("there is already a transport or dialer configured")
	}
	d := connrotation.NewDialer((&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext)
	clientConfig.Dial = d.DialContext
	return d.CloseAll, nil
}

func kubeClientConfigOverrides(s *options.KubeletServer, clientConfig *restclient.Config) {
	setContentTypeForClient(clientConfig, s.ContentType)
	// Override kubeconfig qps/burst settings from flags
	clientConfig.QPS = float32(s.KubeAPIQPS)
	clientConfig.Burst = int(s.KubeAPIBurst)
}

// getNodeName returns the node name according to the cloud provider
// if cloud provider is specified. Otherwise, returns the hostname of the node.
func getNodeName(hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

// setContentTypeForClient sets the appropriate content type into the rest config
// and handles defaulting AcceptContentTypes based on that input.
func setContentTypeForClient(cfg *restclient.Config, contentType string) {
	if len(contentType) == 0 {
		return
	}
	cfg.ContentType = contentType
	switch contentType {
	case runtime.ContentTypeProtobuf:
		cfg.AcceptContentTypes = strings.Join([]string{runtime.ContentTypeProtobuf, runtime.ContentTypeJSON}, ",")
	default:
		// otherwise let the rest client perform defaulting
	}
}

// RunKubelet is responsible for setting up and running a kubelet.  It is used in three different applications:
//
//	1 Integration tests
//	2 Kubelet binary
//	3 Standalone 'kubernetes' binary
//
// Eventually, #2 will be replaced with instances of #3
func RunKubelet(kubeServer *options.KubeletServer, kubeDeps *kubelet.Dependencies, runOnce bool) error {
	hostname, err := nodeutil.GetHostname(kubeServer.HostnameOverride)
	if err != nil {
		return err
	}
	// Query the cloud provider for our node name, default to hostname if kubeDeps.Cloud == nil
	nodeName, err := getNodeName(hostname)
	if err != nil {
		return err
	}
	hostnameOverridden := len(kubeServer.HostnameOverride) > 0
	// Setup event recorder if required.
	makeEventRecorder(context.TODO(), kubeDeps, nodeName)

	nodeIPs, err := nodeutil.ParseNodeIPArgument(kubeServer.NodeIP, kubeServer.CloudProvider)
	if err != nil {
		return fmt.Errorf("bad --node-ip %q: %v", kubeServer.NodeIP, err)
	}

	capabilities.Initialize(capabilities.Capabilities{
		AllowPrivileged: true,
	})

	credentialprovider.SetPreferredDockercfgPath(kubeServer.RootDirectory)
	klog.V(2).InfoS("Using root directory", "path", kubeServer.RootDirectory)

	if kubeDeps.OSInterface == nil {
		kubeDeps.OSInterface = kubecontainer.RealOS{}
	}

	k, err := createAndInitKubelet(kubeServer,
		kubeDeps,
		hostname,
		hostnameOverridden,
		nodeName,
		nodeIPs)
	if err != nil {
		return fmt.Errorf("failed to create kubelet: %w", err)
	}

	// NewMainKubelet should have set up a pod source config if one didn't exist
	// when the builder was run. This is just a precaution.
	if kubeDeps.PodConfig == nil {
		return fmt.Errorf("failed to create kubelet, pod source config was nil")
	}
	podCfg := kubeDeps.PodConfig

	if err := rlimit.SetNumFiles(uint64(kubeServer.MaxOpenFiles)); err != nil {
		klog.ErrorS(err, "Failed to set rlimit on max file handles")
	}

	startKubelet(k, podCfg, &kubeServer.KubeletConfiguration, kubeDeps, kubeServer.EnableServer)
	klog.InfoS("Started kubelet")
	return nil
}

func startKubelet(k kubelet.Bootstrap, podCfg *config.PodConfig, kubeCfg *kubeletconfiginternal.KubeletConfiguration, kubeDeps *kubelet.Dependencies, enableServer bool) {
	// start the kubelet
	go k.Run(podCfg.Updates())

	// start the kubelet server
	if kubeCfg.ReadOnlyPort > 0 {
		go k.ListenAndServeReadOnly(kubeCfg)
	}
}

func createAndInitKubelet(kubeServer *options.KubeletServer,
	kubeDeps *kubelet.Dependencies,
	hostname string,
	hostnameOverridden bool,
	nodeName types.NodeName,
	nodeIPs []net.IP) (k kubelet.Bootstrap, err error) {
	// TODO: block until all sources have delivered at least one update to the channel, or break the sync loop
	// up into "per source" synchronizations

	k, err = kubelet.NewMainKubelet(&kubeServer.KubeletConfiguration,
		kubeDeps,
		&kubeServer.ContainerRuntimeOptions,
		hostname,
		hostnameOverridden,
		nodeName,
		nodeIPs,
		kubeServer.CertDirectory,
		kubeServer.RootDirectory,
		kubeServer.PodLogsDir,
		kubeServer.ImageCredentialProviderConfigFile,
		kubeServer.ImageCredentialProviderBinDir,
		kubeServer.RegisterNode,
		kubeServer.RegisterWithTaints,
		kubeServer.AllowedUnsafeSysctls,
		kubeServer.ExperimentalMounterPath,
		kubeServer.KernelMemcgNotification,
		kubeServer.ExperimentalNodeAllocatableIgnoreEvictionThreshold,
		kubeServer.MinimumGCAge,
		kubeServer.MaxPerPodContainerCount,
		kubeServer.MaxContainerCount,
		kubeServer.RegisterSchedulable,
		kubeServer.KeepTerminatedPodVolumes,
		kubeServer.NodeLabels,
		kubeServer.NodeStatusMaxImages,
		kubeServer.KubeletFlags.SeccompDefault || kubeServer.KubeletConfiguration.SeccompDefault)
	if err != nil {
		return nil, err
	}

	k.BirthCry()

	k.StartGarbageCollection()

	return k, nil
}

// parseResourceList parses the given configuration map into an API
// ResourceList or returns an error.
func parseResourceList(m map[string]string) (v1.ResourceList, error) {
	if len(m) == 0 {
		return nil, nil
	}
	rl := make(v1.ResourceList)
	for k, v := range m {
		switch v1.ResourceName(k) {
		// CPU, memory, local storage, and PID resources are supported.
		case v1.ResourceCPU, v1.ResourceMemory, v1.ResourceEphemeralStorage, pidlimit.PIDs:
			q, err := resource.ParseQuantity(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse quantity %q for %q resource: %w", v, k, err)
			}
			if q.Sign() == -1 {
				return nil, fmt.Errorf("resource quantity for %q cannot be negative: %v", k, v)
			}
			rl[v1.ResourceName(k)] = q
		default:
			return nil, fmt.Errorf("cannot reserve %q resource", k)
		}
	}
	return rl, nil
}

func newTracerProvider(s *options.KubeletServer) (oteltrace.TracerProvider, error) {
	if s.KubeletConfiguration.Tracing == nil {
		return oteltrace.NewNoopTracerProvider(), nil
	}
	hostname, err := nodeutil.GetHostname(s.HostnameOverride)
	if err != nil {
		return nil, fmt.Errorf("could not determine hostname for tracer provider: %w", err)
	}
	resourceOpts := []otelsdkresource.Option{
		otelsdkresource.WithAttributes(
			semconv.ServiceNameKey.String(componentKubelet),
			semconv.HostNameKey.String(hostname),
		),
	}
	tp, err := tracing.NewProvider(context.Background(), s.KubeletConfiguration.Tracing, []otlptracegrpc.Option{}, resourceOpts)
	if err != nil {
		return nil, fmt.Errorf("could not configure tracer provider: %w", err)
	}
	return tp, nil
}

func getCgroupDriverFromCRI(ctx context.Context, s *options.KubeletServer, kubeDeps *kubelet.Dependencies) error {
	klog.V(4).InfoS("Getting CRI runtime configuration information")

	var (
		runtimeConfig *runtimeapi.RuntimeConfigResponse
		err           error
	)
	// Retry a couple of times, hoping that any errors are transient.
	// Fail quickly on known, non transient errors.
	for i := 0; i < 3; i++ {
		runtimeConfig, err = kubeDeps.RemoteRuntimeService.RuntimeConfig(ctx)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok || s.Code() != codes.Unimplemented {
				// We could introduce a backoff delay or jitter, but this is largely catching cases
				// where the runtime is still starting up and we request too early.
				// Give it a little more time.
				time.Sleep(time.Second * 2)
				continue
			}
			// CRI implementation doesn't support RuntimeConfig, fallback
			klog.InfoS("CRI implementation should be updated to support RuntimeConfig when KubeletCgroupDriverFromCRI feature gate has been enabled. Falling back to using cgroupDriver from kubelet config.")
			return nil
		}
	}
	if err != nil {
		return err
	}

	// Calling GetLinux().GetCgroupDriver() won't segfault, but it will always default to systemd
	// which is not intended by the fields not being populated
	linuxConfig := runtimeConfig.GetLinux()
	if linuxConfig == nil {
		return nil
	}

	switch d := linuxConfig.GetCgroupDriver(); d {
	case runtimeapi.CgroupDriver_SYSTEMD:
		s.CgroupDriver = "systemd"
	case runtimeapi.CgroupDriver_CGROUPFS:
		s.CgroupDriver = "cgroupfs"
	default:
		return fmt.Errorf("runtime returned an unknown cgroup driver %d", d)
	}
	klog.InfoS("Using cgroup driver setting received from the CRI runtime", "cgroupDriver", s.CgroupDriver)
	return nil
}
