package admissioncontroller

import (
	"encoding/json"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type patchMapValue struct {
	Op    string        `json:"op"`
	Path  string        `json:"path"`
	Value []interface{} `json:"value,omitempty"`
}

// Offline migration adds a node-unreachable toleration to autonomy Pods. The
// JSON Patch operation must match the state of the original Pod: RFC 6902
// "replace" requires the target member to already exist, while "add" creates
// a missing member. Kubernetes omits spec.tolerations when the Pod has no
// tolerations, so using replace unconditionally makes an otherwise valid Pod
// mutation fail. Keep this distinction covered by tests whenever the Pod
// mutation or its serialization changes.

func mutateOfflineMigration(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{
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
	pt := admissionv1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt
	return &reviewResponse
}

func generatePatch(tolerations []corev1.Toleration) []patchMapValue {
	currentTolerations := make([]interface{}, 0, len(tolerations)+1)
	for _, v := range tolerations {
		if v.Key == corev1.TaintNodeUnreachable {
			continue
		}
		currentTolerations = append(currentTolerations, v)
	}
	currentTolerations = append(currentTolerations, corev1.Toleration{
		Key:      corev1.TaintNodeUnreachable,
		Operator: corev1.TolerationOpExists,
	})
	operation := "replace"
	if len(tolerations) == 0 {
		operation = "add"
	}
	patch := []patchMapValue{
		{
			Op:    operation,
			Path:  "/spec/tolerations",
			Value: currentTolerations,
		},
	}
	return patch
}

func serveOfflineMigration(w http.ResponseWriter, r *http.Request) {
	serve(w, r, mutateOfflineMigration)
}
