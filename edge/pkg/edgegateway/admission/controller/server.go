package controller

import (
	"io/ioutil"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/klog/v2"
	"net/http"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	admissionv1beta1.AddToScheme(scheme)
	admissionv1.AddToScheme(scheme)
}

// AdmissionController checks if an object
// is allowed in the cluster
type AdmissionController interface {
	HandleAdmission(runtime.Object) (runtime.Object, error)
}

// AdmissionControllerServer implements an HTTP server
// for kubernetes validating webhook
// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook
type AdmissionControllerServer struct {
	AdmissionController AdmissionController
}

// NewAdmissionControllerServer instantiates an admission controller server with
// a default codec
func NewAdmissionControllerServer(ac AdmissionController) *AdmissionControllerServer {
	return &AdmissionControllerServer{
		AdmissionController: ac,
	}
}

// ServeHTTP implements http.Server method
func (acs *AdmissionControllerServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		klog.ErrorS(err, "Failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	codec := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{
		Pretty: true,
	})

	obj, _, err := codec.Decode(data, nil, nil)
	if err != nil {
		klog.ErrorS(err, "Failed to decode request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := acs.AdmissionController.HandleAdmission(obj)
	if err != nil {
		klog.ErrorS(err, "failed to process webhook request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := codec.Encode(result, w); err != nil {
		klog.ErrorS(err, "failed to encode response body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
