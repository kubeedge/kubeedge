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
	"sort"
	"strings"
	"sync"

	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
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
// list kind learned for the resource is dropped because the resource maps to
// another kind, after which ListKindFor falls back to the schemes or the
// default. It is called without the change locker held.
type ListKindLearnedFunc func(gvr schema.GroupVersionResource, listKind string)

// RESTMapper maps kinds to resources and back for components that have no
// discovery client, such as the edge MetaServer.
//
// It delegates to Kubernetes RESTMappers, in order:
//  1. a discovery RESTMapper, built with restmapper.NewDiscoveryRESTMapper from
//     the discovery documents seen, see LearnFromDiscovery;
//  2. a meta.DefaultRESTMapper of the types registered in the client-go and
//     KubeEdge schemes, and of CustomResourceDefinitions.
//
// The string rules of UnsafeKindToResource and UnsafeResourceToKind are only
// used for types neither of them knows.
type RESTMapper struct {
	lock sync.RWMutex

	// resources holds the resources of each discovered group version, by name.
	resources map[schema.GroupVersion]map[string]metav1.APIResource
	discovery meta.RESTMapper
	scheme    meta.RESTMapper

	learnedListKind map[schema.GroupVersionResource]string
	builtinListKind map[schema.GroupVersionResource]string

	// changeLocker is held while new discovery data is applied and the
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
	schemeMapper := meta.NewDefaultRESTMapper(nil)
	m := &RESTMapper{
		resources:       map[schema.GroupVersion]map[string]metav1.APIResource{},
		discovery:       restmapper.NewDiscoveryRESTMapper(nil),
		scheme:          schemeMapper,
		learnedListKind: map[schema.GroupVersionResource]string{},
		builtinListKind: map[schema.GroupVersionResource]string{},
	}
	for _, s := range []*runtime.Scheme{scheme.Scheme, kubeedgescheme.Scheme} {
		m.addScheme(schemeMapper, s)
	}
	// Registered directly rather than through the apiextensions scheme, which
	// edgecore does not otherwise depend on.
	for _, version := range []string{"v1", "v1beta1"} {
		gvk := schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: version, Kind: "CustomResourceDefinition"}
		m.addBuiltin(schemeMapper, gvk)
		m.builtinListKind[gvk.GroupVersion().WithResource("customresourcedefinitions")] = gvk.Kind + "List"
	}
	return m
}

var objectMetaAccessor = reflect.TypeOf((*metav1.Object)(nil)).Elem()

// addScheme registers the top-level object types of s in mapper. Kubernetes
// and KubeEdge name the resources of their own types with the standard plural
// rules that meta.DefaultRESTMapper applies.
func (m *RESTMapper) addScheme(mapper *meta.DefaultRESTMapper, s *runtime.Scheme) {
	for gvk, t := range s.AllKnownTypes() {
		if gvk.Version == runtime.APIVersionInternal || strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		// Options and other types without ObjectMeta are not resources.
		if !reflect.PointerTo(t).Implements(objectMetaAccessor) {
			continue
		}
		m.addBuiltin(mapper, gvk)
	}
	// The list types registered for these types give their list kinds.
	for gvk := range s.AllKnownTypes() {
		if !strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		itemGVK := gvk.GroupVersion().WithKind(strings.TrimSuffix(gvk.Kind, "List"))
		if mapping, err := mapper.RESTMapping(itemGVK.GroupKind(), itemGVK.Version); err == nil {
			m.builtinListKind[mapping.Resource] = gvk.Kind
		}
	}
}

// addBuiltin registers gvk in mapper. The scope is not known from the scheme,
// and is not used by the callers of RESTMapper.
func (m *RESTMapper) addBuiltin(mapper *meta.DefaultRESTMapper, gvk schema.GroupVersionKind) {
	mapper.Add(gvk, meta.RESTScopeNamespace)
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

// SetChangeLocker sets the lock held while new discovery data is applied and
// the MappingChangedFuncs run. Storage that keys objects by resource sets the
// lock it holds while computing keys and reading or writing objects, so that a
// mapping change and the migration of the objects it moves are atomic for it:
// an object is either written before the change, and moved by the migration,
// or after it, under the new resource.
//
// The caller of LearnFromDiscovery and RegisterMapping must not hold the
// locker, and the MappingChangedFuncs must not acquire it again.
func (m *RESTMapper) SetChangeLocker(locker sync.Locker) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.changeLocker = locker
}

