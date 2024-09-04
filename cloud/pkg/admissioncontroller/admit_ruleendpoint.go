package admissioncontroller

import (
	"fmt"
	"net/http"
	"strconv"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"

	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
)

func admitRuleEndpoint(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{}
	switch review.Request.Operation {
	case admissionv1.Create:
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
	case admissionv1.Delete, admissionv1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		return &reviewResponse
	default:
		err := fmt.Errorf("unsupported webhook operation %v", review.Request.Operation)
		klog.Warning(err)
		return toAdmissionResponse(err)
	}
}

func validateRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case rulesv1.RuleEndpointTypeServiceBus:
		portStr, exist := ruleEndpoint.Spec.Properties["service_port"]
		if !exist {
			return fmt.Errorf("\"service_port\" property missed in property when ruleEndpoint is \"servicebus\"")
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("port should be integer")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be in range 1-65535")
		}
	}
	return nil
}

func serveRuleEndpoint(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitRuleEndpoint)
}
