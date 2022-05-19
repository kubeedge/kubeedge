package edgeapplication

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeedge/kubeedge/cloud/pkg/groupingcontroller/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/groupingcontroller/edgeapplication/statusmanager"
	groupingv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/grouping/v1alpha1"
)

const (
	LastAppliedTemplateAnnotationKey    = "grouping.kubeedge.io/last-applied-template"
	LastContainedResourcesAnnotationKey = "grouping.kubeedge.io/last-contained-resources"
)

var overriderTargetGVK map[schema.GroupVersionKind]struct{} = map[schema.GroupVersionKind]struct{}{
	{Group: "apps", Version: "v1", Kind: "Deployment"}: {},
}

// Controller is to sync EdgeApplication.
type Controller struct {
	client.Client
	runtime.Serializer
	overridemanager.Overrider
	statusmanager.StatusManager
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.Infof("Reconciling EdgeApplication %s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name)

	edgeApp := &groupingv1alpha1.EdgeApplication{}
	if err := c.Client.Get(ctx, req.NamespacedName, edgeApp); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}

		klog.Errorf("failed to get edgeapplication %s/%s, %v", req.NamespacedName.Namespace, req.NamespacedName.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}

	if !edgeApp.DeletionTimestamp.IsZero() {
		// foreground cascade deletion of OwnerReference
		// will take the responsibility of removing created resources.
		return controllerruntime.Result{}, nil
	}

	annotations := edgeApp.Annotations
	if annotations == nil || annotations[LastContainedResourcesAnnotationKey] == "" {
		// it is a new created EdgeApplication
		// add LastContainedResourcesAnnotation for it
		if err := c.addOrUpdateLastContainedResourcesAnnotation(ctx, edgeApp); err != nil {
			klog.Errorf("failed to add LastContainedResourcesAnnotation to EdgeApplication %s/%s, %v",
				edgeApp.Namespace, edgeApp.Name, err)
			return controllerruntime.Result{Requeue: true}, err
		}
		// it will reconcile at the next time for update
		return controllerruntime.Result{}, nil
	}

	return c.syncEdgeApplication(ctx, edgeApp)
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	if c.Client == nil {
		return fmt.Errorf("client of edgeapplication controller cannot be nil")
	}
	if c.Serializer == nil {
		return fmt.Errorf("serializer of edgeapplication controller cannot be nil")
	}
	if c.StatusManager == nil {
		return fmt.Errorf("status manager of edgeapplication controller cannot be nil")
	}
	if c.Overrider == nil {
		return fmt.Errorf("overrider of edgeapplication controller cannot be nil")
	}
	// start the StatusManager
	c.StatusManager.Start()
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&groupingv1alpha1.EdgeApplication{}).
		Complete(c)
}

