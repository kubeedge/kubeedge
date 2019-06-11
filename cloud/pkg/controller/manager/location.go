package manager

import (
	"fmt"
	"sync"

	"github.com/kubeedge/beehive/pkg/common/log"
	"k8s.io/api/core/v1"
)

// LocationCache cache the map of node, pod, configmap, secret
type LocationCache struct {
	// EdgeNodes is a list of valid edge nodes
	EdgeNodes []string
	// configMapNode is a map, key is namespace/configMapName, value is nodeName
	configMapNode sync.Map
	// secretNode is a map, key is namespace/secretName, value is nodeName
	secretNode sync.Map
	// Services is an array of services
	Services []v1.Service
	// Endpoints is an array of endpoints
	Endpoints []v1.Endpoints
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
	for _, node := range lc.EdgeNodes {
		if node == nodeName {
			return true
		}
	}
	return false
}

// UpdateEdgeNode is to maintain edge nodes name upto-date by querying kubernetes client
func (lc *LocationCache) UpdateEdgeNode(nodeName string) {
	lc.EdgeNodes = append(lc.EdgeNodes, nodeName)
	log.LOGGER.Infof("Edge nodes updated : %v \n", lc.EdgeNodes)
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
func (lc *LocationCache) DeleteNode(name string) {
	for i, v := range lc.EdgeNodes {
		if v == name {
			lc.EdgeNodes = append(lc.EdgeNodes[:i], lc.EdgeNodes[i+1:]...)
		}
	}
}

func (lc *LocationCache) AddService(service v1.Service) {
	lc.Services = append(lc.Services, service)
}

func (lc *LocationCache) UpdateService(service v1.Service) {
	lc.DeleteService(service)
	lc.AddService(service)
}

func (lc *LocationCache) DeleteService(service v1.Service) {
	for i, svc := range lc.Services {
		if svc.Name == service.Name {
			lc.Services = append(lc.Services[:i], lc.Services[i+1:]...)
			break
		}
	}
}

func (lc *LocationCache) AddEndpoints(endpoints v1.Endpoints) {
	lc.Endpoints = append(lc.Endpoints, endpoints)
}

func (lc *LocationCache) UpdateEndpoints(endpoints v1.Endpoints) {
	lc.DeleteEndpoints(endpoints)
	lc.AddEndpoints(endpoints)
}

func (lc *LocationCache) DeleteEndpoints(endpoints v1.Endpoints) {
	for i, eps := range lc.Endpoints {
		if eps.Name == eps.Name {
			lc.Endpoints = append(lc.Endpoints[:i], lc.Endpoints[i+1:]...)
			break
		}
	}
}
