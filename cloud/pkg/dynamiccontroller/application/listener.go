package application

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type SelectorListener struct {
	ID       string
	nodeName string
	gvr      schema.GroupVersionResource
	// e.g. labels and fields(metadata.namespace metadata.name spec.nodename)
	selector LabelFieldSelector

	input chan watch.Event

	messageLayer messagelayer.MessageLayer
}

func NewSelectorListener(ID, nodeName string, gvr schema.GroupVersionResource, selector LabelFieldSelector) *SelectorListener {
	listener := &SelectorListener{
		ID:           ID,
		nodeName:     nodeName,
		gvr:          gvr,
		selector:     selector,
		messageLayer: messagelayer.DynamicControllerMessageLayer(),
	}

	go listener.process()

	return listener
}

func (l *SelectorListener) add(event watch.Event) {
	l.input <- event
}

func (l *SelectorListener) process() {
	for {
		select {
		case event, ok := <-l.input:
			if !ok {
				return
			}

			l.sendWatchEvent(event)

		case <-beehivecontext.Done():
			return
		}
	}
}

func (l *SelectorListener) sendAllObjects(rets []runtime.Object) {
	for _, ret := range rets {
		event := watch.Event{
			Type:   watch.Added,
			Object: ret,
		}
		l.sendWatchEvent(event)
	}
}

func (l *SelectorListener) sendWatchEvent(event watch.Event) {
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
	filter.MessageFilter(filterEvent.Object, l.nodeName)

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
		FillBody(filterEvent.Object)

	if err := l.messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}
