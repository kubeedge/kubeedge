package admissioncontroller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	devicesv1alpha2 "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

// admitFunc is the type we use for all of our validators and mutators
type admitFunc func(admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {
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
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	klog.Infof("sending response: %v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Fatalf("cannot marshal to a valid response %v", err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Fatalf("cannot write response %v", err)
	}
}

func admitDeviceModel(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	var msg string

	switch review.Request.Operation {
	case admissionv1beta1.Create, admissionv1beta1.Update:
		raw := review.Request.Object.Raw
		devicemodel := devicesv1alpha2.DeviceModel{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &devicemodel); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			return toAdmissionResponse(err)
		}
		msg = validateDeviceModel(&devicemodel, &reviewResponse)
	case admissionv1beta1.Delete, admissionv1beta1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		klog.Info("admission validation passed!")
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		reviewResponse.Allowed = false
		msg = msg + "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func validateDeviceModel(devicemodel *devicesv1alpha2.DeviceModel, response *admissionv1beta1.AdmissionResponse) string {
	//device properties must be either Int or String while additional properties is not banned.
	var msg string
	for _, property := range devicemodel.Spec.Properties {
		if property.Type.String == nil && property.Type.Int == nil {
			msg = "Either Int or String must be set"
			response.Allowed = false
		} else if property.Type.String != nil && property.Type.Int != nil {
			msg = "Only one of [Int, String] could be set for properties"
			response.Allowed = false
		}
	}
	return msg
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
func toAdmissionResponse(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func serveDeviceModel(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDeviceModel)
}
