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
	"strings"

	"github.com/blang/semver"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
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
		raw := review.Request.Object.Raw
		upgrade := v1alpha1.NodeUpgradeJob{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &upgrade); err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}

		return admissionResponse(validateNodeUpgradeJob(&upgrade))

	case admissionv1.Update:
		newUpgrade := v1alpha1.NodeUpgradeJob{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(review.Request.Object.Raw, nil, &newUpgrade); err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}

		oldUpgrade := v1alpha1.NodeUpgradeJob{}
		if _, _, err := deserializer.Decode(review.Request.OldObject.Raw, nil, &oldUpgrade); err != nil {
			return admissionResponse(fmt.Errorf("validation failed with error: %v", err))
		}

		// For update, we don't allow update spec fields once an Upgrade is created.
		if !reflect.DeepEqual(oldUpgrade.Spec, newUpgrade.Spec) {
			err := errors.New("spec fields are not allowed to update once it's created")
			return admissionResponse(err)
		}

		return admissionResponse(validateNodeUpgradeJob(&newUpgrade))

	case admissionv1.Delete:
		//no rule defined for above operations, greenlight for all of above.
		return admissionResponse(nil)
	default:
		err := fmt.Errorf("unsupported webhook operation %v", review.Request.Operation)
		return admissionResponse(err)
	}
}

func validateNodeUpgradeJob(upgrade *v1alpha1.NodeUpgradeJob) error {
	// version must be valid
	if !strings.HasPrefix(upgrade.Spec.Version, "v") {
		return fmt.Errorf("version must begin with prefix 'v'")
	}

	_, err := semver.Parse(strings.TrimPrefix(upgrade.Spec.Version, "v"))
	if err != nil {
		return fmt.Errorf("version is not a semver compatible version: %v", err)
	}

	// we must specify NodeNames or LabelSelector, and we can only specify only one
	if len(upgrade.Spec.NodeNames) == 0 && upgrade.Spec.LabelSelector == nil {
		return fmt.Errorf("both NodeNames and LabelSelctor are NOT specified")
	}
	if len(upgrade.Spec.NodeNames) != 0 && upgrade.Spec.LabelSelector != nil {
		return fmt.Errorf("both NodeNames and LabelSelctor are specified")
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

	var upgrade v1alpha1.NodeUpgradeJob
	if err := json.Unmarshal(review.Request.Object.Raw, &upgrade); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		return toAdmissionResponse(err)
	}

	payload := generateNodeUpgradeJobPatch(upgrade.Spec)
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

func generateNodeUpgradeJobPatch(spec v1alpha1.NodeUpgradeJobSpec) []patchValue {
	patch := make([]patchValue, 0)

	// mutate .spec.concurrency to default value 1 if not specified
	if spec.Concurrency == 0 {
		patch = append(patch, patchValue{
			Op:    "replace",
			Path:  "/spec/concurrency",
			Value: 1,
		})
	}
	// mutate .spec.timeoutSeconds to default value 300 if not specified
	if spec.TimeoutSeconds == nil {
		var defaultTimeoutSeconds uint32 = 300
		patch = append(patch, patchValue{
			Op:    "replace",
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
