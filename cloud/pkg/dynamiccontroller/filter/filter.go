package filter

import (
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

// Filter interface
type Filter interface {
	Name() string
	NeedFilter(content interface{}) bool
	FilterResource(targetNode string, obj runtime.Object)
}

var (
	// filters map. dynamiccontroller registers its filters while other
	// modules may already read the map, so filtersLock guards it.
	filters     map[string]Filter
	filtersLock sync.RWMutex
)

func init() {
	filters = make(map[string]Filter)
}

// Register register filter
func Register(f Filter) {
	filtersLock.Lock()
	filters[f.Name()] = f
	filtersLock.Unlock()
	klog.Infof("Filter %s registered successfully", f.Name())
}

// Unregister removes the filter with the given name.
func Unregister(name string) {
	filtersLock.Lock()
	defer filtersLock.Unlock()
	delete(filters, name)
}

// GetFilters returns a copy of the filter map.
func GetFilters() map[string]Filter {
	filtersLock.RLock()
	defer filtersLock.RUnlock()
	copied := make(map[string]Filter, len(filters))
	for name, f := range filters {
		copied[name] = f
	}
	return copied
}

// MessageFilter filter message according to specify policy
func MessageFilter(content interface{}, targetNode string) {
	for _, f := range GetFilters() {
		if !f.NeedFilter(content) {
			continue
		}
		if objList, ok := content.(*unstructured.UnstructuredList); ok {
			for i := range objList.Items {
				f.FilterResource(targetNode, &objList.Items[i])
			}
			continue
		}
		if obj, ok := content.(*unstructured.Unstructured); ok {
			f.FilterResource(targetNode, obj)
		}
	}
}
