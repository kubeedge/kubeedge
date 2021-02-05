package admissioncontroller

import (
	"encoding/json"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const (
	Exists    = "Exists"
	NoExecute = "NoExecute"
)

type patchMapValue struct {
	Op    string              `json:"op"`
	Path  string              `json:"path"`
	Value []map[string]string `json:"value,omitempty"`
}

func mutateOfflineMigration(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}

	var pod corev1.Pod
	if err := json.Unmarshal(review.Request.Object.Raw, &pod); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		return toAdmissionResponse(err)
	}

	payload := generatePatch(pod.Spec.Tolerations)

	if len(payload) == 0 {
		return &reviewResponse
	}

	patch, err := json.Marshal(payload)
	if err != nil {
		return toAdmissionResponse(err)
	}

	reviewResponse.Patch = patch
	pt := admissionv1beta1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt
	return &reviewResponse
}

func generatePatch(tolerations []corev1.Toleration) []patchMapValue {
	patch := []patchMapValue{{
		Op:   "add",
		Path: "/spec/template/spec/tolerations",
		Value: []map[string]string{{
			"key":      corev1.TaintNodeUnreachable,
			"operator": Exists,
			"effect":   NoExecute,
		}},
	}}
	if len(tolerations) > 0 {
		for _, toleration := range tolerations {
			if toleration.Key == corev1.TaintNodeUnreachable {
				if toleration.Effect == "NoExecute" &&
					toleration.Operator == "Exists" && toleration.TolerationSeconds == nil {
					return nil
				}
				toleration.TolerationSeconds = nil
				patch[0].Op = "replace"
			}
		}
	}

	return patch
}

func serveOfflineMigration(w http.ResponseWriter, r *http.Request) {
	serve(w, r, mutateOfflineMigration)
}
