package metaserver

import (
	"context"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func TestKeyFuncObj(t *testing.T) {
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
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			if test.stdResult != key {
				t.Errorf("KeyFuncObj Case failed,wanted result:%+v,acctual result:%+v", test.stdResult, key)
			}
		})
	}
}

func TestKeyFuncReq(t *testing.T) {
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
	}
	resolver := newTestRequestInfoResolver()
	for k, v := range Cases {
		t.Run("parseKey", func(t *testing.T) {
			req, err := http.NewRequest(v.method, v.url, nil)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			apiRequestInfo, err := resolver.NewRequestInfo(req)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			ctx := request.WithRequestInfo(context.TODO(), apiRequestInfo)
			key, err := KeyFuncReq(ctx, "")
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			if key != stdResult[k] {
				t.Errorf("failed to parse req context, wanted(%v),get(%v)", stdResult[k], key)
			}
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
			if test.stdResult != parseResult {
				t.Errorf("ParseKey Case failed, key:%v,wanted result:%+v,acctual result:%+v", test.key, test.stdResult, parseResult)
			}
		})
	}
}
