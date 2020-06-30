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
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/klog"
	"k8s.io/kube-proxy/config/v1alpha1"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/apis"
	kubeproxyconfig "k8s.io/kubernetes/pkg/proxy/apis/config"
	proxyconfigscheme "k8s.io/kubernetes/pkg/proxy/apis/config/scheme"
	"k8s.io/kubernetes/pkg/proxy/iptables"
	"k8s.io/kubernetes/pkg/proxy/ipvs"
	"k8s.io/kubernetes/pkg/proxy/userspace"
	utilipset "k8s.io/kubernetes/pkg/util/ipset"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilipvs "k8s.io/kubernetes/pkg/util/ipvs"
	"k8s.io/kubernetes/pkg/util/oom"
	"k8s.io/utils/exec"
)

const (
	proxyModeUserspace = "userspace"
	proxyModeIPTables  = "iptables"
	proxyModeIPVS      = "ipvs"
)

// proxyRun defines the interface to run a specified ProxyServer
type proxyRun interface {
	Run() error
	CleanupAndExit() error
}

// Options contains everything necessary to create and run a proxy server.
type Options struct {
	// config is the proxy server's configuration object.
	config *kubeproxyconfig.KubeProxyConfiguration
	// proxyServer is the interface to run the proxy server
	proxyServer proxyRun
	// errCh is the channel that errors will be sent
	errCh chan error
}

// NewOptions returns initialized Options
func NewOptions() *Options {
	opts := &Options{
		errCh: make(chan error),
	}
	config, err := opts.ApplyDefaults(new(kubeproxyconfig.KubeProxyConfiguration))
	if err != nil {
		klog.Fatal(err)
	}
	opts.config = config
	return opts
}

func (o *Options) errorHandler(err error) {
	o.errCh <- err
}

// Run runs the specified ProxyServer.
func (o *Options) Run() error {
	defer close(o.errCh)

	proxyServer, err := NewProxyServer(o)
	if err != nil {
		return err
	}

	o.proxyServer = proxyServer
	return o.runLoop()
}

// runLoop will watch on the update change of the proxy server's configuration file.
// Return an error when updated
func (o *Options) runLoop() error {
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

	// use IPVS mode as default
	out.Mode = proxyModeIPVS

	return out, nil
}

// ProxyServer represents all the parameters required to start the Kubernetes proxy server. All
// fields are required.
type ProxyServer struct {
	IptInterface           utiliptables.Interface
	IpvsInterface          utilipvs.Interface
	IpsetInterface         utilipset.Interface
	execer                 exec.Interface
	Proxier                proxy.Provider
	ConntrackConfiguration kubeproxyconfig.KubeProxyConntrackConfiguration
	Conntracker            Conntracker // if nil, ignored
	ProxyMode              string
	CleanupIPVS            bool
	MetricsBindAddress     string
	EnableProfiling        bool
	UseEndpointSlices      bool
	OOMScoreAdj            *int32
	ConfigSyncPeriod       time.Duration

	servicesMap  sync.Map
	endpointsMap sync.Map
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

// Run runs the specified ProxyServer. This should never exit (unless CleanupAndExit is set).
func (s *ProxyServer) Run() error {
	var oomAdjuster *oom.OOMAdjuster
	if s.OOMScoreAdj != nil {
		oomAdjuster = oom.NewOOMAdjuster()
		if err := oomAdjuster.ApplyOOMScoreAdj(0, int(*s.OOMScoreAdj)); err != nil {
			klog.V(2).Info(err)
		}
	}

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
			klog.Warning(api.EventTypeWarning, err.Error(), message)
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

	// Just loop forever for now...
	s.Proxier.SyncLoop()
	return nil
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

// CleanupAndExit remove iptables rules
func (s *ProxyServer) CleanupAndExit() error {
	encounteredError := userspace.CleanupLeftovers(s.IptInterface)
	encounteredError = iptables.CleanupLeftovers(s.IptInterface) || encounteredError
	encounteredError = ipvs.CleanupLeftovers(s.IpvsInterface, s.IptInterface, s.IpsetInterface, s.CleanupIPVS) || encounteredError
	if encounteredError {
		return errors.New("encountered an error while tearing down rules")
	}
	return nil
}
