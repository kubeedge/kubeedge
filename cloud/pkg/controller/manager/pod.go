package manager

import (
	"reflect"
	"sync"

	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"

	"github.com/kubeedge/beehive/pkg/common/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
			log.LOGGER.Warnf("event type: %s unsupported", re.Type)
		}
	}
}

// Events return a channel, can receive all pod event
func (pm *PodManager) Events() chan watch.Event {
	return pm.mergedEvents
}

// NewPodManager create PodManager from config
func NewPodManager(kubeClient *kubernetes.Clientset, namespace, nodeName string) (*PodManager, error) {
	var lw *cache.ListWatch
	if "" == nodeName {
		lw = cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", namespace, fields.Everything())
	} else {
		selector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
		lw = cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", namespace, selector)
	}
	realEvents := make(chan watch.Event, config.PodEventBuffer)
	mergedEvents := make(chan watch.Event, config.PodEventBuffer)
	rh := NewCommonResourceEventHandler(realEvents)
	si := cache.NewSharedInformer(lw, &v1.Pod{}, 0)
	si.AddEventHandler(rh)

	pm := &PodManager{realEvents: realEvents, mergedEvents: mergedEvents}

	stopNever := make(chan struct{})
	go si.Run(stopNever)
	go pm.merge()

	return pm, nil
}
