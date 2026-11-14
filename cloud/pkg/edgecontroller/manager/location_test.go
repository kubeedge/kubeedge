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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonconst "github.com/kubeedge/kubeedge/common/constants"
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

// configMapEnvVar returns an env var sourced from a configmap key.
func configMapEnvVar(name string) v1.EnvVar {
	return v1.EnvVar{
		Name: "CONFIG_MAP_KEY",
		ValueFrom: &v1.EnvVarSource{
			ConfigMapKeyRef: &v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: name},
				Key:                  "key",
			},
		},
	}
}

// secretEnvVar returns an env var sourced from a secret key.
func secretEnvVar(name string) v1.EnvVar {
	return v1.EnvVar{
		Name: "SECRET_KEY",
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: name},
				Key:                  "key",
			},
		},
	}
}

// configMapEnvFrom returns an envFrom source sourced from a configmap.
func configMapEnvFrom(name string) v1.EnvFromSource {
	return v1.EnvFromSource{
		ConfigMapRef: &v1.ConfigMapEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: name}},
	}
}

// secretEnvFrom returns an envFrom source sourced from a secret.
func secretEnvFrom(name string) v1.EnvFromSource {
	return v1.EnvFromSource{
		SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: name}},
	}
}

// configMapVolume returns a volume sourced from a configmap.
func configMapVolumeSource(name string) v1.VolumeSource {
	return v1.VolumeSource{
		ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: name}},
	}
}

// secretVolumeSource returns a volume sourced from a secret.
func secretVolumeSource(name string) v1.VolumeSource {
	return v1.VolumeSource{
		Secret: &v1.SecretVolumeSource{SecretName: name},
	}
}

// uniqueStrings returns values with duplicates removed, preserving first-seen order.
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// syncMapLen returns the number of entries stored in a sync.Map.
func syncMapLen(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// podRefsCase describes a pod and the configmaps/secrets it references.
type podRefsCase struct {
	name           string
	pod            v1.Pod
	wantConfigMaps []string
	wantSecrets    []string
}

// podRefsCases returns pods covering every container type and volume source that
// PodConfigMapsAndSecrets has to resolve.
func podRefsCases() []podRefsCase {
	const namespace = "default"

	return []podRefsCase{
		{
			name: "init container env refs",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "init-env"},
				Spec: v1.PodSpec{
					NodeName: "node-init-env",
					InitContainers: []v1.Container{{
						Name: "init",
						Env: []v1.EnvVar{
							configMapEnvVar("init-env-cm"),
							secretEnvVar("init-env-secret"),
						},
					}},
				},
			},
			wantConfigMaps: []string{"init-env-cm"},
			wantSecrets:    []string{"init-env-secret"},
		},
		{
			name: "init container envFrom refs",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "init-envfrom"},
				Spec: v1.PodSpec{
					NodeName: "node-init-envfrom",
					InitContainers: []v1.Container{{
						Name: "init",
						EnvFrom: []v1.EnvFromSource{
							configMapEnvFrom("init-envfrom-cm"),
							secretEnvFrom("init-envfrom-secret"),
						},
					}},
				},
			},
			wantConfigMaps: []string{"init-envfrom-cm"},
			wantSecrets:    []string{"init-envfrom-secret"},
		},
		{
			name: "ephemeral container env and envFrom refs",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "ephemeral"},
				Spec: v1.PodSpec{
					NodeName: "node-ephemeral",
					EphemeralContainers: []v1.EphemeralContainer{{
						EphemeralContainerCommon: v1.EphemeralContainerCommon{
							Name: "debugger",
							Env: []v1.EnvVar{
								configMapEnvVar("ephemeral-env-cm"),
								secretEnvVar("ephemeral-env-secret"),
							},
							EnvFrom: []v1.EnvFromSource{
								configMapEnvFrom("ephemeral-envfrom-cm"),
								secretEnvFrom("ephemeral-envfrom-secret"),
							},
						},
					}},
				},
			},
			wantConfigMaps: []string{"ephemeral-env-cm", "ephemeral-envfrom-cm"},
			wantSecrets:    []string{"ephemeral-env-secret", "ephemeral-envfrom-secret"},
		},
		{
			name: "regular containers, volumes and imagePullSecrets",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "regular"},
				Spec: v1.PodSpec{
					NodeName: "node-regular",
					Volumes: []v1.Volume{
						{Name: "cm", VolumeSource: configMapVolumeSource("vol-cm")},
						{Name: "secret", VolumeSource: secretVolumeSource("vol-secret")},
						{
							Name: "projected",
							VolumeSource: v1.VolumeSource{
								Projected: &v1.ProjectedVolumeSource{
									Sources: []v1.VolumeProjection{
										{ConfigMap: &v1.ConfigMapProjection{LocalObjectReference: v1.LocalObjectReference{Name: "projected-cm"}}},
										{Secret: &v1.SecretProjection{LocalObjectReference: v1.LocalObjectReference{Name: "projected-secret"}}},
									},
								},
							},
						},
					},
					Containers: []v1.Container{{
						Name: "app",
						EnvFrom: []v1.EnvFromSource{
							configMapEnvFrom("regular-envfrom-cm"),
							secretEnvFrom("regular-envfrom-secret"),
						},
						Env: []v1.EnvVar{
							configMapEnvVar("regular-env-cm"),
							secretEnvVar("regular-env-secret"),
						},
					}},
					ImagePullSecrets: []v1.LocalObjectReference{{Name: "pull-secret"}},
				},
			},
			wantConfigMaps: []string{"vol-cm", "projected-cm", "regular-envfrom-cm", "regular-env-cm"},
			wantSecrets:    []string{"vol-secret", "projected-secret", "regular-envfrom-secret", "regular-env-secret", "pull-secret"},
		},
		{
			name: "duplicate init container refs",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "duplicate"},
				Spec: v1.PodSpec{
					NodeName: "node-duplicate",
					InitContainers: []v1.Container{
						{
							Name: "init-a",
							Env: []v1.EnvVar{
								configMapEnvVar("dup-cm"),
								secretEnvVar("dup-secret"),
							},
						},
						{
							Name: "init-b",
							Env: []v1.EnvVar{
								configMapEnvVar("dup-cm"),
								secretEnvVar("dup-secret"),
							},
						},
					},
				},
			},
			wantConfigMaps: []string{"dup-cm"},
			wantSecrets:    []string{"dup-secret"},
		},
		{
			name: "pod without refs",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "no-refs"},
				Spec: v1.PodSpec{
					NodeName:   "node-no-refs",
					Containers: []v1.Container{{Name: "app", Image: "busybox"}},
				},
			},
			wantConfigMaps: nil,
			wantSecrets:    nil,
		},
	}
}

