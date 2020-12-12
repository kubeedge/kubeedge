/*
Copyright 2019 The KubeEdge Authors.

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

package manager

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	configMapKey    = "ObjectMeta1/VolumeConfig1"
	configMapVolume = "VolumeConfig1"
	nodes           = []string{"Node1", "Node2"}
	objectMeta      = "ObjectMeta1"
	secretKey       = "ObjectMeta1/VolumeSecret1"
	secretVolume    = "VolumeSecret1"
)

// TestAddOrUpdatePod is function to test AddOrUpdatePod
func TestAddOrUpdatePod(t *testing.T) {
	pod := v1.Pod{
		Spec: v1.PodSpec{
			NodeName: "Node1",
			Volumes: []v1.Volume{{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: configMapVolume}},
					Secret:    &v1.SecretVolumeSource{SecretName: secretVolume},
				},
			}},
			Containers: []v1.Container{{
				EnvFrom: []v1.EnvFromSource{{
					ConfigMapRef: &v1.ConfigMapEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: "ContainerConfig1"}},
					SecretRef:    &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: "ContainerSecret1"}},
				}},
			}},
			ImagePullSecrets: []v1.LocalObjectReference{{Name: "ImageSecret1"}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: objectMeta,
			Name:      "Object1",
		},
	}
	locationCache := LocationCache{}
	locationCache.configMapNode.Store(configMapKey, "Node1")
	locationCache.secretNode.Store(secretKey, nodes)
	tests := []struct {
		name string
		lc   *LocationCache
		pod  v1.Pod
	}{
		{
			name: "TestAddOrUpdatePod(): Case 1: LocationCache is empty",
			lc:   &LocationCache{},
			pod:  pod,
		},
		{
			name: "TestAddOrUpdatePod(): Case 2: LocationCache is not empty",
			lc:   &locationCache,
			pod:  pod,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.AddOrUpdatePod(test.pod)
		})
	}
}

// TestConfigMapNodes is function to test ConfigMapNodes
func TestConfigMapNodes(t *testing.T) {
	locationCache := LocationCache{}
	locationCache.configMapNode.Store(configMapKey, nodes)
	tests := []struct {
		name          string
		lc            *LocationCache
		namespace     string
		configMapName string
		nodes         []string
	}{
		{
			name:  "TestConfigMapNodes(): Case 1: LocationCache is empty",
			lc:    &LocationCache{},
			nodes: nil,
		},
		{
			name:          "TestConfigMapNodes(): Case 2: LocationCache is not empty",
			lc:            &locationCache,
			namespace:     objectMeta,
			configMapName: configMapVolume,
			nodes:         nodes,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if nodes := test.lc.ConfigMapNodes(test.namespace, test.configMapName); !reflect.DeepEqual(nodes, test.nodes) {
				t.Errorf("Manager.TestConfigMapNodes() case failed: got = %v, Want = %v", nodes, test.nodes)
			}
		})
	}
}

// TestSecretNodes is function to test SecretNodes
func TestSecretNodes(t *testing.T) {
	locationCache := LocationCache{}
	locationCache.secretNode.Store(secretKey, nodes)
	tests := []struct {
		name       string
		lc         *LocationCache
		namespace  string
		secretName string
		nodes      []string
	}{
		{
			name:  "TestSecretNodes(): Case 1: LocationCache is empty",
			lc:    &LocationCache{},
			nodes: nil,
		},
		{
			name:       "TestSecretNodes(): Case 2: LocationCache is not empty",
			lc:         &locationCache,
			namespace:  objectMeta,
			secretName: secretVolume,
			nodes:      nodes,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if nodes := test.lc.SecretNodes(test.namespace, test.secretName); !reflect.DeepEqual(nodes, test.nodes) {
				t.Errorf("Manager.TestSecretNodes() case failed: got = %v, Want = %v", nodes, test.nodes)
			}
		})
	}
}

// TestDeleteConfigMap is function to test DeleteConfigMap
func TestDeleteConfigMap(t *testing.T) {
	locationCache := LocationCache{}
	locationCache.configMapNode.Store(configMapKey, nodes)
	tests := []struct {
		name          string
		lc            *LocationCache
		namespace     string
		configMapName string
		errorWant     bool
	}{
		{
			name:          "TestDeleteConfigMap(): delete configMap from cache",
			lc:            &locationCache,
			namespace:     objectMeta,
			configMapName: configMapVolume,
			errorWant:     false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteConfigMap(test.namespace, test.configMapName)
			if _, got := test.lc.configMapNode.Load(configMapKey); !reflect.DeepEqual(got, test.errorWant) {
				t.Errorf("Manager.TestDeleteConfigMap() case failed: got = %v, Want = %v", got, test.errorWant)
			}
		})
	}
}

// TestDeleteSecret is function to test DeleteSecret
func TestDeleteSecret(t *testing.T) {
	locationCache := LocationCache{}
	locationCache.secretNode.Store(secretKey, nodes)
	tests := []struct {
		name       string
		lc         *LocationCache
		namespace  string
		secretName string
		errorWant  bool
	}{
		{
			name:       "TestDeleteSecret(): delete secret from cache",
			lc:         &locationCache,
			namespace:  objectMeta,
			secretName: secretVolume,
			errorWant:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteSecret(test.namespace, test.secretName)
			if _, got := test.lc.secretNode.Load(secretKey); !reflect.DeepEqual(got, test.errorWant) {
				t.Errorf("Manager.TestDeleteSecret() case failed: got = %v, Want = %v", got, test.errorWant)
			}
		})
	}
}

// TestIsEdgeNode is function to test IsEdgeNode
func TestIsEdgeNode(t *testing.T) {
	nodeName := nodes[0]
	locationCache := LocationCache{}
	locationCache.EdgeNodes.Store(nodeName, "OK")

	tests := []struct {
		name     string
		lc       *LocationCache
		nodeName string
		want     bool
	}{
		{
			name:     "TestIsEdgeNode() Case: Node is edgenode",
			lc:       &locationCache,
			nodeName: nodeName,
			want:     true,
		},
		{
			name:     "TestIsEdgeNode() Case: Node is not edgenode",
			lc:       &locationCache,
			nodeName: "notExistNode",
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.lc.IsEdgeNode(test.nodeName); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Manager.TestIsEdgeNode() case failed: got = %v, want = %v", got, test.want)
			}
		})
	}
}

// TestGetNodeStatus is function to test GetNodeStatus
func TestGetNodeStatus(t *testing.T) {
	nodeOK := nodes[0]
	nodeUnknown := nodes[1]
	locationCache := LocationCache{}
	locationCache.EdgeNodes.Store(nodeOK, "OK")
	locationCache.EdgeNodes.Store(nodeUnknown, "Unknown")

	tests := []struct {
		name     string
		lc       *LocationCache
		nodeName string
		want     string
		exist    bool
	}{
		{
			name:     "TestGetNodeStatus() Case: Node status is OK",
			lc:       &locationCache,
			nodeName: nodeOK,
			want:     "OK",
			exist:    true,
		},
		{
			name:     "TestGetNodeStatus() Case: Node status is Unknown",
			lc:       &locationCache,
			nodeName: nodeUnknown,
			want:     "Unknown",
			exist:    true,
		},
		{
			name:     "TestGetNodeStatus() Case: Node not exist",
			lc:       &locationCache,
			nodeName: "notExistNode",
			want:     "",
			exist:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, exist := test.lc.GetNodeStatus(test.nodeName); !reflect.DeepEqual(got, test.want) || !reflect.DeepEqual(exist, test.exist) {
				t.Errorf("Manager.TestGetNodeStatus() case failed: gotStatus = %v,gotExist = %v, wantStatus = %v.  wantExist: %v", got, exist, test.want, test.exist)
			}
		})
	}
}

// TestUpdateEdgeNode is function to test UpdateEdgeNode
func TestUpdateEdgeNode(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	locationCache.EdgeNodes.Store(nodeName, "")

	tests := []struct {
		name string
		lc   *LocationCache
		want string
	}{
		{
			name: "TestUpdateEdgeNode() Case: Node status update to OK",
			lc:   &locationCache,
			want: "OK",
		},
		{
			name: "TestUpdateEdgeNode() Case: Node status update to Unknown",
			lc:   &locationCache,
			want: "Unknown",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.UpdateEdgeNode(nodeName, test.want)
			if got, _ := test.lc.EdgeNodes.Load(nodeName); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Manager.TestUpdateEdgeNode() case failed: got = %v, want = %v.", got, test.want)
			}
		})
	}
}

// TestDeleteNode is function to test DeleteNode
func TestDeleteNode(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	locationCache.EdgeNodes.Store(nodeName, "OK")

	tests := []struct {
		name     string
		lc       *LocationCache
		nodeName string
		want     bool
	}{
		{
			name:     "TestDeleteNode() Case: Delete exist node",
			lc:       &locationCache,
			nodeName: nodeName,
			want:     false,
		},
		{
			name:     "TestDeleteNode() Case: Delete not exist node",
			lc:       &locationCache,
			nodeName: "notExistNode",
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteNode(test.nodeName)
			if _, exist := test.lc.EdgeNodes.Load(test.nodeName); !reflect.DeepEqual(exist, test.want) {
				t.Errorf("Manager.TestDeleteNode() case failed: exist = %v, want = %v.", exist, test.want)
			}
		})
	}
}

// TestGetService is function to test GetService
func TestGetService(t *testing.T) {
	locationCache := LocationCache{}
	svc1 := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Selector: map[string]string{
				"selector": "selector",
			},
			Ports: []v1.ServicePort{
				{
					Name:     "port",
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}
	locationCache.services.Store("default/svc1", svc1)
	var nilSvc v1.Service

	tests := []struct {
		name    string
		lc      *LocationCache
		svcName string
		svc     v1.Service
		want    bool
	}{
		{
			name:    "TestGetService() Case: Get exist service",
			lc:      &locationCache,
			svcName: "default/svc1",
			svc:     svc1,
			want:    true,
		},
		{
			name:    "TestGetService() Case: Get not exist service",
			lc:      &locationCache,
			svcName: "default/svc2",
			svc:     nilSvc,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, exist := test.lc.GetService(test.svcName)
			if !reflect.DeepEqual(exist, test.want) {
				t.Errorf("Manager.TestGetService() case failed: exist: %v, want: %v", exist, test.want)
				return
			}
			if exist && !reflect.DeepEqual(svc, test.svc) {
				t.Errorf("Manager.TestGetService() case failed: svc: %v, want: %v", svc, test.svc)
				return
			}
		})
	}
}

// TestGetAllService is function to test GetAllService
func TestGetAllService(t *testing.T) {
	lc := LocationCache{}
	svclist := []v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "svc1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "svc2",
				Namespace: "default",
			},
		},
	}

	for _, v := range svclist {
		lc.services.Store(v.GetNamespace()+"/"+v.GetName(), v)
	}

	t.Run("TestGetAllService() Case: Get all service", func(t *testing.T) {
		got := lc.GetAllServices()
		if len(got) != len(svclist) {
			t.Errorf("Manager.TestGetAllService() case failed: len(got): %v, len(svclist): %v", len(got), len(svclist))
			return
		}
		m := map[string]v1.Service{}
		for _, svc := range got {
			m[svc.GetNamespace()+"/"+svc.GetName()] = svc
		}

		for _, svc := range svclist {
			if _, ok := m[svc.GetNamespace()+"/"+svc.GetName()]; !ok {
				t.Errorf("Manager.TestGetAllService() case failed: service not exist in GetAllService() result. got: %v want: %v ", got, svc)
			}
		}
	})
}

// TestAddOrUpdateService is function to test AddOrUpdateService
func TestAddOrUpdateService(t *testing.T) {
	locationCache := LocationCache{}
	svc1 := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Selector: map[string]string{
				"selector": "selector",
			},
			Ports: []v1.ServicePort{
				{
					Name:     "port",
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}

	tests := []struct {
		name string
		lc   *LocationCache
		svc  v1.Service
	}{
		{
			name: "TestAddOrUpdateService() Case: Add service",
			lc:   &locationCache,
			svc:  svc1,
		},
		{
			name: "TestAddOrUpdateService() Case: Update service",
			lc:   &locationCache,
			svc:  svc1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.AddOrUpdateService(test.svc)
			svc, _ := test.lc.services.Load(test.svc.GetNamespace() + "/" + test.svc.GetName())
			if !reflect.DeepEqual(svc, test.svc) {
				t.Errorf("Manager.TestAddOrUpdateService() case failed: got: %v want: %v", svc, test.svc)
			}
		})
	}
}

// TestDeleteService is function to test DeleteService
func TestDeleteService(t *testing.T) {
	locationCache := LocationCache{}
	svc1 := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "default",
		},
	}

	tests := []struct {
		name string
		lc   *LocationCache
		svc  v1.Service
	}{
		{
			name: "TestDeleteService() Case: Delete a exist service",
			lc:   &locationCache,
			svc:  svc1,
		},
		{
			name: "TestDeleteService() Case: Delete not exist service",
			lc:   &locationCache,
			svc:  svc1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteService(test.svc)
			svcName := test.svc.GetNamespace() + "/" + test.svc.GetName()
			if svc, exist := test.lc.GetService(svcName); exist {
				t.Errorf("Manager.TestDeleteService() case failed: service still exist after delete. %v", svc)
			}
		})
	}
}

// TestAddOrUpdateServicePods is function to test AddOrUpdateServicePods
func TestAddOrUpdateServicePods(t *testing.T) {
	locationCache := LocationCache{}
	pods1 := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
			},
		},
	}

	pods2 := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
			},
		},
	}

	tests := []struct {
		name    string
		lc      *LocationCache
		svcName string
		pods    []v1.Pod
	}{
		{
			name:    "TestAddOrUpdateServicePods() Case: Add a service pods relation",
			lc:      &locationCache,
			svcName: "default/svc1",
			pods:    pods1,
		},
		{
			name:    "TestAddOrUpdateServicePods() Case: Update a exist service pods relation",
			lc:      &locationCache,
			svcName: "default/svc1",
			pods:    pods2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.AddOrUpdateServicePods(test.svcName, test.pods)
			got, _ := test.lc.servicePods.Load(test.svcName)

			if !reflect.DeepEqual(got, test.pods) {
				t.Errorf("Manager.TestAddOrUpdateServicePods() case failed: got: %v, want: %v", got, test.pods)
			}
		})
	}
}

// TestDeleteServicePods is function to test DeleteServicePods
func TestDeleteServicePods(t *testing.T) {
	locationCache := LocationCache{}
	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
			},
		},
	}

	endpoints := v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "default",
		},
	}

	locationCache.servicePods.Store(endpoints.GetNamespace()+"/"+endpoints.GetName(), pods)

	tests := []struct {
		name string
		lc   *LocationCache
		svc  v1.Service
		ep   v1.Endpoints
	}{
		{
			name: "TestDeleteServicePods() Case: Delete a exist service",
			lc:   &locationCache,
			ep:   endpoints,
		},
		{
			name: "TestDeleteServicePods() Case: Delete non-existent service",
			lc:   &LocationCache{},
			ep:   endpoints,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteServicePods(test.ep)
			epName := test.ep.GetNamespace() + "/" + test.ep.GetName()
			if svcPods, exist := test.lc.servicePods.Load(epName); exist {
				t.Errorf("Manager.TestDeleteServicePods() case failed: servicePods still exist after delete. %v", svcPods)
			}
		})
	}
}

// TestGetServicePods is function to test GetServicePods
func TestGetServicePods(t *testing.T) {
	locationCache := LocationCache{}
	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
			},
		},
	}
	locationCache.servicePods.Store("default/svc1", pods)
	locationCache.servicePods.Store("default/svc2", "invalidPods")

	tests := []struct {
		name    string
		lc      *LocationCache
		svcName string
		pods    []v1.Pod
		exist   bool
	}{
		{
			name:    "TestGetServicePods() Case: Get exist servicePods",
			lc:      &locationCache,
			svcName: "default/svc1",
			pods:    pods,
			exist:   true,
		},
		{
			name:    "TestGetServicePods() Case: Get invalid servicePods",
			lc:      &locationCache,
			svcName: "default/svc2",
			exist:   false,
		},

		{
			name:    "TestGetServicePods() Case: Get not exist servicePods",
			lc:      &locationCache,
			svcName: "default/svc3",
			exist:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, exist := test.lc.GetServicePods(test.svcName)
			if !reflect.DeepEqual(exist, test.exist) {
				t.Errorf("Manager.TestGetServicePods() case failed: exist: %v, want: %v", exist, test.exist)
				return
			}
			if exist && !reflect.DeepEqual(got, pods) {
				t.Errorf("Manager.TestGetServicePods() case failed: got: %v, want: %v", got, pods)
			}
		})
	}
}

// TestAddOrUpdateEndpoints is function to test AddOrUpdateEndpoints
func TestAddOrUpdateEndpoints(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]

	tests := []struct {
		name string
		lc   *LocationCache
		ep   v1.Endpoints
	}{
		{
			name: "TestAddOrUpdateEndpoints() Case: Add endpoints",
			lc:   &locationCache,
			ep: v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ep1",
					Namespace: "default",
				},
				Subsets: []v1.EndpointSubset{},
			},
		},
		{
			name: "TestAddOrUpdateEndpoints() Case: Update endpoints",
			lc:   &locationCache,
			ep: v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ep1",
					Namespace: "default",
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP:       "10.0.0.1",
								NodeName: &nodeName,
							},
							{
								IP:       "10.0.0.2",
								NodeName: &nodeName,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.AddOrUpdateEndpoints(test.ep)
			ep, _ := test.lc.endpoints.Load(test.ep.GetNamespace() + "/" + test.ep.GetName())
			if !reflect.DeepEqual(ep, test.ep) {
				t.Errorf("Manager.TestAddOrUpdateService() case failed: got: %v want: %v", ep, test.ep)
			}
		})
	}
}

// TestDeleteEndpoints is function to test DeleteEndpoints
func TestDeleteEndpoints(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	ep1 := v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.0.0.1",
						NodeName: &nodeName,
					},
					{
						IP:       "10.0.0.2",
						NodeName: &nodeName,
					},
				},
			},
		},
	}

	locationCache.endpoints.Store(ep1.GetNamespace()+"/"+ep1.GetName(), ep1)

	tests := []struct {
		name string
		lc   *LocationCache
		ep   v1.Endpoints
	}{
		{
			name: "TestDeleteEndpoints() Case: Delete a exist service",
			lc:   &locationCache,
			ep:   ep1,
		},
		{
			name: "TestDeleteEndpoints() Case: Delete not exist service",
			lc:   &LocationCache{},
			ep:   ep1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.DeleteEndpoints(test.ep)
			if got, exist := test.lc.endpoints.Load(test.ep.GetNamespace() + "/" + test.ep.GetName()); exist {
				t.Errorf("Manager.TestDeleteEndpoints() case failed: endpoints still exits after delete. %v", got)
			}
		})
	}
}

// TestGetAllEndpoints is function to test GetAllEndpoints
func TestGetAllEndpoints(t *testing.T) {
	lc := LocationCache{}
	nodeName := nodes[0]
	eplist := []v1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ep1",
				Namespace: "default",
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP:       "10.0.0.1",
							NodeName: &nodeName,
						},
						{
							IP:       "10.0.0.2",
							NodeName: &nodeName,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ep2",
				Namespace: "default",
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP:       "10.0.0.3",
							NodeName: &nodeName,
						},
						{
							IP:       "10.0.0.4",
							NodeName: &nodeName,
						},
					},
				},
			},
		},
	}

	for _, ep := range eplist {
		lc.endpoints.Store(ep.GetNamespace()+"/"+ep.GetName(), ep)
	}

	t.Run("TestGetAllEndpoints() Case: Get all endpoints", func(t *testing.T) {
		got := lc.GetAllEndpoints()
		if len(got) != len(eplist) {
			t.Errorf("Manager.TestGetAllEndpoints() case failed: len(got): %v, len(eplist): %v", len(got), len(eplist))
		}
		m := map[string]v1.Endpoints{}
		for _, ep := range got {
			m[ep.GetNamespace()+"/"+ep.GetName()] = ep
		}

		for _, ep := range eplist {
			if _, ok := m[ep.GetNamespace()+"/"+ep.GetName()]; !ok {
				t.Errorf("Manager.TestGetAllEndpoints() case failed: endpoints not exist in GetAllEndpoints() result. got: %v want: %v ", got, ep)
			}
		}
	})
}

// TestIsEndpointsUpdated is function to test IsEndpointsUpdated
func TestIsEndpointsUpdated(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	ep1 := v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ep1",
			Namespace: "default",
		},
	}
	ep2 := v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ep1",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.0.0.1",
						NodeName: &nodeName,
					},
					{
						IP:       "10.0.0.2",
						NodeName: &nodeName,
					},
				},
			},
		},
	}

	locationCache.endpoints.Store(ep1.GetNamespace()+"/"+ep1.GetName(), ep1)
	locationCache.endpoints.Store("invalid/endpoints", "invalidEndpoints")

	tests := []struct {
		name string
		lc   *LocationCache
		ep   v1.Endpoints
		want bool
	}{
		{
			name: "TestIsEndpointsUpdated() Case: No changed",
			lc:   &locationCache,
			ep:   ep1,
			want: false,
		},
		{
			name: "TestIsEndpointsUpdated() Case: Update subsets",
			lc:   &locationCache,
			ep:   ep2,
			want: true,
		},
		{
			name: "TestIsEndpointsUpdated() Case: Not found",
			lc:   &LocationCache{},
			ep:   ep1,
			want: true,
		},
		{
			name: "TestIsEndpointsUpdated() Case: Update a invalid value",
			lc:   &locationCache,
			ep: v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "endpoints",
					Namespace: "invalid",
				},
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.lc.IsEndpointsUpdated(test.ep); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Manager.TestIsEndpointsUpdated() case failed: got: %v, want: %v", got, test.want)
			}
		})
	}
}
