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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	kubeedgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
)

func TestRESTMapperBuiltinTypes(t *testing.T) {
	m := NewRESTMapper()
	tests := []struct {
		gvr  schema.GroupVersionResource
		kind string
	}{
		{schema.GroupVersionResource{Version: "v1", Resource: "pods"}, "Pod"},
		{schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "ConfigMap"},
		{schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}, "ServiceAccount"},
		{schema.GroupVersionResource{Version: "v1", Resource: "endpoints"}, "Endpoints"},
		{schema.GroupVersionResource{Version: "v1", Resource: "services"}, "Service"},
		{schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, "DaemonSet"},
		{schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, "Ingress"},
		{schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, "NetworkPolicy"},
		{schema.GroupVersionResource{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"}, "RuntimeClass"},
		{schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}, "Lease"},
		{schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}, "EndpointSlice"},
		{schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, "CustomResourceDefinition"},
		{schema.GroupVersionResource{Group: "devices.kubeedge.io", Version: "v1beta1", Resource: "devicemodels"}, "DeviceModel"},
		{schema.GroupVersionResource{Group: "operations.kubeedge.io", Version: "v1alpha2", Resource: "nodeupgradejobs"}, "NodeUpgradeJob"},
		{schema.GroupVersionResource{Group: "policy.kubeedge.io", Version: "v1alpha1", Resource: "serviceaccountaccesses"}, "ServiceAccountAccess"},
	}
	for _, tt := range tests {
		t.Run(tt.gvr.String(), func(t *testing.T) {
			gvk := tt.gvr.GroupVersion().WithKind(tt.kind)
			assert.True(t, m.HasMapping(tt.gvr))
			assert.Equal(t, gvk, m.KindFor(tt.gvr))
			assert.Equal(t, tt.gvr, m.ResourceFor(gvk))
			assert.Equal(t, tt.kind+"List", m.ListKindFor(tt.gvr).Kind)
		})
	}
}

// TestRESTMapperKubeEdgeCRDs checks the mapping of every KubeEdge CRD version
// registered in the KubeEdge scheme against the CRD manifests.
func TestRESTMapperKubeEdgeCRDs(t *testing.T) {
	m := NewRESTMapper()
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "build", "crds", "*", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var crd apiextensionsv1.CustomResourceDefinition
		require.NoError(t, yaml.Unmarshal(data, &crd), file)

		listKind := crd.Spec.Names.ListKind
		if listKind == "" {
			// Defaulted by the API server.
			listKind = crd.Spec.Names.Kind + "List"
		}
		registered := 0
		for _, version := range crd.Spec.Versions {
			gvk := schema.GroupVersionKind{Group: crd.Spec.Group, Version: version.Name, Kind: crd.Spec.Names.Kind}
			if !kubeedgescheme.Scheme.Recognizes(gvk) {
				continue
			}
			registered++
			gvr := gvk.GroupVersion().WithResource(crd.Spec.Names.Plural)
			assert.Equal(t, gvr, m.ResourceFor(gvk), file)
			assert.Equal(t, gvk, m.KindFor(gvr), file)
			assert.Equal(t, listKind, m.ListKindFor(gvr).Kind, file)
		}
		assert.NotZero(t, registered, "no version of %s is registered in the KubeEdge scheme", file)
	}
}

// mappingCases are kinds the string rules get wrong in at least one direction.
var mappingCases = []struct {
	gvr  schema.GroupVersionResource
	kind string
}{
	{schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumnodes"}, "CiliumNode"},
	{schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumnetworkpolicies"}, "CiliumNetworkPolicy"},
	{schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "gateways"}, "Gateway"},
	{schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "virtualservices"}, "VirtualService"},
	{schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}, "HTTPRoute"},
	{schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors"}, "ServiceMonitor"},
	{schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers"}, "ClusterIssuer"},
	{schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "databases"}, "Database"},
	{schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "apikeys"}, "APIKey"},
	{schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "mailboxes"}, "Mailbox"},
	{schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1alpha1", Resource: "podchaos"}, "PodChaos"},
}

func TestRESTMapperGuessesUnknownTypes(t *testing.T) {
	m := NewRESTMapper()
	for _, tt := range mappingCases {
		gvk := tt.gvr.GroupVersion().WithKind(tt.kind)
		assert.False(t, m.HasMapping(tt.gvr))
		assert.Equal(t, tt.gvr.GroupVersion().WithKind(UnsafeResourceToKind(tt.gvr.Resource)), m.KindFor(tt.gvr))
		assert.Equal(t, tt.gvr.GroupVersion().WithResource(UnsafeKindToResource(tt.kind)), m.ResourceFor(gvk))
	}
}

