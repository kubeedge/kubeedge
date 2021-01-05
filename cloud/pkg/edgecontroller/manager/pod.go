package manager

import (
	"reflect"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// CachePod is the struct save pod data for check pod is really changed
type CachePod struct {
	metav1.ObjectMeta
	Spec v1.PodSpec
}

// PodManager is a manager watch pod change event
type PodManager struct {
	// events from watch kubernetes api server
	realEvents chan watch.Event

	// events merged
	mergedEvents chan watch.Event

	// pods, key is UID, value is *v1.Pod
	pods sync.Map
}

func (pm *PodManager) isPodUpdated(old *CachePod, new *v1.Pod) bool {
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
			pm.pods.Store(pod.UID, &CachePod{ObjectMeta: pod.ObjectMeta, Spec: pod.Spec})
			if pod.DeletionTimestamp == nil {
				pm.mergedEvents <- re
			} else {
				re.Type = watch.Modified
				pm.mergedEvents <- re
			}
		case watch.Deleted:
			pm.pods.Delete(pod.UID)
			pm.mergedEvents <- re
		case watch.Modified:
			value, ok := pm.pods.Load(pod.UID)
			pm.pods.Store(pod.UID, &CachePod{ObjectMeta: pod.ObjectMeta, Spec: pod.Spec})
			if ok {
				cachedPod := value.(*CachePod)
				if pm.isPodUpdated(cachedPod, pod) {
					pm.mergedEvents <- re
				}
			} else {
				pm.mergedEvents <- re
			}
		default:
			klog.Warningf("event type: %s unsupported", re.Type)
		}
	}
}

// Events return a channel, can receive all pod event
func (pm *PodManager) Events() chan watch.Event {
	return pm.mergedEvents
}

// NewPodManager create PodManager from config
func NewPodManager(si cache.SharedIndexInformer) (*PodManager, error) {
	realEvents := make(chan watch.Event, config.Config.Buffer.PodEvent)
	mergedEvents := make(chan watch.Event, config.Config.Buffer.PodEvent)
	rh := NewCommonResourceEventHandler(realEvents)
	si.AddEventHandler(rh)
	pm := &PodManager{realEvents: realEvents, mergedEvents: mergedEvents}
	go pm.merge()
	return pm, nil
}
