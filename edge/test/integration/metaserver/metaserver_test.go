package metaserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
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
		})
		AfterEach(func() {
		})
		It("Test CRD Map in MetaServer", func() {
			type T struct {
				Method string
				Path   string
				Kind   string
				Status int
			}
			cases := map[string]T{
				"Unusual Case: List ServiceEntry": {"GET", "/apis/networking.istio.io/v1alpha3/namespaces/default/serviceentries", "ServiceEntryList", http.StatusOK},
				"Unusual Case: List Gateway":      {"GET", "/apis/networking.istio.io/v1alpha3/namespaces/default/gateways", "GatewayList", http.StatusOK},
				"Unusual Case: Get ServiceEntry":  {"GET", "/apis/networking.istio.io/v1alpha3/namespaces/default/serviceentries/test-serviceentry", "ServiceEntry", http.StatusOK},
				"Unusual Case: Get Gateway":       {"GET", "/apis/networking.istio.io/v1alpha3/namespaces/default/gateways/test-gateway", "Gateway", http.StatusOK},
			}

			time.Sleep(time.Second * 90)
			client := http.Client{}
			url := "http://127.0.0.1:10550"
			for _, v := range cases {
				request, err := http.NewRequest(v.Method, url+v.Path, nil)
				Expect(err).Should(BeNil())
				response, err := client.Do(request)
				Expect(err).Should(BeNil())
				isEqual := v.Status == response.StatusCode
				Expect(isEqual).Should(BeTrue(), "Expected response status %v, Got %v", v.Status, response.Status)

				contents, err := ioutil.ReadAll(response.Body)
				if err != nil {
					common.Fatalf("HTTP Response reading has failed: %v", err)
				}
				var obj *unstructured.Unstructured
				err = json.Unmarshal(contents, &obj)
				isEqual = obj.GetKind() == v.Kind
				Expect(isEqual).Should(BeTrue(), "Expected response kind %v, Got %v", v.Kind, obj.GetKind())
			}
		})
	})
})
