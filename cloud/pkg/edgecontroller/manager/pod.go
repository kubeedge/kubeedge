/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manager

import (
	"reflect"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/eventmanager"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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
func NewPodManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*PodManager, error) {
	realEvents := make(chan watch.Event, config.Buffer.PodEvent)
	mergedEvents := make(chan watch.Event, config.Buffer.PodEvent)
	rh := eventmanager.NewGenericResourceEventHandler(realEvents)
	si.AddEventHandler(rh)
	pm := &PodManager{realEvents: realEvents, mergedEvents: mergedEvents}
	go pm.merge()
	return pm, nil
}
