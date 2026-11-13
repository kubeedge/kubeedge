/*
Copyright 2026 The KubeEdge Authors.

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
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPodConfigMapsAndSecretsReferenceKinds covers resource references not
// exercised by the existing AddOrUpdatePod node-mapping regression tests.
func TestPodConfigMapsAndSecretsReferenceKinds(t *testing.T) {
	tests := []struct {
		name       string
		spec       v1.PodSpec
		configMaps []string
		secrets    []string
	}{
		{
			name: "projected volume sources",
			spec: v1.PodSpec{Volumes: []v1.Volume{{
				Name: "credentials",
				VolumeSource: v1.VolumeSource{Projected: &v1.ProjectedVolumeSource{
					Sources: []v1.VolumeProjection{
						{ConfigMap: &v1.ConfigMapProjection{
							LocalObjectReference: v1.LocalObjectReference{Name: "projected-config"},
						}},
						{Secret: &v1.SecretProjection{
							LocalObjectReference: v1.LocalObjectReference{Name: "projected-secret"},
						}},
					},
				}},
			}}},
			configMaps: []string{"projected-config"},
			secrets:    []string{"projected-secret"},
		},
		{
			name: "environment key selectors",
			spec: v1.PodSpec{Containers: []v1.Container{{
				Name: "app",
				Env: []v1.EnvVar{
					{Name: "API_URL", ValueFrom: &v1.EnvVarSource{
						ConfigMapKeyRef: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{Name: "env-config"},
							Key:                  "url",
						},
					}},
					{Name: "API_TOKEN", ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{Name: "env-secret"},
							Key:                  "token",
						},
					}},
					// A downward-API field reference is not a ConfigMap or Secret.
					{Name: "POD_NAME", ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"},
					}},
				},
			}}},
			configMaps: []string{"env-config"},
			secrets:    []string{"env-secret"},
		},
		{
			name: "image pull secrets",
			spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
				{Name: "registry-auth"},
			}},
			secrets: []string{"registry-auth"},
		},
		{
			name: "no referenced resources",
			spec: v1.PodSpec{Containers: []v1.Container{{
				Name: "app", Env: []v1.EnvVar{{Name: "STATIC", Value: "constant"}},
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const namespace, node = "test-ns", "edge-node-1"
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "test-pod"},
				Spec:       tt.spec,
			}
			pod.Spec.NodeName = node
			cache := &LocationCache{}
			gotConfigMaps, gotSecrets := cache.PodConfigMapsAndSecrets(pod)
			assertResourceNames(t, "ConfigMaps", gotConfigMaps, tt.configMaps)
			assertResourceNames(t, "Secrets", gotSecrets, tt.secrets)

			cache.AddOrUpdatePod(pod)
			for _, name := range tt.configMaps {
				if got := cache.ConfigMapNodes(namespace, name); !reflect.DeepEqual(got, []string{node}) {
					t.Errorf("ConfigMap %q nodes = %v, want [%s]", name, got, node)
				}
			}
			for _, name := range tt.secrets {
				if got := cache.SecretNodes(namespace, name); !reflect.DeepEqual(got, []string{node}) {
					t.Errorf("Secret %q nodes = %v, want [%s]", name, got, node)
				}
			}
		})
	}
}

func assertResourceNames(t *testing.T, resource string, got, want []string) {
	t.Helper()
	// Order is not part of the API contract; do not mutate cache-owned slices.
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if !reflect.DeepEqual(gotSorted, wantSorted) {
		t.Errorf("%s = %v, want %v", resource, gotSorted, wantSorted)
	}
}
