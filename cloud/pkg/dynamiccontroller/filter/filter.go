package filter

import (
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
	// filters map
	filters map[string]Filter
)

func init() {
	filters = make(map[string]Filter)
}

// Register register filter
func Register(f Filter) {
	filters[f.Name()] = f
	klog.Infof("Filter %s registered successfully", f.Name())
}

// GetFilters gets filter map
func GetFilters() map[string]Filter {
	return filters
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
