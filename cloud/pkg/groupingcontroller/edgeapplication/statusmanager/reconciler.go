package statusmanager

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/apps"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	groupingv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/grouping/v1alpha1"
)

var caredGVKs map[schema.GroupVersionKind]struct{} = map[schema.GroupVersionKind]struct{}{
	{Group: "apps", Version: "v1", Kind: "Deployment"}: {},
}

type statusReconciler struct {
	schema.GroupVersionKind
	runtime.Serializer
	client.Client
}

func (r *statusReconciler) Reconcile(ctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	edgeApp := &groupingv1alpha1.EdgeApplication{}
	if err := r.Client.Get(ctx, request.NamespacedName, edgeApp); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		klog.Errorf("failed to get edgeApp %s/%s, %v", request.Namespace, request.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}

	if !edgeApp.GetDeletionTimestamp().IsZero() {
		return controllerruntime.Result{}, nil
	}

	return r.sync(ctx, edgeApp)
}

func (r *statusReconciler) sync(ctx context.Context, edgeApp *groupingv1alpha1.EdgeApplication) (controllerruntime.Result, error) {
	objs, err := GetContainedResourceObjs(edgeApp, r.Serializer)
	if err != nil {
		klog.Errorf("failed to get objects of manifests in edgeApp, %v", err)
		return controllerruntime.Result{Requeue: true}, err
	}

	for _, obj := range objs {
		gvk := obj.GroupVersionKind()
		if gvk != r.GroupVersionKind {
			// it's not managed by this reconciler
			continue
		}
		if _, ok := caredGVKs[gvk]; !ok {
			available := availableIfExists{}
			if err := r.updateStatus(ctx, edgeApp, obj, available); err != nil {
				klog.Errorf("failed to update status for edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
		} else {
			// Currently only deployment are cared
			available := deploymentAvailable{}
			if err := r.updateStatus(ctx, edgeApp, obj, available); err != nil {
				klog.Errorf("failed to update status for edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
		}
	}
	return controllerruntime.Result{}, nil
}

func (r *statusReconciler) updateStatus(
	ctx context.Context,
	edgeApp *groupingv1alpha1.EdgeApplication,
	obj *unstructured.Unstructured,
	available available) error {
	isAvailable, err := available.IsAvailable(ctx, r.Client, obj)
	if err != nil {
		return fmt.Errorf("failed to check the availability of obj %s/%s, gvk: %s, %v",
			obj.GetNamespace(), obj.GetName(), obj.GroupVersionKind(), err)
	}
	if isAvailable {
		return r.update(ctx, edgeApp, obj, groupingv1alpha1.EdgeAppAvailable)
	}
	return r.update(ctx, edgeApp, obj, groupingv1alpha1.EdgeAppProcessing)
}

func (r *statusReconciler) update(
	ctx context.Context,
	edgeApp *groupingv1alpha1.EdgeApplication,
	obj *unstructured.Unstructured,
	status groupingv1alpha1.ManifestCondition) error {
	isSame := func(identifier groupingv1alpha1.ResourceIdentifier, obj *unstructured.Unstructured) bool {
		return identifier.Group == obj.GroupVersionKind().Group &&
			identifier.Version == obj.GroupVersionKind().Version &&
			identifier.Kind == obj.GroupVersionKind().Kind &&
			identifier.Namespace == obj.GetNamespace() &&
			identifier.Name == obj.GetName()
	}

	var statusInEdgeApp *groupingv1alpha1.ManifestStatus
	for i, status := range edgeApp.Status.WorkloadStatus {
		if isSame(status.Identifier, obj) {
			statusInEdgeApp = &edgeApp.Status.WorkloadStatus[i]
			break
		}
	}
	if statusInEdgeApp == nil {
		// not found, add a new entry for it
		edgeApp.Status.WorkloadStatus = append(edgeApp.Status.WorkloadStatus, groupingv1alpha1.ManifestStatus{
			Identifier: groupingv1alpha1.ResourceIdentifier{
				Group:     obj.GroupVersionKind().Group,
				Version:   obj.GroupVersionKind().Version,
				Kind:      obj.GroupVersionKind().Kind,
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
			Condition: status,
		})
	} else if statusInEdgeApp.Condition == status {
		// no need to update status
		return nil
	} else {
		statusInEdgeApp.Condition = status
	}

	if err := r.Client.Update(ctx, edgeApp); err != nil {
		return fmt.Errorf("failed to update status of EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
	}
	return nil
}

type available interface {
	IsAvailable(context.Context, client.Client, runtime.Object) (bool, error)
}

type availableIfExists struct{}

func (e availableIfExists) IsAvailable(ctx context.Context, client client.Client, obj runtime.Object) (bool, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return false, fmt.Errorf("failed to convert object to unstructured")
	}

	gvk := unstructuredObj.GroupVersionKind()
	ns, name := unstructuredObj.GetNamespace(), unstructuredObj.GetName()
	curObj := &unstructured.Unstructured{}
	curObj.SetGroupVersionKind(gvk)
	if err := client.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, curObj); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get obj %s/%s, gvk: %s, %v", ns, name, gvk, err)
	}
	return true, nil
}

type deploymentAvailable struct{}

func (d deploymentAvailable) IsAvailable(ctx context.Context, client client.Client, obj runtime.Object) (bool, error) {
	deploy, ok := obj.(*apps.Deployment)
	if !ok {
		return false, fmt.Errorf("failed to convert objecto to deployment")
	}
	curDeploy := &apps.Deployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: deploy.Namespace, Name: deploy.Name}, curDeploy); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get deployment %s/%s, %v", deploy.Namespace, deploy.Name, err)
	}

	return deploy.Status.ReadyReplicas == deploy.Spec.Replicas, nil
}
