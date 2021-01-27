package controller

import (
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"unsafe"
)

// these conversions are copied from https://github.com/kubernetes/kubernetes/blob/4db3a096ce8ac730b2280494422e1c4cf5fe875e/pkg/apis/admission/v1beta1/zz_generated.conversion.go
// to avoid copying in kubernetes/kubernetes
// they are sightly modified to remove complexity

func convertV1beta1AdmissionReviewToAdmissionAdmissionReview(in *admissionv1beta1.AdmissionReview, out *admissionv1.AdmissionReview) {
	if in.Request != nil {
		if out.Request == nil {
			out.Request = &admissionv1.AdmissionRequest{}
		}
		in, out := &in.Request, &out.Request
		*out = new(admissionv1.AdmissionRequest)
		convertV1beta1AdmissionRequestToAdmissionAdmissionRequest(*in, *out)
	} else {
		out.Request = nil
	}
	out.Response = (*admissionv1.AdmissionResponse)(unsafe.Pointer(in.Response)) // #nosec
}

func convertV1beta1AdmissionRequestToAdmissionAdmissionRequest(in *admissionv1beta1.AdmissionRequest, out *admissionv1.AdmissionRequest) {
	out.UID = types.UID(in.UID)
	out.Kind = in.Kind
	out.Resource = in.Resource
	out.SubResource = in.SubResource
	out.RequestKind = (*metav1.GroupVersionKind)(unsafe.Pointer(in.RequestKind))             // #nosec
	out.RequestResource = (*metav1.GroupVersionResource)(unsafe.Pointer(in.RequestResource)) // #nosec
	out.RequestSubResource = in.RequestSubResource
	out.Name = in.Name
	out.Namespace = in.Namespace
	out.Operation = admissionv1.Operation(in.Operation)
	out.Object = in.Object
	out.OldObject = in.OldObject
	out.Options = in.Options
}

func convertAdmissionAdmissionReviewToV1beta1AdmissionReview(in *admissionv1.AdmissionReview, out *admissionv1beta1.AdmissionReview) {
	if in.Request != nil {
		if out.Request == nil {
			out.Request = &admissionv1beta1.AdmissionRequest{}
		}
		in, out := &in.Request, &out.Request
		*out = new(admissionv1beta1.AdmissionRequest)
		convertAdmissionAdmissionRequestToV1beta1AdmissionRequest(*in, *out)
	} else {
		out.Request = nil
	}
	out.Response = (*admissionv1beta1.AdmissionResponse)(unsafe.Pointer(in.Response)) // #nosec
}

func convertAdmissionAdmissionRequestToV1beta1AdmissionRequest(in *admissionv1.AdmissionRequest, out *admissionv1beta1.AdmissionRequest) {
	out.UID = types.UID(in.UID)
	out.Kind = in.Kind
	out.Resource = in.Resource
	out.SubResource = in.SubResource
	out.RequestKind = (*metav1.GroupVersionKind)(unsafe.Pointer(in.RequestKind))             // #nosec
	out.RequestResource = (*metav1.GroupVersionResource)(unsafe.Pointer(in.RequestResource)) // #nosec
	out.RequestSubResource = in.RequestSubResource
	out.Name = in.Name
	out.Namespace = in.Namespace
	out.Operation = admissionv1beta1.Operation(in.Operation)
	out.Object = in.Object
	out.OldObject = in.OldObject
	out.Options = in.Options
}
