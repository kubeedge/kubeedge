package registry

import (
	"fmt"
	v1 "k8s.io/api/core/v1"

	"github.com/go-chassis/go-chassis/core/registry"
	utiltags "github.com/go-chassis/go-chassis/pkg/util/tags"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
)

const (
	// EdgeRegistry constant string
	EdgeRegistry = "edge"
)

// init initialize the plugin of edge meta registry
func init() { registry.InstallServiceDiscovery(EdgeRegistry, NewServiceDiscovery) }

// ServiceDiscovery to represent the object of service center to call the APIs of service center
type ServiceDiscovery struct {
	metaClient client.CoreInterface
	Name       string
}

func NewServiceDiscovery(options registry.Options) registry.ServiceDiscovery {
	c := context.GetContext(context.MsgCtxTypeChannel)
	return &ServiceDiscovery{
		metaClient: client.New(c),
		Name:       EdgeRegistry,
	}
}

// GetAllMicroServices Get all MicroService information.
func (r *ServiceDiscovery) GetAllMicroServices() ([]*registry.MicroService, error) {
	return nil, nil
}

// FindMicroServiceInstances find micro-service instances (subnets)
func (r *ServiceDiscovery) FindMicroServiceInstances(consumerID, microServiceName string, tags utiltags.Tags) ([]*registry.MicroServiceInstance, error) {
	name, namespace, _, _, servicePort, err := common.ParseServiceName(microServiceName)
	if err != nil {
		log.LOGGER.Errorf("parse micro service name error: %v", err)
		return nil, err
	}

	service, err := r.metaClient.Services(namespace).Get(name)
	if err != nil {
		log.LOGGER.Errorf("get service failed, error: %v", err)
		return nil, err
	}
	pods, err := r.metaClient.Services(namespace).GetPods(name)
	if err != nil {
		log.LOGGER.Errorf("get service pod list failed, error: %v", err)
		return nil, err
	}

	var microServiceInstance []*registry.MicroServiceInstance
	var httpPort v1.ServicePort
	for _, port := range service.Spec.Ports {
		if port.Name == "http" && port.Protocol == "TCP" && int(port.Port) == servicePort {
			httpPort = port
			break
		}
	}
	for _, p := range pods {
		for _, container := range p.Spec.Containers {
			for _, containPort := range container.Ports {
				if containPort.ContainerPort == int32(httpPort.TargetPort.IntValue()) {
					microServiceInstance = append(microServiceInstance, &registry.MicroServiceInstance{
						InstanceID:   "",
						ServiceID:    name + "." + namespace,
						HostName:     "",
						EndpointsMap: map[string]string{"rest": fmt.Sprintf("%s:%d", p.Status.HostIP, containPort.HostPort)},
					})
				}
			}
		}
	}

	return microServiceInstance, nil
}

// GetMicroServiceID get microServiceID
func (r *ServiceDiscovery) GetMicroServiceID(appID, microServiceName, version, env string) (string, error) {
	return "", nil
}

// GetMicroServiceInstances return instances
func (r *ServiceDiscovery) GetMicroServiceInstances(consumerID, providerID string) ([]*registry.MicroServiceInstance, error) {
	return nil, nil
}

// GetMicroService return service
func (r *ServiceDiscovery) GetMicroService(microServiceID string) (*registry.MicroService, error) {
	return nil, nil
}

// AutoSync updating the cache manager
func (r *ServiceDiscovery) AutoSync() {}

// Close close all websocket connection
func (r *ServiceDiscovery) Close() error { return nil }
