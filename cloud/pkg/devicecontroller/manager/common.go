package manager

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

// Manager define the interface of a Manager, configmapManager and podManager implement it
type Manager interface {
	Events() chan EventWithOldObject
}

// Filter filter the event before enqueuing the key
type EventFilter interface {
	// Create returns true if the Create event should be processed
	Create(obj interface{}) bool

	// Delete returns true if the Delete event should be processed
	Delete(obj interface{}) bool

	// Update returns true if the Update event should be processed
	Update(oldObj, newObj interface{}) bool
}

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events      chan EventWithOldObject
	eventFilter EventFilter
}

// EventWithOldObject wraps a watch.Event with old runtime.Object
// OldObject is the last known state of the object
// This is due to deviceController need to compare oldObj with newObj when processing device update
// Maybe we could refactor this.
type EventWithOldObject struct {
	Event     watch.Event
	OldObject interface{}
}

func (c *CommonResourceEventHandler) obj2Event(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", obj)
		return
	}
	c.events <- EventWithOldObject{
		Event: watch.Event{Type: t, Object: eventObj},
	}
}

func (c *CommonResourceEventHandler) obj2EventForUpdate(t watch.EventType, oldObj, newObj interface{}) {
	oldValue, ok := oldObj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", oldObj)
		return
	}
	newValue, ok := newObj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", newObj)
		return
	}
	c.events <- EventWithOldObject{
		Event: watch.Event{
			Type:   t,
			Object: newValue,
		},
		OldObject: oldValue,
	}
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
	c.obj2EventForUpdate(watch.Modified, oldObj, newObj)
}

// OnDelete handle Delete event
func (c *CommonResourceEventHandler) OnDelete(obj interface{}) {
	if c.eventFilter != nil && !c.eventFilter.Delete(obj) {
		return
	}
	c.obj2Event(watch.Deleted, obj)
}

// NewCommonResourceEventHandler create CommonResourceEventHandler used by deviceManager and deviceModelManager
func NewCommonResourceEventHandler(events chan EventWithOldObject, eventFilter EventFilter) *CommonResourceEventHandler {
	return &CommonResourceEventHandler{
		events:      events,
		eventFilter: eventFilter,
	}
}
