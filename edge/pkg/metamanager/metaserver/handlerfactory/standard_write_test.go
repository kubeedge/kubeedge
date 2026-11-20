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

package handlerfactory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/descriptorpb"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

// What client-go sends for a built-in kind when it is configured for protobuf.
const clientGoProtobufAccept = "application/vnd.kubernetes.protobuf, application/json"

// fakeStorage replaces the storage's Create and Update and records the object
// each one received, which is what MetaServer sends to the cloud.
type fakeStorage struct {
	patches *gomonkey.Patches
	got     runtime.Object
	calls   int
	err     error
}

func newFakeStorage(t *testing.T) *fakeStorage {
	fs := &fakeStorage{patches: gomonkey.NewPatches()}
	t.Cleanup(fs.patches.Reset)

	fs.patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{Store: &genericregistry.Store{
			NewFunc: func() runtime.Object { return &unstructured.Unstructured{} },
		}}, nil
	})
	fs.patches.ApplyMethod((*storage.REST)(nil), "Create",
		func(_ *storage.REST, _ context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
			fs.calls++
			fs.got = obj
			if fs.err != nil {
				return nil, fs.err
			}
			return obj, nil
		})
	fs.patches.ApplyMethod((*storage.REST)(nil), "Update",
		func(_ *storage.REST, ctx context.Context, _ string, objInfo rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
			fs.calls++
			obj, err := objInfo.UpdatedObject(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			fs.got = obj
			if fs.err != nil {
				return nil, false, fs.err
			}
			return obj, false, nil
		})
	return fs
}

// gotContent returns the object the storage received as a map, as it goes to the cloud.
func (fs *fakeStorage) gotContent(t *testing.T) map[string]interface{} {
	t.Helper()
	require.Equal(t, 1, fs.calls, "storage was not called exactly once")
	u, ok := fs.got.(*unstructured.Unstructured)
	require.True(t, ok, "storage got %T, want *unstructured.Unstructured", fs.got)
	return u.UnstructuredContent()
}

// protobufBody encodes obj as client-go does, with its apiVersion and kind in the envelope.
func protobufBody(t *testing.T, obj runtime.Object) []byte {
	t.Helper()
	gvks, _, err := kubescheme.Scheme.ObjectKinds(obj)
	require.NoError(t, err)
	obj.GetObjectKind().SetGroupVersionKind(gvks[0])
	body := &bytes.Buffer{}
	require.NoError(t, protobuf.NewSerializer(kubescheme.Scheme, kubescheme.Scheme).Encode(obj, body))
	return body.Bytes()
}

func serve(h http.Handler, method, path string, info *request.RequestInfo, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", clientGoProtobufAccept)
	req = req.WithContext(request.WithRequestInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func nodeUpdateInfo(name string) *request.RequestInfo {
	return &request.RequestInfo{IsResourceRequest: true, Verb: "update", APIVersion: "v1", Resource: "nodes", Name: name}
}

func eventCreateInfo() *request.RequestInfo {
	return &request.RequestInfo{IsResourceRequest: true, Verb: "create", APIVersion: "v1", Resource: "events", Namespace: "default"}
}

func calicoCreateInfo(resource string) *request.RequestInfo {
	return &request.RequestInfo{IsResourceRequest: true, Verb: "create", APIGroup: "crd.projectcalico.org", APIVersion: "v1", Resource: resource}
}

// The case this change exists for: calico-node PUTs its v1.Node as protobuf.
// The cloud gets it as an object with apiVersion and kind, and the answer is JSON.
func TestUpdateProtobufNode(t *testing.T) {
	fs := newFakeStorage(t)
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "edge-34-237-223-165",
			Annotations: map[string]string{"projectcalico.org/IPv4Address": "10.100.120.155/24"},
		},
		Spec: corev1.NodeSpec{PodCIDR: "10.4.3.0/24"},
	}

	w := serve(NewFactory().Update(nodeUpdateInfo(node.Name)), http.MethodPut, "/api/v1/nodes/"+node.Name,
		nodeUpdateInfo(node.Name), runtime.ContentTypeProtobuf, protobufBody(t, node))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, runtime.ContentTypeJSON, w.Header().Get("Content-Type"))
	got := fs.gotContent(t)
	assert.Equal(t, "v1", got["apiVersion"])
	assert.Equal(t, "Node", got["kind"])
	podCIDR, _, _ := unstructured.NestedString(got, "spec", "podCIDR")
	assert.Equal(t, "10.4.3.0/24", podCIDR)
	address, _, _ := unstructured.NestedString(got, "metadata", "annotations", "projectcalico.org/IPv4Address")
	assert.Equal(t, "10.100.120.155/24", address)
}