func TestRESTMapperRegisterMapping(t *testing.T) {
	m := NewRESTMapper()
	for _, tt := range mappingCases {
		t.Run(tt.gvr.String(), func(t *testing.T) {
			gvk := tt.gvr.GroupVersion().WithKind(tt.kind)
			assert.True(t, m.RegisterMapping(tt.gvr, tt.kind))
			assert.False(t, m.RegisterMapping(tt.gvr, tt.kind), "registering a known mapping again must be a no-op")
			assert.True(t, m.HasMapping(tt.gvr))
			assert.Equal(t, gvk, m.KindFor(tt.gvr))
			assert.Equal(t, tt.gvr, m.ResourceFor(gvk))
			assert.Equal(t, tt.kind+"List", m.ListKindFor(tt.gvr).Kind)
		})
	}

	// Invalid input is ignored.
	assert.False(t, m.RegisterMapping(schema.GroupVersionResource{Version: "v1"}, "Pod"))
	assert.False(t, m.RegisterMapping(schema.GroupVersionResource{Version: "v1", Resource: "pods"}, ""))
	assert.False(t, m.RegisterMapping(schema.GroupVersionResource{Version: "v1", Resource: "pods/status"}, "Pod"))
}

func TestRESTMapperMappingChangedListener(t *testing.T) {
	type change struct {
		gvk            schema.GroupVersionKind
		oldGVR, newGVR schema.GroupVersionResource
	}
	m := NewRESTMapper()
	var changes []change
	m.AddMappingChangedListener(func(gvk schema.GroupVersionKind, oldGVR, newGVR schema.GroupVersionResource) {
		changes = append(changes, change{gvk, oldGVR, newGVR})
	})

	gateways := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "gateways"}
	m.RegisterMapping(gateways, "Gateway")
	require.Len(t, changes, 1)
	assert.Equal(t, change{
		gvk:    gateways.GroupVersion().WithKind("Gateway"),
		oldGVR: gateways.GroupVersion().WithResource("gatewaies"),
		newGVR: gateways,
	}, changes[0])

	// The guess was right: the resource returned by ResourceFor does not change.
	m.RegisterMapping(schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}, "Widget")
	// Builtin type: already known.
	m.RegisterMapping(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "ConfigMap")
	assert.Len(t, changes, 1)
}

