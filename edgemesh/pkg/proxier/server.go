/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

-----------------------------------------------------------------------------
CHANGELOG
KubeEdge Authors:
- Remove useless functions and adjust logic
- Use MsgProcess replaced with informer

*/

package proxier

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	gerrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/component-base/configz"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/component-base/version"
	"k8s.io/klog/v2"
	"k8s.io/kube-proxy/config/v1alpha1"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/master/ports"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/apis"
	kubeproxyconfig "k8s.io/kubernetes/pkg/proxy/apis/config"
	proxyconfigscheme "k8s.io/kubernetes/pkg/proxy/apis/config/scheme"
	kubeproxyconfigv1alpha1 "k8s.io/kubernetes/pkg/proxy/apis/config/v1alpha1"
	"k8s.io/kubernetes/pkg/proxy/apis/config/validation"
	"k8s.io/kubernetes/pkg/proxy/config"
	"k8s.io/kubernetes/pkg/proxy/healthcheck"
	"k8s.io/kubernetes/pkg/proxy/iptables"
	"k8s.io/kubernetes/pkg/proxy/ipvs"
	"k8s.io/kubernetes/pkg/proxy/userspace"
	proxyutil "k8s.io/kubernetes/pkg/proxy/util"
	"k8s.io/kubernetes/pkg/util/filesystem"
	utilipset "k8s.io/kubernetes/pkg/util/ipset"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilipvs "k8s.io/kubernetes/pkg/util/ipvs"
	"k8s.io/kubernetes/pkg/util/oom"
	"k8s.io/utils/exec"
)

const (
	proxyModeUserspace   = "userspace"
	proxyModeIPTables    = "iptables"
	proxyModeIPVS        = "ipvs"
	proxyModeKernelspace = "kernelspace"
)

// proxyRun defines the interface to run a specified ProxyServer
type proxyRun interface {
	Run() error
	CleanupAndExit() error
}

// Options contains everything necessary to create and run a proxy server.
type Options struct {
	// ConfigFile is the location of the proxy server's configuration file.
	ConfigFile string
	// WriteConfigTo is the path where the default configuration will be written.
	WriteConfigTo string
	// CleanupAndExit, when true, makes the proxy server clean up iptables and ipvs rules, then exit.
	CleanupAndExit bool
	// CleanupIPVS, when true, makes the proxy server clean up ipvs rules before running.
	CleanupIPVS bool
	// WindowsService should be set to true if kube-proxy is running as a service on Windows.
	// Its corresponding flag only gets registered in Windows builds
	WindowsService bool
	// config is the proxy server's configuration object.
	config *kubeproxyconfig.KubeProxyConfiguration
	// watcher is used to watch on the update change of ConfigFile
	watcher filesystem.FSWatcher
	// proxyServer is the interface to run the proxy server
	proxyServer proxyRun
	// errCh is the channel that errors will be sent
	errCh chan error

	// The fields below here are placeholders for flags that can't be directly mapped into
	// config.KubeProxyConfiguration.
	//
	// TODO remove these fields once the deprecated flags are removed.

	// master is used to override the kubeconfig's URL to the apiserver.
	master string
	// healthzPort is the port to be used by the healthz server.
	healthzPort int32
	// metricsPort is the port to be used by the metrics server.
	metricsPort int32

	// hostnameOverride, if set from the command line flag, takes precedence over the `HostnameOverride` value from the config file
	hostnameOverride string
}

// NewOptions returns initialized Options
func NewOptions() *Options {
	return &Options{
		config:      new(kubeproxyconfig.KubeProxyConfiguration),
		healthzPort: ports.ProxyHealthzPort,
		metricsPort: ports.ProxyStatusPort,
		CleanupIPVS: true,
		errCh:       make(chan error),
	}
}