func (c *Controller) syncEdgeApplication(ctx context.Context, edgeApp *groupingv1alpha1.EdgeApplication) (controllerruntime.Result, error) {
	// 1. get manifests, set ownerReference and apply overrides to all target resources
	manifests := edgeApp.Spec.WorkloadTemplate.Manifests
	overriderInfos := getAllOverriders(edgeApp)
	var modifiedTmpls []*unstructured.Unstructured
	for _, manifest := range manifests {
		copy := manifest.DeepCopy()
		unstructuredObj := &unstructured.Unstructured{}
		if _, _, err := c.Serializer.Decode(copy.Raw, nil, unstructuredObj); err != nil {
			klog.Errorf("failed to get the unstructured of manifest: %s, %v", copy.Raw, err)
			return controllerruntime.Result{Requeue: true}, err
		}
		setOwnerReference(unstructuredObj, edgeApp)
		if !needOverride(manifest.Object) {
			modifiedTmpls = append(modifiedTmpls, unstructuredObj)
			continue
		}

		// apply overriders
		for _, info := range overriderInfos {
			if err := c.Overrider.ApplyOverrides(unstructuredObj, info); err != nil {
				klog.Errorf("failed to apply override of %s to template, %v, template: %s", info.TargetNodeGroup, err, string(copy.Raw))
				return controllerruntime.Result{Requeue: true}, err
			}
			modifiedTmpls = append(modifiedTmpls, unstructuredObj)
		}
	}

	// 2. apply all templates
	for _, tmpl := range modifiedTmpls {
		if err := c.applyTemplate(ctx, tmpl); err != nil {
			klog.Errorf("failed to apply overridden template of EdgeApplication %s/%s, %v, template: %v", edgeApp.Namespace, edgeApp.Name, err, tmpl)
			return controllerruntime.Result{Requeue: true}, err
		}
		klog.V(4).Infof("successfully applied overridden template of EdgeApplication %s/%s, template: %v", edgeApp.Namespace, edgeApp.Name, tmpl)
	}

	// 3. delete resources that have been removed from the manifests
	lastContainedResourcesInfos, err := c.getLastContainedResourceInfos(edgeApp)
	if err != nil {
		klog.Errorf("failed to get infos of last contained resources in EdgeApplication %s/%s, %v",
			edgeApp.Namespace, edgeApp.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	currentContainedResourcesInfos, err := statusmanager.GetContainedResourceInfos(edgeApp, c.Serializer)
	if err != nil {
		klog.Errorf("failed to get infos of current contained resources in EdgeApplication %s/%s, %v",
			edgeApp.Namespace, edgeApp.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	deleted := getDeletedResources(lastContainedResourcesInfos, currentContainedResourcesInfos)
	for _, info := range deleted {
		if err := c.removeResource(ctx, info); err != nil {
			klog.Errorf("failed to remove resource %s/%s of gvk %s, %v",
				info.Namespace, info.Name, schema.GroupVersionKind{Group: info.Group, Version: info.Version, Kind: info.Kind}, err)
			return controllerruntime.Result{Requeue: true}, err
		}
		klog.V(4).Infof("successfully remove resource %s/%s of gvk %s, %v",
			info.Namespace, info.Name, schema.GroupVersionKind{Group: info.Group, Version: info.Version, Kind: info.Kind}, err)
	}

	// 4. update the LastContainedResourcesAnnotation
	if err := c.addOrUpdateLastContainedResourcesAnnotation(ctx, edgeApp); err != nil {
		klog.Errorf("failed to update annotation of EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
		return controllerruntime.Result{Requeue: true}, err
	}
	return controllerruntime.Result{}, nil
}

func (c *Controller) ifObjExists(ctx context.Context, obj *unstructured.Unstructured) (bool, runtime.Object, error) {
	ns, name := obj.GetNamespace(), obj.GetName()
	gvk := obj.GetObjectKind().GroupVersionKind()
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.SetGroupVersionKind(gvk)
	if err := c.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, unstructuredObj); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to get obj %s/%s of gvk %s, %v", ns, name, gvk, err)
	}
	return true, unstructuredObj, nil
}

func (c *Controller) update(ctx context.Context, tmpl *unstructured.Unstructured, curObj runtime.Object) error {
	same, err := isSameAsLastApplied(tmpl, curObj)
	if err != nil {
		// error occurs when comparing the overridden template with the last applied template
		return err
	} else if err == nil && same {
		// nothing to do for this template
		return nil
	}

	// The existing object has different last applied template than what is specified in the EdgeApplication.
	// Update the object with the template in EdgeApplication.
	//
	// TODO:
	// Currently, we just use the new template in EdgeApplication to overwrite the existing object.
	// Maybe we should do it with strategic merge patch.
	if err := c.Client.Update(ctx, tmpl); err != nil {
		return fmt.Errorf("failed to update object with template %s, %v", tmpl, err)
	}

	return nil
}

// applyTemplate will apply the passed-in template
// If the object has already existed, it will update it when it is different from what specified in the template
// If the object does not exist, it will create it according to the template
func (c *Controller) applyTemplate(ctx context.Context, tmpl *unstructured.Unstructured) error {
	ns, name := tmpl.GetNamespace(), tmpl.GetName()
	gvk := tmpl.GroupVersionKind()
	exists, curObj, err := c.ifObjExists(ctx, tmpl)
	if err != nil {
		klog.Errorf("failed to check the existence of obj %s/%s, gvk: %s, %v", ns, name, gvk, err)
		return err
	}

	if exists {
		// the obj has already exited in the cluster
		// try to update it
		klog.V(4).Infof("object %s/%s of gvk %s has already existed, try to update it with template: %v", ns, name, gvk, tmpl)
		if err := c.update(ctx, tmpl, curObj); err != nil {
			klog.Errorf("failed to update the object %s/%s, gvk: %s, %v", ns, name, gvk, err)
			return err
		}
		return nil
	}

	klog.V(4).Infof("try to create object %s/%s of gvk %s with template: %v", ns, name, gvk, tmpl)
	if err := c.Client.Create(ctx, tmpl); err != nil {
		klog.Errorf("failed to create the object %s/%s of gvk %s with template: %v, %v", ns, name, gvk, tmpl, err)
		return err
	}
	// create the object successfully, notify the StatusManager to
	// watch its status.
	return c.StatusManager.WatchStatus(statusmanager.ResourceInfo{
		Group:     gvk.Group,
		Version:   gvk.Version,
		Kind:      gvk.Kind,
		Namespace: ns,
		Name:      name,
	})
}

func (c *Controller) removeResource(ctx context.Context, info statusmanager.ResourceInfo) error {
	unstructuredObj := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{
		Group:   info.Group,
		Version: info.Version,
		Kind:    info.Kind,
	}
	unstructuredObj.SetGroupVersionKind(gvk)
	if err := c.Client.Get(ctx, types.NamespacedName{Namespace: info.Namespace, Name: info.Name}, unstructuredObj); err != nil && apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get obj %s/%s of gvk %s, %v", info.Namespace, info.Name, gvk, err)
	}

	if err := c.Client.Delete(ctx, unstructuredObj); err != nil && apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obj %s/%s of gvk %s, %v", info.Namespace, info.Name, gvk, err)
	}
	return nil
}

func (c *Controller) getLastContainedResourceInfos(edgeApp *groupingv1alpha1.EdgeApplication) ([]statusmanager.ResourceInfo, error) {
	anno := edgeApp.Annotations
	if anno == nil || anno[LastContainedResourcesAnnotationKey] == "" {
		return nil, fmt.Errorf("failed to get last contained resources of EdgeApplication %s/%s for annotation not existing",
			edgeApp.Namespace, edgeApp.Name)
	}

	infos := []statusmanager.ResourceInfo{}
	annoValue := anno[LastContainedResourcesAnnotationKey]
	lastContainedResourceJsons := strings.Split(annoValue, ",")
	for _, js := range lastContainedResourceJsons {
		info := &statusmanager.ResourceInfo{}
		if err := json.Unmarshal([]byte(js), info); err != nil {
			return nil, fmt.Errorf("failed to unmarshal containedResourceInfo %s, %v", string(js), err)
		}

		infos = append(infos, *info)
	}
	return infos, nil
}

// addOrUpdateLastContainedResourcesAnnotation will add the ContainedResourcesAnnotation to the EdgeApplication,
// if the annotation has already existed, it will be updated it according to resources in manifests.
func (c *Controller) addOrUpdateLastContainedResourcesAnnotation(ctx context.Context, edgeApp *groupingv1alpha1.EdgeApplication) error {
	if edgeApp.Annotations == nil {
		edgeApp.Annotations = make(map[string]string)
	}

	infos, err := statusmanager.GetContainedResourceInfos(edgeApp, c.Serializer)
	if err != nil {
		return fmt.Errorf("failed to get infos of current contained resources, %v", err)
	}

	containedResources := []string{}
	for _, info := range infos {
		infoJSON, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal containedResourceInfo %s in edgeApp %s/%s, %v",
				info, edgeApp.Namespace, edgeApp.Name, err)
		}
		containedResources = append(containedResources, string(infoJSON))
	}

	edgeApp.Annotations[LastContainedResourcesAnnotationKey] = strings.Join(containedResources, ",")
	if err := c.Client.Update(ctx, edgeApp); err != nil {
		return fmt.Errorf("failed to update edgeApp, %v", err)
	}

	return nil
}

