package application

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	edgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type SelectorListener struct {
	id       string
	nodeName string
	gvr      schema.GroupVersionResource
	// e.g. labels and fields(metadata.namespace metadata.name spec.nodename)
	selector LabelFieldSelector
}

func NewSelectorListener(ID, nodeName string, gvr schema.GroupVersionResource, selector LabelFieldSelector) *SelectorListener {
	return &SelectorListener{id: ID, nodeName: nodeName, gvr: gvr, selector: selector}
}

func (l *SelectorListener) sendAllObjects(rets []runtime.Object, handler *CommonResourceEventHandler) {
	for _, ret := range rets {
		event := watch.Event{
			Type:   watch.Added,
			Object: ret,
		}
		l.sendObj(event, handler.messageLayer)
	}
}

// groupVersionKind returns the GVK of obj from the schemes of CloudCore's typed
// informers (client-go for built-in types, KubeEdge for its CRDs), like
// util.SetMetaType does, but without modifying obj. It falls back to the
// listener's GVR, e.g. for objects of dynamic informers.
func (l *SelectorListener) groupVersionKind(obj runtime.Object) schema.GroupVersionKind {
	for _, s := range []*runtime.Scheme{scheme.Scheme, edgescheme.Scheme} {
		if kinds, _, err := s.ObjectKinds(obj); err == nil && len(kinds) > 0 {
			return kinds[0]
		}
	}
	return l.gvr.GroupVersion().WithKind(util.UnsafeResourceToKind(l.gvr.Resource))
}

func (l *SelectorListener) sendObj(event watch.Event, messageLayer messagelayer.MessageLayer) {
	accessor, err := meta.Accessor(event.Object)
	if err != nil {
		klog.Error(err)
		return
	}
	klog.V(4).Infof("[dynamiccontroller/selectorListener] listener(%v) is sending obj %v", *l, accessor.GetName())
	// do not send obj if obj does not match listener's selector
	if !l.selector.MatchObj(event.Object) {
		return
	}
	// filter message
	filterEvent := *(event.DeepCopy())
	content, err := convertToUnstructured(filterEvent.Object)
	if err != nil {
		klog.Errorf("convertToUnstructured error %v", err)
		return
	}
	filter.MessageFilter(content, l.nodeName)

	// Objects listed from the informer cache (sendAllObjects) can still have
	// an empty TypeMeta (the event handler sets it asynchronously, on the
	// shared object), and the edge cannot decode an object without kind into
	// unstructured.
	if content.GetAPIVersion() == "" || content.GetKind() == "" {
		gvk := l.groupVersionKind(event.Object)
		if content.GetAPIVersion() == "" {
			content.SetAPIVersion(gvk.GroupVersion().String())
		}
		if content.GetKind() == "" {
			content.SetKind(gvk.Kind)
		}
	}

	namespace := accessor.GetNamespace()
	if namespace == "" {
		namespace = models.NullNamespace
	}
	resourceType := strings.ToLower(util.UnsafeResourceToKind(l.gvr.Resource))
	resource, err := messagelayer.BuildResource(l.nodeName, namespace, resourceType, accessor.GetName())
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}

	var operation string
	switch filterEvent.Type {
	case watch.Added:
		operation = model.InsertOperation
	case watch.Modified:
		operation = model.UpdateOperation
	case watch.Deleted:
		operation = model.DeleteOperation
	default:
		klog.Warningf("event type: %s unsupported", filterEvent.Type)
		return
	}

	msg := model.NewMessage("").
		SetResourceVersion(accessor.GetResourceVersion()).
		BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, operation).
		FillBody(content.DeepCopyObject())

	if err := messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}

func convertToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstructuredObj}, nil
}
