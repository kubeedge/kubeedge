package application

import (
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type SelectorListener struct {
	id       string
	nodeName string
	gvr      schema.GroupVersionResource
	// e.g. labels and fields(metadata.namespace metadata.name spec.nodename)
	selector LabelFieldSelector
}

func NewSelectorListener(nodeName string, gvr schema.GroupVersionResource, selector LabelFieldSelector) *SelectorListener {
	return &SelectorListener{id: uuid.New().String(), nodeName: nodeName, gvr: gvr, selector: selector}
}

func (l *SelectorListener) sendAllObjects(rets []runtime.Object, messageLayer messagelayer.MessageLayer) {
	for _, ret := range rets {
		event := watch.Event{
			Type:   watch.Added,
			Object: ret,
		}
		l.sendObj(event, messageLayer)
	}
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

	namespace := accessor.GetNamespace()
	if namespace == "" {
		namespace = v2.NullNamespace
	}
	kind := util.UnsafeResourceToKind(l.gvr.Resource)
	resourceType := strings.ToLower(kind)
	resource, err := messagelayer.BuildResource(l.nodeName, namespace, resourceType, accessor.GetName())
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}

	var operation string
	switch event.Type {
	case watch.Added:
		operation = model.InsertOperation
	case watch.Modified:
		operation = model.UpdateOperation
	case watch.Deleted:
		operation = model.DeleteOperation
	default:
		klog.Warningf("event type: %s unsupported", event.Type)
		return
	}

	msg := model.NewMessage("").
		SetResourceVersion(accessor.GetResourceVersion()).
		BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, operation).
		FillBody(event.Object)

	if err := messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}
