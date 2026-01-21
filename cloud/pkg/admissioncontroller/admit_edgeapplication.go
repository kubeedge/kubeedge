package admissioncontroller

import (
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"

	"github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

func admitEdgeApplication(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	switch review.Request.Operation {
	case admissionv1.Create, admissionv1.Update:
		raw := review.Request.Object.Raw
		edgeapp := v1alpha1.EdgeApplication{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &edgeapp); err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}

		return admissionResponse(validateEdgeApplication(edgeapp.Spec.WorkloadScope.TargetNodeGroups))

	case admissionv1.Delete:
		//no rule defined for above operations, greenlight for all of above.
		return admissionResponse(nil)
	default:
		err := fmt.Errorf("unsupported webhook operation %v", review.Request.Operation)
		return admissionResponse(err)
	}
}

func validateEdgeApplication(targetNodeGroups []v1alpha1.TargetNodeGroup) error {
	names := make(map[string]bool)
	for _, targetNodeGroup := range targetNodeGroups {
		name := targetNodeGroup.Name
		if names[name] {
			return fmt.Errorf("duplicate Name '%s' found in TargetNodeGroups", name)
		}
		names[name] = true
	}
	return nil
}

func serveEdgeApplication(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitEdgeApplication)
}