// TestRESTMapperReplaceMapping checks that a learned mapping replaced by an
// authoritative one leaves no stale entry behind in either direction.
func TestRESTMapperReplaceMapping(t *testing.T) {
	type change struct {
		oldGVR, newGVR schema.GroupVersionResource
	}
	m := NewRESTMapper()
	var changes []change
	m.AddMappingChangedListener(func(_ schema.GroupVersionKind, oldGVR, newGVR schema.GroupVersionResource) {
		changes = append(changes, change{oldGVR, newGVR})
	})

	gv := schema.GroupVersion{Group: "networking.istio.io", Version: "v1alpha3"}
	gateway := gv.WithKind("Gateway")
	gatewaies := gv.WithResource("gatewaies")
	gateways := gv.WithResource("gateways")

	// 1. An initial mapping Gateway -> gatewaies is learned.
	require.True(t, m.RegisterMapping(gatewaies, "Gateway"))
	assert.Equal(t, gatewaies, m.ResourceFor(gateway))
	assert.True(t, m.HasMapping(gatewaies))

	// 2. The authoritative discovery mapping Gateway -> gateways is learned later.
	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv.String(),
		APIResources: []metav1.APIResource{{Name: "gateways", Namespaced: true, Kind: "Gateway"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, m.LearnFromDiscovery(doc))

	// 3. ResourceFor(Gateway) returns gateways.
	assert.Equal(t, gateways, m.ResourceFor(gateway))
	assert.Equal(t, gateway, m.KindFor(gateways))

	// 4. gatewaies is no longer an authoritative mapping for Gateway: its kind
	// is guessed again, like any resource never learned.
	assert.False(t, m.HasMapping(gatewaies))
	assert.Equal(t, gv.WithKind(UnsafeResourceToKind("gatewaies")), m.KindFor(gatewaies))

	// Learning gatewaies did not change the resource of Gateway, which was
	// already guessed as gatewaies. Learning gateways did, and the listener was
	// told so: this is the change the stored objects are migrated with.
	assert.Equal(t, []change{{oldGVR: gatewaies, newGVR: gateways}}, changes)
}

// TestRESTMapperReplaceKindOfResource checks that a resource learned for
// another kind no longer maps the old kind to it.
func TestRESTMapperReplaceKindOfResource(t *testing.T) {
	m := NewRESTMapper()
	gv := schema.GroupVersion{Group: "example.com", Version: "v1"}
	widgets := gv.WithResource("widgets")

	require.True(t, m.RegisterMapping(widgets, "Gadget"))
	require.True(t, m.RegisterMapping(widgets, "Widget"))

	assert.Equal(t, gv.WithKind("Widget"), m.KindFor(widgets))
	assert.Equal(t, widgets, m.ResourceFor(gv.WithKind("Widget")))
	// Gadget falls back to the guess instead of still pointing to widgets.
	assert.Equal(t, gv.WithResource(UnsafeKindToResource("Gadget")), m.ResourceFor(gv.WithKind("Gadget")))
}

func TestRESTMapperLearnFromObject(t *testing.T) {
	m := NewRESTMapper()
	nodes := schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumnodes"}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("cilium.io/v2")
	obj.SetKind("CiliumNode")
	m.LearnFromObject(nodes, obj)
	assert.Equal(t, "CiliumNode", m.KindFor(nodes).Kind)

	// An empty list still tells the kind through the list kind.
	identities := schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumidentities"}
	list := &unstructured.UnstructuredList{}
	list.SetAPIVersion("cilium.io/v2")
	list.SetKind("CiliumIdentityList")
	m.LearnFromObject(identities, list)
	assert.Equal(t, "CiliumIdentity", m.KindFor(identities).Kind)

	// Objects of another group version, without a kind, or lists without a
	// list kind are ignored.
	endpoints := schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumendpoints"}
	other := &unstructured.Unstructured{}
	other.SetAPIVersion("cilium.io/v2alpha1")
	other.SetKind("CiliumEndpoint")
	m.LearnFromObject(endpoints, other)
	m.LearnFromObject(endpoints, &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "cilium.io/v2"}})
	badList := &unstructured.UnstructuredList{}
	badList.SetAPIVersion("cilium.io/v2")
	badList.SetKind("CiliumEndpoint")
	m.LearnFromObject(endpoints, badList)
	m.LearnFromObject(endpoints, nil)
	assert.False(t, m.HasMapping(endpoints))

	// A Status returned instead of the object is not learned as its kind.
	examples := schema.GroupVersionResource{Version: "v1", Resource: "examples"}
	status := &unstructured.Unstructured{}
	status.SetAPIVersion("v1")
	status.SetKind("Status")
	m.LearnFromObject(examples, status)
	assert.False(t, m.HasMapping(examples))
}

func TestRESTMapperLearnFromAPIResourceList(t *testing.T) {
	m := NewRESTMapper()
	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: "networking.istio.io/v1alpha3",
		APIResources: []metav1.APIResource{
			{Name: "gateways", SingularName: "gateway", Namespaced: true, Kind: "Gateway"},
			{Name: "gateways/status", Namespaced: true, Kind: "Gateway"},
			{Name: "virtualservices", SingularName: "virtualservice", Namespaced: true, Kind: "VirtualService"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, 2, m.LearnFromDiscovery(doc))
	assert.Equal(t, 0, m.LearnFromDiscovery(doc), "learning the same document again must be a no-op")

	gateways := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "gateways"}
	assert.Equal(t, gateways, m.ResourceFor(gateways.GroupVersion().WithKind("Gateway")))
	assert.Equal(t, "VirtualService", m.KindFor(gateways.GroupVersion().WithResource("virtualservices")).Kind)
	assert.False(t, m.HasMapping(gateways.GroupVersion().WithResource("gateways/status")))
}