// TestPodConfigMapsAndSecrets is function to test PodConfigMapsAndSecrets
func TestPodConfigMapsAndSecrets(t *testing.T) {
	for _, test := range podRefsCases() {
		t.Run(test.name, func(t *testing.T) {
			locationCache := &LocationCache{}
			configMaps, secrets := locationCache.PodConfigMapsAndSecrets(test.pod)

			// Multiplicity in the returned slices is intentionally not asserted.
			assert.ElementsMatch(t, uniqueStrings(test.wantConfigMaps), uniqueStrings(configMaps), "unexpected configmaps")
			assert.ElementsMatch(t, uniqueStrings(test.wantSecrets), uniqueStrings(secrets), "unexpected secrets")
			if len(test.wantConfigMaps) == 0 {
				assert.Empty(t, configMaps, "expected no configmaps")
			}
			if len(test.wantSecrets) == 0 {
				assert.Empty(t, secrets, "expected no secrets")
			}
		})
	}
}

// TestPodConfigMapsAndSecretsNodes verifies that pods are mapped to their node
// for every configmap and secret they reference, including those referenced
// only by init or ephemeral containers.
func TestPodConfigMapsAndSecretsNodes(t *testing.T) {
	const namespace = "default"

	for _, test := range podRefsCases() {
		t.Run(test.name, func(t *testing.T) {
			locationCache := &LocationCache{}
			locationCache.AddOrUpdatePod(test.pod)

			wantNodes := []string{test.pod.Spec.NodeName}
			for _, name := range test.wantConfigMaps {
				if nodes := locationCache.ConfigMapNodes(namespace, name); !reflect.DeepEqual(nodes, wantNodes) {
					t.Errorf("Manager.TestPodConfigMapsAndSecretsNodes() configmap %q: got = %v, want = %v", name, nodes, wantNodes)
				}
			}
			for _, name := range test.wantSecrets {
				if nodes := locationCache.SecretNodes(namespace, name); !reflect.DeepEqual(nodes, wantNodes) {
					t.Errorf("Manager.TestPodConfigMapsAndSecretsNodes() secret %q: got = %v, want = %v", name, nodes, wantNodes)
				}
			}

			if got, want := syncMapLen(&locationCache.configMapNode), len(uniqueStrings(test.wantConfigMaps)); got != want {
				t.Errorf("Manager.TestPodConfigMapsAndSecretsNodes() configmap node entries: got = %d, want = %d", got, want)
			}
			if got, want := syncMapLen(&locationCache.secretNode), len(uniqueStrings(test.wantSecrets)); got != want {
				t.Errorf("Manager.TestPodConfigMapsAndSecretsNodes() secret node entries: got = %d, want = %d", got, want)
			}
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
	locationCache.EdgeNodes.Store(nodeName, commonconst.MessageSuccessfulContent)

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

// TestUpdateEdgeNode is function to test UpdateEdgeNode
func TestUpdateEdgeNode(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	locationCache.EdgeNodes.Store(nodeName, "")

	tests := []struct {
		name string
		lc   *LocationCache
		want bool
	}{
		{
			name: "TestUpdateEdgeNode() Case: Node status update to OK",
			lc:   &locationCache,
			want: true,
		},
		{
			name: "TestUpdateEdgeNode() Case: Node status update to Unknown",
			lc:   &locationCache,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.lc.UpdateEdgeNode(nodeName)
			if _, ok := test.lc.EdgeNodes.Load(nodeName); !ok {
				t.Errorf("Manager.TestUpdateEdgeNode() case failed: got = %v, want = %v.", ok, test.want)
			}
		})
	}
}

// TestDeleteNode is function to test DeleteNode
func TestDeleteNode(t *testing.T) {
	locationCache := LocationCache{}
	nodeName := nodes[0]
	locationCache.EdgeNodes.Store(nodeName, commonconst.MessageSuccessfulContent)

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
