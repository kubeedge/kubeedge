package admissioncontroller

import (
	"context"
	"encoding/json"
	"io/ioutil"
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
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Fatalf("contentType=%s, expect application/json", contentType)
		return
	}

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := admissionv1beta1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := admissionv1beta1.AdmissionReview{}

	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Fatalf("decode failed with error: %v", err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		responseAdmissionReview.Response = hook(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	klog.V(4).Infof("sending response: %+v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Fatalf("cannot marshal to a valid response %v", err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Fatalf("cannot write response %v", err)
	}
}
