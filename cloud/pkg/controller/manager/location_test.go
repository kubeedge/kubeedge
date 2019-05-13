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

	"k8s.io/api/core/v1"
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
