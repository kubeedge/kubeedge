package manager

import (
	"fmt"
	"reflect"
	"sync"

	"k8s.io/api/core/v1"
)

// LocationCache cache the map of node, pod, configmap, secret
type LocationCache struct {
	// EdgeNodes is a map, key is nodeName, value is Status
	EdgeNodes sync.Map
	// configMapNode is a map, key is namespace/configMapName, value is nodeName
	configMapNode sync.Map
	// secretNode is a map, key is namespace/secretName, value is nodeName
	secretNode sync.Map
	// services is a map, key is namespace/serviceName, value is v1.Service
	services sync.Map
	// endpoints is a map, key is namespace/endpointsName, value is v1.endpoints
	endpoints sync.Map
	// servicePods is a map, key is namespace/serviceName, value is []v1.Pod
	servicePods sync.Map
}

// PodConfigMapsAndSecrets return configmaps and secrets used by pod
func (lc *LocationCache) PodConfigMapsAndSecrets(pod v1.Pod) (configMaps, secrets []string) {
	for _, v := range pod.Spec.Volumes {
		if v.ConfigMap != nil {
			configMaps = append(configMaps, v.ConfigMap.Name)
		}
		if v.Secret != nil {
			secrets = append(secrets, v.Secret.SecretName)
		}
	}
	// used by envs
	for _, s := range pod.Spec.Containers {
		for _, ef := range s.EnvFrom {
			if ef.ConfigMapRef != nil {
				configMaps = append(configMaps, ef.ConfigMapRef.Name)
			}
			if ef.SecretRef != nil {
				secrets = append(secrets, ef.SecretRef.Name)
			}
		}
	}
	// used by ImagePullSecrets
	for _, s := range pod.Spec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	return
}

func (lc *LocationCache) newNodes(oldNodes []string, node string) []string {
	for _, n := range oldNodes {
		if n == node {
			return oldNodes
		}
	}
	return append(oldNodes, node)
}

// AddOrUpdatePod add pod to node, pod to configmap, configmap to pod, pod to secret, secret to pod relation
func (lc *LocationCache) AddOrUpdatePod(pod v1.Pod) {
	configMaps, secrets := lc.PodConfigMapsAndSecrets(pod)
	for _, c := range configMaps {
		configMapKey := fmt.Sprintf("%s/%s", pod.Namespace, c)
		// update configMapPod
		value, ok := lc.configMapNode.Load(configMapKey)
		var newNodes []string
		if ok {
			nodes, _ := value.([]string)
			newNodes = lc.newNodes(nodes, pod.Spec.NodeName)
		} else {
			newNodes = []string{pod.Spec.NodeName}
		}
		lc.configMapNode.Store(configMapKey, newNodes)
	}

	for _, s := range secrets {
		secretKey := fmt.Sprintf("%s/%s", pod.Namespace, s)
		// update secretPod
		value, ok := lc.secretNode.Load(secretKey)
		var newNodes []string
		if ok {
			nodes, _ := value.([]string)
			newNodes = lc.newNodes(nodes, pod.Spec.NodeName)
		} else {
			newNodes = []string{pod.Spec.NodeName}
		}
		lc.secretNode.Store(secretKey, newNodes)
	}
}

// ConfigMapNodes return all nodes which deploy pod on with configmap
func (lc *LocationCache) ConfigMapNodes(namespace, name string) (nodes []string) {
	configMapKey := fmt.Sprintf("%s/%s", namespace, name)
	value, ok := lc.configMapNode.Load(configMapKey)
	if ok {
		if nodes, ok := value.([]string); ok {
			return nodes
		}
	}
	return
}

// SecretNodes return all nodes which deploy pod on with secret
func (lc *LocationCache) SecretNodes(namespace, name string) (nodes []string) {
	secretKey := fmt.Sprintf("%s/%s", namespace, name)
	value, ok := lc.secretNode.Load(secretKey)
	if ok {
		if nodes, ok := value.([]string); ok {
			return nodes
		}
	}
	return
}

//IsEdgeNode checks weather node is edge node or not
func (lc *LocationCache) IsEdgeNode(nodeName string) bool {
	_, ok := lc.EdgeNodes.Load(nodeName)
	return ok
}

