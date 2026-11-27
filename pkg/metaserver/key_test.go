package metaserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

func TestKeyFuncObj(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		// group version kind namespace name
		attr      []string
		stdResult string
	}{
		{
			attr:      []string{"", "v1", "Pod", "default", "pods-foo"},
			stdResult: "/core/v1/pods/default/pods-foo",
		},
		{
			attr:      []string{"", "v1", "Endpoints", "default", "pods-foo"},
			stdResult: "/core/v1/endpoints/default/pods-foo",
		},
		{
			attr:      []string{"", "v1", "Configmap", "default", "pods-foo"},
			stdResult: "/core/v1/configmaps/default/pods-foo",
		},
		{
			attr:      []string{"", "v1", "KindFoo", "ns-bar", "name-whatever"},
			stdResult: "/core/v1/kindfoos/ns-bar/name-whatever",
		},
		{
			attr:      []string{"apps", "v1", "Deployment", "ns-bar", "name-whatever"},
			stdResult: "/apps/v1/deployments/ns-bar/name-whatever",
		},
		{
			attr:      []string{"", "v1", "KindFoo", "", "name-whatever"},
			stdResult: "/core/v1/kindfoos/null/name-whatever",
		},
	}
	for _, test := range cases {
		t.Run("parseKey", func(t *testing.T) {
			var obj unstructured.Unstructured
			gvk := schema.GroupVersionKind{
				Group:   test.attr[0],
				Version: test.attr[1],
				Kind:    test.attr[2],
			}
			obj.SetGroupVersionKind(gvk)
			obj.SetNamespace(test.attr[3])
			obj.SetName(test.attr[4])

			key, err := KeyFuncObj(&obj)
			assert.NoError(err)
			assert.Equal(test.stdResult, key)
		})
	}
}

func TestKeyFuncReq(t *testing.T) {
	assert := assert.New(t)

	namespaceAll := metav1.NamespaceAll
	Cases := []struct { //copy by requestinfo_test.go
		method              string
		url                 string
		expectedVerb        string
		expectedAPIPrefix   string
		expectedAPIGroup    string
		expectedAPIVersion  string
		expectedNamespace   string
		expectedResource    string
		expectedSubresource string
		expectedName        string
		expectedParts       []string
	}{
		// resource paths
		{"GET", "/api/v1/namespaces", "list", "api", "", "v1", "", "namespaces", "", "", []string{"namespaces"}},
		{"GET", "/api/v1/namespaces/other", "get", "api", "", "v1", "other", "namespaces", "", "other", []string{"namespaces", "other"}},

		{"GET", "/api/v1/namespaces/other/pods", "list", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/namespaces/other/pods/foo", "get", "api", "", "v1", "other", "pods", "", "foo", []string{"pods", "foo"}},
		{"HEAD", "/api/v1/namespaces/other/pods/foo", "get", "api", "", "v1", "other", "pods", "", "foo", []string{"pods", "foo"}},
		{"GET", "/api/v1/pods", "list", "api", "", "v1", namespaceAll, "pods", "", "", []string{"pods"}},
		{"HEAD", "/api/v1/pods", "list", "api", "", "v1", namespaceAll, "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/namespaces/other/pods/foo", "get", "api", "", "v1", "other", "pods", "", "foo", []string{"pods", "foo"}},
		{"GET", "/api/v1/namespaces/other/pods", "list", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},

		// special verbs
		{"GET", "/api/v1/proxy/namespaces/other/pods/foo", "proxy", "api", "", "v1", "other", "pods", "", "foo", []string{"pods", "foo"}},
		{"GET", "/api/v1/proxy/namespaces/other/pods/foo/subpath/not/a/subresource", "proxy", "api", "", "v1", "other", "pods", "", "foo", []string{"pods", "foo", "subpath", "not", "a", "subresource"}},
		{"GET", "/api/v1/watch/pods", "watch", "api", "", "v1", namespaceAll, "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/pods?watch=true", "watch", "api", "", "v1", namespaceAll, "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/pods?watch=false", "list", "api", "", "v1", namespaceAll, "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/watch/namespaces/other/pods", "watch", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/namespaces/other/pods?watch=1", "watch", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},
		{"GET", "/api/v1/namespaces/other/pods?watch=0", "list", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},

		// deletecollection verb identification
		{"DELETE", "/api/v1/nodes", "deletecollection", "api", "", "v1", "", "nodes", "", "", []string{"nodes"}},
		{"DELETE", "/api/v1/nodes/node-foo", "deletecollection", "api", "", "v1", "", "nodes", "", "node-foo", []string{"nodes"}},
		{"DELETE", "/api/v1/namespaces", "deletecollection", "api", "", "v1", "", "namespaces", "", "", []string{"namespaces"}},
		{"DELETE", "/api/v1/namespaces/other/pods", "deletecollection", "api", "", "v1", "other", "pods", "", "", []string{"pods"}},
		{"DELETE", "/apis/extensions/v1/namespaces/other/pods", "deletecollection", "api", "extensions", "v1", "other", "pods", "", "", []string{"pods"}},

		// api group identification
		{"POST", "/apis/extensions/v1/namespaces/other/pods", "create", "api", "extensions", "v1", "other", "pods", "", "", []string{"pods"}},

		// api version identification
		{"POST", "/apis/extensions/v1beta3/namespaces/other/pods", "create", "api", "extensions", "v1beta3", "other", "pods", "", "", []string{"pods"}},

		// non-resource api pass through
		{method: "GET", url: "/version"},
	}
	stdResult := []string{
		"/core/v1/namespaces/null/null",
		"/core/v1/namespaces/null/other", //a namespace called other

		"/core/v1/pods/other/null",
		"/core/v1/pods/other/foo",
		"/core/v1/pods/other/foo",
		"/core/v1/pods/null/null",
		"/core/v1/pods/null/null",
		"/core/v1/pods/other/foo",
		"/core/v1/pods/other/null",

		"/core/v1/pods/other/foo",
		"/core/v1/pods/other/foo",
		"/core/v1/pods/null/null",
		"/core/v1/pods/null/null",
		"/core/v1/pods/null/null",
		"/core/v1/pods/other/null",
		"/core/v1/pods/other/null",
		"/core/v1/pods/other/null",

		"/core/v1/nodes/null/null",
		"/core/v1/nodes/null/node-foo",
		"/core/v1/namespaces/null/null",
		"/core/v1/pods/other/null",
		"/extensions/v1/pods/other/null",

		"/extensions/v1/pods/other/null",

		"/extensions/v1beta3/pods/other/null",

		"/version",
	}
	resolver := newTestRequestInfoResolver()
	for k, v := range Cases {
		t.Run("parseKey", func(t *testing.T) {
			req, err := http.NewRequest(v.method, v.url, nil)
			assert.NoError(err)

			apiRequestInfo, err := resolver.NewRequestInfo(req)
			assert.NoError(err)

			ctx := request.WithRequestInfo(context.TODO(), apiRequestInfo)
			key, err := KeyFuncReq(ctx, "")
			assert.NoError(err)
			assert.Equal(stdResult[k], key)
		})
	}
}
func newTestRequestInfoResolver() *request.RequestInfoFactory {
	return &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
}

// TestSaveMeta is function to initialize all global variable and test SaveMeta
func TestParseKey(t *testing.T) {
	assert := assert.New(t)

	type result struct {
		gvr       schema.GroupVersionResource
		namespace string
		name      string
	}
	cases := []struct {
		key       string
		stdResult result
	}{
		{
			// Success Case
			key: "/core/v1/pods/default/pod-foo",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				namespace: "default",
				name:      "pod-foo",
			},
		},
		{
			// Success Case
			key: "/core/v1/endpoints",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "endpoints",
				},
				namespace: "",
				name:      "",
			},
		},
		{
			// Success Case
			key: "/core/v1/endpoints/",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "endpoints",
				},
				namespace: "",
				name:      "",
			},
		},
		{
			// Success test
			key: "/core/v1/endpoints/default",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "endpoints",
				},
				namespace: "default",
				name:      "",
			},
		},
		{
			// Success test
			key: "/core/v1/endpoints/null/null",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "endpoints",
				},
				namespace: "",
				name:      "",
			},
		},
		{
			// Fail test
			key:       "/",
			stdResult: result{},
		},
		{
			// Fail test
			key:       "abc",
			stdResult: result{},
		},
		{
			// Fail test
			key:       "///////",
			stdResult: result{},
		},
		{
			// Specially success test, ParseKey is not responsible for verifying the validity of the content
			key: "/core/v1/endpoints",
			stdResult: result{
				gvr: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "endpoints",
				},
				namespace: "",
				name:      "",
			},
		},
	}

	// run the test cases
	for _, test := range cases {
		t.Run("parseKey", func(t *testing.T) {
			gvr, ns, name := ParseKey(test.key)
			parseResult := result{gvr, ns, name}
			assert.Equal(test.stdResult, parseResult)
		})
	}
}