// Complete completes all the required options.
func (o *Options) Complete() error {
	if len(o.ConfigFile) == 0 && len(o.WriteConfigTo) == 0 {
		klog.Warning("WARNING: all flags other than --config, --write-config-to, and --cleanup are deprecated. Please begin using a config file ASAP.")
		o.config.HealthzBindAddress = addressFromDeprecatedFlags(o.config.HealthzBindAddress, o.healthzPort)
		o.config.MetricsBindAddress = addressFromDeprecatedFlags(o.config.MetricsBindAddress, o.metricsPort)
	}

	// Load the config file here in Complete, so that Validate validates the fully-resolved config.
	if len(o.ConfigFile) > 0 {
		c, err := o.loadConfigFromFile(o.ConfigFile)
		if err != nil {
			return err
		}
		o.config = c

		if err := o.initWatcher(); err != nil {
			return err
		}
	}

	if err := o.processHostnameOverrideFlag(); err != nil {
		return err
	}

	return utilfeature.DefaultMutableFeatureGate.SetFromMap(o.config.FeatureGates)
}

// Creates a new filesystem watcher and adds watches for the config file.
func (o *Options) initWatcher() error {
	fswatcher := filesystem.NewFsnotifyWatcher()
	err := fswatcher.Init(o.eventHandler, o.errorHandler)
	if err != nil {
		return err
	}
	err = fswatcher.AddWatch(o.ConfigFile)
	if err != nil {
		return err
	}
	o.watcher = fswatcher
	return nil
}

func (o *Options) eventHandler(ent fsnotify.Event) {
	eventOpIs := func(Op fsnotify.Op) bool {
		return ent.Op&Op == Op
	}
	if eventOpIs(fsnotify.Write) || eventOpIs(fsnotify.Rename) {
		// error out when ConfigFile is updated
		o.errCh <- fmt.Errorf("content of the proxy server's configuration file was updated")
		return
	}
	o.errCh <- nil
}

func (o *Options) errorHandler(err error) {
	o.errCh <- err
}

// processHostnameOverrideFlag processes hostname-override flag
func (o *Options) processHostnameOverrideFlag() error {
	// Check if hostname-override flag is set and use value since configFile always overrides
	if len(o.hostnameOverride) > 0 {
		hostName := strings.TrimSpace(o.hostnameOverride)
		if len(hostName) == 0 {
			return fmt.Errorf("empty hostname-override is invalid")
		}
		o.config.HostnameOverride = strings.ToLower(hostName)
	}

	return nil
}

// Validate validates all the required options.
func (o *Options) Validate() error {
	if errs := validation.Validate(o.config); len(errs) != 0 {
		return errs.ToAggregate()
	}

	return nil
}

// Run runs the specified ProxyServer.
func (o *Options) Run() error {
	defer close(o.errCh)
	if len(o.WriteConfigTo) > 0 {
		return o.writeConfigFile()
	}

	proxyServer, err := NewProxyServer(o)
	if err != nil {
		return err
	}

	if o.CleanupAndExit {
		return proxyServer.CleanupAndExit()
	}

	o.proxyServer = proxyServer
	return o.runLoop()
}

// runLoop will watch on the update change of the proxy server's configuration file.
// Return an error when updated
func (o *Options) runLoop() error {
	if o.watcher != nil {
		o.watcher.Run()
	}

	// run the proxy in goroutine
	go func() {
		err := o.proxyServer.Run()
		o.errCh <- err
	}()

	for {
		err := <-o.errCh
		if err != nil {
			return err
		}
	}
}

func (o *Options) writeConfigFile() (err error) {
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(proxyconfigscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return fmt.Errorf("unable to locate encoder -- %q is not a supported media type", mediaType)
	}

	encoder := proxyconfigscheme.Codecs.EncoderForVersion(info.Serializer, v1alpha1.SchemeGroupVersion)

	configFile, err := os.Create(o.WriteConfigTo)
	if err != nil {
		return err
	}

	defer func() {
		ferr := configFile.Close()
		if ferr != nil && err == nil {
			err = ferr
		}
	}()

	if err = encoder.Encode(o.config, configFile); err != nil {
		return err
	}

	klog.Infof("Wrote configuration to: %s\n", o.WriteConfigTo)

	return nil
}