// kube-proxy posts its startup Event as protobuf.
func TestCreateProtobufEvent(t *testing.T) {
	fs := newFakeStorage(t)
	event := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: "kube-proxy.1", Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{Kind: "Node", Name: "edge-1"},
		Reason:         "Starting",
	}

	w := serve(NewFactory().Create(eventCreateInfo()), http.MethodPost, "/api/v1/namespaces/default/events",
		eventCreateInfo(), runtime.ContentTypeProtobuf, protobufBody(t, event))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	got := fs.gotContent(t)
	assert.Equal(t, "Event", got["kind"])
	assert.Equal(t, "Starting", got["reason"])
}

// A JSON body is not decoded into edgecore's types, so a field those types do
// not have, such as one added by a newer Kubernetes, still reaches the cloud.
func TestCreateJSONKeepsFieldsEdgecoreDoesNotKnow(t *testing.T) {
	fs := newFakeStorage(t)
	body := `{"apiVersion":"v1","kind":"Event","metadata":{"name":"e","namespace":"default"},"reason":"Starting","fieldFromANewerRelease":"kept"}`

	w := serve(NewFactory().Create(eventCreateInfo()), http.MethodPost, "/api/v1/namespaces/default/events",
		eventCreateInfo(), runtime.ContentTypeJSON, []byte(body))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	got := fs.gotContent(t)
	assert.Equal(t, "kept", got["fieldFromANewerRelease"])
	assert.NotContains(t, got, "source", "JSON bodies must not gain the zero-value fields of a typed object")
}

// Calico's own writes are custom resources, which are never protobuf. Their
// JSON bodies reach the cloud as they were sent.
func TestCreateJSONCustomResource(t *testing.T) {
	fs := newFakeStorage(t)
	const block = `{"apiVersion":"crd.projectcalico.org/v1","kind":"IPAMBlock",` +
		`"metadata":{"name":"10-4-4-0-26"},"spec":{"cidr":"10.4.4.0/26","affinity":"host:edge-52-21-33-192"}}`

	w := serve(NewFactory().Create(calicoCreateInfo("ipamblocks")), http.MethodPost, "/apis/crd.projectcalico.org/v1/ipamblocks",
		calicoCreateInfo("ipamblocks"), runtime.ContentTypeJSON, []byte(block))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	got := fs.gotContent(t)
	spec, _, _ := unstructured.NestedStringMap(got, "spec")
	assert.Equal(t, map[string]string{"cidr": "10.4.4.0/26", "affinity": "host:edge-52-21-33-192"}, spec)
}

func TestCreateYAMLCustomResource(t *testing.T) {
	fs := newFakeStorage(t)
	const pool = "apiVersion: crd.projectcalico.org/v1\nkind: IPPool\nmetadata:\n  name: default-ipv4\nspec:\n  cidr: 10.4.0.0/16\n"

	w := serve(NewFactory().Create(calicoCreateInfo("ippools")), http.MethodPost, "/apis/crd.projectcalico.org/v1/ippools",
		calicoCreateInfo("ippools"), "application/yaml", []byte(pool))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	got := fs.gotContent(t)
	assert.Equal(t, "IPPool", got["kind"])
	cidr, _, _ := unstructured.NestedString(got, "spec", "cidr")
	assert.Equal(t, "10.4.0.0/16", cidr)
}

