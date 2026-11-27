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

func TestAddOrUpdatePodResourceNodeMappings(t *testing.T) {
	const configMapName, secretName = "app-config", "app-secret"
	lc := &LocationCache{}
	checkNodes := func(stage, namespace string, want []string) {
		t.Helper()
		for _, resource := range []struct {
			name  string
			nodes []string
		}{
			{"ConfigMap", lc.ConfigMapNodes(namespace, configMapName)},
			{"Secret", lc.SecretNodes(namespace, secretName)},
		} {
			// Node ordering is not part of the lookup contract. Sort a copy
			// so the assertion does not modify the cache's returned slice.
			got := append([]string(nil), resource.nodes...)
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %s nodes in namespace %q = %v, want %v", stage, resource.name, namespace, got, want)
			}
		}
	}

	// Referencing each resource through both a volume and an environment
	// source must still register the node only once.
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "first"},
		Spec: v1.PodSpec{
			NodeName: "edge-2",
			Volumes: []v1.Volume{
				{Name: "config", VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: configMapName}},
				}},
				{Name: "secret", VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{SecretName: secretName},
				}},
			},
			Containers: []v1.Container{{Name: "app", EnvFrom: []v1.EnvFromSource{
				{ConfigMapRef: &v1.ConfigMapEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: configMapName}}},
				{SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: secretName}}},
			}}},
		},
	}
	lc.AddOrUpdatePod(pod)
	checkNodes("duplicate references", "team-a", []string{"edge-2"})
	checkNodes("unused namespace", "team-b", nil)

	lc.AddOrUpdatePod(pod)
	checkNodes("repeated pod update", "team-a", []string{"edge-2"})
	pod.Name = "second"
	lc.AddOrUpdatePod(pod)
	checkNodes("two pods on one node", "team-a", []string{"edge-2"})

	pod.Namespace, pod.Name, pod.Spec.NodeName = "team-b", "other-team", "edge-3"
	lc.AddOrUpdatePod(pod)
	checkNodes("same names in another namespace", "team-a", []string{"edge-2"})
	checkNodes("same names in another namespace", "team-b", []string{"edge-3"})

	pod.Namespace, pod.Name, pod.Spec.NodeName = "team-a", "third", "edge-1"
	lc.AddOrUpdatePod(pod)
	checkNodes("second node", "team-a", []string{"edge-1", "edge-2"})
	checkNodes("second node", "team-b", []string{"edge-3"})
}
