package admissioncontroller

import (
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	devicesv1beta1 "github.com/kubeedge/api/apis/devices/v1beta1"
)

func admitDevice(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	var msg string

	switch review.Request.Operation {
	case admissionv1.Create, admissionv1.Update:
		raw := review.Request.Object.Raw
		device := devicesv1beta1.Device{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &device); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			return toAdmissionResponse(err)
		}
		msg = validateDevice(&device, &reviewResponse)
	case admissionv1.Delete, admissionv1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		reviewResponse.Allowed = false
		msg = "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func validateDevice(device *devicesv1beta1.Device, response *admissionv1.AdmissionResponse) string {
	//device properties name must be unique.
	var msg string
	size := len(device.Spec.Properties)
	for i := range device.Spec.Properties {
		for j := i + 1; j < size; j++ {
			if device.Spec.Properties[i].Name == device.Spec.Properties[j].Name {
				msg = "property names must be unique."
				response.Allowed = false
				return msg
			}
		}
	}

	return msg
}

func serveDevice(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDevice)
}
