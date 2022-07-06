package admissioncontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionregistrationv1beta1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	"k8s.io/klog/v2"
)

func registerValidateWebhook(client admissionregistrationv1beta1client.ValidatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1beta1.ValidatingWebhookConfiguration) error {
	for _, hook := range webhooks {
		existing, err := client.Get(context.Background(), hook.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil && existing != nil {
			existing.Webhooks = hook.Webhooks
			klog.Infof("Updating ValidatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Update(context.Background(), existing, metav1.UpdateOptions{}); err != nil {
				return err
			}
		} else {
			klog.Infof("Creating ValidatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Create(context.Background(), &hook, metav1.CreateOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func registerMutatingWebhook(client admissionregistrationv1beta1client.MutatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1beta1.MutatingWebhookConfiguration) error {
	for _, hook := range webhooks {
		existing, err := client.Get(context.Background(), hook.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil && existing != nil {
			existing.Webhooks = hook.Webhooks
			klog.Infof("Updating MutatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Update(context.Background(), existing, metav1.UpdateOptions{}); err != nil {
				return err
			}
		} else {
			klog.Infof("Creating MutatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Create(context.Background(), &hook, metav1.CreateOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

// hookFunc is the type we use for all of our validators and mutators
type hookFunc func(admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, hook hookFunc) {
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		errStr := fmt.Sprintf("Invalid Content-Type, actual %s, expect application/json", contentType)
		http.Error(w, errStr, http.StatusUnsupportedMediaType)
		return
	}

	if r.Body == nil {
		http.Error(w, "Parse body failed", http.StatusInternalServerError)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Read body failed: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}

	// The AdmissionReview that will be returned
	var responseAdmissionReview admissionv1beta1.AdmissionReview
	// The AdmissionReview that was sent to the webhook
	var requestedAdmissionReview admissionv1beta1.AdmissionReview
	if _, _, err := codecs.UniversalDeserializer().Decode(body, nil, &requestedAdmissionReview); err != nil {
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		responseAdmissionReview.Response = hook(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	klog.V(4).Infof("sending response: %+v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		errStr := fmt.Sprintf("Could not encode response: %v", err)
		klog.Error(errStr)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Errorf("Cannot write response %v", err)
	}
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
func toAdmissionResponse(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
