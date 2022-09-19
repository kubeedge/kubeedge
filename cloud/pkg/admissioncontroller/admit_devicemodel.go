package admissioncontroller

import (
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	devicesv1alpha2 "github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

func admitDeviceModel(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	var msg string

	switch review.Request.Operation {
	case admissionv1.Create, admissionv1.Update:
		raw := review.Request.Object.Raw
		devicemodel := devicesv1alpha2.DeviceModel{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &devicemodel); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			return toAdmissionResponse(err)
		}
		msg = validateDeviceModel(&devicemodel, &reviewResponse)
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

func validateDeviceModel(devicemodel *devicesv1alpha2.DeviceModel, response *admissionv1.AdmissionResponse) string {
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

func serveDeviceModel(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDeviceModel)
}
