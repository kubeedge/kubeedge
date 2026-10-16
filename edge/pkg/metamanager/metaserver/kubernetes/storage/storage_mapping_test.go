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

func TestListLearnsMappingFromCloudResponse(t *testing.T) {
	// The list returned by the cloud is empty, its kind still tells the mapping.
	list := []byte(`{"apiVersion":"list.cilium.io/v2","kind":"CiliumNodeList","metadata":{"resourceVersion":"1"},"items":[]}`)
	patches := fakeCloud(t, true, func(string) ([]byte, bool) { return list, true })
	defer patches.Reset()

	gvr := schema.GroupVersionResource{Group: "list.cilium.io", Version: "v2", Resource: "ciliumnodes"}
	require.False(t, util.DefaultRESTMapper().HasMapping(gvr))

	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	obj, err := rest.List(resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource), &metainternalversion.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, "CiliumNodeList", obj.GetObjectKind().GroupVersionKind().Kind)

	assert.True(t, util.DefaultRESTMapper().HasMapping(gvr))
	assert.Equal(t, "CiliumNode", util.KindFor(gvr).Kind)
	assert.Equal(t, gvr, util.ResourceFor(gvr.GroupVersion().WithKind("CiliumNode")))
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
	list := []byte(`{"apiVersion":"offline.example.com/v1","kind":"DatabaseCollection","metadata":{"resourceVersion":"1"},` +
		`"items":[{"apiVersion":"offline.example.com/v1","kind":"Database","metadata":{"name":"db1","namespace":"default"}}]}`)
	patches := fakeCloud(t, true, func(string) ([]byte, bool) { return list, true })
	defer patches.Reset()

	gvr := schema.GroupVersionResource{Group: "offline.example.com", Version: "v1", Resource: "databases"}
	ctx := resourceRequest("list", gvr.Group, gvr.Version, gvr.Resource)
	rest := &REST{Agent: &agent.Agent{Applications: sync.Map{}}}
	_, err := rest.List(ctx, &metainternalversion.ListOptions{})
	require.NoError(t, err)

	assert.Equal(t, gvr.GroupVersion().WithKind("Database"), util.KindFor(gvr))
	// What the local storage, sqlite/store.go, and decorateList use offline.
	assert.Equal(t, gvr.GroupVersion().WithKind("DatabaseCollection"), util.ListKindFor(gvr))
	local := &unstructured.UnstructuredList{}
	decorateList(ctx, local)
	assert.Equal(t, "DatabaseCollection", local.GetKind())
}
