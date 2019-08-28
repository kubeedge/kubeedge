package admissioncontroller

import (
	"net/http"
	"reflect"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"

	devicesv1alpha1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
)

func admitDevice(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	var msg string
	switch review.Request.Operation {
	case admissionv1beta1.Create:
		raw := review.Request.Object.Raw
		device := devicesv1alpha1.Device{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &device); err != nil {
			klog.Errorf("Validation failed with error: %v", err)
			return toAdmissionResponse(err)
		}

		un := unstructured.Unstructured{}
		if err := un.UnmarshalJSON(raw); err != nil {
			klog.Errorf("Failed to unmarshall object (%v) with error: %v", string(raw), err)
			return toAdmissionResponse(err)
		}

		mdSpec, ok := un.Object["spec"].(map[string]interface{})
		if ok {
			mdProto, ok := mdSpec["protocol"].(map[string]interface{})
			if ok {
				keys := reflect.ValueOf(mdProto).MapKeys()
				//device should only have one protocol defined.
				if len(keys) > 1 {
					reviewResponse.Allowed = false
					msg = msg + " multiple protocols found!"
				}
			}
		} else {
			reviewResponse.Allowed = false
			msg = msg + " cannot get spec of device!"
		}
	case admissionv1beta1.Update, admissionv1beta1.Delete, admissionv1beta1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		klog.Info("Admission validation passed!")
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		reviewResponse.Allowed = false
		msg = msg + " Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func serveDevice(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDevice)
}
