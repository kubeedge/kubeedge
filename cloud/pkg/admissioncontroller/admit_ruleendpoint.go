package admissioncontroller

import (
	"net/http"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	rulesv1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
)

func admitRuleEndpoint(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	var msg string
	switch review.Request.Operation {
	case admissionv1beta1.Create:
		raw := review.Request.Object.Raw
		ruleEndpoint := rulesv1.RuleEndpoint{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &ruleEndpoint); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			msg = err.Error()
			break
		}
		reviewResponse.Allowed = true
		klog.Info("admission validation passed!")
	case admissionv1beta1.Delete, admissionv1beta1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		klog.Info("admission validation passed!")
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		msg = msg + "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func serveRuleEndpoint(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitRuleEndpoint)
}
