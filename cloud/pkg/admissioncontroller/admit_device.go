package admissioncontroller

import (
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func admitDevice(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	var msg string

	switch review.Request.Operation {
	case admissionv1.Create, admissionv1.Update:
		// TODO add admission for device v1beta1
	case admissionv1.Delete, admissionv1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		reviewResponse.Allowed = false
		msg = "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func serveDevice(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDevice)
}