// HasMapping reports whether the kind of gvr is known from discovery or the
// schemes, i.e. whether KindFor does not need to guess it.
func (m *RESTMapper) HasMapping(gvr schema.GroupVersionResource) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.kindForLocked(gvr)
	return ok
}

// ResourceFor returns the resource of gvk.
func (m *RESTMapper) ResourceFor(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	m.lock.RLock()
	defer m.lock.RUnlock()
	gvr, _ := m.resourceForLocked(gvk)
	return gvr
}

func (m *RESTMapper) resourceForLocked(gvk schema.GroupVersionKind) (schema.GroupVersionResource, bool) {
	for _, mapper := range []meta.RESTMapper{m.discovery, m.scheme} {
		if mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version); err == nil {
			return mapping.Resource, true
		}
	}
	return gvk.GroupVersion().WithResource(UnsafeKindToResource(gvk.Kind)), false
}

// KindFor returns the kind of gvr.
func (m *RESTMapper) KindFor(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	m.lock.RLock()
	defer m.lock.RUnlock()
	gvk, _ := m.kindForLocked(gvr)
	return gvk
}

func (m *RESTMapper) kindForLocked(gvr schema.GroupVersionResource) (schema.GroupVersionKind, bool) {
	for _, mapper := range []meta.RESTMapper{m.discovery, m.scheme} {
		if gvk, err := mapper.KindFor(gvr); err == nil {
			return gvk, true
		}
	}
	return gvr.GroupVersion().WithKind(UnsafeResourceToKind(gvr.Resource)), false
}

// ListKindFor returns the kind of a list of gvr, from, in order:
//  1. the list kind returned by the API server for gvr, see LearnListKind and
//     RegisterListKind;
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

// LearnListKind records the kind of list, a list returned by the API server
// for gvr. Objects other than lists are ignored.
func (m *RESTMapper) LearnListKind(gvr schema.GroupVersionResource, list runtime.Object) {
	if list == nil || gvr.Resource == "" || !meta.IsListType(list) {
		return
	}
	gvk := list.GetObjectKind().GroupVersionKind()
	if gvk.Kind == "" || gvk.GroupVersion() != gvr.GroupVersion() {
		return
	}
	m.RegisterListKind(gvr, gvk.Kind)
}

// isMetaKind reports whether kind is one of the kinds the API server returns
// instead of the requested object, e.g. a Status on failure, which must not be
// learned as the list kind of the requested resource.
func isMetaKind(kind string) bool {
	switch kind {
	case "Status", "Table", "PartialObjectMetadata", "PartialObjectMetadataList", "WatchEvent":
		return true
	}
	return false
}

// RegisterMapping records that the resource gvr serves kind, as a discovery
// document of its group version would. Another resource of the group version
// previously recorded for kind is dropped, since a kind has one resource. It
// returns true if the mapping was not known before.
func (m *RESTMapper) RegisterMapping(gvr schema.GroupVersionResource, kind string) bool {
	if gvr.Resource == "" || kind == "" || strings.Contains(gvr.Resource, "/") {
		return false
	}
	gv := gvr.GroupVersion()
	m.lock.RLock()
	resources := map[string]metav1.APIResource{}
	for name, r := range m.resources[gv] {
		if name != gvr.Resource && r.Kind != kind {
			resources[name] = r
		}
	}
	m.lock.RUnlock()
	resources[gvr.Resource] = metav1.APIResource{Name: gvr.Resource, Kind: kind, Namespaced: true}
	return m.apply(map[schema.GroupVersion]map[string]metav1.APIResource{gv: resources}) > 0
}

