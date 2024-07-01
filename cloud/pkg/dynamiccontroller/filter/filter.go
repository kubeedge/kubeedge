/*
Copyright 2022 The KubeEdge Authors.

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

package filter

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// Filter defines the resource filtering interface for a single gvr
type Filter interface {
	// Name return filter name
	Name() string
	// NeedFilter checks whether the resource should be process by the filter
	NeedFilter(obj runtime.Object) bool
	// FilterResource filters resources that do not match this filter or
	// modifies the content of resources based on certain information,
	// for example, modifying the Kube-APIserver address for edge pod.
	FilterResource(targetNode string, obj runtime.Object)
}

// FiltersChain defines a Filter array.
type FiltersChain struct {
	filters []Filter
}

func newFiltersChain() *FiltersChain {
	return &FiltersChain{filters: make([]Filter, 0)}
}

func (fc *FiltersChain) Add(filter Filter) {
	fc.filters = append(fc.filters, filter)
}

func (fc *FiltersChain) Process(content runtime.Object, targetNode string) {
	for _, f := range fc.filters {
		if !f.NeedFilter(content) {
			continue
		}

		f.FilterResource(targetNode, content)
	}
}

var (
	// resourceFilters maps GroupVersionResource to a list of resource
	// filter chain, for one type of resource, there may be several filters.
	resourceFilters = make(map[schema.GroupVersionResource]*FiltersChain)
)

// RegisterFilter register filter for GroupVersionResource.
// All GroupVersionResource filters should register themselves.
func RegisterFilter(gvr schema.GroupVersionResource, filter Filter) {
	klog.Infof("Register filter %s for gvr: %s", filter.Name(), gvr.String())

	if _, exist := resourceFilters[gvr]; !exist {
		resourceFilters[gvr] = newFiltersChain()
	}

	resourceFilters[gvr].Add(filter)
}

func GetFilterChainFor(gvr schema.GroupVersionResource) *FiltersChain {
	return resourceFilters[gvr]
}
