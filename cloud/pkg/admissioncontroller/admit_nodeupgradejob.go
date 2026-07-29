/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admissioncontroller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha1 "github.com/kubeedge/api/apis/operations/v1alpha1"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
)

func serveNodeUpgradeJob(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitNodeUpgradeJob)
}

func serveMutatingNodeUpgradeJob(w http.ResponseWriter, r *http.Request) {
	serve(w, r, mutatingNodeUpgradeJob)
}

func admitNodeUpgradeJob(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	switch review.Request.Operation {
	case admissionv1.Create:
		upgrade, err := decodeNodeUpgradeJob(review.Request.Object.Raw)
		if err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}
		return admissionResponse(validateNodeUpgradeJob(upgrade))

	case admissionv1.Update:
		newUpgrade, err := decodeNodeUpgradeJob(review.Request.Object.Raw)
		if err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}
		oldUpgrade, err := decodeNodeUpgradeJob(review.Request.OldObject.Raw)
		if err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}

		// For update, we don't allow update spec fields once an Upgrade is created.
		if !reflect.DeepEqual(oldUpgrade, newUpgrade) {
			err := errors.New("spec fields are not allowed to update once it's created")
			return admissionResponse(err)
		}

		return admissionResponse(validateNodeUpgradeJob(newUpgrade))

	case admissionv1.Delete:
		//no rule defined for above operations, greenlight for all of above.
		return admissionResponse(nil)
	default:
		err := fmt.Errorf("unsupported webhook operation %v", review.Request.Operation)
		return admissionResponse(err)
	}
}

type nodeUpgradeJobSpec struct {
	Version        string
	TimeoutSeconds *uint32
	NodeNames      []string
	LabelSelector  *metav1.LabelSelector
	Image          string
	Concurrency    int32
}

func decodeNodeUpgradeJob(raw []byte) (*nodeUpgradeJobSpec, error) {
	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(raw, nil, nil)
	if err != nil {
		return nil, err
	}

	switch {
	case gvk.Group == operationsv1alpha1.GroupName && gvk.Version == operationsv1alpha1.Version && gvk.Kind == "NodeUpgradeJob":
		upgrade, ok := obj.(*operationsv1alpha1.NodeUpgradeJob)
		if !ok {
			return nil, fmt.Errorf("decoded object is %T, want *v1alpha1.NodeUpgradeJob", obj)
		}
		return nodeUpgradeJobSpecFromV1Alpha1(upgrade.Spec), nil
	case gvk.Group == operationsv1alpha2.GroupName && gvk.Version == operationsv1alpha2.Version && gvk.Kind == "NodeUpgradeJob":
		upgrade, ok := obj.(*operationsv1alpha2.NodeUpgradeJob)
		if !ok {
			return nil, fmt.Errorf("decoded object is %T, want *v1alpha2.NodeUpgradeJob", obj)
		}
		return nodeUpgradeJobSpecFromV1Alpha2(upgrade.Spec), nil
	default:
		return nil, fmt.Errorf("unsupported NodeUpgradeJob GVK %s", gvk.String())
	}
}

func nodeUpgradeJobSpecFromV1Alpha1(spec operationsv1alpha1.NodeUpgradeJobSpec) *nodeUpgradeJobSpec {
	return &nodeUpgradeJobSpec{
		Version:        spec.Version,
		TimeoutSeconds: spec.TimeoutSeconds,
		NodeNames:      spec.NodeNames,
		LabelSelector:  spec.LabelSelector,
		Image:          spec.Image,
		Concurrency:    spec.Concurrency,
	}
}

func nodeUpgradeJobSpecFromV1Alpha2(spec operationsv1alpha2.NodeUpgradeJobSpec) *nodeUpgradeJobSpec {
	return &nodeUpgradeJobSpec{
		Version:        spec.Version,
		TimeoutSeconds: spec.TimeoutSeconds,
		NodeNames:      spec.NodeNames,
		LabelSelector:  spec.LabelSelector,
		Image:          spec.Image,
		Concurrency:    spec.Concurrency,
	}
}

func validateNodeUpgradeJob(upgrade *nodeUpgradeJobSpec) error {
	if !validation.ValidateVersion(upgrade.Version) {
		return fmt.Errorf("invalid version %s", upgrade.Version)
	}
	// Image is a optional field.
	if upgrade.Image != "" && !validation.ValidateImageRepo(upgrade.Image) {
		return fmt.Errorf("invalid image repo %s", upgrade.Image)
	}
	// we must specify NodeNames or LabelSelector, and we can only specify only one
	if len(upgrade.NodeNames) == 0 && upgrade.LabelSelector == nil {
		return fmt.Errorf("both NodeNames and LabelSelector are NOT specified")
	}
	if len(upgrade.NodeNames) != 0 && upgrade.LabelSelector != nil {
		return fmt.Errorf("both NodeNames and LabelSelector are specified")
	}

	return nil
}

func admissionResponse(err error) *admissionv1.AdmissionResponse {
	if err != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func mutatingNodeUpgradeJob(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}

	upgrade, err := decodeNodeUpgradeJob(review.Request.Object.Raw)
	if err != nil {
		klog.Errorf("Could not decode raw object: %v", err)
		return toAdmissionResponse(err)
	}

	payload := generateNodeUpgradeJobPatch(upgrade)
	if len(payload) == 0 {
		return &reviewResponse
	}

	patch, err := json.Marshal(payload)
	if err != nil {
		return toAdmissionResponse(err)
	}

	reviewResponse.Patch = patch
	pt := admissionv1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt
	return &reviewResponse
}

func generateNodeUpgradeJobPatch(spec *nodeUpgradeJobSpec) []patchValue {
	patch := make([]patchValue, 0)

	// mutate .spec.concurrency to default value 1 if not specified
	if spec.Concurrency == 0 {
		patch = append(patch, patchValue{
			Op:    "add",
			Path:  "/spec/concurrency",
			Value: 1,
		})
	}
	// mutate .spec.timeoutSeconds to default value 300 if not specified
	if spec.TimeoutSeconds == nil {
		var defaultTimeoutSeconds uint32 = 300
		patch = append(patch, patchValue{
			Op:    "add",
			Path:  "/spec/timeoutSeconds",
			Value: &defaultTimeoutSeconds,
		})
	}

	return patch
}

type patchValue struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}
