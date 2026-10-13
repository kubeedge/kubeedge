/*
Copyright 2026 The KubeEdge Authors.

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

package util

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	kubeedgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
)

// MappingChangedFunc is called when ResourceFor starts returning a different
// resource for a kind. Storage that keys objects by resource uses it to move
// the objects saved under the old resource. It is called with the change
// locker held, see SetChangeLocker.
type MappingChangedFunc func(gvk schema.GroupVersionKind, oldGVR, newGVR schema.GroupVersionResource)

// ListKindLearnedFunc is called when the list kind of a resource is learned
// from a list returned by the API server, and with an empty listKind when the
// list kind learned for the resource is dropped because the resource is
// learned for another kind, after which ListKindFor falls back to the schemes
// or the default. It is called without the change locker held.
type ListKindLearnedFunc func(gvr schema.GroupVersionResource, listKind string)

// RESTMapper maps kinds to resources and back.
//
// Lookups are answered, in order, from:
//  1. mappings learned at runtime from authoritative sources: API discovery
//     documents, and objects returned by the API server for a known resource;
//  2. the resource types registered in the client-go and KubeEdge schemes,
//     and CustomResourceDefinitions;
//  3. the string rules of UnsafeKindToResource and UnsafeResourceToKind, only
//     for types never seen through 1 or 2.
type RESTMapper struct {
	lock sync.RWMutex

	learnedKindToResource map[schema.GroupVersionKind]schema.GroupVersionResource
	learnedResourceToKind map[schema.GroupVersionResource]schema.GroupVersionKind
	learnedListKind       map[schema.GroupVersionResource]string

	builtinKindToResource map[schema.GroupVersionKind]schema.GroupVersionResource
	builtinResourceToKind map[schema.GroupVersionResource]schema.GroupVersionKind
	builtinListKind       map[schema.GroupVersionResource]string

	// changeLocker is held while a new mapping is applied and the
	// MappingChangedFuncs run.
	changeLocker      sync.Locker
	listeners         []MappingChangedFunc
	listKindListeners []ListKindLearnedFunc
}

var defaultRESTMapper = NewRESTMapper()

// DefaultRESTMapper returns the RESTMapper shared by the process.
func DefaultRESTMapper() *RESTMapper {
	return defaultRESTMapper
}

// NewRESTMapper returns a RESTMapper that knows the types of the client-go and
// KubeEdge schemes, and CustomResourceDefinitions.
func NewRESTMapper() *RESTMapper {
	m := &RESTMapper{
		learnedKindToResource: map[schema.GroupVersionKind]schema.GroupVersionResource{},
		learnedResourceToKind: map[schema.GroupVersionResource]schema.GroupVersionKind{},
		learnedListKind:       map[schema.GroupVersionResource]string{},
		builtinKindToResource: map[schema.GroupVersionKind]schema.GroupVersionResource{},
		builtinResourceToKind: map[schema.GroupVersionResource]schema.GroupVersionKind{},
		builtinListKind:       map[schema.GroupVersionResource]string{},
	}
	for _, s := range []*runtime.Scheme{scheme.Scheme, kubeedgescheme.Scheme} {
		m.addScheme(s)
	}
	// Registered directly rather than through the apiextensions scheme, which
	// edgecore does not otherwise depend on.
	for _, version := range []string{"v1", "v1beta1"} {
		gvk := schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: version, Kind: "CustomResourceDefinition"}
		m.addBuiltin(gvk)
		m.builtinListKind[m.builtinKindToResource[gvk]] = gvk.Kind + "List"
	}
	return m
}

var objectMetaAccessor = reflect.TypeOf((*metav1.Object)(nil)).Elem()

// addScheme indexes the top-level object types of s. Kubernetes and KubeEdge
// name the resources of their own types with the standard plural rules, so
// meta.UnsafeGuessKindToResource is exact for them.
func (m *RESTMapper) addScheme(s *runtime.Scheme) {
	for gvk, t := range s.AllKnownTypes() {
		if gvk.Version == runtime.APIVersionInternal || strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		// Options and other types without ObjectMeta are not resources.
		if !reflect.PointerTo(t).Implements(objectMetaAccessor) {
			continue
		}
		m.addBuiltin(gvk)
	}
	// The list types registered for the indexed types give their list kinds.
	for gvk := range s.AllKnownTypes() {
		if !strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		itemGVK := gvk.GroupVersion().WithKind(strings.TrimSuffix(gvk.Kind, "List"))
		if gvr, ok := m.builtinKindToResource[itemGVK]; ok {
			m.builtinListKind[gvr] = gvk.Kind
		}
	}
}

func (m *RESTMapper) addBuiltin(gvk schema.GroupVersionKind) {
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	m.builtinKindToResource[gvk] = gvr
	m.builtinResourceToKind[gvr] = gvk
}

// AddMappingChangedListener registers fn to be called whenever ResourceFor
// starts returning a different resource for a kind.
func (m *RESTMapper) AddMappingChangedListener(fn MappingChangedFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.listeners = append(m.listeners, fn)
}

// AddListKindLearnedListener registers fn to be called whenever the list kind
// of a resource is learned.
func (m *RESTMapper) AddListKindLearnedListener(fn ListKindLearnedFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.listKindListeners = append(m.listKindListeners, fn)
}

// SetChangeLocker sets the lock held while a new mapping is applied and the
// MappingChangedFuncs run. Storage that keys objects by resource sets the lock
// it holds while computing keys and reading or writing objects, so that a
// mapping change and the migration of the objects it moves are atomic for it:
// an object is either written before the change, and moved by the migration,
// or after it, under the new resource.
//
// The caller of RegisterMapping must not hold the locker, and the
// MappingChangedFuncs must not acquire it again.
func (m *RESTMapper) SetChangeLocker(locker sync.Locker) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.changeLocker = locker
}

// RegisterMapping records the authoritative mapping between gvr and kind. It
// returns true if the mapping was not known before.
func (m *RESTMapper) RegisterMapping(gvr schema.GroupVersionResource, kind string) bool {
	if gvr.Resource == "" || kind == "" || strings.Contains(gvr.Resource, "/") {
		return false
	}
	gvk := gvr.GroupVersion().WithKind(kind)

	m.lock.RLock()
	known := m.isLearnedLocked(gvr, gvk)
	locker := m.changeLocker
	m.lock.RUnlock()
	if known {
		return false
	}

	// Hold the change locker while the mapping is applied and the listeners
	// move the objects of the kinds whose resource changes, so that storage
	// never sees the new mapping before its objects have been moved.
	if locker != nil {
		locker.Lock()
	}
	unlock := func() {
		if locker != nil {
			locker.Unlock()
		}
	}

	m.lock.Lock()
	if m.isLearnedLocked(gvr, gvk) {
		m.lock.Unlock()
		unlock()
		return false
	}
	changes, droppedListKinds := m.applyLocked(gvr, gvk)
	listeners := append([]MappingChangedFunc(nil), m.listeners...)
	listKindListeners := append([]ListKindLearnedFunc(nil), m.listKindListeners...)
	m.lock.Unlock()

	klog.V(4).Infof("[metaserver] learned mapping %s <-> %s", gvk.String(), gvr.String())
	for _, c := range changes {
		for _, fn := range listeners {
			fn(c.gvk, c.oldGVR, c.newGVR)
		}
	}
	unlock()

	// Tell the list kind listeners about the list kinds dropped with the
	// replaced mappings, so that a persisted list kind is dropped as well.
	for _, dropped := range droppedListKinds {
		for _, fn := range listKindListeners {
			fn(dropped, "")
		}
	}
	return true
}

func (m *RESTMapper) isLearnedLocked(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind) bool {
	return m.learnedKindToResource[gvk] == gvr && m.learnedResourceToKind[gvr] == gvk
}

type resourceChange struct {
	gvk            schema.GroupVersionKind
	oldGVR, newGVR schema.GroupVersionResource
}

// applyLocked records gvr <-> gvk and returns the kinds whose resource changed,
// and the resources whose learned list kind was dropped. Besides gvk, the kind
// previously learned for gvr loses its mapping and falls back to the schemes or
// the guess.
func (m *RESTMapper) applyLocked(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind) ([]resourceChange, []schema.GroupVersionResource) {
	var droppedListKinds []schema.GroupVersionResource
	dropListKind := func(r schema.GroupVersionResource) {
		if _, ok := m.learnedListKind[r]; ok {
			delete(m.learnedListKind, r)
			droppedListKinds = append(droppedListKinds, r)
		}
	}

	affected := []schema.GroupVersionKind{gvk}
	if previous, ok := m.learnedResourceToKind[gvr]; ok && previous != gvk {
		affected = append(affected, previous)
	}
	before := make([]schema.GroupVersionResource, len(affected))
	for i, k := range affected {
		before[i] = m.resourceForLocked(k)
	}

	// Drop the entries of the mappings being replaced, so that a resource or
	// kind learned before is not still reported as authoritative.
	if previous, ok := m.learnedKindToResource[gvk]; ok && previous != gvr && m.learnedResourceToKind[previous] == gvk {
		delete(m.learnedResourceToKind, previous)
		dropListKind(previous)
	}
	if previous, ok := m.learnedResourceToKind[gvr]; ok && previous != gvk {
		if m.learnedKindToResource[previous] == gvr {
			delete(m.learnedKindToResource, previous)
		}
		// The list kind learned for gvr belonged to the previous kind.
		dropListKind(gvr)
	}
	m.learnedKindToResource[gvk] = gvr
	m.learnedResourceToKind[gvr] = gvk

	var changes []resourceChange
	for i, k := range affected {
		if after := m.resourceForLocked(k); after != before[i] {
			changes = append(changes, resourceChange{gvk: k, oldGVR: before[i], newGVR: after})
		}
	}
	return changes, droppedListKinds
}

// RegisterListKind records listKind, returned by the API server for a list of
// gvr. It returns true if it was not known before.
func (m *RESTMapper) RegisterListKind(gvr schema.GroupVersionResource, listKind string) bool {
	if gvr.Resource == "" || listKind == "" || strings.Contains(gvr.Resource, "/") || isMetaKind(listKind) {
		return false
	}
	m.lock.Lock()
	if m.learnedListKind[gvr] == listKind {
		m.lock.Unlock()
		return false
	}
	m.learnedListKind[gvr] = listKind
	listeners := append([]ListKindLearnedFunc(nil), m.listKindListeners...)
	m.lock.Unlock()

	klog.V(4).Infof("[metaserver] learned list kind %s of %s", listKind, gvr.String())
	for _, fn := range listeners {
		fn(gvr, listKind)
	}
	return true
}

// HasMapping reports whether the kind of gvr comes from an authoritative
// source, i.e. whether KindFor does not need to guess it.
func (m *RESTMapper) HasMapping(gvr schema.GroupVersionResource) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if _, ok := m.learnedResourceToKind[gvr]; ok {
		return true
	}
	_, ok := m.builtinResourceToKind[gvr]
	return ok
}

// ResourceFor returns the resource of gvk.
func (m *RESTMapper) ResourceFor(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.resourceForLocked(gvk)
}

func (m *RESTMapper) resourceForLocked(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	if gvr, ok := m.learnedKindToResource[gvk]; ok {
		return gvr
	}
	if gvr, ok := m.builtinKindToResource[gvk]; ok {
		return gvr
	}
	return gvk.GroupVersion().WithResource(UnsafeKindToResource(gvk.Kind))
}

// KindFor returns the kind of gvr.
func (m *RESTMapper) KindFor(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if gvk, ok := m.learnedResourceToKind[gvr]; ok {
		return gvk
	}
	if gvk, ok := m.builtinResourceToKind[gvr]; ok {
		return gvk
	}
	return gvr.GroupVersion().WithKind(UnsafeResourceToKind(gvr.Resource))
}

// ListKindFor returns the kind of a list of gvr, from, in order:
//  1. the list kind returned by the API server for gvr, see LearnFromObject
//     and RegisterListKind;
//  2. the list type registered in the schemes, for the types they know;
//  3. the default list kind of Kubernetes: the kind of gvr followed by "List".
//
// Discovery documents don't carry list kinds, so for a CRD whose
// spec.names.listKind is not <kind>List, the list kind is only right once a
// list of it has been returned by the API server, or restored from storage.
func (m *RESTMapper) ListKindFor(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	m.lock.RLock()
	listKind, ok := m.learnedListKind[gvr]
	if !ok {
		listKind, ok = m.builtinListKind[gvr]
	}
	m.lock.RUnlock()
	if ok {
		return gvr.GroupVersion().WithKind(listKind)
	}
	gvk := m.KindFor(gvr)
	gvk.Kind += "List"
	return gvk
}

// LearnFromObject records the mapping between gvr and the kind of obj, where
// obj is an object or a list of objects returned by the API server for gvr.
func (m *RESTMapper) LearnFromObject(gvr schema.GroupVersionResource, obj runtime.Object) {
	if obj == nil || gvr.Resource == "" {
		return
	}
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind == "" || gvk.GroupVersion() != gvr.GroupVersion() || isMetaKind(gvk.Kind) {
		return
	}
	if !meta.IsListType(obj) {
		m.RegisterMapping(gvr, gvk.Kind)
		return
	}

	listKind := gvk.Kind
	if kind := listItemKind(obj, gvr.GroupVersion(), listKind); kind != "" {
		m.RegisterMapping(gvr, kind)
	}
	// After the kind, since a new kind for gvr drops the list kind learned before.
	m.RegisterListKind(gvr, listKind)
}

// listItemKind returns the kind of the items of list, from the items when they
// carry it, and otherwise from listKind when it is <kind>List. It returns ""
// when the kind is not known: a list kind such as DatabaseCollection does not
// tell the kind of its items.
func listItemKind(list runtime.Object, gv schema.GroupVersion, listKind string) string {
	if items, err := meta.ExtractList(list); err == nil {
		for _, item := range items {
			itemGVK := item.GetObjectKind().GroupVersionKind()
			// Items without a kind are given listKind without its "List" suffix
			// when decoded, which is not a kind when listKind has no such suffix.
			if itemGVK.Kind != "" && itemGVK.Kind != listKind && itemGVK.GroupVersion() == gv && !isMetaKind(itemGVK.Kind) {
				return itemGVK.Kind
			}
		}
	}
	if kind := strings.TrimSuffix(listKind, "List"); kind != listKind && kind != "" {
		return kind
	}
	return ""
}

// isMetaKind reports whether kind is one of the kinds the API server returns
// instead of the requested object, e.g. a Status on failure, which must not be
// learned as the kind of the requested resource.
func isMetaKind(kind string) bool {
	switch kind {
	case "Status", "Table", "PartialObjectMetadata", "PartialObjectMetadataList", "WatchEvent":
		return true
	}
	return false
}

// LearnFromDiscovery records the mappings of a discovery document: either the
// APIResourceList served at /api/<version> and /apis/<group>/<version>, or the
// aggregated APIGroupDiscoveryList served at /api and /apis. Other documents
// are ignored. It returns the number of mappings that were not known before.
func (m *RESTMapper) LearnFromDiscovery(data []byte) int {
	var typeMeta metav1.TypeMeta
	if err := json.Unmarshal(data, &typeMeta); err != nil {
		return 0
	}
	var mappings []schema.GroupVersionResource
	kinds := map[schema.GroupVersionResource]string{}
	register := func(gvr schema.GroupVersionResource, kind string) {
		if gvr.Resource == "" || kind == "" {
			return
		}
		mappings = append(mappings, gvr)
		kinds[gvr] = kind
	}

	switch {
	case typeMeta.Kind == "APIResourceList":
		var list metav1.APIResourceList
		if err := json.Unmarshal(data, &list); err != nil {
			return 0
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return 0
		}
		for _, r := range list.APIResources {
			// Subresources, such as pods/status, are not stored on their own.
			if strings.Contains(r.Name, "/") {
				continue
			}
			register(gv.WithResource(r.Name), r.Kind)
		}

	case typeMeta.Kind == "APIGroupDiscoveryList" && typeMeta.APIVersion == apidiscoveryv2.SchemeGroupVersion.String():
		var list apidiscoveryv2.APIGroupDiscoveryList
		if err := json.Unmarshal(data, &list); err != nil {
			return 0
		}
		for _, group := range list.Items {
			for _, version := range group.Versions {
				gv := schema.GroupVersion{Group: group.Name, Version: version.Version}
				for _, r := range version.Resources {
					if r.ResponseKind != nil {
						register(gv.WithResource(r.Resource), r.ResponseKind.Kind)
					}
				}
			}
		}

	case typeMeta.Kind == "APIGroupDiscoveryList" && typeMeta.APIVersion == apidiscoveryv2beta1.SchemeGroupVersion.String():
		var list apidiscoveryv2beta1.APIGroupDiscoveryList
		if err := json.Unmarshal(data, &list); err != nil {
			return 0
		}
		for _, group := range list.Items {
			for _, version := range group.Versions {
				gv := schema.GroupVersion{Group: group.Name, Version: version.Version}
				for _, r := range version.Resources {
					if r.ResponseKind != nil {
						register(gv.WithResource(r.Resource), r.ResponseKind.Kind)
					}
				}
			}
		}
	}

	// A kind served by several resources of a group version, like the
	// ExternalMetricValueList of every metric of external.metrics.k8s.io, has
	// no single resource: it is not learned, and keeps resolving as before,
	// instead of moving from one resource to another.
	resourcesOfKind := map[schema.GroupVersionKind]int{}
	for _, gvr := range mappings {
		resourcesOfKind[gvr.GroupVersion().WithKind(kinds[gvr])]++
	}
	learned := 0
	for _, gvr := range mappings {
		if resourcesOfKind[gvr.GroupVersion().WithKind(kinds[gvr])] > 1 {
			continue
		}
		if m.RegisterMapping(gvr, kinds[gvr]) {
			learned++
		}
	}
	return learned
}

// ResourceFor returns the resource of gvk using the default RESTMapper.
func ResourceFor(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return defaultRESTMapper.ResourceFor(gvk)
}

// KindFor returns the kind of gvr using the default RESTMapper.
func KindFor(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	return defaultRESTMapper.KindFor(gvr)
}

// ListKindFor returns the kind of a list of gvr using the default RESTMapper.
func ListKindFor(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	return defaultRESTMapper.ListKindFor(gvr)
}