// addressFromDeprecatedFlags returns server address from flags
// passed on the command line based on the following rules:
// 1. If port is 0, disable the server (e.g. set address to empty).
// 2. Otherwise, set the port portion of the config accordingly.
func addressFromDeprecatedFlags(addr string, port int32) string {
	if port == 0 {
		return ""
	}
	return proxyutil.AppendPortIfNeeded(addr, port)
}

// newLenientSchemeAndCodecs returns a scheme that has only v1alpha1 registered into
// it and a CodecFactory with strict decoding disabled.
func newLenientSchemeAndCodecs() (*runtime.Scheme, *serializer.CodecFactory, error) {
	lenientScheme := runtime.NewScheme()
	if err := kubeproxyconfig.AddToScheme(lenientScheme); err != nil {
		return nil, nil, fmt.Errorf("failed to add kube-proxy config API to lenient scheme: %v", err)
	}
	if err := kubeproxyconfigv1alpha1.AddToScheme(lenientScheme); err != nil {
		return nil, nil, fmt.Errorf("failed to add kube-proxy config v1alpha1 API to lenient scheme: %v", err)
	}
	lenientCodecs := serializer.NewCodecFactory(lenientScheme, serializer.DisableStrict)
	return lenientScheme, &lenientCodecs, nil
}

// loadConfigFromFile loads the contents of file and decodes it as a
// KubeProxyConfiguration object.
func (o *Options) loadConfigFromFile(file string) (*kubeproxyconfig.KubeProxyConfiguration, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return o.loadConfig(data)
}

// loadConfig decodes a serialized KubeProxyConfiguration to the internal type.
func (o *Options) loadConfig(data []byte) (*kubeproxyconfig.KubeProxyConfiguration, error) {

	configObj, gvk, err := proxyconfigscheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		// Try strict decoding first. If that fails decode with a lenient
		// decoder, which has only v1alpha1 registered, and log a warning.
		// The lenient path is to be dropped when support for v1alpha1 is dropped.
		if !runtime.IsStrictDecodingError(err) {
			return nil, gerrors.Wrap(err, "failed to decode")
		}

		_, lenientCodecs, lenientErr := newLenientSchemeAndCodecs()
		if lenientErr != nil {
			return nil, lenientErr
		}

		configObj, gvk, lenientErr = lenientCodecs.UniversalDecoder().Decode(data, nil, nil)
		if lenientErr != nil {
			// Lenient decoding failed with the current version, return the
			// original strict error.
			return nil, fmt.Errorf("failed lenient decoding: %v", err)
		}

		// Continue with the v1alpha1 object that was decoded leniently, but emit a warning.
		klog.Warningf("using lenient decoding as strict decoding failed: %v", err)
	}

	proxyConfig, ok := configObj.(*kubeproxyconfig.KubeProxyConfiguration)
	if !ok {
		return nil, fmt.Errorf("got unexpected config type: %v", gvk)
	}
	return proxyConfig, nil
}

// ApplyDefaults applies the default values to Options.
func (o *Options) ApplyDefaults(in *kubeproxyconfig.KubeProxyConfiguration) (*kubeproxyconfig.KubeProxyConfiguration, error) {
	external, err := proxyconfigscheme.Scheme.ConvertToVersion(in, v1alpha1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}

	proxyconfigscheme.Scheme.Default(external)

	internal, err := proxyconfigscheme.Scheme.ConvertToVersion(external, kubeproxyconfig.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}

	out := internal.(*kubeproxyconfig.KubeProxyConfiguration)

	return out, nil
}

