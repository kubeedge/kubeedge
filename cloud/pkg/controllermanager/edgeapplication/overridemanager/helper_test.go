/*
Copyright 2024 The KubeEdge Authors.

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

package overridemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestConvertToPod(t *testing.T) {
	assert := assert.New(t)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "test-container", Image: "test-image"},
			},
		},
	}

	unstructuredPod, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	assert.NoError(err)

	result, err := ConvertToPod(&unstructured.Unstructured{Object: unstructuredPod})
	assert.NoError(err)
	assert.Equal(pod.Name, result.Name)
	assert.Equal(pod.Spec.Containers[0].Name, result.Spec.Containers[0].Name)
}

func TestConvertToReplicaSet(t *testing.T) {
	assert := assert.New(t)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-rs",
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: new(int32),
		},
	}

	unstructuredRS, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rs)
	assert.NoError(err)

	result, err := ConvertToReplicaSet(&unstructured.Unstructured{Object: unstructuredRS})
	assert.NoError(err)
	assert.Equal(rs.Name, result.Name)
	assert.Equal(rs.Spec.Replicas, result.Spec.Replicas)
}

func TestConvertToDeployment(t *testing.T) {
	assert := assert.New(t)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32),
		},
	}

	unstructuredDeployment, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
	assert.NoError(err)

	result, err := ConvertToDeployment(&unstructured.Unstructured{Object: unstructuredDeployment})
	assert.NoError(err)
	assert.Equal(deployment.Name, result.Name)
	assert.Equal(deployment.Spec.Replicas, result.Spec.Replicas)
}

func TestConvertToDaemonSet(t *testing.T) {
	assert := assert.New(t)

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ds",
		},
		Spec: appsv1.DaemonSetSpec{
			RevisionHistoryLimit: new(int32),
		},
	}

	unstructuredDS, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ds)
	assert.NoError(err)

	result, err := ConvertToDaemonSet(&unstructured.Unstructured{Object: unstructuredDS})
	assert.NoError(err)
	assert.Equal(ds.Name, result.Name)
	assert.Equal(ds.Spec.RevisionHistoryLimit, result.Spec.RevisionHistoryLimit)
}

func TestConvertToStatefulSet(t *testing.T) {
	assert := assert.New(t)

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ss",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: new(int32),
		},
	}

	unstructuredSS, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ss)
	assert.NoError(err)

	result, err := ConvertToStatefulSet(&unstructured.Unstructured{Object: unstructuredSS})
	assert.NoError(err)
	assert.Equal(ss.Name, result.Name)
	assert.Equal(ss.Spec.Replicas, result.Spec.Replicas)
}

func TestConvertToJob(t *testing.T) {
	assert := assert.New(t)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
		Spec: batchv1.JobSpec{
			Completions: new(int32),
		},
	}

	unstructuredJob, err := runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	assert.NoError(err)

	result, err := ConvertToJob(&unstructured.Unstructured{Object: unstructuredJob})
	assert.NoError(err)
	assert.Equal(job.Name, result.Name)
	assert.Equal(job.Spec.Completions, result.Spec.Completions)
}