// LearnFromDiscovery records the resources of a discovery document: either the
// APIResourceList served at /api/<version> and /apis/<group>/<version>, or the
// aggregated APIGroupDiscoveryList served at /api and /apis. The resources of
// each group version in the document replace those known before. Other
// documents are ignored. It returns the number of resources whose kind became
// known, or changed.
func (m *RESTMapper) LearnFromDiscovery(data []byte) int {
	var typeMeta metav1.TypeMeta
	if err := json.Unmarshal(data, &typeMeta); err != nil {
		return 0
	}

	var lists map[schema.GroupVersion]*metav1.APIResourceList
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
		lists = map[schema.GroupVersion]*metav1.APIResourceList{gv: &list}

	case typeMeta.Kind == "APIGroupDiscoveryList" && typeMeta.APIVersion == apidiscoveryv2.SchemeGroupVersion.String():
		var list apidiscoveryv2.APIGroupDiscoveryList
		if err := json.Unmarshal(data, &list); err != nil {
			return 0
		}
		_, lists, _ = discovery.SplitGroupsAndResources(list)

	case typeMeta.Kind == "APIGroupDiscoveryList" && typeMeta.APIVersion == apidiscoveryv2beta1.SchemeGroupVersion.String():
		var list apidiscoveryv2beta1.APIGroupDiscoveryList
		if err := json.Unmarshal(data, &list); err != nil {
			return 0
		}
		_, lists, _ = discovery.SplitGroupsAndResourcesV2Beta1(list)

	default:
		return 0
	}

	update := map[schema.GroupVersion]map[string]metav1.APIResource{}
	for gv, list := range lists {
		resources := map[string]metav1.APIResource{}
		for _, r := range list.APIResources {
			// Subresources, such as pods/status, are not stored on their own.
			if r.Name == "" || r.Kind == "" || strings.Contains(r.Name, "/") {
				continue
			}
			resources[r.Name] = metav1.APIResource{Name: r.Name, Kind: r.Kind, Namespaced: r.Namespaced}
		}
		if len(resources) > 0 {
			update[gv] = resources
		}
	}
	return m.apply(update)
}

type resourceChange struct {
	gvk            schema.GroupVersionKind
	oldGVR, newGVR schema.GroupVersionResource
}