//
func (lc *LocationCache) GetNodeStatus(nodeName string) (string, bool) {
	value, ok := lc.EdgeNodes.Load(nodeName)
	status, ok := value.(string)
	return status, ok
}

// UpdateEdgeNode is to maintain edge nodes name upto-date by querying kubernetes client
func (lc *LocationCache) UpdateEdgeNode(nodeName string, status string) {
	lc.EdgeNodes.Store(nodeName, status)
}

// DeleteConfigMap from cache
func (lc *LocationCache) DeleteConfigMap(namespace, name string) {
	lc.configMapNode.Delete(fmt.Sprintf("%s/%s", namespace, name))
}

// DeleteSecret from cache
func (lc *LocationCache) DeleteSecret(namespace, name string) {
	lc.secretNode.Delete(fmt.Sprintf("%s/%s", namespace, name))
}

// DeleteNode from cache
func (lc *LocationCache) DeleteNode(nodeName string) {
	lc.EdgeNodes.Delete(nodeName)
}

// AddOrUpdateService in cache
func (lc *LocationCache) AddOrUpdateService(service v1.Service) {
	lc.services.Store(fmt.Sprintf("%s/%s", service.Namespace, service.Name), service)
}

// DeleteService from cache
func (lc *LocationCache) DeleteService(service v1.Service) {
	lc.services.Delete(fmt.Sprintf("%s/%s", service.Namespace, service.Name))
}

// GetAllServices from cache
func (lc *LocationCache) GetAllServices() []v1.Service {
	services := []v1.Service{}
	lc.services.Range(func(key interface{}, value interface{}) bool {
		svc, ok := value.(v1.Service)
		if ok {
			services = append(services, svc)
		}
		return true
	})
	return services
}

// GetService from cache
func (lc *LocationCache) GetService(name string) (v1.Service, bool) {
	value, ok := lc.services.Load(name)
	if !ok {
		return v1.Service{}, ok
	}
	svc, ok := value.(v1.Service)
	return svc, ok
}

// AddOrUpdateServicePods in cache
func (lc *LocationCache) AddOrUpdateServicePods(name string, value []v1.Pod) {
	lc.servicePods.Store(name, value)
}

// DeleteServicePods from cache
func (lc *LocationCache) DeleteServicePods(endpoints v1.Endpoints) {
	lc.servicePods.Delete(fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name))
}

// GetServicePods from cache
func (lc *LocationCache) GetServicePods(name string) ([]v1.Pod, bool) {
	value, ok := lc.servicePods.Load(name)
	if !ok {
		return []v1.Pod{}, ok
	}
	pods, ok := value.([]v1.Pod)
	return pods, ok
}

// AddOrUpdateEndpoints in cache
func (lc *LocationCache) AddOrUpdateEndpoints(endpoints v1.Endpoints) {
	lc.endpoints.Store(fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name), endpoints)
}

// DeleteEndpoints in cache
func (lc *LocationCache) DeleteEndpoints(endpoints v1.Endpoints) {
	lc.endpoints.Delete(fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name))
}

// IsEndpointsUpdated checks if endpoints is actually updated
func (lc *LocationCache) IsEndpointsUpdated(new v1.Endpoints) bool {
	eps, ok := lc.endpoints.Load(fmt.Sprintf("%s/%s", new.Namespace, new.Name))
	if !ok {
		// return true because the endpoint was not found in cache
		return !ok
	}
	old, ok := eps.(v1.Endpoints)
	if !ok {
		return !ok
	}
	old.ObjectMeta.ResourceVersion = new.ObjectMeta.ResourceVersion
	old.ObjectMeta.Generation = new.ObjectMeta.Generation
	old.ObjectMeta.Annotations = new.ObjectMeta.Annotations
	// return true if ObjectMeta or Subsets changed, else false
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Subsets, new.Subsets)
}

// GetAllEndpoints from cache
func (lc *LocationCache) GetAllEndpoints() []v1.Endpoints {
	endpoints := []v1.Endpoints{}
	lc.endpoints.Range(func(key interface{}, value interface{}) bool {
		eps, ok := value.(v1.Endpoints)
		if ok {
			endpoints = append(endpoints, eps)
		}
		return true
	})
	return endpoints
}
