package application

import (
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
)

type SelectorListener struct {
	id       string
	nodeName string
	gvr      schema.GroupVersionResource
	// e.g. lables and fields(metadata.namespace metadata.name spec.nodename)
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
	accessor, _ := meta.Accessor(event.Object)
	klog.V(4).Infof("[dynamiccontroller/selectorListener] listener(%v) is sending obj %v", *l, accessor.GetName())
	// do not send obj if obj does not match listener's selector
	if !l.selector.MatchObj(event.Object) {
		return
	}

	msg := model.NewMessage("")
	accessor, err := meta.Accessor(event.Object)
	if err != nil {
		klog.Error(err)
		return
	}

	msg.SetResourceVersion(accessor.GetResourceVersion())
	namespace := accessor.GetNamespace()
	if namespace == "" {
		namespace = v2.NullNamespace
	}
	resource, err := messagelayer.BuildResource(l.nodeName, namespace, l.gvr.Resource, accessor.GetName())
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}
	msg.Content = event.Object
	switch event.Type {
	case watch.Added:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
	case watch.Modified:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
	case watch.Deleted:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
	default:
		klog.Warningf("event type: %s unsupported", event.Type)
	}
	if err := messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}
