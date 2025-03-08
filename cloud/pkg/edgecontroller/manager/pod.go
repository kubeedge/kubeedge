package manager

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

// PodManager is a manager watch pod change event
type PodManager struct {
	// events from watch kubernetes api server
	realEvents chan watch.Event

	// events merged
	mergedEvents chan watch.Event
}

func isPodUpdated(old v1.Pod, new v1.Pod) bool {
	// does not care fields
	old.ObjectMeta.ResourceVersion = new.ObjectMeta.ResourceVersion
	old.ObjectMeta.Generation = new.ObjectMeta.Generation

	// return true if ObjectMeta or Spec changed, else false
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Spec, new.Spec)
}

func (pm *PodManager) merge() {
	for re := range pm.realEvents {
		pod := re.Object.(*v1.Pod)
		switch re.Type {
		case watch.Added:
			if pod.DeletionTimestamp == nil {
				pm.mergedEvents <- re
			} else {
				re.Type = watch.Modified
				pm.mergedEvents <- re
			}
		case watch.Deleted:
			pm.mergedEvents <- re
		case watch.Modified:
			pm.mergedEvents <- re
		default:
			klog.Warningf("event type: %s unsupported", re.Type)
		}
	}
}

// Events return a channel, can receive all pod event
func (pm *PodManager) Events() chan watch.Event {
	return pm.mergedEvents
}

var _ EventFilter = &podEventFilter{}

type podEventFilter struct{}

func (*podEventFilter) Create(_ interface{}) bool {
	return true
}

func (*podEventFilter) Delete(_ interface{}) bool {
	return true
}

func (*podEventFilter) Update(oldObj, newObj interface{}) bool {
	curPod := newObj.(*v1.Pod)
	oldPod := oldObj.(*v1.Pod)

	return isPodUpdated(*oldPod, *curPod)
}

// NewPodManager create PodManager from config
func NewPodManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*PodManager, error) {
	realEvents := make(chan watch.Event, config.Buffer.PodEvent)
	mergedEvents := make(chan watch.Event, config.Buffer.PodEvent)
	rh := NewCommonResourceEventHandler(realEvents, &podEventFilter{})
	if _, err := si.AddEventHandler(rh); err != nil {
		return nil, err
	}
	pm := &PodManager{realEvents: realEvents, mergedEvents: mergedEvents}
	go pm.merge()
	return pm, nil
}