// TestKeyFuncObjMatchesKeyFuncReq checks that an object is saved under the key
// it is requested with, which KeyFuncReq builds from the resource of the URL.
func TestKeyFuncObjMatchesKeyFuncReq(t *testing.T) {
	resolver := &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
	keyOfRequest := func(url string) string {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.NoError(t, err)
		info, err := resolver.NewRequestInfo(req)
		assert.NoError(t, err)
		key, err := KeyFuncReq(request.WithRequestInfo(context.TODO(), info), "")
		assert.NoError(t, err)
		return key
	}
	newObj := func(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion(apiVersion)
		obj.SetKind(kind)
		obj.SetNamespace(namespace)
		obj.SetName(name)
		return obj
	}

	// Types of the known schemes are mapped without having to learn them.
	for url, obj := range map[string]*unstructured.Unstructured{
		"/api/v1/namespaces/default/configmaps/foo":                        newObj("v1", "ConfigMap", "default", "foo"),
		"/api/v1/namespaces/default/endpoints/foo":                         newObj("v1", "Endpoints", "default", "foo"),
		"/apis/coordination.k8s.io/v1/namespaces/default/leases/foo":       newObj("coordination.k8s.io/v1", "Lease", "default", "foo"),
		"/apis/networking.k8s.io/v1/namespaces/default/ingresses/foo":      newObj("networking.k8s.io/v1", "Ingress", "default", "foo"),
		"/apis/devices.kubeedge.io/v1beta1/namespaces/default/devices/foo": newObj("devices.kubeedge.io/v1beta1", "Device", "default", "foo"),
	} {
		key, err := KeyFuncObj(obj)
		assert.NoError(t, err)
		assert.Equal(t, keyOfRequest(url), key, url)
	}

	// The resource of a CRD with an irregular plural is only known once learned.
	gateway := newObj("networking.istio.io/v1alpha3", "Gateway", "default", "foo")
	url := "/apis/networking.istio.io/v1alpha3/namespaces/default/gateways/foo"
	key, err := KeyFuncObj(gateway)
	assert.NoError(t, err)
	assert.Equal(t, "/networking.istio.io/v1alpha3/gatewaies/default/foo", key)

	util.DefaultRESTMapper().RegisterMapping(schema.GroupVersionResource{
		Group: "networking.istio.io", Version: "v1alpha3", Resource: "gateways",
	}, "Gateway")
	key, err = KeyFuncObj(gateway)
	assert.NoError(t, err)
	assert.Equal(t, keyOfRequest(url), key)
}