// apply replaces the resources of the group versions of update and rebuilds
// the discovery RESTMapper. The MappingChangedFuncs are told about the kinds
// whose resource changed, with the change locker held, and the
// ListKindLearnedFuncs about the list kinds dropped because their resource
// now maps to another kind. It returns the number of resources whose kind
// became known, or changed.
func (m *RESTMapper) apply(update map[schema.GroupVersion]map[string]metav1.APIResource) int {
	m.lock.RLock()
	changed := m.changedLocked(update)
	locker := m.changeLocker
	m.lock.RUnlock()
	if len(changed) == 0 {
		return 0
	}

	// Hold the change locker while the mapping is applied and the listeners
	// move the objects of the kinds whose resource changes, so that storage
	// never sees the new mapping before its objects have been moved.
	if locker != nil {
		locker.Lock()
	}
	m.lock.Lock()
	changed = m.changedLocked(update)
	if len(changed) == 0 {
		m.lock.Unlock()
		if locker != nil {
			locker.Unlock()
		}
		return 0
	}

	// The kinds and resources whose mapping may change.
	type kindBefore struct {
		gvk   schema.GroupVersionKind
		known bool
	}
	kinds := map[schema.GroupVersionKind]schema.GroupVersionResource{}
	gvrs := map[schema.GroupVersionResource]kindBefore{}
	for _, gv := range changed {
		for _, resources := range []map[string]metav1.APIResource{m.resources[gv], update[gv]} {
			for name, r := range resources {
				gvr := gv.WithResource(name)
				kinds[gv.WithKind(r.Kind)], _ = m.resourceForLocked(gv.WithKind(r.Kind))
				gvk, known := m.kindForLocked(gvr)
				gvrs[gvr] = kindBefore{gvk: gvk, known: known}
			}
		}
		m.resources[gv] = update[gv]
	}
	m.discovery = restmapper.NewDiscoveryRESTMapper(m.groupResourcesLocked())

	// A resource is learned when its kind becomes known, or changes.
	learned := 0
	for gvr, before := range gvrs {
		if after, known := m.kindForLocked(gvr); known && (!before.known || after != before.gvk) {
			learned++
		}
	}

	var changes []resourceChange
	for gvk, before := range kinds {
		if after, _ := m.resourceForLocked(gvk); after != before {
			changes = append(changes, resourceChange{gvk: gvk, oldGVR: before, newGVR: after})
		}
	}
	var droppedListKinds []schema.GroupVersionResource
	for gvr, before := range gvrs {
		if after, _ := m.kindForLocked(gvr); after != before.gvk {
			if _, ok := m.learnedListKind[gvr]; ok {
				delete(m.learnedListKind, gvr)
				droppedListKinds = append(droppedListKinds, gvr)
			}
		}
	}
	listeners := append([]MappingChangedFunc(nil), m.listeners...)
	listKindListeners := append([]ListKindLearnedFunc(nil), m.listKindListeners...)
	m.lock.Unlock()

	klog.V(4).Infof("[metaserver] learned the resources of %v", changed)
	for _, c := range changes {
		for _, fn := range listeners {
			fn(c.gvk, c.oldGVR, c.newGVR)
		}
	}
	if locker != nil {
		locker.Unlock()
	}

	// Tell the list kind listeners about the dropped list kinds, so that a
	// persisted list kind is dropped as well.
	for _, gvr := range droppedListKinds {
		for _, fn := range listKindListeners {
			fn(gvr, "")
		}
	}
	return learned
}

// changedLocked returns the group versions whose resources differ in update.
func (m *RESTMapper) changedLocked(update map[schema.GroupVersion]map[string]metav1.APIResource) []schema.GroupVersion {
	var changed []schema.GroupVersion
	for gv, resources := range update {
		if !reflect.DeepEqual(m.resources[gv], resources) {
			changed = append(changed, gv)
		}
	}
	sort.Slice(changed, func(i, j int) bool { return changed[i].String() < changed[j].String() })
	return changed
}

// groupResourcesLocked returns the discovered resources in the form taken by
// restmapper.NewDiscoveryRESTMapper. A kind served by several resources of a
// group version, like the ExternalMetricValueList of every metric of
// external.metrics.k8s.io, has no single resource: it is left out, and keeps
// resolving as before, instead of moving from one resource to another.
func (m *RESTMapper) groupResourcesLocked() []*restmapper.APIGroupResources {
	groups := map[string]*restmapper.APIGroupResources{}
	for gv, resources := range m.resources {
		resourcesOfKind := map[string]int{}
		for _, r := range resources {
			resourcesOfKind[r.Kind]++
		}
		var list []metav1.APIResource
		for _, r := range resources {
			if resourcesOfKind[r.Kind] == 1 {
				list = append(list, r)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

		group, ok := groups[gv.Group]
		if !ok {
			group = &restmapper.APIGroupResources{
				Group:              metav1.APIGroup{Name: gv.Group},
				VersionedResources: map[string][]metav1.APIResource{},
			}
			groups[gv.Group] = group
		}
		group.Group.Versions = append(group.Group.Versions, metav1.GroupVersionForDiscovery{GroupVersion: gv.String(), Version: gv.Version})
		group.VersionedResources[gv.Version] = list
	}

	ret := make([]*restmapper.APIGroupResources, 0, len(groups))
	for _, group := range groups {
		sort.Slice(group.Group.Versions, func(i, j int) bool { return group.Group.Versions[i].Version < group.Group.Versions[j].Version })
		group.Group.PreferredVersion = group.Group.Versions[0]
		ret = append(ret, group)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].Group.Name < ret[j].Group.Name })
	return ret
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
