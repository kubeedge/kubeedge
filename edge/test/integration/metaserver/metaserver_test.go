package metaserver

import (
	"context"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

var (
	gw = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1beta1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": "gateways.networking.istio.io",
			},
			"spec": map[string]interface{}{
				"group": "networking.istio.io",
				"names": map[string]string{
					"kind":     "Gateway",
					"plural":   "gateways",
					"singular": "gateway",
				},
				"scope":   "Namespaced",
				"version": "v1alpha3",
			},
		},
	}

	se = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1beta1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": "serviceentries.networking.istio.io",
			},
			"spec": map[string]interface{}{
				"group": "networking.istio.io",
				"names": map[string]string{
					"kind":     "ServiceEntry",
					"plural":   "serviceentries",
					"singular": "serviceentry",
				},
				"scope":   "Namespaced",
				"version": "v1alpha3",
			},
		},
	}
)

var _ = Describe("Test MetaServer", func() {
	Context("Test Access MetaServer at local", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
		})
		It("Test NotFound response", func() {
			var (
				coreAPIPrefix       = "api"
				coreAPIGroupVersion = schema.GroupVersion{Group: "", Version: "v1"}
				prefix              = "apis"
				//testGroupVersion    = schema.GroupVersion{Group: "test-group", Version: "test-version"}
			)
			type T struct {
				Method string
				Path   string
				Status int
			}
			cases := map[string]T{
				// Positive checks to make sure everything is wired correctly
				"List Core Cluster-Scope API":   {"GET", "/" + coreAPIPrefix + "/" + coreAPIGroupVersion.Version + "/nodes", http.StatusOK},
				"List Core Namespace-Scope API": {"GET", "/" + coreAPIPrefix + "/" + coreAPIGroupVersion.Version + "/namespaces/ns/pods", http.StatusOK},
				"List Cluster-Scope API":        {"GET", "/" + prefix + "/apiextensions.k8s.io/v1beta1/customresourcedefinitions", http.StatusOK},
				"List Namespace-Scope API":      {"GET", "/" + prefix + "/apps/v1/namespaces/ns-foo/jobs", http.StatusOK},

				"Get Core Cluster-Scope API":   {"GET", "/" + coreAPIPrefix + "/" + coreAPIGroupVersion.Version + "/nodes/node-foo", http.StatusNotFound},
				"Get Core Namespace-Scope API": {"GET", "/" + coreAPIPrefix + "/" + coreAPIGroupVersion.Version + "/namespaces/ns/pods/pod-foo", http.StatusNotFound},
				"Get Cluster-Scope API":        {"GET", "/" + prefix + "/apiextensions.k8s.io/v1beta1/customresourcedefinitions/crd-foo", http.StatusNotFound},
				"Get Namespace-Scope API":      {"GET", "/" + prefix + "/apps/v1/namespaces/ns-foo/jobs/job-foo", http.StatusNotFound},

				"Get Core Cluster-Scope API with extra segment": {"GET", "/" + coreAPIPrefix + "/" + coreAPIGroupVersion.Version + "/nodes/node-foo/baz", http.StatusNotFound},
				//"Watch with bad method":                         {"POST", "/" + prefix + "/" + testGroupVersion.Group + "/" + testGroupVersion.Version + "/watch/namespaces/ns/simples/", http.StatusMethodNotAllowed},
				//"Watch param with bad method": {"POST", "/" + prefix + "/" + testGroupVersion.Group + "/" + testGroupVersion.Version + "/namespaces/ns-foo/simples?watch=true", http.StatusMethodNotAllowed},
			}
			client := http.Client{}
			url := "http://127.0.0.1:10550"
			for _, v := range cases {
				request, err := http.NewRequest(v.Method, url+v.Path, nil)
				Expect(err).Should(BeNil())
				response, err := client.Do(request)
				Expect(err).Should(BeNil())
				isEqual := v.Status == response.StatusCode
				Expect(isEqual).Should(BeTrue(), "Expected response status %v, Got %v", v.Status, response.Status)
			}
		})
	})

	Context("Test CRDMap in MetaServer", func() {
		BeforeEach(func() {
			err := imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), se)
			if err != nil {
				common.Fatalf("%s", err)
				return
			}
			err = imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), gw)
			if err != nil {
				common.Fatalf("%s", err)
				return
			}
			_ = util.InitCrdMap()
		})
		AfterEach(func() {
			_ = imitator.DefaultV2Client.DeleteObj(context.TODO(), se)
			_ = imitator.DefaultV2Client.DeleteObj(context.TODO(), gw)
		})
		It("Test CRD Map in MetaServer", func() {
			type T struct {
				kind     string
				resource string
			}
			cases := map[string]T{
				"usual Case: ServiceEntry": {"ServiceEntry", "serviceentries"},
				"Unusual Case: Gateway":    {"Gateway", "gateways"},
			}
			for _, v := range cases {
				Expect(strings.Compare(util.UnsafeResourceToKind(v.resource), v.kind)).Should(BeZero())
				Expect(strings.Compare(util.UnsafeKindToResource(v.kind), v.resource)).Should(BeZero())
			}
		})
	})
})
