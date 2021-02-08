package watchhook

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

var (
	hooksLock sync.Mutex
	// hooks is a map from hook.id to hook
	hooks = make(map[string]*WatchHook)
)

func AddHook(hook *WatchHook) error {
	hooksLock.Lock()
	defer hooksLock.Unlock()
	if _, exists := hooks[hook.id]; exists {
		return fmt.Errorf("unable to add hook %v because it was already registered", hook.id)
	}
	hooks[hook.id] = hook
	return nil
}

func DeleteHook(id string) error {
	hooksLock.Lock()
	defer hooksLock.Unlock()
	if _, exists := hooks[id]; exists {
		delete(hooks, id)
		return nil
	}
	return fmt.Errorf("unable to delete %q because it was not registered", id)
}

// Trigger trigger the corresponding hook to serve watch based on the event passed in
func Trigger(e watch.Event) {
	key, err := metaserver.KeyFuncObj(e.Object)
	if err != nil {
		klog.Errorf("failed to get key, %v", err)
		return
	}
	gvr, ns, name := metaserver.ParseKey(key)
	for _, hook := range hooks {
		compGVR, compNS, compName, compRev := true, true, true, true
		if !hook.GetGVR().Empty() {
			compGVR = hook.GetGVR() == gvr
		}
		if hook.GetNamespace() != "" {
			compNS = hook.GetNamespace() == ns
		}
		if hook.GetName() != "" {
			compName = hook.GetName() == name
		}
		if hook.GetResourceVersion() != 0 {
			accessor, err := meta.Accessor(e.Object)
			if err != nil {
				klog.Errorf("failed to get accessor, %v", err)
				return
			}
			rev, err := etcd3.Versioner.ParseResourceVersion(accessor.GetResourceVersion())
			if err != nil {
				klog.Errorf("failed to parse resource version, %v", err)
				return
			}
			compRev = hook.GetResourceVersion() < rev
		}
		if compGVR && compNS && compName && compRev {
			utilruntime.Must(hook.Do(e))
		}
	}
}
