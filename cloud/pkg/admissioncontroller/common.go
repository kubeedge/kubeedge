package admissioncontroller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionregistrationv1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

// isRegistrationRetriable reports whether a failed registration can be resolved by
// reading the webhook configuration again and writing it once more. A conflict means
// another writer updated the configuration between our get and update, an already
// exists error means another writer created it between our get and create.
func isRegistrationRetriable(err error) bool {
	return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err)
}

func registerValidateWebhook(client admissionregistrationv1client.ValidatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1.ValidatingWebhookConfiguration) error {
	for _, hook := range webhooks {
		err := retry.OnError(retry.DefaultRetry, isRegistrationRetriable, func() error {
			existing, err := client.Get(context.Background(), hook.Name, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
				klog.Infof("Creating ValidatingWebhookConfiguration: %v", hook.Name)
				_, err = client.Create(context.Background(), &hook, metav1.CreateOptions{})
				return err
			}
			existing.Webhooks = hook.Webhooks
			klog.Infof("Updating ValidatingWebhookConfiguration: %v", hook.Name)
			_, err = client.Update(context.Background(), existing, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func registerMutatingWebhook(client admissionregistrationv1client.MutatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1.MutatingWebhookConfiguration) error {
	for _, hook := range webhooks {
		err := retry.OnError(retry.DefaultRetry, isRegistrationRetriable, func() error {
			existing, err := client.Get(context.Background(), hook.Name, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
				klog.Infof("Creating MutatingWebhookConfiguration: %v", hook.Name)
				_, err = client.Create(context.Background(), &hook, metav1.CreateOptions{})
				return err
			}
			existing.Webhooks = hook.Webhooks
			klog.Infof("Updating MutatingWebhookConfiguration: %v", hook.Name)
			_, err = client.Update(context.Background(), existing, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// hookFunc is the type we use for all of our validators and mutators
type hookFunc func(admissionv1.AdmissionReview) *admissionv1.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, hook hookFunc) {
	var body []byte
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, constants.MaxRespBodyLength)
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := admissionv1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := admissionv1.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"))

	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Errorf("decode failed with error: %v", err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		responseAdmissionReview.Response = hook(requestedAdmissionReview)
		// Return the same UID
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	}

	klog.V(4).Infof("sending response: %+v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Errorf("cannot marshal to a valid response %v", err)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Errorf("cannot write response %v", err)
		return
	}
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
func toAdmissionResponse(err error) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
