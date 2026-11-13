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

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/agent"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	fakeclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// fakeCloud patches the connection to the cloud and answers every application
// with the response returned by respond for its key.
func fakeCloud(t *testing.T, connected bool, respond func(key string) ([]byte, bool)) *gomonkey.Patches {
	t.Helper()
	return gomonkey.NewPatches().
		ApplyFunc(connect.IsConnected, func() bool { return connected }).
		ApplyFunc(beehiveContext.SendSync, func(_ string, msg model.Message, _ time.Duration) (model.Message, error) {
			var app metaserver.Application
			data, err := json.Marshal(msg.GetContent())
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &app))
			body, ok := respond(app.Key)
			if ok {
				app.Status = metaserver.Approved
				app.RespBody = body
			} else {
				app.Status = metaserver.Failed
				app.Reason = "not found"
			}
			content, err := json.Marshal(app)
			require.NoError(t, err)
			return model.Message{Content: content}, nil
		})
}

func resourceRequest(verb, group, version, resource string) context.Context {
	return request.WithRequestInfo(context.TODO(), &request.RequestInfo{
		IsResourceRequest: true,
		Verb:              verb,
		APIPrefix:         "apis",
		APIGroup:          group,
		APIVersion:        version,
		Resource:          resource,
	})
}

// discoveryDoc returns the APIResourceList of gv with the given resource and
// kind pairs.
func discoveryDoc(t *testing.T, gv schema.GroupVersion, resourceKinds ...string) []byte {
	t.Helper()
	list := &metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv.String(),
	}
	for i := 0; i+1 < len(resourceKinds); i += 2 {
		list.APIResources = append(list.APIResources, metav1.APIResource{Name: resourceKinds[i], Namespaced: true, Kind: resourceKinds[i+1]})
	}
	doc, err := json.Marshal(list)
	require.NoError(t, err)
	return doc
}

// cloudServing answers discovery requests for gv with doc, and any other
// request with body.
func cloudServing(gv schema.GroupVersion, doc, body []byte) func(string) ([]byte, bool) {
	return func(key string) ([]byte, bool) {
		if key == discoveryPath(gv) {
			return doc, true
		}
		return body, true
	}
}

// fakeLocalStorage replaces the imitator, which needs a database.
func fakeLocalStorage(patches *gomonkey.Patches) {
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(context.Context, []byte, string) error { return nil },
		InsertOrUpdateObjF:            func(context.Context, runtime.Object) error { return nil },
		GetPassThroughObjF: func(context.Context, string) ([]byte, error) {
			return nil, errors.New("not found")
		},
	})
}

// TestListLearnsMappingFromDiscovery checks that a list of a resource whose
// kind is not known fetches the discovery document of its group version, and
// records the list kind of the list.
func TestListLearnsMappingFromDiscovery(t *testing.T) {
	gv := schema.GroupVersion{Group: "list.cilium.io", Version: "v2"}
	gvr := gv.WithResource("ciliumnodes")
	list := []byte(`{"apiVersion":"list.cilium.io/v2","kind":"CiliumNodeList","metadata":{"resourceVersion":"1"},"items":[]}`)
	patches := fakeCloud(t, true, cloudServing(gv, discoveryDoc(t, gv, "ciliumnodes", "CiliumNode"), list))
	defer patches.Reset()
	fakeLocalStorage(patches)
	require.False(t, util.DefaultRESTMapper().HasMapping(gvr))

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	obj, err := rest.List(resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource), &metainternalversion.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, "CiliumNodeList", obj.GetObjectKind().GroupVersionKind().Kind)

	assert.True(t, util.DefaultRESTMapper().HasMapping(gvr))
	assert.Equal(t, "CiliumNode", util.KindFor(gvr).Kind)
	assert.Equal(t, gvr, util.ResourceFor(gv.WithKind("CiliumNode")))
	assert.Equal(t, "CiliumNodeList", util.ListKindFor(gvr).Kind)
}

func TestDecorateListUsesMapping(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "decorate.cilium.io", Version: "v2", Resource: "ciliumidentities"}
	ctx := resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource)

	list := &unstructured.UnstructuredList{}
	decorateList(ctx, list)
	assert.Equal(t, "CiliumidentityList", list.GetKind(), "unknown kinds are guessed")

	util.DefaultRESTMapper().RegisterMapping(gvr, "CiliumIdentity")
	list = &unstructured.UnstructuredList{}
	decorateList(ctx, list)
	assert.Equal(t, gvr.GroupVersion().WithKind("CiliumIdentityList"), list.GroupVersionKind())
}

