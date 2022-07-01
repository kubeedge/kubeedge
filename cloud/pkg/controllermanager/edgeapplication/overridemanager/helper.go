/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/helper/unstructured.go
- pick some functions to handle apis, including pod, replicaset, deployment, deamonset, statefulset
  and job.
*/

package overridemanager

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConvertToPod converts a Pod object from unstructured to typed.
func ConvertToPod(obj *unstructured.Unstructured) (*corev1.Pod, error) {
	typedObj := &corev1.Pod{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// ConvertToReplicaSet converts a ReplicaSet object from unstructured to typed.
func ConvertToReplicaSet(obj *unstructured.Unstructured) (*appsv1.ReplicaSet, error) {
	typedObj := &appsv1.ReplicaSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// ConvertToDeployment converts a Deployment object from unstructured to typed.
func ConvertToDeployment(obj *unstructured.Unstructured) (*appsv1.Deployment, error) {
	typedObj := &appsv1.Deployment{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// ConvertToDaemonSet converts a DaemonSet object from unstructured to typed.
func ConvertToDaemonSet(obj *unstructured.Unstructured) (*appsv1.DaemonSet, error) {
	typedObj := &appsv1.DaemonSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// ConvertToStatefulSet converts a StatefulSet object from unstructured to typed.
func ConvertToStatefulSet(obj *unstructured.Unstructured) (*appsv1.StatefulSet, error) {
	typedObj := &appsv1.StatefulSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}

// ConvertToJob converts a Job object from unstructured to typed.
func ConvertToJob(obj *unstructured.Unstructured) (*batchv1.Job, error) {
	typedObj := &batchv1.Job{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
		return nil, err
	}

	return typedObj, nil
}
