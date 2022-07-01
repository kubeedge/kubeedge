package statusmanager

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

type statusReconciler struct {
	schema.GroupVersionKind
	runtime.Serializer
	client.Client
	overridemanager.Overrider
	ReoncileTriggerChan chan event.GenericEvent
}

func (r *statusReconciler) Reconcile(ctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	edgeApp := &appsv1alpha1.EdgeApplication{}
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

func (r *statusReconciler) sync(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication) (controllerruntime.Result, error) {
	tmplInfos, err := utils.GetTemplatesInfosOfEdgeApp(edgeApp, r.Serializer)
	if err != nil {
		klog.Errorf("failed to get infos of templates in edgeApp %s/%s, continue with what has been got", edgeApp.Namespace, edgeApp.Name, err)
	}

	for _, tmplInfo := range tmplInfos {
		tmpl := tmplInfo.Template
		gvk := tmpl.GroupVersionKind()
		if gvk != r.GroupVersionKind {
			// it's not managed by this reconciler
			continue
		}
		if _, ok := constants.OverriderTargetGVK[gvk]; !ok {
			if err := r.updateStatus(ctx, edgeApp, tmplInfo, availableIfExists{}); err != nil {
				klog.Errorf("failed to update status for edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
				return controllerruntime.Result{Requeue: true}, err
			}
		} else {
			// Apply overriders to get the actual template that applied to
			// each nodegroup. Currently, only NameOverrider is applied.
			overrideInfos := utils.GetAllOverriders(edgeApp)
			for _, overrideInfo := range overrideInfos {
				copy := tmpl.DeepCopy()
				if err := r.Overrider.ApplyOverrides(copy, overrideInfo); err != nil {
					klog.Errorf("failed to apply overrides to template %s/%s of gvk %s when updating status of edgeapp %s/%s, %v",
						copy.GetNamespace(), copy.GetName(), gvk, edgeApp.Namespace, edgeApp.Name, err)
					continue
				}
				// TODO:
				// Currently, only deployment will be override. So, we just use the deploymentAvailable here.
				// It a temporary strategy for convinence. When we need to support more GVK, a generic strategy
				// is needed.
				newTmplInfo := &utils.TemplateInfo{Ordinal: tmplInfo.Ordinal, Template: copy}
				if err := r.updateStatus(ctx, edgeApp, newTmplInfo, deploymentAvailable{}); err != nil {
					klog.Errorf("failed to update status for edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
					return controllerruntime.Result{Requeue: true}, err
				}
			}
		}
	}
	// Trigger reconciliation of edgeapplication controller
	// Considering that if an object created by edgeapplication controller is deleted by others,
	// only this controller watches its event, thus if it status in edgeapplication resource is unchanged,
	// the deleted object will not be created until the next resync.
	r.ReoncileTriggerChan <- event.GenericEvent{Object: edgeApp.DeepCopy()}
	return controllerruntime.Result{}, nil
}

func (r *statusReconciler) updateStatus(
	ctx context.Context,
	edgeApp *appsv1alpha1.EdgeApplication,
	tmplInfo *utils.TemplateInfo,
	available available) error {
	info := utils.GetResourceInfoOfTemplateInfo(tmplInfo)
	isAvailable, err := available.IsAvailable(ctx, r.Client, info)
	if err != nil {
		klog.Errorf("failed to check the availability of obj %s/%s, %s/%s, kind: %s, %v",
			info.Namespace, info.Name, info.Group, info.Version, info.Kind, err)
	}
	if isAvailable {
		return r.update(ctx, edgeApp, info, appsv1alpha1.EdgeAppAvailable)
	}
	return r.update(ctx, edgeApp, info, appsv1alpha1.EdgeAppProcessing)
}

func (r *statusReconciler) update(
	ctx context.Context,
	edgeApp *appsv1alpha1.EdgeApplication,
	info utils.ResourceInfo,
	status appsv1alpha1.ManifestCondition) error {
	newStatus := appsv1alpha1.ManifestStatus{
		Identifier: appsv1alpha1.ResourceIdentifier{
			Ordinal:   info.Ordinal,
			Group:     info.Group,
			Version:   info.Version,
			Kind:      info.Kind,
			Namespace: info.Namespace,
			Name:      info.Name,
		},
		Condition: status,
	}

	var statusInEdgeApp *appsv1alpha1.ManifestStatus
	for i, status := range edgeApp.Status.WorkloadStatus {
		// find the existing status that need update
		if status.Identifier.Ordinal == info.Ordinal {
			if utils.IsIdentifierSameAsResourceInfo(status.Identifier, info) || utils.IsInitStatus(&status) {
				statusInEdgeApp = &edgeApp.Status.WorkloadStatus[i]
				break
			}
		}
	}
	if statusInEdgeApp == nil {
		// not found, add a new entry for it
		edgeApp.Status.WorkloadStatus = append(edgeApp.Status.WorkloadStatus, newStatus)
	} else if utils.IsInitStatus(statusInEdgeApp) || statusInEdgeApp.Condition != status {
		// the existing status needs to be updated
		*statusInEdgeApp = newStatus
	} else {
		// no need to update status
		klog.V(4).Infof("obj %s/%s of gvk %s/%s, %s has same status as what in edgeapp %s/%s, skip update status",
			info.Namespace, info.Name, info.Group, info.Version, info.Kind, edgeApp.Namespace, edgeApp.Name)
		return nil
	}

	sort.Slice(edgeApp.Status.WorkloadStatus, func(i, j int) bool {
		return edgeApp.Status.WorkloadStatus[i].Identifier.Ordinal < edgeApp.Status.WorkloadStatus[j].Identifier.Ordinal
	})
	if err := r.Client.Status().Update(ctx, edgeApp); err != nil {
		return fmt.Errorf("failed to update status of EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
	}
	klog.V(4).Infof("successfully update status of edgeapp %s/%s for obj %s/%s of gvk %s/%s, Kind %s with value %s",
		edgeApp.Namespace, edgeApp.Name, info.Namespace, info.Name, info.Group, info.Version, info.Kind, status)
	return nil
}

type available interface {
	IsAvailable(context.Context, client.Client, utils.ResourceInfo) (bool, error)
}

var _ available = availableIfExists{}
var _ available = deploymentAvailable{}

type availableIfExists struct{}

func (e availableIfExists) IsAvailable(ctx context.Context, client client.Client, info utils.ResourceInfo) (bool, error) {
	obj, err := getObjAccordingToResourceInfo(ctx, client, info)
	if err != nil {
		return false, err
	}
	if obj == nil {
		return false, nil
	}
	return true, nil
}

type deploymentAvailable struct{}

func (d deploymentAvailable) IsAvailable(ctx context.Context, client client.Client, info utils.ResourceInfo) (bool, error) {
	obj, err := getObjAccordingToResourceInfo(ctx, client, info)
	if err != nil {
		return false, err
	}
	if obj == nil {
		return false, err
	}

	deploy := &appsv1.Deployment{}
	if err := client.Scheme().Convert(obj, deploy, nil); err != nil {
		return false, fmt.Errorf("failed to convert unstructured to deployment for %s/%s, %v", info.Namespace, info.Name, err)
	}

	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
}

func getObjAccordingToResourceInfo(ctx context.Context, client client.Client, info utils.ResourceInfo) (*unstructured.Unstructured, error) {
	gvk := schema.GroupVersionKind{Group: info.Group, Version: info.Version, Kind: info.Kind}
	curObj := &unstructured.Unstructured{}
	curObj.SetGroupVersionKind(gvk)
	if err := client.Get(ctx, types.NamespacedName{Namespace: info.Namespace, Name: info.Name}, curObj); err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(4).Infof("cannot find obj %s/%s of gvk %s", info.Namespace, info.Name, gvk)
			return nil, nil
		}
		klog.Errorf("failed to get obj %s/%s, gvk: %s, %v", info.Namespace, info.Name, gvk, err)
		return nil, err
	}
	return curObj, nil
}
