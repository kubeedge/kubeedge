package admissioncontroller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/cloud/test/httpfake"
)

func TestRegisterValidateWebhook(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		existingHooks []admissionregistrationv1.ValidatingWebhookConfiguration
		newHooks      []admissionregistrationv1.ValidatingWebhookConfiguration
		expectedError bool
	}{
		{
			name:          "Register new webhook",
			existingHooks: []admissionregistrationv1.ValidatingWebhookConfiguration{},
			newHooks: []admissionregistrationv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-webhook"},
					Webhooks:   []admissionregistrationv1.ValidatingWebhook{{Name: "test.webhook.com"}},
				},
			},
			expectedError: false,
		},
		{
			name: "Update existing webhook",
			existingHooks: []admissionregistrationv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-webhook"},
					Webhooks:   []admissionregistrationv1.ValidatingWebhook{{Name: "old.test-webhook.com"}},
				},
			},
			newHooks: []admissionregistrationv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-webhook"},
					Webhooks:   []admissionregistrationv1.ValidatingWebhook{{Name: "new.test-webhook.com"}},
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			for _, hook := range tc.existingHooks {
				_, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), &hook, metav1.CreateOptions{})
				assert.NoError(err)
			}

			err := registerValidateWebhook(clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations(), tc.newHooks)

			if tc.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)

				for _, hook := range tc.newHooks {
					registeredHook, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), hook.Name, metav1.GetOptions{})
					assert.NoError(err)
					assert.Equal(hook.Webhooks, registeredHook.Webhooks)
				}
			}
		})
	}
}

func TestRegisterMutatingWebhook(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		existingHooks []admissionregistrationv1.MutatingWebhookConfiguration
		newHooks      []admissionregistrationv1.MutatingWebhookConfiguration
		expectedError bool
	}{
		{
			name:          "Register new mutating webhook",
			existingHooks: []admissionregistrationv1.MutatingWebhookConfiguration{},
			newHooks: []admissionregistrationv1.MutatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-mutating-webhook"},
					Webhooks:   []admissionregistrationv1.MutatingWebhook{{Name: "test.mutating.webhook.com"}},
				},
			},
			expectedError: false,
		},
		{
			name: "Update existing mutating webhook",
			existingHooks: []admissionregistrationv1.MutatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-mutating-webhook"},
					Webhooks:   []admissionregistrationv1.MutatingWebhook{{Name: "old.mutating.webhook.com"}},
				},
			},
			newHooks: []admissionregistrationv1.MutatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-mutating-webhook"},
					Webhooks:   []admissionregistrationv1.MutatingWebhook{{Name: "new.mutating.webhook.com"}},
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			for _, hook := range tc.existingHooks {
				_, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), &hook, metav1.CreateOptions{})
				assert.NoError(err)
			}

			err := registerMutatingWebhook(clientset.AdmissionregistrationV1().MutatingWebhookConfigurations(), tc.newHooks)

			if tc.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)

				for _, hook := range tc.newHooks {
					registeredHook, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), hook.Name, metav1.GetOptions{})
					assert.NoError(err)
					assert.Equal(hook.Webhooks, registeredHook.Webhooks)
				}
			}
		})
	}
}

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

func TestToAdmissionResponse(t *testing.T) {
	assert := assert.New(t)

	err := errors.New("test error")
	response := toAdmissionResponse(err)

	assert.NotNil(response)
	assert.NotNil(response.Result)
	assert.Equal(err.Error(), response.Result.Message)
}
