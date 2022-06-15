package edgeapplication

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/statusmanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

// Controller is to sync EdgeApplication.
type Controller struct {
	client.Client
	runtime.Serializer
	overridemanager.Overrider
	statusmanager.StatusManager
	UseServerSideApply   bool
	ReconcileTriggerChan chan event.GenericEvent
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.Infof("Reconciling EdgeApplication %s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name)

	edgeApp := &appsv1alpha1.EdgeApplication{}
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
	c.ReconcileTriggerChan = make(chan event.GenericEvent)
	c.StatusManager.SetReconcileTriggerChan(c.ReconcileTriggerChan)
	// start the StatusManager
	if err := c.StatusManager.Start(); err != nil {
		return fmt.Errorf("fail to start StatusManager, %v", err)
	}
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.EdgeApplication{}).
		Watches(&source.Channel{Source: c.ReconcileTriggerChan}, &handler.EnqueueRequestForObject{}).
		Complete(c)
}

func (c *Controller) syncEdgeApplication(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication) (controllerruntime.Result, error) {
	// 1. get manifests, set ownerReference and apply overrides to all target resources
	// It will traverse all templates in EdgeApplication. If error occurs during traverse,
	// it will log the error and continue.
	modifiedTmplInfos := []*utils.TemplateInfo{}
	errs := []error{}
	overriderInfos := utils.GetAllOverriders(edgeApp)
	tmplInfos, err := utils.GetTemplatesInfosOfEdgeApp(edgeApp, c.Serializer)
	if err != nil {
		klog.Errorf("failed to get all templates from edgeapp %s/%s, %v, continue with what got", edgeApp.Namespace, edgeApp.Name, err)
		errs = append(errs, err)
	}
	for _, tmplInfo := range tmplInfos {
		tmpl := tmplInfo.Template
		setOwnerReference(tmpl, edgeApp)
		if tmpl.GroupVersionKind() == constants.ServiceGVK {
			addRangeNodeGroupAnnotation(tmpl)
			modifiedTmplInfos = append(modifiedTmplInfos, tmplInfo)
			continue
		}

		if !needOverride(tmpl) {
			klog.V(4).Infof("obj %s/%s of gvk %s does not need override, skip override",
				tmpl.GetNamespace(), tmpl.GetName(), tmpl.GroupVersionKind())
			modifiedTmplInfos = append(modifiedTmplInfos, tmplInfo)
			continue
		}

		// apply overriders
		//
		// TODO: consider the situation that not all the overrides have been applied successfully
		// If one succeeded and another failed, the status of edgeApp will only contain the successful
		// one, and have no status about the failed one.
		for _, info := range overriderInfos {
			copy := tmpl.DeepCopy()
			klog.V(4).Infof("override obj %s/%s of gvk %s", copy.GetNamespace(), copy.GetName(), copy.GroupVersionKind())
			if err := c.Overrider.ApplyOverrides(copy, info); err != nil {
				klog.Errorf("failed to apply override of nodegroup %s to obj %s/%s of gvk %s, %v",
					info.TargetNodeGroup, copy.GetNamespace(), copy.GetName(), copy.GroupVersionKind(), err)
				errs = append(errs, err)
				continue
			}
			modifiedTmplInfos = append(modifiedTmplInfos, &utils.TemplateInfo{Ordinal: tmplInfo.Ordinal, Template: copy})
		}
	}

	// 2. remove status that do not need
	if err := c.updateStatus(ctx, edgeApp, modifiedTmplInfos); err != nil {
		klog.Errorf("failed to update status for EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
		errs = append(errs, err)
	}

	// 3. apply all templates
	// It will create/update the resource in the template and notify the status manager
	// to monitor its status.
	for _, tmplInfo := range modifiedTmplInfos {
		tmpl := tmplInfo.Template
		if err := c.applyTemplate(ctx, tmpl); err != nil {
			klog.Errorf("failed to apply overridden template of EdgeApplication %s/%s, %v, template: %v", edgeApp.Namespace, edgeApp.Name, err, tmpl)
			errs = append(errs, err)
			continue
		}
		klog.V(4).Infof("successfully applied overridden template of EdgeApplication %s/%s, template: %v", edgeApp.Namespace, edgeApp.Name, tmpl)
	}

	// 4. delete resources that have been removed from the manifests
	if err := c.deleteRedundantResources(ctx, edgeApp, modifiedTmplInfos); err != nil {
		klog.Errorf("failed to delete redundant resource for EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
		errs = append(errs, err)
	}

	// 5. update the LastContainedResourcesAnnotation
	if err := c.addOrUpdateLastContainedResourcesAnnotation(ctx, edgeApp, modifiedTmplInfos); err != nil {
		klog.Errorf("failed to update annotation of EdgeApplication %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
		errs = append(errs, err)
	}

	return controllerruntime.Result{}, errors.NewAggregate(errs)
}

func (c *Controller) deleteRedundantResources(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication, currentTmplInfos []*utils.TemplateInfo) error {
	lastContainedResourcesInfos, err := c.getLastContainedResourceInfos(edgeApp)
	if err != nil {
		klog.Errorf("failed to get infos of last contained resources in EdgeApplication %s/%s, %v",
			edgeApp.Namespace, edgeApp.Name, err)
		return err
	}
	currentContainedResourceInfos := make([]utils.ResourceInfo, len(currentTmplInfos))
	for i := range currentTmplInfos {
		currentContainedResourceInfos[i] = utils.GetResourceInfoOfTemplateInfo(currentTmplInfos[i])
	}
	deleted := getDeletedResources(lastContainedResourcesInfos, currentContainedResourceInfos)
	for _, info := range deleted {
		if err := c.removeResource(ctx, info); err != nil {
			klog.Errorf("failed to remove resource %s/%s of gvk %s, %v",
				info.Namespace, info.Name, schema.GroupVersionKind{Group: info.Group, Version: info.Version, Kind: info.Kind}, err)
			return err
		}
		klog.V(4).Infof("successfully remove resource %s/%s of gvk %s, %v",
			info.Namespace, info.Name, schema.GroupVersionKind{Group: info.Group, Version: info.Version, Kind: info.Kind}, err)
	}
	return nil
}

func (c *Controller) updateStatus(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication, tmplInfos []*utils.TemplateInfo) error {
	newStatus := []appsv1alpha1.ManifestStatus{}
	tmplMap := map[int][]*utils.TemplateInfo{}
	for _, tmplInfo := range tmplInfos {
		ordinal := tmplInfo.Ordinal
		if _, ok := tmplMap[ordinal]; !ok {
			tmplMap[ordinal] = []*utils.TemplateInfo{tmplInfo}
		} else {
			tmplMap[ordinal] = append(tmplMap[ordinal], tmplInfo)
		}
	}

	// remove redundant status entries, these status do not have any corresponding
	// template in this edgeapplication
	statusExists := map[string]struct{}{}
	for _, status := range edgeApp.Status.WorkloadStatus {
		id := status.Identifier
		for _, tmplInfo := range tmplMap[id.Ordinal] {
			resourceInfo := utils.GetResourceInfoOfTemplateInfo(tmplInfo)
			if utils.IsIdentifierSameAsResourceInfo(id, resourceInfo) {
				// this status still need to retain
				statusExists[resourceInfo.String()] = struct{}{}
				newStatus = append(newStatus, status)
			}
		}
	}

	// add missed status entry to ensure each passed-in template have its corresponding status
	for _, tmplInfo := range tmplInfos {
		resourceInfo := utils.GetResourceInfoOfTemplateInfo(tmplInfo)
		if _, ok := statusExists[resourceInfo.String()]; !ok {
			// this tmpl does not have relate status entry, add a new entry for it
			newStatus = append(newStatus, appsv1alpha1.ManifestStatus{
				Condition: appsv1alpha1.EdgeAppProcessing,
				Identifier: appsv1alpha1.ResourceIdentifier{
					Ordinal:   resourceInfo.Ordinal,
					Group:     resourceInfo.Group,
					Version:   resourceInfo.Kind,
					Kind:      resourceInfo.Kind,
					Namespace: resourceInfo.Namespace,
					Name:      resourceInfo.Name,
				},
			})
		}
	}

	// ensure each template have its corresponding status
	// Because of error, some entries in edgeApp.Spec.WorkloadTemplate.Manifests cannot
	// be parsed as an template object or cannot applied override to it. These entries should
	// also have its status, though they are not elements of passed-in argument tmplInfos.
	for ordinal := 0; ordinal < len(edgeApp.Spec.WorkloadTemplate.Manifests); ordinal++ {
		find := false
		for _, status := range newStatus {
			if status.Identifier.Ordinal == ordinal {
				find = true
				break
			}
		}
		if !find {
			newStatus = append(newStatus, appsv1alpha1.ManifestStatus{
				Condition: appsv1alpha1.EdgeAppProcessing,
				Identifier: appsv1alpha1.ResourceIdentifier{
					Ordinal: ordinal,
				},
			})
		}
	}

	sort.Slice(newStatus, func(i, j int) bool {
		if newStatus[i].Identifier.Ordinal != newStatus[j].Identifier.Ordinal {
			return newStatus[i].Identifier.Ordinal < newStatus[j].Identifier.Ordinal
		}
		return newStatus[i].Identifier.Name < newStatus[j].Identifier.Name
	})

	if equality.Semantic.DeepEqual(newStatus, edgeApp.Status.WorkloadStatus) {
		klog.V(4).Infof("newStatus is same as the current status in edgeApp %s/%s, skip update status",
			edgeApp.Namespace, edgeApp.Name)
		return nil
	}

	edgeApp.Status.WorkloadStatus = newStatus
	return c.Client.Status().Update(ctx, edgeApp)
}

func (c *Controller) ifObjExists(ctx context.Context, obj *unstructured.Unstructured) (bool, *unstructured.Unstructured, error) {
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

func (c *Controller) updateTemplate(ctx context.Context, tmpl *unstructured.Unstructured, curObj *unstructured.Unstructured) error {
	if _, ok := curObj.GetAnnotations()[constants.LastAppliedTemplateAnnotationKey]; !ok {
		klog.Warningf("cannot find LastAppliedTemplateAnnotation on obj %s/%s of gvk %s, update it with new template",
			curObj.GetNamespace(), curObj.GetName(), curObj.GroupVersionKind())
		if err := c.Client.Update(ctx, tmpl); err != nil {
			return fmt.Errorf("failed to update object with template %s, %v", tmpl, err)
		}
		return nil
	}

	same, err := isSameAsLastApplied(tmpl, curObj)
	if err != nil {
		// error occurs when comparing the overridden template with the last applied template
		return err
	} else if err == nil && same {
		// nothing to do for this template
		return nil
	}

	// The existing object has different last applied template than what is specified in the EdgeApplication.
	// Update the object with the template in EdgeApplication, and update its LastAppliedTemplateAnnotation.
	if err := addOrUpdateLastAppliedTemplateAnnotation(tmpl); err != nil {
		return fmt.Errorf("failed to add LastAppliedTemplateAnnotation to obj %s/%s of gvk %s, %v",
			tmpl.GetNamespace(), tmpl.GetName(), tmpl.GroupVersionKind(), err)
	}
	if err := c.update(ctx, tmpl, curObj); err != nil {
		return fmt.Errorf("failed to update object %s/%s of gvk %s, %v",
			curObj.GetNamespace(), curObj.GetName(), curObj.GroupVersionKind(), err)
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
		if err := c.updateTemplate(ctx, tmpl, curObj); err != nil {
			klog.Errorf("failed to update the object %s/%s, gvk: %s, %v", ns, name, gvk, err)
			return err
		}
	} else {
		klog.V(4).Infof("try to create object %s/%s of gvk %s with template: %v", ns, name, gvk, tmpl)
		if err := addOrUpdateLastAppliedTemplateAnnotation(tmpl); err != nil {
			return fmt.Errorf("failed to add LastAppliedTemplateAnnotation to obj %s/%s of gvk %s, %v",
				tmpl.GetNamespace(), tmpl.GetName(), tmpl.GroupVersionKind(), err)
		}
		if err := c.Client.Create(ctx, tmpl); err != nil {
			klog.Errorf("failed to create the object %s/%s of gvk %s with template: %v, %v", ns, name, gvk, tmpl, err)
			return err
		}
	}
	// notify the StatusManager to watch its status.
	return c.StatusManager.WatchStatus(utils.ResourceInfo{
		Group:     gvk.Group,
		Version:   gvk.Version,
		Kind:      gvk.Kind,
		Namespace: ns,
		Name:      name,
	})
}

func (c *Controller) removeResource(ctx context.Context, info utils.ResourceInfo) error {
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

func (c *Controller) getLastContainedResourceInfos(edgeApp *appsv1alpha1.EdgeApplication) ([]utils.ResourceInfo, error) {
	anno := edgeApp.Annotations
	if anno == nil || anno[constants.LastContainedResourcesAnnotationKey] == "" {
		klog.Infof("cannot get last contained resources of EdgeApplication %s/%s for annotation not existing, possibly it is a new-created edgeapp",
			edgeApp.Namespace, edgeApp.Name)
		return []utils.ResourceInfo{}, nil
	}

	infos := []utils.ResourceInfo{}
	annoValue := anno[constants.LastContainedResourcesAnnotationKey]
	if err := json.Unmarshal([]byte(annoValue), &infos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LastContainedResourcesAnnotation on edgeapp %s/%s", edgeApp.Namespace, edgeApp.Name)
	}
	return infos, nil
}

// addOrUpdateLastContainedResourcesAnnotation will add the ContainedResourcesAnnotation to the EdgeApplication,
// if the annotation has already existed, it will be updated it according to resources in manifests.
func (c *Controller) addOrUpdateLastContainedResourcesAnnotation(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication, tmplInfos []*utils.TemplateInfo) error {
	if edgeApp.Annotations == nil {
		edgeApp.Annotations = make(map[string]string)
	}

	resourceInfos := make([]*utils.ResourceInfo, len(tmplInfos))
	for i := range tmplInfos {
		info := utils.GetResourceInfoOfTemplateInfo(tmplInfos[i])
		resourceInfos[i] = &info
	}
	sort.Slice(resourceInfos, func(i, j int) bool { return resourceInfos[i].String() < resourceInfos[j].String() })
	infosJSON, err := json.Marshal(resourceInfos)
	if err != nil {
		return fmt.Errorf("failed to marshal infos %v, %v", resourceInfos, err)
	}

	oldAnno := edgeApp.Annotations[constants.LastContainedResourcesAnnotationKey]
	if oldAnno == string(infosJSON) {
		klog.V(4).Infof("skip update last-applied-resources annotation of edgeapp %s/%s for same value", edgeApp.Namespace, edgeApp.Name)
		return nil
	}
	edgeApp.Annotations[constants.LastContainedResourcesAnnotationKey] = string(infosJSON)
	return c.Client.Update(ctx, edgeApp)
}

func (c *Controller) update(ctx context.Context, tmpl *unstructured.Unstructured, curObj *unstructured.Unstructured) error {
	if c.UseServerSideApply {
		if err := c.Client.Update(ctx, tmpl); err != nil {
			return fmt.Errorf("failed to update object with template %s, %v", tmpl, err)
		}
		return nil
	}

	// use client-side apply
	var oldJSON, newJSON, curObjectJSON, newObjectJSON []byte
	var err error
	anno, ok := curObj.GetAnnotations()[constants.LastAppliedTemplateAnnotationKey]
	if !ok {
		return fmt.Errorf("cannot find last-applied-template annotation on obj %s/%s of gvk %s",
			curObj.GetNamespace(), curObj.GetName(), curObj.GroupVersionKind())
	}
	oldJSON = []byte(anno)
	if newJSON, err = tmpl.MarshalJSON(); err != nil {
		return fmt.Errorf("failed to serialize template as json %v, %s", tmpl, err)
	}
	mergePatch, err := jsonpatch.CreateMergePatch(oldJSON, newJSON)
	if err != nil {
		return fmt.Errorf("cannot get merge patch for error %v, old json: %s, new json: %s", err, oldJSON, newJSON)
	}

	if curObjectJSON, err = curObj.MarshalJSON(); err != nil {
		return fmt.Errorf("failed to serialize current obj as json, which is %s/%s of gvk %s, err: %v",
			curObj.GetNamespace(), curObj.GetName(), curObj.GroupVersionKind(), err)
	}
	if newObjectJSON, err = jsonpatch.MergePatch(curObjectJSON, mergePatch); err != nil {
		return fmt.Errorf("failed to apply json merge patch to current obj %s/%s of gvk %s, merge patch: %s, err: %v",
			curObj.GetNamespace(), curObj.GetName(), curObj.GroupVersionKind(), string(mergePatch), err)
	}
	newObj := &unstructured.Unstructured{}
	if _, _, err = c.Serializer.Decode(newObjectJSON, nil, newObj); err != nil {
		return fmt.Errorf("failed to decode json of new object as new object for error: %v, json: %s", err, string(newObjectJSON))
	}

	if err := c.Client.Patch(ctx, newObj, client.MergeFrom(curObj)); err != nil {
		return fmt.Errorf("failed to update obj as %v, %v", newObj, err)
	}
	return nil
}

func addOrUpdateLastAppliedTemplateAnnotation(obj *unstructured.Unstructured) error {
	objJSON, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to unmarshal obj %s/%s of gvk %s, %v",
			obj.GetNamespace(), obj.GetName(), obj.GroupVersionKind(), err)
	}
	if obj.GetAnnotations() == nil {
		annotations := map[string]string{
			constants.LastAppliedTemplateAnnotationKey: string(objJSON),
		}
		obj.SetAnnotations(annotations)
		return nil
	}
	annotations := obj.GetAnnotations()
	annotations[constants.LastAppliedTemplateAnnotationKey] = string(objJSON)
	obj.SetAnnotations(annotations)
	return nil
}

func setOwnerReference(obj *unstructured.Unstructured, edgeApp *appsv1alpha1.EdgeApplication) {
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

	if lastApplied, ok := annots[constants.LastAppliedTemplateAnnotationKey]; ok {
		if string(objJSON) == lastApplied {
			return true, nil
		}
		return false, nil
	}

	return false, fmt.Errorf("cannot find last applied template in annotation, %v, possibly it is not created by EdgeApplication Controller", err)
}

// needOverride determines if a obj needs override, according to its gvk.
func needOverride(obj runtime.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()
	_, ok := constants.OverriderTargetGVK[gvk]
	return ok
}

// getDeletedResources will return a slice of all deleted resourceInfo, which
// are in oldInfos but not in newInfos.
func getDeletedResources(oldInfos, newInfos []utils.ResourceInfo) []utils.ResourceInfo {
	deleted := []utils.ResourceInfo{}
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

func addRangeNodeGroupAnnotation(obj *unstructured.Unstructured) {
	anno := obj.GetAnnotations()
	if anno == nil {
		obj.SetAnnotations(
			map[string]string{nodegroup.ServiceTopologyAnnotation: nodegroup.ServiceTopologyRangeNodegroup},
		)
		return
	}
	anno[nodegroup.ServiceTopologyAnnotation] = nodegroup.ServiceTopologyRangeNodegroup
	obj.SetAnnotations(anno)
}
