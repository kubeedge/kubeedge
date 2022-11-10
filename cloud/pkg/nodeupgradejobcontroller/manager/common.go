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

package manager

import (
	"k8s.io/client-go/tools/cache"
)

type AddFunc func(obj interface{})
type UpdateFunc func(oldObj, newObj interface{})
type DeleteFunc func(obj interface{})
type FilterFunc func(obj interface{}) bool

// resourceEventHandler wraps a cache.ResourceEventHandlerFuncs
// implement the interface cache.ResourceEventHandler
type resourceEventHandler struct {
	*cache.ResourceEventHandlerFuncs
}

// filteringResourceEventHandler wraps a cache.FilteringResourceEventHandler
// there's a provided filter to all events coming
// implement the interface cache.ResourceEventHandler
type filteringResourceEventHandler struct {
	*cache.FilteringResourceEventHandler
}

func NewResourceEventHandler(filterFunc FilterFunc, addFunc AddFunc, updateFunc UpdateFunc, deleteFunc DeleteFunc) cache.ResourceEventHandler {
	return &resourceEventHandler{
		ResourceEventHandlerFuncs: &cache.ResourceEventHandlerFuncs{
			AddFunc:    addFunc,
			UpdateFunc: updateFunc,
			DeleteFunc: deleteFunc,
		},
	}
}

func NewFilterResourceEventHandler(filterFunc FilterFunc, addFunc AddFunc, updateFunc UpdateFunc, deleteFunc DeleteFunc) cache.ResourceEventHandler {
	return &filteringResourceEventHandler{
		FilteringResourceEventHandler: &cache.FilteringResourceEventHandler{
			FilterFunc: filterFunc,
			Handler: &cache.ResourceEventHandlerFuncs{
				AddFunc:    addFunc,
				UpdateFunc: updateFunc,
				DeleteFunc: deleteFunc,
			},
		},
	}
}

// // OnAdd handle Add event
// func (h *filteringResourceEventHandler) OnAdd(obj interface{}) {
// 	h.OnAdd(obj)
// }

// // OnUpdate handle Update event
// func (h *filteringResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
// 	h.OnUpdate(oldObj, newObj)
// }

// // OnDelete handle Delete event
// func (h *filteringResourceEventHandler) OnDelete(obj interface{}) {
// 	h.OnDelete(obj)
// }
