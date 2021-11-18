package metaserver_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/helpers"
)

const (
	CrdHandler         = "/crd"
	CrdInstanceHandler = "/crdinstance"
)

var (
	gatewaysName       = "gateways.networking.istio.io"
	gatewaysKind       = "Gateway"
	gatewaysPlural     = "gateways"
	serviceentryName   = "serviceentries.networking.istio.io"
	serviceentryKind   = "ServiceEntry"
	serviceentryPlural = "serviceentries"
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
			url := "http://" + constants.DefaultMetaServerAddr
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
			isCRDDeployed := helpers.HandleAddAndDeleteCRDs(http.MethodPut, ctx.Cfg.TestManager+CrdHandler, gatewaysName, gatewaysKind, gatewaysPlural)
			Expect(isCRDDeployed).Should(BeTrue())
			isCRDDeployed = helpers.HandleAddAndDeleteCRDs(http.MethodPut, ctx.Cfg.TestManager+CrdHandler, serviceentryName, serviceentryKind, serviceentryPlural)
			Expect(isCRDDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			isCRDDeployed = helpers.HandleAddAndDeleteCRDInstances(http.MethodPut, ctx.Cfg.TestManager+CrdInstanceHandler, "test-gateway", gatewaysKind)
			Expect(isCRDDeployed).Should(BeTrue())
			isCRDDeployed = helpers.HandleAddAndDeleteCRDInstances(http.MethodPut, ctx.Cfg.TestManager+CrdInstanceHandler, "test-serviceentry", serviceentryKind)
			Expect(isCRDDeployed).Should(BeTrue())
		})
		AfterEach(func() {
			IsCRDDeleted := helpers.HandleAddAndDeleteCRDs(http.MethodDelete, ctx.Cfg.TestManager+CrdHandler, gatewaysName, gatewaysKind, gatewaysPlural)
			Expect(IsCRDDeleted).Should(BeTrue())
			IsCRDDeleted = helpers.HandleAddAndDeleteCRDs(http.MethodDelete, ctx.Cfg.TestManager+CrdHandler, serviceentryName, serviceentryKind, serviceentryPlural)
			Expect(IsCRDDeleted).Should(BeTrue())
			IsCRDDeleted = helpers.HandleAddAndDeleteCRDInstances(http.MethodDelete, ctx.Cfg.TestManager+CrdInstanceHandler, "test-gateway", gatewaysKind)
			Expect(IsCRDDeleted).Should(BeTrue())
			IsCRDDeleted = helpers.HandleAddAndDeleteCRDInstances(http.MethodDelete, ctx.Cfg.TestManager+CrdInstanceHandler, "test-serviceentry", serviceentryKind)
			Expect(IsCRDDeleted).Should(BeTrue())
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

			client := http.Client{}
			url := "http://" + constants.DefaultMetaServerAddr
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