// ProxyServer represents all the parameters required to start the Kubernetes proxy server. All
// fields are required.
type ProxyServer struct {
	Client                 clientset.Interface
	EventClient            v1core.EventsGetter
	IptInterface           utiliptables.Interface
	IpvsInterface          utilipvs.Interface
	IpsetInterface         utilipset.Interface
	execer                 exec.Interface
	Proxier                proxy.Provider
	Broadcaster            record.EventBroadcaster
	Recorder               record.EventRecorder
	ConntrackConfiguration kubeproxyconfig.KubeProxyConntrackConfiguration
	Conntracker            Conntracker // if nil, ignored
	ProxyMode              string
	NodeRef                *v1.ObjectReference
	CleanupIPVS            bool
	MetricsBindAddress     string
	BindAddressHardFail    bool
	EnableProfiling        bool
	UseEndpointSlices      bool
	OOMScoreAdj            *int32
	ConfigSyncPeriod       time.Duration
	HealthzServer          healthcheck.ProxierHealthUpdater
	servicesMap            sync.Map
	endpointsMap           sync.Map
}

// createClients creates a kube client and an event client from the given config and masterOverride.
// TODO remove masterOverride when CLI flags are removed.
func createClients(config componentbaseconfig.ClientConnectionConfiguration, masterOverride string) (clientset.Interface, v1core.EventsGetter, error) {
	var kubeConfig *rest.Config
	var err error

	if len(config.Kubeconfig) == 0 && len(masterOverride) == 0 {
		klog.Info("Neither kubeconfig file nor master URL was specified. Falling back to in-cluster config.")
		kubeConfig, err = rest.InClusterConfig()
	} else {
		// This creates a client, first loading any specified kubeconfig
		// file, and then overriding the Master flag, if non-empty.
		kubeConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: config.Kubeconfig},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterOverride}}).ClientConfig()
	}
	if err != nil {
		return nil, nil, err
	}

	kubeConfig.AcceptContentTypes = config.AcceptContentTypes
	kubeConfig.ContentType = config.ContentType
	kubeConfig.QPS = config.QPS
	kubeConfig.Burst = int(config.Burst)

	client, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	eventClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	return client, eventClient.CoreV1(), nil
}

func serveHealthz(hz healthcheck.ProxierHealthUpdater, errCh chan error) {
	if hz == nil {
		return
	}

	fn := func() {
		err := hz.Run()
		if err != nil {
			klog.Errorf("healthz server failed: %v", err)
			if errCh != nil {
				errCh <- fmt.Errorf("healthz server failed: %v", err)
				// if in hardfail mode, never retry again
				blockCh := make(chan error)
				<-blockCh
			}
		} else {
			klog.Errorf("healthz server returned without error")
		}
	}
	go wait.Until(fn, 5*time.Second, wait.NeverStop)
}

func serveMetrics(bindAddress, proxyMode string, enableProfiling bool, errCh chan error) {
	if len(bindAddress) == 0 {
		return
	}

	proxyMux := mux.NewPathRecorderMux("kube-proxy")
	healthz.InstallHandler(proxyMux)
	proxyMux.HandleFunc("/proxyMode", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fmt.Fprintf(w, "%s", proxyMode)
	})

	//lint:ignore SA1019 See the Metrics Stability Migration KEP
	proxyMux.Handle("/metrics", legacyregistry.Handler())

	if enableProfiling {
		routes.Profiling{}.Install(proxyMux)
	}

	configz.InstallHandler(proxyMux)

	fn := func() {
		err := http.ListenAndServe(bindAddress, proxyMux)
		if err != nil {
			err = fmt.Errorf("starting metrics server failed: %v", err)
			utilruntime.HandleError(err)
			if errCh != nil {
				errCh <- err
				// if in hardfail mode, never retry again
				blockCh := make(chan error)
				<-blockCh
			}
		}
	}
	go wait.Until(fn, 5*time.Second, wait.NeverStop)
}