func TestWatchLearnsMappingFromDiscovery(t *testing.T) {
	gv := schema.GroupVersion{Group: "watch.istio.io", Version: "v1alpha3"}
	gvr := gv.WithResource("gateways")
	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv.String(),
		APIResources: []metav1.APIResource{{Name: "gateways", Namespaced: true, Kind: "Gateway"}},
	})
	require.NoError(t, err)

	var cached []string
	patches := fakeCloud(t, true, func(key string) ([]byte, bool) {
		if key == "/apis/watch.istio.io/v1alpha3" {
			return doc, true
		}
		return nil, false
	})
	defer patches.Reset()
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(_ context.Context, _ []byte, key string) error {
			cached = append(cached, key)
			return nil
		},
	})

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	info, _ := request.RequestInfoFrom(resourceRequest("watch", gv.Group, gv.Version, "gateways"))
	rest.ensureMapping(context.TODO(), info)

	assert.Equal(t, gvr, util.ResourceFor(gv.WithKind("Gateway")))
	assert.Equal(t, []string{"/apis/watch.istio.io/v1alpha3"}, cached, "the discovery document is cached for offline use")
}

func TestWatchLearnsMappingFromCachedDiscoveryWhenOffline(t *testing.T) {
	gv := schema.GroupVersion{Group: "offline.istio.io", Version: "v1alpha3"}
	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv.String(),
		APIResources: []metav1.APIResource{{Name: "gateways", Namespaced: true, Kind: "Gateway"}},
	})
	require.NoError(t, err)

	patches := fakeCloud(t, false, func(string) ([]byte, bool) { return nil, false })
	defer patches.Reset()
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		GetPassThroughObjF: func(_ context.Context, key string) ([]byte, error) {
			if key == "/apis/offline.istio.io/v1alpha3" {
				return doc, nil
			}
			return nil, errors.New("not found")
		},
	})

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	info, _ := request.RequestInfoFrom(resourceRequest("watch", gv.Group, gv.Version, "gateways"))
	rest.ensureMapping(context.TODO(), info)

	assert.Equal(t, gv.WithResource("gateways"), util.ResourceFor(gv.WithKind("Gateway")))
}

func TestEnsureMappingSkipsKnownResources(t *testing.T) {
	patches := fakeCloud(t, true, func(string) ([]byte, bool) {
		t.Fatal("no discovery request is expected for a known resource")
		return nil, false
	})
	defer patches.Reset()

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	info, _ := request.RequestInfoFrom(resourceRequest("watch", "apps", "v1", "daemonsets"))
	rest.ensureMapping(context.TODO(), info)
	rest.ensureMapping(context.TODO(), &request.RequestInfo{Path: "/version"})
	rest.ensureMapping(context.TODO(), nil)
}

func TestDiscoveryPath(t *testing.T) {
	assert.Equal(t, "/api/v1", discoveryPath(schema.GroupVersion{Version: "v1"}))
	assert.Equal(t, "/apis/cilium.io/v2", discoveryPath(schema.GroupVersion{Group: "cilium.io", Version: "v2"}))
}

// TestCustomListKindServedOffline checks that a list kind which differs from
// <kind>List, learned from a list returned by the cloud, is used for lists
// served from the local storage.
func TestCustomListKindServedOffline(t *testing.T) {
	gv := schema.GroupVersion{Group: "offline.example.com", Version: "v1"}
	gvr := gv.WithResource("databases")
	list := []byte(`{"apiVersion":"offline.example.com/v1","kind":"DatabaseCollection","metadata":{"resourceVersion":"1"},` +
		`"items":[{"apiVersion":"offline.example.com/v1","kind":"Database","metadata":{"name":"db1","namespace":"default"}}]}`)
	patches := fakeCloud(t, true, cloudServing(gv, discoveryDoc(t, gv, "databases", "Database"), list))
	defer patches.Reset()
	fakeLocalStorage(patches)

	ctx := resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource)
	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	_, err := rest.List(ctx, &metainternalversion.ListOptions{})
	require.NoError(t, err)

	assert.Equal(t, gv.WithKind("Database"), util.KindFor(gvr))
	// What the local storage, sqlite/store.go, and decorateList use offline.
	assert.Equal(t, gv.WithKind("DatabaseCollection"), util.ListKindFor(gvr))
	local := &unstructured.UnstructuredList{}
	decorateList(ctx, local)
	assert.Equal(t, "DatabaseCollection", local.GetKind())
}