// The cloud updates the object named in the body, so a body that names another
// object than the URL must be refused here.
func TestUpdateRefusesNameMismatch(t *testing.T) {
	for _, tc := range []struct {
		name        string
		contentType string
		body        []byte
	}{
		{"json", runtime.ContentTypeJSON, []byte(`{"apiVersion":"v1","kind":"Node","metadata":{"name":"node-b"}}`)},
		{"protobuf", runtime.ContentTypeProtobuf, protobufBody(t, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-b"}})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeStorage(t)
			w := serve(NewFactory().Update(nodeUpdateInfo("node-a")), http.MethodPut, "/api/v1/nodes/node-a",
				nodeUpdateInfo("node-a"), tc.contentType, tc.body)

			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Equal(t, 0, fs.calls)
		})
	}
}

func TestCreateRefusesBadBodies(t *testing.T) {
	unknownKind := &runtime.Unknown{
		TypeMeta: runtime.TypeMeta{APIVersion: "example.com/v1", Kind: "Widget"},
		Raw:      []byte{},
	}
	unknownKindBody := &bytes.Buffer{}
	require.NoError(t, protobuf.NewSerializer(kubescheme.Scheme, kubescheme.Scheme).Encode(unknownKind, unknownKindBody))

	for _, tc := range []struct {
		name        string
		contentType string
		body        []byte
		wantCode    int
	}{
		{"protobuf of a kind that is not built in", runtime.ContentTypeProtobuf, unknownKindBody.Bytes(), http.StatusUnsupportedMediaType},
		{"protobuf content type, not protobuf", runtime.ContentTypeProtobuf, []byte(`{"kind":"Event"}`), http.StatusBadRequest},
		{"truncated protobuf", runtime.ContentTypeProtobuf, []byte("k8s\x00\x0a\x05"), http.StatusBadRequest},
		{"malformed JSON", runtime.ContentTypeJSON, []byte(`{"kind":`), http.StatusBadRequest},
		{"JSON without kind", runtime.ContentTypeJSON, []byte(`{"metadata":{"name":"e"}}`), http.StatusBadRequest},
		{"unsupported content type", "text/plain", []byte("hello"), http.StatusUnsupportedMediaType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeStorage(t)
			w := serve(NewFactory().Create(eventCreateInfo()), http.MethodPost, "/api/v1/namespaces/default/events",
				eventCreateInfo(), tc.contentType, tc.body)

			assert.Equal(t, tc.wantCode, w.Code, w.Body.String())
			assert.Equal(t, 0, fs.calls)
		})
	}
}

func TestCreateRefusesTooLargeProtobufBody(t *testing.T) {
	fs := newFakeStorage(t)
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "e", Namespace: "default"},
		Message:    strings.Repeat("x", 4*1024*1024),
	}

	w := serve(NewFactory().Create(eventCreateInfo()), http.MethodPost, "/api/v1/namespaces/default/events",
		eventCreateInfo(), runtime.ContentTypeProtobuf, protobufBody(t, event))

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Equal(t, 0, fs.calls)
}

func TestWriteStorageError(t *testing.T) {
	fs := newFakeStorage(t)
	fs.err = errors.New("cloud unreachable")

	w := serve(NewFactory().Create(eventCreateInfo()), http.MethodPost, "/api/v1/namespaces/default/events",
		eventCreateInfo(), runtime.ContentTypeProtobuf, protobufBody(t, &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "e"}}))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "cloud unreachable")

	w = serve(NewFactory().Update(nodeUpdateInfo("n")), http.MethodPut, "/api/v1/nodes/n",
		nodeUpdateInfo("n"), runtime.ContentTypeProtobuf, protobufBody(t, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "n"}}))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "cloud unreachable")
}