// Run runs the specified ProxyServer.  This should never exit (unless CleanupAndExit is set).
// TODO: At the moment, Run() cannot return a nil error, otherwise it's caller will never exit. Update callers of Run to handle nil errors.
func (s *ProxyServer) Run() error {
	// To help debugging, immediately log version
	klog.Infof("Version: %+v", version.Get())

	// TODO(vmarmol): Use container config for this.
	var oomAdjuster *oom.OOMAdjuster
	if s.OOMScoreAdj != nil {
		oomAdjuster = oom.NewOOMAdjuster()
		if err := oomAdjuster.ApplyOOMScoreAdj(0, int(*s.OOMScoreAdj)); err != nil {
			klog.V(2).Info(err)
		}
	}

	if s.Broadcaster != nil && s.EventClient != nil {
		s.Broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: s.EventClient.Events("")})
	}

	// TODO(thockin): make it possible for healthz and metrics to be on the same port.

	var errCh chan error
	if s.BindAddressHardFail {
		errCh = make(chan error)
	}

	// Start up a healthz server if requested
	serveHealthz(s.HealthzServer, errCh)

	// Start up a metrics server if requested
	serveMetrics(s.MetricsBindAddress, s.ProxyMode, s.EnableProfiling, errCh)

	// Tune conntrack, if requested
	// Conntracker is always nil for windows
	if s.Conntracker != nil {
		max, err := getConntrackMax(s.ConntrackConfiguration)
		if err != nil {
			return err
		}
		if max > 0 {
			err := s.Conntracker.SetMax(max)
			if err != nil {
				if err != errReadOnlySysFS {
					return err
				}
				// errReadOnlySysFS is caused by a known docker issue (https://github.com/docker/docker/issues/24000),
				// the only remediation we know is to restart the docker daemon.
				// Here we'll send an node event with specific reason and message, the
				// administrator should decide whether and how to handle this issue,
				// whether to drain the node and restart docker.  Occurs in other container runtimes
				// as well.
				// TODO(random-liu): Remove this when the docker bug is fixed.
				const message = "CRI error: /sys is read-only: " +
					"cannot modify conntrack limits, problems may arise later (If running Docker, see docker issue #24000)"
				s.Recorder.Eventf(s.NodeRef, api.EventTypeWarning, err.Error(), message)
			}
		}

		if s.ConntrackConfiguration.TCPEstablishedTimeout != nil && s.ConntrackConfiguration.TCPEstablishedTimeout.Duration > 0 {
			timeout := int(s.ConntrackConfiguration.TCPEstablishedTimeout.Duration / time.Second)
			if err := s.Conntracker.SetTCPEstablishedTimeout(timeout); err != nil {
				return err
			}
		}

		if s.ConntrackConfiguration.TCPCloseWaitTimeout != nil && s.ConntrackConfiguration.TCPCloseWaitTimeout.Duration > 0 {
			timeout := int(s.ConntrackConfiguration.TCPCloseWaitTimeout.Duration / time.Second)
			if err := s.Conntracker.SetTCPCloseWaitTimeout(timeout); err != nil {
				return err
			}
		}
	}

	noProxyName, err := labels.NewRequirement(apis.LabelServiceProxyName, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	noHeadlessEndpoints, err := labels.NewRequirement(v1.IsHeadlessService, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*noProxyName, *noHeadlessEndpoints)

	// Make informers that filter out objects that want a non-default service proxy.
	informerFactory := informers.NewSharedInformerFactoryWithOptions(s.Client, s.ConfigSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}))

	// Create configs (i.e. Watches for Services and Endpoints or EndpointSlices)
	// Note: RegisterHandler() calls need to happen before creation of Sources because sources
	// only notify on changes, and the initial update (on process start) may be lost if no handlers
	// are registered yet.
	serviceConfig := config.NewServiceConfig(informerFactory.Core().V1().Services(), s.ConfigSyncPeriod)
	serviceConfig.RegisterEventHandler(s.Proxier)
	go serviceConfig.Run(wait.NeverStop)

	if s.UseEndpointSlices {
		endpointSliceConfig := config.NewEndpointSliceConfig(informerFactory.Discovery().V1beta1().EndpointSlices(), s.ConfigSyncPeriod)
		endpointSliceConfig.RegisterEventHandler(s.Proxier)
		go endpointSliceConfig.Run(wait.NeverStop)
	} else {
		endpointsConfig := config.NewEndpointsConfig(informerFactory.Core().V1().Endpoints(), s.ConfigSyncPeriod)
		endpointsConfig.RegisterEventHandler(s.Proxier)
		go endpointsConfig.Run(wait.NeverStop)
	}

	// This has to start after the calls to NewServiceConfig and NewEndpointsConfig because those
	// functions must configure their shared informer event handlers first.
	informerFactory.Start(wait.NeverStop)

	if utilfeature.DefaultFeatureGate.Enabled(features.ServiceTopology) {
		// Make an informer that selects for our nodename.
		currentNodeInformerFactory := informers.NewSharedInformerFactoryWithOptions(s.Client, s.ConfigSyncPeriod,
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", s.NodeRef.Name).String()
			}))
		nodeConfig := config.NewNodeConfig(currentNodeInformerFactory.Core().V1().Nodes(), s.ConfigSyncPeriod)
		nodeConfig.RegisterEventHandler(s.Proxier)
		go nodeConfig.Run(wait.NeverStop)

		// This has to start after the calls to NewNodeConfig because that must
		// configure the shared informer event handler first.
		currentNodeInformerFactory.Start(wait.NeverStop)
	}

	go s.Proxier.SyncLoop()

	return <-errCh
}