func TestRESTMapperLearnFromAggregatedDiscovery(t *testing.T) {
	ciliumNodes := schema.GroupVersionResource{Group: "cilium.io", Version: "v2", Resource: "ciliumnodes"}
	podChaos := schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1alpha1", Resource: "podchaos"}

	v2Doc, err := json.Marshal(&apidiscoveryv2.APIGroupDiscoveryList{
		TypeMeta: metav1.TypeMeta{Kind: "APIGroupDiscoveryList", APIVersion: "apidiscovery.k8s.io/v2"},
		Items: []apidiscoveryv2.APIGroupDiscovery{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "cilium.io"},
				Versions: []apidiscoveryv2.APIVersionDiscovery{{
					Version: "v2",
					Resources: []apidiscoveryv2.APIResourceDiscovery{
						{Resource: "ciliumnodes", ResponseKind: &metav1.GroupVersionKind{Group: "cilium.io", Version: "v2", Kind: "CiliumNode"}},
						{Resource: "ciliumnoresponsekind"},
					},
				}},
			},
		},
	})
	require.NoError(t, err)

	v2beta1Doc, err := json.Marshal(&apidiscoveryv2beta1.APIGroupDiscoveryList{
		TypeMeta: metav1.TypeMeta{Kind: "APIGroupDiscoveryList", APIVersion: "apidiscovery.k8s.io/v2beta1"},
		Items: []apidiscoveryv2beta1.APIGroupDiscovery{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "chaos-mesh.org"},
				Versions: []apidiscoveryv2beta1.APIVersionDiscovery{{
					Version: "v1alpha1",
					Resources: []apidiscoveryv2beta1.APIResourceDiscovery{
						{Resource: "podchaos", ResponseKind: &metav1.GroupVersionKind{Group: "chaos-mesh.org", Version: "v1alpha1", Kind: "PodChaos"}},
					},
				}},
			},
		},
	})
	require.NoError(t, err)

	m := NewRESTMapper()
	assert.Equal(t, 1, m.LearnFromDiscovery(v2Doc))
	assert.Equal(t, 1, m.LearnFromDiscovery(v2beta1Doc))
	assert.Equal(t, "CiliumNodeList", m.ListKindFor(ciliumNodes).Kind)
	assert.Equal(t, podChaos, m.ResourceFor(podChaos.GroupVersion().WithKind("PodChaos")))
	assert.False(t, m.HasMapping(ciliumNodes.GroupVersion().WithResource("ciliumnoresponsekind")))
}

func TestRESTMapperIgnoresOtherDocuments(t *testing.T) {
	m := NewRESTMapper()
	for _, doc := range []string{
		``,
		`not json`,
		`{"major":"1","minor":"32"}`,
		`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`,
		`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"a/b/c","resources":[{"name":"things","kind":"Thing"}]}`,
		`{"kind":"APIGroupDiscoveryList","apiVersion":"apidiscovery.k8s.io/v1","items":[]}`,
	} {
		assert.Zero(t, m.LearnFromDiscovery([]byte(doc)), doc)
	}
}

func TestRESTMapperListKindFor(t *testing.T) {
	m := NewRESTMapper()

	// Types of the schemes use their registered list types.
	assert.Equal(t, "ConfigMapList", m.ListKindFor(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Kind)
	assert.Equal(t, "CustomResourceDefinitionList", m.ListKindFor(schema.GroupVersionResource{
		Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions",
	}).Kind)

	// Without a list kind learned, the default <kind>List is used.
	databases := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "databases"}
	m.RegisterMapping(databases, "Database")
	assert.Equal(t, databases.GroupVersion().WithKind("DatabaseList"), m.ListKindFor(databases))
}

