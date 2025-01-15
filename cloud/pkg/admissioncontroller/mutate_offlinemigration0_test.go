package main

import (
	"testing"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMutateOfflineMigration(t *testing.T) {
	// Create a sample AdmissionReview object
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: "12345",
			Kind: metav1.GroupVersionKind{
				Group:   "example.com",
				Version: "v1",
				Kind:    "ExampleResource",
			},
			Resource: metav1.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "exampleresources",
			},
			Operation: admissionv1.Create,
		},
	}

	// Call the function under test
	response := mutateOfflineMigration(review)

	// Check if the response is as expected
	if response == nil {
		t.Fatal("Expected a non-nil AdmissionResponse")
	}

	if !response.Allowed {
		t.Error("Expected Allowed to be true, got false")
	}
}