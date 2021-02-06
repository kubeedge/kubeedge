package watchhook

import (
	"sync"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

type WatchHook struct {
	GVR             schema.GroupVersionResource
	Namespace       string
	Name            string
	ResourceVersion uint64
	id              string
	Receiver
	lock sync.Mutex
}

func NewWatchHook(key string, rev uint64, receiver Receiver) *WatchHook {
	id := uuid.New().String()
	gvr, ns, name := metaserver.ParseKey(key)
	wh := &WatchHook{
		id:              id,
		GVR:             gvr,
		Namespace:       ns,
		Name:            name,
		ResourceVersion: rev,
		Receiver:        receiver,
	}
	utilruntime.Must(AddHook(wh))
	return wh
}

func (h *WatchHook) Do(event watch.Event) error {
	h.Lock()
	defer h.UnLock()
	utilruntime.Must(h.Receive(event))
	return nil
}

func (h *WatchHook) GetGVR() schema.GroupVersionResource {
	return h.GVR
}

func (h *WatchHook) GetNamespace() string {
	return h.Namespace
}

func (h *WatchHook) GetName() string {
	return h.Name
}
func (h *WatchHook) GetResourceVersion() uint64 {
	return h.ResourceVersion
}

func (h *WatchHook) Stop() {
	h.Lock()
	defer h.UnLock()
	if err := DeleteHook(h.id); err != nil {
		klog.Error(err)
	}
}

func (h *WatchHook) Lock() {
	h.lock.Lock()
}

func (h *WatchHook) UnLock() {
	h.lock.Unlock()
}
