package overridemanager

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	errorutil "k8s.io/apimachinery/pkg/util/errors"

	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

// overrideOption define the JSONPatch operator
type overrideOption struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// OverrideManager manages override operation
type Overrider interface {
	ApplyOverrides(rawObjs *unstructured.Unstructured, overrideInfo OverriderInfo) error
}

type OverriderInfo struct {
	TargetNodeGroup string
	Overriders      *appsv1alpha1.Overriders
}

type OverrideManager struct {
	Overriders []Overrider
}

func (o *OverrideManager) ApplyOverrides(rawObjs *unstructured.Unstructured, overrideInfo OverriderInfo) error {
	errs := []error{}
	for _, overrider := range o.Overriders {
		if err := overrider.ApplyOverrides(rawObjs, overrideInfo); err != nil {
			errs = append(errs, fmt.Errorf("failed to override the obj, %v", err))
		}
	}
	return errorutil.NewAggregate(errs)
}

// applyJSONPatch applies the override on to the given unstructured object.
func applyJSONPatch(obj *unstructured.Unstructured, overrides []overrideOption) error {
	jsonPatchBytes, err := json.Marshal(overrides)
	if err != nil {
		return err
	}

	patch, err := jsonpatch.DecodePatch(jsonPatchBytes)
	if err != nil {
		return err
	}

	objectJSONBytes, err := obj.MarshalJSON()
	if err != nil {
		return err
	}

	patchedObjectJSONBytes, err := patch.Apply(objectJSONBytes)
	if err != nil {
		return err
	}

	err = obj.UnmarshalJSON(patchedObjectJSONBytes)
	return err
}