// The protobuf limit applies to the body as sent, not to its larger JSON form.
func TestCreateAcceptsProtobufBodyThatGrowsAsJSON(t *testing.T) {
	fs := newFakeStorage(t)
	info := &request.RequestInfo{IsResourceRequest: true, Verb: "create", APIVersion: "v1", Resource: "configmaps", Namespace: "default"}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "default"},
		BinaryData: map[string][]byte{"b": make([]byte, 2_600_000)}, // over 3 MiB once base64 in JSON
	}

	w := serve(NewFactory().Create(info), http.MethodPost, "/api/v1/namespaces/default/configmaps",
		info, runtime.ContentTypeProtobuf, protobufBody(t, cm))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "ConfigMap", fs.gotContent(t)["kind"])
}

// Kinds of built-in groups other than core reach storage too. Subresources such
// as pods/eviction are refused for JSON and protobuf alike, so they are not here.
func TestCreateProtobufOtherBuiltInGroups(t *testing.T) {
	for _, tc := range []struct {
		obj  runtime.Object
		info *request.RequestInfo
		path string
	}{
		{
			&coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "l", Namespace: "default"}},
			&request.RequestInfo{IsResourceRequest: true, Verb: "create", APIGroup: "coordination.k8s.io", APIVersion: "v1", Resource: "leases", Namespace: "default"},
			"/apis/coordination.k8s.io/v1/namespaces/default/leases",
		},
		{
			&discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"}, AddressType: discoveryv1.AddressTypeIPv4},
			&request.RequestInfo{IsResourceRequest: true, Verb: "create", APIGroup: "discovery.k8s.io", APIVersion: "v1", Resource: "endpointslices", Namespace: "default"},
			"/apis/discovery.k8s.io/v1/namespaces/default/endpointslices",
		},
	} {
		gvks, _, err := kubescheme.Scheme.ObjectKinds(tc.obj)
		require.NoError(t, err)
		want := gvks[0]
		t.Run(want.String(), func(t *testing.T) {
			fs := newFakeStorage(t)
			w := serve(NewFactory().Create(tc.info), http.MethodPost, tc.path, tc.info, runtime.ContentTypeProtobuf, protobufBody(t, tc.obj))

			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			got := fs.gotContent(t)
			assert.Equal(t, want.GroupVersion().String(), got["apiVersion"])
			assert.Equal(t, want.Kind, got["kind"])
		})
	}
}

// protobufWithExtraFields encodes raw, an encoded object of the given kind, with
// fields its type lacks, as a newer client would send them: extra appended to the
// object, and extraInField to its message field number field.
func protobufWithExtraFields(t *testing.T, apiVersion, kind string, raw, extra []byte, field protowire.Number, extraInField []byte) []byte {
	t.Helper()
	raw = append(raw, extra...)
	if extraInField != nil {
		// A singular message field that occurs twice is merged, so this adds to the one already in raw.
		raw = bytesField(raw, field, extraInField)
	}
	body := &bytes.Buffer{}
	require.NoError(t, protobuf.NewSerializer(kubescheme.Scheme, kubescheme.Scheme).Encode(&runtime.Unknown{
		TypeMeta: runtime.TypeMeta{APIVersion: apiVersion, Kind: kind},
		Raw:      raw,
	}, body))
	return body.Bytes()
}

func protobufNodeWithExtraFields(t *testing.T, extra, extraInSpec []byte) []byte {
	t.Helper()
	raw, err := (&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "n"}}).Marshal()
	require.NoError(t, err)
	return protobufWithExtraFields(t, "v1", "Node", raw, extra, 2, extraInSpec)
}

