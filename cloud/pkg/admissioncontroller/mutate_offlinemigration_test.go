package admissioncontroller

import (
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGeneratePatchUsesAddWhenTolerationsAreAbsent(t *testing.T) {
	patch := generatePatch(nil)
	if len(patch) != 1 {
		t.Fatalf("expected one patch operation, got %d", len(patch))
	}
	if patch[0].Op != "add" {
		t.Fatalf("expected add operation for an absent tolerations field, got %q", patch[0].Op)
	}
	if patch[0].Path != "/spec/tolerations" {
		t.Fatalf("unexpected patch path %q", patch[0].Path)
	}
}

func TestGeneratePatchUsesReplaceWhenTolerationsExist(t *testing.T) {
	patch := generatePatch([]corev1.Toleration{{Key: "example.com/key"}})
	if len(patch) != 1 {
		t.Fatalf("expected one patch operation, got %d", len(patch))
	}
	if patch[0].Op != "replace" {
		t.Fatalf("expected replace operation for an existing tolerations field, got %q", patch[0].Op)
	}
}

func TestMutateOfflineMigrationAddsTolerationsToPodWithoutTolerations(t *testing.T) {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "autonomy-pod",
			Namespace: "default",
		},
	}
	raw, err := json.Marshal(pod)
	if err != nil {
		t.Fatal(err)
	}

	response := mutateOfflineMigration(admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object:    runtime.RawExtension{Raw: raw},
		},
	})

	if !response.Allowed {
		t.Fatalf("expected Pod mutation to be allowed: %v", response.Result)
	}
	if response.PatchType == nil || *response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Fatalf("expected JSON patch type, got %v", response.PatchType)
	}
	var patch []patchMapValue
	if err := json.Unmarshal(response.Patch, &patch); err != nil {
		t.Fatal(err)
	}
	if len(patch) != 1 || patch[0].Op != "add" {
		t.Fatalf("expected one add operation for missing tolerations, got %#v", patch)
	}
}