func TestLearnListKindOnlyFromResourceRequests(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "guard.example.com", Version: "v1", Resource: "widgets"}
	list := &unstructured.UnstructuredList{}
	list.SetAPIVersion("guard.example.com/v1")
	list.SetKind("WidgetCollection")

	learnListKind(context.TODO(), list)
	learnListKind(request.WithRequestInfo(context.TODO(), &request.RequestInfo{Path: "/version"}), list)
	status := resourceRequest("update", gvr.Group, gvr.Version, gvr.Resource)
	info, _ := request.RequestInfoFrom(status)
	info.Subresource = "status"
	learnListKind(status, list)
	assert.NotEqual(t, "WidgetCollection", util.ListKindFor(gvr).Kind)

	learnListKind(resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource), list)
	assert.Equal(t, "WidgetCollection", util.ListKindFor(gvr).Kind)
}

// TestGetLearnsMappingBeforeSaving checks that a get of a resource whose kind
// is not known fetches the discovery document of its group version before the
// object is saved, so that it is saved under the requested resource.
func TestGetLearnsMappingBeforeSaving(t *testing.T) {
	gv := schema.GroupVersion{Group: "get.example.com", Version: "v1"}
	gvr := gv.WithResource("apikeys")
	obj := []byte(`{"apiVersion":"get.example.com/v1","kind":"APIKey","metadata":{"name":"k","namespace":"default","resourceVersion":"1"}}`)
	patches := fakeCloud(t, true, cloudServing(gv, discoveryDoc(t, gv, "apikeys", "APIKey"), obj))
	defer patches.Reset()
	var savedUnder schema.GroupVersionResource
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(context.Context, []byte, string) error { return nil },
		InsertOrUpdateObjF: func(_ context.Context, o runtime.Object) error {
			savedUnder = util.ResourceFor(o.GetObjectKind().GroupVersionKind())
			return nil
		},
	})

	ctx := request.WithRequestInfo(context.TODO(), &request.RequestInfo{
		IsResourceRequest: true, Verb: "get", APIPrefix: "apis",
		APIGroup: gvr.Group, APIVersion: gvr.Version, Resource: gvr.Resource, Namespace: "default", Name: "k",
	})
	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	_, err := rest.Get(ctx, "", &metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, gvr, savedUnder, "apikeys, not the guessed apikeies")
}

// TestEnsureMappingRequestsDiscoveryOnce checks that the discovery document of
// a group version that does not resolve the watched resource is not requested
// again for every watch.
func TestEnsureMappingRequestsDiscoveryOnce(t *testing.T) {
	gv := schema.GroupVersion{Group: "once.metrics.k8s.io", Version: "v1beta1"}
	// Two resources share the kind: the document does not resolve them.
	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv.String(),
		APIResources: []metav1.APIResource{
			{Name: "queue_length", Namespaced: true, Kind: "ExternalMetricValueList"},
			{Name: "requests", Namespaced: true, Kind: "ExternalMetricValueList"},
		},
	})
	require.NoError(t, err)

	requests := 0
	patches := fakeCloud(t, true, func(key string) ([]byte, bool) {
		if key == "/apis/once.metrics.k8s.io/v1beta1" {
			requests++
			return doc, true
		}
		return nil, false
	})
	defer patches.Reset()
	patches.ApplyGlobalVar(&imitator.DefaultV2Client, fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(context.Context, []byte, string) error { return nil },
	})

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	info, _ := request.RequestInfoFrom(resourceRequest("watch", gv.Group, gv.Version, "queue_length"))
	rest.ensureMapping(context.TODO(), info)
	rest.ensureMapping(context.TODO(), info)

	assert.Equal(t, 1, requests)
	assert.False(t, util.DefaultRESTMapper().HasMapping(gv.WithResource("queue_length")))
}