func bytesField(b []byte, num protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

func stringField(num protowire.Number, v string) []byte {
	return bytesField(nil, num, []byte(v))
}

func varintField(num protowire.Number, v uint64) []byte {
	b := protowire.AppendTag(nil, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

// Decoding a protobuf body into edgecore's types loses the fields they lack, and
// an update then removes them in the cloud. Such a body is refused, even when the
// field is 0 or empty: a newer pointer field set to 0 differs from an unset one.
func TestProtobufRefusesFieldsEdgecoreLacks(t *testing.T) {
	for _, tc := range []struct {
		name               string
		extra, extraInSpec []byte
		wantField          string
	}{
		{"in the object", stringField(99, "new"), nil, "(field 99)"},
		{"in a nested message", nil, stringField(99, "new"), "spec.(field 99)"},
		{"in a message the types lack", stringField(98, string(stringField(1, "new"))), nil, "(field 98)"},
		{"set to 0", nil, varintField(97, 0), "spec.(field 97)"},
		{"set to an empty string", stringField(98, ""), nil, "(field 98)"},
		{"set to an empty message", nil, stringField(99, string(stringField(1, ""))), "spec.(field 99)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeStorage(t)
			w := serve(NewFactory().Update(nodeUpdateInfo("n")), http.MethodPut, "/api/v1/nodes/n",
				nodeUpdateInfo("n"), runtime.ContentTypeProtobuf, protobufNodeWithExtraFields(t, tc.extra, tc.extraInSpec))

			assert.Equal(t, http.StatusUnsupportedMediaType, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), tc.wantField)
			assert.Contains(t, w.Body.String(), "send it as JSON")
			assert.Equal(t, 0, fs.calls)
		})
	}
}

// gogo's decoder skips a group whose end tag names another field, but the check
// cannot read it. That is the client's error, so 400 rather than 500.
func TestProtobufMalformedUnknownFieldIsBadRequest(t *testing.T) {
	extra := protowire.AppendTag(nil, 50, protowire.StartGroupType)
	extra = protowire.AppendTag(extra, 51, protowire.EndGroupType)

	fs := newFakeStorage(t)
	w := serve(NewFactory().Update(nodeUpdateInfo("n")), http.MethodPut, "/api/v1/nodes/n",
		nodeUpdateInfo("n"), runtime.ContentTypeProtobuf, protobufNodeWithExtraFields(t, extra, nil))

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "failed to read the protobuf request body")
	assert.Equal(t, 0, fs.calls)
}

// Clients from Kubernetes 1.33 on always send Pod status.observedGeneration and
// status.conditions[].observedGeneration, which edgecore's types lack. A 0 there
// loses nothing, so the body is accepted; any other value is refused.
func TestProtobufAlwaysSentPodFields(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		generation, condGeneration uint64
		wantField                  string
	}{
		{"both 0", 0, 0, ""},
		{"status set", 3, 0, "status.(field 17)"},
		{"condition set", 0, 3, "status.conditions.(field 7)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := (&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default"}}).Marshal()
			require.NoError(t, err)
			cond, err := (&corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}).Marshal()
			require.NoError(t, err)
			cond = append(cond, varintField(7, tc.condGeneration)...)
			status := append(bytesField(nil, 2, cond), varintField(17, tc.generation)...)
			info := &request.RequestInfo{IsResourceRequest: true, Verb: "update", APIVersion: "v1", Resource: "pods", Namespace: "default", Name: "p"}

			fs := newFakeStorage(t)
			w := serve(NewFactory().Update(info), http.MethodPut, "/api/v1/namespaces/default/pods/p",
				info, runtime.ContentTypeProtobuf, protobufWithExtraFields(t, "v1", "Pod", raw, nil, 3, status))

			if tc.wantField == "" {
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				conditions, _, _ := unstructured.NestedSlice(fs.gotContent(t), "status", "conditions")
				assert.Len(t, conditions, 1)
				return
			}
			assert.Equal(t, http.StatusUnsupportedMediaType, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), tc.wantField)
			assert.Equal(t, 0, fs.calls)
		})
	}
}