func setOwnerReference(obj *unstructured.Unstructured, edgeApp *groupingv1alpha1.EdgeApplication) {
	toAdd := metav1.OwnerReference{
		APIVersion:         edgeApp.APIVersion,
		BlockOwnerDeletion: pointer.BoolPtr(true),
		Controller:         pointer.BoolPtr(true),
		Kind:               edgeApp.Kind,
		Name:               edgeApp.Name,
		UID:                edgeApp.UID,
	}
	ownerReferences := obj.GetOwnerReferences()
	if ownerReferences == nil {
		ownerReferences = []metav1.OwnerReference{toAdd}
		obj.SetOwnerReferences(ownerReferences)
		return
	}

	// check if the OwnerReference has already existed
	for i := range ownerReferences {
		ownerReference := &ownerReferences[i]
		if ownerReference.APIVersion == edgeApp.APIVersion &&
			*ownerReference.Controller &&
			ownerReference.Kind == edgeApp.Kind {
			// one obj can only have one edgeApp as its owner
			// so we overwrite this entry.
			ownerReference.Name = edgeApp.Name
			ownerReference.UID = edgeApp.UID
			obj.SetOwnerReferences(ownerReferences)
			return
		}
	}

	// add a new entry to its OwnerReferences
	ownerReferences = append(ownerReferences, toAdd)
	obj.SetOwnerReferences(ownerReferences)
}

