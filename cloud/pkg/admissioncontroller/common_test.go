package admissioncontroller

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"

	"github.com/kubeedge/kubeedge/cloud/test/httpfake"
)

func TestServe(t *testing.T) {
	w := httpfake.NewResponseWriter()
	hookfn := func(admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	t.Run("not json content-type", func(_ *testing.T) {
		serve(w, &http.Request{
			Header: map[string][]string{
				"Content-Type": {"application/xml"},
			},
		}, hookfn)
	})

	t.Run("decode body failed", func(_ *testing.T) {
		raw := "{"
		serve(w, &http.Request{
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(raw))),
		}, hookfn)
	})

	t.Run("handle hook func", func(_ *testing.T) {
		raw := "{\"request\": {\"uid\": \"1\"}}"
		serve(w, &http.Request{
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(raw))),
		}, hookfn)
	})
}