// Clients from Kubernetes 1.37 on always send CSINode status, which edgecore's
// types lack. An empty status loses nothing, so the body is accepted.
func TestProtobufAlwaysSentCSINodeStatus(t *testing.T) {
	for _, version := range []struct {
		name string
		obj  interface{ Marshal() ([]byte, error) }
	}{
		{"v1", &storagev1.CSINode{ObjectMeta: metav1.ObjectMeta{Name: "n"}}},
		{"v1beta1", &storagev1beta1.CSINode{ObjectMeta: metav1.ObjectMeta{Name: "n"}}},
	} {
		for _, tc := range []struct {
			name      string
			status    []byte
			wantField string
		}{
			{"empty", nil, ""},
			{"storageHealth set", stringField(1, string(stringField(1, "driver"))), "(field 3)"},
		} {
			t.Run(version.name+" "+tc.name, func(t *testing.T) {
				raw, err := version.obj.Marshal()
				require.NoError(t, err)
				info := &request.RequestInfo{IsResourceRequest: true, Verb: "update", APIGroup: "storage.k8s.io", APIVersion: version.name, Resource: "csinodes", Name: "n"}

				fs := newFakeStorage(t)
				w := serve(NewFactory().Update(info), http.MethodPut, "/apis/storage.k8s.io/"+version.name+"/csinodes/n",
					info, runtime.ContentTypeProtobuf, protobufWithExtraFields(t, "storage.k8s.io/"+version.name, "CSINode", raw, bytesField(nil, 3, tc.status), 0, nil))

				if tc.wantField == "" {
					require.Equal(t, http.StatusOK, w.Code, w.Body.String())
					return
				}
				assert.Equal(t, http.StatusUnsupportedMediaType, w.Code, w.Body.String())
				assert.Contains(t, w.Body.String(), tc.wantField)
				assert.Equal(t, 0, fs.calls)
			})
		}
	}
}

// The check needs the descriptor of every message a built-in kind can hold.
func TestProtobufDescriptorsOfAllBuiltInKinds(t *testing.T) {
	for gvk, typ := range kubescheme.Scheme.AllKnownTypes() {
		obj := reflect.New(typ).Interface()
		if _, ok := obj.(descriptorMessage); !ok {
			continue // not a protobuf message, so the protobuf decoder cannot produce it
		}
		name, err := builtInMessages.messageOf(obj)
		require.NoError(t, err, gvk.String())

		seen := map[string]bool{}
		var walk func(name string)
		walk = func(name string) {
			if seen[name] {
				return
			}
			seen[name] = true
			m, err := builtInMessages.lookup(name)
			require.NoError(t, err, "%v message %s", gvk, name)
			for _, f := range m.GetField() {
				if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
					walk(f.GetTypeName())
				}
			}
		}
		walk(name)
	}
}

// The always-sent fields must name real messages, or they never match.
func TestProtobufAlwaysSentFieldsNameKnownMessages(t *testing.T) {
	for _, typ := range kubescheme.Scheme.AllKnownTypes() {
		if obj, ok := reflect.New(typ).Interface().(descriptorMessage); ok {
			_, err := builtInMessages.messageOf(obj)
			require.NoError(t, err)
		}
	}
	for f := range alwaysSentFields {
		m, err := builtInMessages.lookup(f.message)
		require.NoError(t, err)
		assert.Nil(t, fieldByNumber(m, f.number), "%s field %d is now in edgecore's types", f.message, f.number)
	}
}

// A failure of the field check is a bug in edgecore, so it is logged as an error.
func TestProtobufCheckFailureIsLogged(t *testing.T) {
	patches := gomonkey.ApplyFunc(droppedProtobufFields, func([]byte, runtime.Object) ([]string, error) {
		return nil, errors.New("no descriptor")
	})
	defer patches.Reset()
	// The limiter is global, so a run with -count would otherwise use it up.
	limiter := failureErrors
	failureErrors = rate.NewLimiter(rate.Inf, 0)
	defer func() { failureErrors = limiter }()
	var logged string
	patches.ApplyFunc(klog.Errorf, func(format string, args ...interface{}) { logged = fmt.Sprintf(format, args...) })

	fs := newFakeStorage(t)
	w := serve(NewFactory().Update(nodeUpdateInfo("n")), http.MethodPut, "/api/v1/nodes/n",
		nodeUpdateInfo("n"), runtime.ContentTypeProtobuf, protobufNodeWithExtraFields(t, nil, nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	assert.Contains(t, logged, "failed to convert the protobuf body of PUT /api/v1/nodes/n")
	assert.Contains(t, logged, "no descriptor")
	assert.Equal(t, 0, fs.calls)
}
