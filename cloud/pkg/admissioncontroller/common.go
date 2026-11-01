package admissioncontroller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionregistrationv1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

func registerValidateWebhook(client admissionregistrationv1client.ValidatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1.ValidatingWebhookConfiguration) error {
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

func registerMutatingWebhook(client admissionregistrationv1client.MutatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1.MutatingWebhookConfiguration) error {
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

// errNilAdmissionRequest is returned when a decoded AdmissionReview carries no request.
var errNilAdmissionRequest = errors.New("the request field of the AdmissionReview is nil")

// hookFunc is the type we use for all of our validators and mutators
type hookFunc func(admissionv1.AdmissionReview) *admissionv1.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, hook hookFunc) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
		return
	}

	// AdmissionReview requests are JSON. Parse the media type instead of doing
	// an exact comparison so valid values such as
	// "application/json; charset=utf-8" are accepted.
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", r.Header.Get("Content-Type"))
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	if r.Body == nil {
		http.Error(w, "request body is required", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, constants.MaxRespBodyLength)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("read request body failed: %v", err)
		http.Error(w, "read request body failed", http.StatusBadRequest)
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
	} else if requestedAdmissionReview.Request == nil {
		// Every hook dereferences the request, so reject the review here
		// instead of letting the hook panic on a nil pointer.
		klog.Errorf("invalid admission review: %v", errNilAdmissionRequest)
		responseAdmissionReview.Response = toAdmissionResponse(errNilAdmissionRequest)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
