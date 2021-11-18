package admissioncontroller

import (
	"fmt"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/klog/v2"

	rulesv1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
)

func admitRuleEndpoint(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	switch review.Request.Operation {
	case admissionv1beta1.Create:
		raw := review.Request.Object.Raw
		ruleEndpoint := rulesv1.RuleEndpoint{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &ruleEndpoint); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			return toAdmissionResponse(err)
		}
		err := validateRuleEndpoint(&ruleEndpoint)
		if err != nil {
			return toAdmissionResponse(err)
		}
		reviewResponse.Allowed = true
		return &reviewResponse
	case admissionv1beta1.Delete, admissionv1beta1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		return &reviewResponse
	default:
		err := fmt.Errorf("Unsupported webhook operation %v", review.Request.Operation)
		klog.Warning(err)
		return toAdmissionResponse(err)
	}
}

func validateRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case rulesv1.RuleEndpointTypeServiceBus:
		_, exist := ruleEndpoint.Spec.Properties["service_port"]
		if !exist {
			return fmt.Errorf("\"service_port\" property missed in property when ruleEndpoint is \"servicebus\"")
		}
	}
	return nil
}

func serveRuleEndpoint(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitRuleEndpoint)
}
