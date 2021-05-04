package registry

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/registry"
	utiltags "github.com/go-chassis/go-chassis/pkg/util/tags"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/cache"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
)

const (
	// EdgeRegistry constant string
	EdgeRegistry = "edge"
)

// TODO Remove the init method, because it will cause invalid logs to be printed when the program is running @kadisi
// init initialize the plugin of edge meta registry
func init() { registry.InstallServiceDiscovery(EdgeRegistry, NewEdgeServiceDiscovery) }

// EdgeServiceDiscovery to represent the object of service center to call the APIs of service center
type EdgeServiceDiscovery struct {
	metaClient client.CoreInterface
	Name       string
}

func NewEdgeServiceDiscovery(options registry.Options) registry.ServiceDiscovery {
	return &EdgeServiceDiscovery{
		metaClient: client.New(),
		Name:       EdgeRegistry,
	}
}

// GetAllMicroServices Get all MicroService information.
func (esd *EdgeServiceDiscovery) GetAllMicroServices() ([]*registry.MicroService, error) {
	return nil, nil
}

// FindMicroServiceInstances find micro-service instances (subnets)
func (esd *EdgeServiceDiscovery) FindMicroServiceInstances(consumerID, microServiceName string, tags utiltags.Tags) ([]*registry.MicroServiceInstance, error) {
	// parse microServiceName
	name, namespace, port, err := parseServiceURL(microServiceName)
	if err != nil {
		return nil, err
	}
	// get service
	service, err := esd.getService(name, namespace)
	if err != nil {
		return nil, err
	}
	// get pods
	pods, err := esd.getPods(name, namespace)
	if err != nil {
		return nil, err
	}
	// get targetPort
	var targetPort int
	for _, p := range service.Spec.Ports {
		if p.Protocol == "TCP" && int(p.Port) == port {
			targetPort = p.TargetPort.IntValue()
			break
		}
	}
	// port not found
	if targetPort == 0 {
		klog.Errorf("[EdgeMesh] port %d not found in svc: %s.%s", port, namespace, name)
		return nil, fmt.Errorf("port %d not found in svc: %s.%s", port, namespace, name)
	}

	// gen
	var microServiceInstances []*registry.MicroServiceInstance
	var hostPort int32
	// all pods share the same hostport, get from pods[0]
	if pods[0].Spec.HostNetwork {
		// host network
		hostPort = int32(targetPort)
	} else {
		// container network
		for _, container := range pods[0].Spec.Containers {
			for _, port := range container.Ports {
				if port.ContainerPort == int32(targetPort) {
					hostPort = port.HostPort
				}
			}
		}
	}
	for _, p := range pods {
		if p.Status.Phase == v1.PodRunning {
			microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
				InstanceID:   "",
				ServiceID:    name + "." + namespace,
				HostName:     "",
				EndpointsMap: map[string]string{"rest": fmt.Sprintf("%s:%d", p.Status.HostIP, hostPort)},
			})
		}
	}

	return microServiceInstances, nil
}

// GetMicroServiceID get microServiceID
func (esd *EdgeServiceDiscovery) GetMicroServiceID(appID, microServiceName, version, env string) (string, error) {
	return "", nil
}

// GetMicroServiceInstances return instances
func (esd *EdgeServiceDiscovery) GetMicroServiceInstances(consumerID, providerID string) ([]*registry.MicroServiceInstance, error) {
	return nil, nil
}

// GetMicroService return service
func (esd *EdgeServiceDiscovery) GetMicroService(microServiceID string) (*registry.MicroService, error) {
	return nil, nil
}

// AutoSync updating the cache manager
func (esd *EdgeServiceDiscovery) AutoSync() {}

// Close close all websocket connection
func (esd *EdgeServiceDiscovery) Close() error { return nil }

// parseServiceURL parses serviceURL to ${service_name}.${namespace}.svc.${cluster}:${port}, keeps with k8s service
func parseServiceURL(serviceURL string) (string, string, int, error) {
	var port int
	var err error
	serviceURLSplit := strings.Split(serviceURL, ":")
	if len(serviceURLSplit) == 1 {
		// default
		port = 80
	} else if len(serviceURLSplit) == 2 {
		port, err = strconv.Atoi(serviceURLSplit[1])
		if err != nil {
			klog.Errorf("[EdgeMesh] service url %s invalid", serviceURL)
			return "", "", 0, err
		}
	} else {
		klog.Errorf("[EdgeMesh] service url %s invalid", serviceURL)
		err = fmt.Errorf("service url %s invalid", serviceURL)
		return "", "", 0, err
	}
	name, namespace := common.SplitServiceKey(serviceURLSplit[0])
	return name, namespace, port, nil
}

// getService get k8s service from either lruCache or metaManager
func (esd *EdgeServiceDiscovery) getService(name, namespace string) (*v1.Service, error) {
	var svc *v1.Service
	key := fmt.Sprintf("service.%s.%s", namespace, name)
	// try to get from cache
	v, ok := cache.GetMeshCache().Get(key)
	if ok {
		svc, ok = v.(*v1.Service)
		if !ok {
			klog.Errorf("[EdgeMesh] service %s from cache with invalid type", key)
			return nil, fmt.Errorf("service %s from cache with invalid type", key)
		}
		klog.Infof("[EdgeMesh] get service %s from cache", key)
	} else {
		// get from metaClient
		var err error
		svc, err = esd.metaClient.Services(namespace).Get(name)
		if err != nil {
			klog.Errorf("[EdgeMesh] get service from metaClient failed, error: %v", err)
			return nil, err
		}
		cache.GetMeshCache().Add(key, svc)
		klog.Infof("[EdgeMesh] get service %s from metaClient", key)
	}
	return svc, nil
}

// getPods get service pods from either lruCache or metaManager
func (esd *EdgeServiceDiscovery) getPods(name, namespace string) ([]v1.Pod, error) {
	var pods []v1.Pod
	key := fmt.Sprintf("pods.%s.%s", namespace, name)
	// try to get from cache
	v, ok := cache.GetMeshCache().Get(key)
	if ok {
		pods, ok = v.([]v1.Pod)
		if !ok {
			klog.Errorf("[EdgeMesh] pods %s from cache with invalid type", key)
			return nil, fmt.Errorf("pods %s from cache with invalid type", key)
		}
		if len(pods) == 0 {
			klog.Errorf("[EdgeMesh] pod list %s is empty", key)
			return nil, fmt.Errorf("pod list %s is empty", key)
		}
		klog.Infof("[EdgeMesh] get pods %s from cache", key)
	} else {
		// get from metaClient
		var err error
		pods, err = esd.metaClient.Services(namespace).GetPods(name)
		if err != nil {
			klog.Errorf("[EdgeMesh] get pods from metaClient failed, error: %v", err)
			return nil, err
		}
		if len(pods) == 0 {
			klog.Errorf("[EdgeMesh] pod list %s is empty", key)
			return nil, fmt.Errorf("pod list %s is empty", key)
		}
		cache.GetMeshCache().Add(key, pods)
		klog.Infof("[EdgeMesh] get pods %s from metaClient", key)
	}
	return pods, nil
}