func getConntrackMax(config kubeproxyconfig.KubeProxyConntrackConfiguration) (int, error) {
	if config.MaxPerCore != nil && *config.MaxPerCore > 0 {
		floor := 0
		if config.Min != nil {
			floor = int(*config.Min)
		}
		scaled := int(*config.MaxPerCore) * goruntime.NumCPU()
		if scaled > floor {
			klog.V(3).Infof("getConntrackMax: using scaled conntrack-max-per-core")
			return scaled, nil
		}
		klog.V(3).Infof("getConntrackMax: using conntrack-min")
		return floor, nil
	}
	return 0, nil
}

// CleanupAndExit remove iptables rules and exit if success return nil
func (s *ProxyServer) CleanupAndExit() error {
	encounteredError := userspace.CleanupLeftovers(s.IptInterface)
	encounteredError = iptables.CleanupLeftovers(s.IptInterface) || encounteredError
	encounteredError = ipvs.CleanupLeftovers(s.IpvsInterface, s.IptInterface, s.IpsetInterface, s.CleanupIPVS) || encounteredError
	if encounteredError {
		return errors.New("encountered an error while tearing down rules")
	}

	return nil
}

// getResourceType returns the resource type as a string
func getResourceType(resource string) string {
	str := strings.Split(resource, "/")
	if len(str) == 3 {
		return str[1]
	} else if len(str) == 5 {
		return str[3]
	} else {
		return resource
	}
}

func getServices(msg model.Message, resourceType string) ([]v1.Service, error) {
	content, err := json.Marshal(msg.GetContent())
	if err != nil {
		klog.V(3).Infof("[EdgeMesh] failed to marshal message: %v, err: %v", msg, err)
	}

	serviceList := make([]v1.Service, 0)
	switch resourceType {
	case constants.ResourceTypeService:
		var s v1.Service
		err = json.Unmarshal(content, &s)
		if err != nil {
			klog.Errorf("[EdgeMesh] failed to unmarshal message to service, err: %v", err)
			return nil, err
		}
		serviceList = append(serviceList, s)
	case constants.ResourceTypeServiceList:
		var s []v1.Service
		err = json.Unmarshal(content, &s)
		if err != nil {
			klog.Errorf("[EdgeMesh] failed to unmarshal message to service, err: %v", err)
			return nil, err
		}
		serviceList = append(serviceList, s...)
	}

	return serviceList, nil
}

