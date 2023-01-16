package manager

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// Manager define the interface of a Manager, configmapManager and podManager implement it
type Manager interface {
	Events() chan watch.Event
}

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events      chan watch.Event
	eventFilter EventFilter
}

func (c *CommonResourceEventHandler) obj2Event(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", obj)
		return
	}
	// All obj from client has been removed the information of apiversion/kind called MetaType,
	// it is fatal to decode the obj as unstructured.Unstructure or unstructured.UnstructureList at edge.
	err := util.SetMetaType(eventObj)
	if err != nil {
		klog.Warningf("failed to set meta type :%v", err)
	}

	c.events <- watch.Event{Type: t, Object: eventObj}
}

// OnAdd handle Add event
func (c *CommonResourceEventHandler) OnAdd(obj interface{}) {
	if c.eventFilter != nil && !c.eventFilter.Create(obj) {
		return
	}
	c.obj2Event(watch.Added, obj)
}

// OnUpdate handle Update event
func (c *CommonResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if c.eventFilter != nil && !c.eventFilter.Update(oldObj, newObj) {
		return
	}
	c.obj2Event(watch.Modified, newObj)
}

// OnDelete handle Delete event
func (c *CommonResourceEventHandler) OnDelete(obj interface{}) {
	if c.eventFilter != nil && !c.eventFilter.Delete(obj) {
		return
	}
	c.obj2Event(watch.Deleted, obj)
}

// NewCommonResourceEventHandler create CommonResourceEventHandler used by configmapManager and podManager
func NewCommonResourceEventHandler(events chan watch.Event, filter EventFilter) *CommonResourceEventHandler {
	return &CommonResourceEventHandler{events: events, eventFilter: filter}
}