// isSameAsLastApplied will check if the curObj has the same specified fileds as objInEdgeApp.
// It assumes that fields of the obj in cluster are same as the value of last-applied-template annotation.
func isSameAsLastApplied(objInEdgeApp *unstructured.Unstructured, curObj runtime.Object) (bool, error) {
	accessor := meta.NewAccessor()
	annots, err := accessor.Annotations(curObj)
	if err != nil {
		return false, fmt.Errorf("failed to get annotations of object, %v", err)
	}

	objJSON, err := objInEdgeApp.MarshalJSON()
	if err != nil {
		return false, fmt.Errorf("failed to marshal json of obj %s/%s, gvk: %s, %v",
			objInEdgeApp.GetNamespace(), objInEdgeApp.GetName(), objInEdgeApp.GroupVersionKind(), err)
	}

	if lastApplied, ok := annots[LastAppliedTemplateAnnotationKey]; ok {
		if string(objJSON) == lastApplied {
			return true, nil
		}
		return false, nil
	}

	return false, fmt.Errorf("cannot find last applied template in annotation, %v, possibly it is not created by EdgeApplication Controller", err)
}

func getAllOverriders(edgeApp *groupingv1alpha1.EdgeApplication) []overridemanager.OverriderInfo {
	infos := make([]overridemanager.OverriderInfo, 0, len(edgeApp.Spec.WorkloadScope.TargetNodeGroups))
	for index := range edgeApp.Spec.WorkloadScope.TargetNodeGroups {
		copied := edgeApp.Spec.WorkloadScope.TargetNodeGroups[index].Overriders.DeepCopy()
		infos = append(infos, overridemanager.OverriderInfo{
			TargetNodeGroup: edgeApp.Spec.WorkloadScope.TargetNodeGroups[index].Name,
			Overriders:      copied,
		})
	}
	return infos
}

// needOverride determines if a obj needs override, according to its gvk.
func needOverride(obj runtime.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()
	_, ok := overriderTargetGVK[gvk]
	return ok
}

// getDeletedResources will return a slice of all deleted resourceInfo, which
// are in oldInfos but not in newInfos.
func getDeletedResources(oldInfos, newInfos []statusmanager.ResourceInfo) []statusmanager.ResourceInfo {
	deleted := []statusmanager.ResourceInfo{}
	newInfoStrs := make(map[string]struct{})
	for _, info := range newInfos {
		newInfoStrs[info.String()] = struct{}{}
	}
	for _, info := range oldInfos {
		if _, ok := newInfoStrs[info.String()]; !ok {
			deleted = append(deleted, info)
		}
	}
	return deleted
}