func getEndpoints(msg model.Message, resourceType string) ([]v1.Endpoints, error) {
	content, err := json.Marshal(msg.GetContent())
	if err != nil {
		klog.V(3).Infof("[EdgeMesh] failed to marshal message: %v, err: %v", msg, err)
	}

	endpointsList := make([]v1.Endpoints, 0)
	switch resourceType {
	case constants.ResourceTypeEndpoints:
		var e v1.Endpoints
		err = json.Unmarshal(content, &e)
		if err != nil {
			klog.Errorf("[EdgeMesh] failed to unmarshal message to service, err: %v", err)
			return nil, err
		}
		endpointsList = append(endpointsList, e)
	case constants.ResourceTypeEndpointsList:
		var e []v1.Endpoints
		err = json.Unmarshal(content, &e)
		if err != nil {
			klog.Errorf("[EdgeMesh] failed to unmarshal message to service list, err: %v", err)
			return nil, err
		}
		endpointsList = append(endpointsList, e...)
	}

	return endpointsList, nil
}

func (s *ProxyServer) handleServices(serviceList []v1.Service, operation string) {
	for _, service := range serviceList {
		key := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
		switch operation {
		case "insert":
			s.servicesMap.Store(key, service)
			s.Proxier.OnServiceAdd(&service)
			s.Proxier.OnServiceSynced()
		case "update":
			value, loadOk := s.servicesMap.Load(key)
			oldService, isService := value.(v1.Service)
			if !loadOk || !isService {
				s.servicesMap.Store(key, service)
				s.Proxier.OnServiceAdd(&service)
				s.Proxier.OnServiceSynced()
			} else {
				s.servicesMap.Store(key, service)
				s.Proxier.OnServiceUpdate(&oldService, &service)
				s.Proxier.OnServiceSynced()
			}
		case "delete":
			s.servicesMap.Delete(key)
			s.Proxier.OnServiceDelete(&service)
			s.Proxier.OnServiceSynced()
		default:
			klog.Warningf("[EdgeMesh] invalid %s operation on services", operation)
		}
	}
}

func (s *ProxyServer) handleEndpoints(endpointsList []v1.Endpoints, operation string) {
	for _, endpoint := range endpointsList {
		key := fmt.Sprintf("%s/%s", endpoint.Namespace, endpoint.Name)
		switch operation {
		case "insert":
			s.endpointsMap.Store(key, endpoint)
			s.Proxier.OnEndpointsAdd(&endpoint)
			s.Proxier.OnEndpointsSynced()
		case "update":
			value, loadOk := s.servicesMap.Load(key)
			oldEndpoint, isEndpoint := value.(v1.Endpoints)
			if !loadOk || !isEndpoint {
				s.endpointsMap.Store(key, endpoint)
				s.Proxier.OnEndpointsAdd(&endpoint)
				s.Proxier.OnEndpointsSynced()
			} else {
				s.endpointsMap.Store(key, endpoint)
				s.Proxier.OnEndpointsUpdate(&oldEndpoint, &endpoint)
				s.Proxier.OnEndpointsSynced()
			}
		case "delete":
			s.endpointsMap.Delete(key)
			s.Proxier.OnEndpointsDelete(&endpoint)
			s.Proxier.OnEndpointsSynced()
		default:
			klog.Warningf("[EdgeMesh] invalid %s operation on endpoints", operation)
		}
	}
}

// MsgProcess handle messages and trigger proxier if necessary
func (s *ProxyServer) MsgProcess(msg model.Message) {
	operation := msg.GetOperation()
	switch getResourceType(msg.GetResource()) {
	case constants.ResourceTypeService:
		serviceList, err := getServices(msg, constants.ResourceTypeService)
		if err != nil {
			klog.Error(err)
		}
		s.handleServices(serviceList, operation)
	case constants.ResourceTypeServiceList:
		serviceList, err := getServices(msg, constants.ResourceTypeServiceList)
		if err != nil {
			klog.Error(err)
		}
		s.handleServices(serviceList, operation)
	case constants.ResourceTypeEndpoints:
		endpointsList, err := getEndpoints(msg, constants.ResourceTypeEndpoints)
		if err != nil {
			klog.Error(err)
		}
		s.handleEndpoints(endpointsList, operation)
	case constants.ResourceTypeEndpointsList:
		endpointsList, err := getEndpoints(msg, constants.ResourceTypeEndpointsList)
		if err != nil {
			klog.Error(err)
		}
		s.handleEndpoints(endpointsList, operation)
	}
}