// TestRESTMapperCustomListKind checks that a list kind which differs from
// <kind>List, as allowed by spec.names.listKind of a CRD, is kept.
func TestRESTMapperCustomListKind(t *testing.T) {
	gv := schema.GroupVersion{Group: "example.com", Version: "v1"}
	databases := gv.WithResource("databases")
	newList := func(items ...string) *unstructured.UnstructuredList {
		list := &unstructured.UnstructuredList{}
		list.SetAPIVersion(gv.String())
		list.SetKind("DatabaseCollection")
		for _, name := range items {
			item := unstructured.Unstructured{}
			item.SetAPIVersion(gv.String())
			item.SetKind("Database")
			item.SetName(name)
			list.Items = append(list.Items, item)
		}
		return list
	}

	t.Run("list with items", func(t *testing.T) {
		m := NewRESTMapper()
		m.LearnFromObject(databases, newList("db1"))
		assert.Equal(t, gv.WithKind("Database"), m.KindFor(databases))
		assert.Equal(t, gv.WithKind("DatabaseCollection"), m.ListKindFor(databases))
		assert.Equal(t, databases, m.ResourceFor(gv.WithKind("Database")))
	})

	t.Run("empty list, kind from discovery", func(t *testing.T) {
		m := NewRESTMapper()
		doc, err := json.Marshal(&metav1.APIResourceList{
			TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
			GroupVersion: gv.String(),
			APIResources: []metav1.APIResource{{Name: "databases", Namespaced: true, Kind: "Database"}},
		})
		require.NoError(t, err)
		m.LearnFromDiscovery(doc)
		// An empty list does not tell the kind of its items: Database is kept,
		// DatabaseCollection is not mistaken for a kind.
		m.LearnFromObject(databases, newList())
		assert.Equal(t, gv.WithKind("Database"), m.KindFor(databases))
		assert.Equal(t, gv.WithKind("DatabaseCollection"), m.ListKindFor(databases))
	})

	t.Run("items decoded without a kind", func(t *testing.T) {
		m := NewRESTMapper()
		// Items without a kind are given DatabaseCollection by the decoder.
		data := []byte(`{"apiVersion":"example.com/v1","kind":"DatabaseCollection","items":[{"metadata":{"name":"db1"}}]}`)
		list := &unstructured.UnstructuredList{}
		require.NoError(t, json.Unmarshal(data, list))
		m.LearnFromObject(databases, list)
		assert.False(t, m.HasMapping(databases), "DatabaseCollection must not be learned as the kind of the items")
		assert.Equal(t, gv.WithKind("DatabaseCollection"), m.ListKindFor(databases))
	})

	t.Run("list kind dropped when the resource is learned for another kind", func(t *testing.T) {
		m := NewRESTMapper()
		m.LearnFromObject(databases, newList("db1"))
		m.RegisterMapping(databases, "Store")
		assert.Equal(t, gv.WithKind("StoreList"), m.ListKindFor(databases))
	})

	t.Run("listeners", func(t *testing.T) {
		m := NewRESTMapper()
		var learned []string
		m.AddListKindLearnedListener(func(gvr schema.GroupVersionResource, listKind string) {
			learned = append(learned, gvr.Resource+"="+listKind)
		})
		assert.True(t, m.RegisterListKind(databases, "DatabaseCollection"))
		assert.False(t, m.RegisterListKind(databases, "DatabaseCollection"))
		assert.False(t, m.RegisterListKind(databases, "Status"))
		assert.False(t, m.RegisterListKind(databases.GroupVersion().WithResource("databases/status"), "DatabaseCollection"))
		assert.Equal(t, []string{"databases=DatabaseCollection"}, learned)
	})
}

type recordingLocker struct {
	sync.Mutex
	held  bool
	locks int
}

func (l *recordingLocker) Lock() {
	l.Mutex.Lock()
	l.held = true
	l.locks++
}

func (l *recordingLocker) Unlock() {
	l.held = false
	l.Mutex.Unlock()
}

// TestRESTMapperChangeLocker checks that a mapping change is applied, and its
// listeners run, with the change locker held.
func TestRESTMapperChangeLocker(t *testing.T) {
	m := NewRESTMapper()
	locker := &recordingLocker{}
	m.SetChangeLocker(locker)

	gv := schema.GroupVersion{Group: "networking.istio.io", Version: "v1alpha3"}
	gateways := gv.WithResource("gateways")
	var heldDuringListener bool
	var resourceSeenByListener schema.GroupVersionResource
	m.AddMappingChangedListener(func(gvk schema.GroupVersionKind, _, _ schema.GroupVersionResource) {
		heldDuringListener = locker.held
		resourceSeenByListener = m.ResourceFor(gvk)
	})

	require.True(t, m.RegisterMapping(gateways, "Gateway"))
	assert.True(t, heldDuringListener)
	assert.Equal(t, gateways, resourceSeenByListener, "the listener sees the new mapping")
	assert.False(t, locker.held)
	assert.Equal(t, 1, locker.locks)

	// A known mapping does not take the locker.
	assert.False(t, m.RegisterMapping(gateways, "Gateway"))
	assert.Equal(t, 1, locker.locks)
}

// TestRESTMapperReplacedKindIsMoved checks that the kind previously learned
// for a resource is reported as moved when the resource is learned for
// another kind, so that its stored objects are moved too.
func TestRESTMapperReplacedKindIsMoved(t *testing.T) {
	m := NewRESTMapper()
	gv := schema.GroupVersion{Group: "example.com", Version: "v1"}
	widgets := gv.WithResource("widgets")
	require.True(t, m.RegisterMapping(widgets, "Gadget"))

	type change struct {
		gvk            schema.GroupVersionKind
		oldGVR, newGVR schema.GroupVersionResource
	}
	var changes []change
	m.AddMappingChangedListener(func(gvk schema.GroupVersionKind, oldGVR, newGVR schema.GroupVersionResource) {
		changes = append(changes, change{gvk, oldGVR, newGVR})
	})
	require.True(t, m.RegisterMapping(widgets, "Widget"))

	// Widget was already guessed as widgets, only Gadget moves: it falls back
	// to its guessed resource.
	assert.Equal(t, []change{
		{gvk: gv.WithKind("Gadget"), oldGVR: widgets, newGVR: gv.WithResource("gadgets")},
	}, changes)
}
