/*
Copyright 2020 The Kubernetes Authors.

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

package patch

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type patchType string

func (p patchType) Key() string {
	return strings.Split(string(p), ".")[0]
}

const (
	specPatch   patchType = "spec"
	statusPatch patchType = "status"
)

var (
	preserveUnstructuredKeys = map[string]bool{
		"kind":       true,
		"apiVersion": true,
		"metadata":   true,
	}
)

func unstructuredHasStatus(u *unstructured.Unstructured) bool {
	_, ok := u.Object["status"]
	return ok
}

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	// If the incoming object is already unstructured, perform a deep copy first
	// otherwise DefaultUnstructuredConverter ends up returning the inner map without
	// making a copy.
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject()
	}
	rawMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: rawMap}, nil
}

// unsafeUnstructuredCopy returns a shallow copy of the unstructured object given as input.
// It copies the common fields such as `kind`, `apiVersion`, `metadata` and the patchType specified.
//
// It's not safe to modify any of the keys in the returned unstructured object, the result should be treated as read-only.
func unsafeUnstructuredCopy(obj *unstructured.Unstructured, focus patchType, isConditionsSetter bool) *unstructured.Unstructured {
	// Create the return focused-unstructured object with a preallocated map.
	res := &unstructured.Unstructured{Object: make(map[string]interface{}, len(obj.Object))}

	// Ranges over the keys of the unstructured object, think of this as the very top level of an object
	// when submitting a yaml to kubectl or a client.
	// These would be keys like `apiVersion`, `kind`, `metadata`, `spec`, `status`, etc.
	for key := range obj.Object {
		value := obj.Object[key]

		// Perform a shallow copy only for the keys we're interested in, or the ones that should be always preserved.
		if key == focus.Key() || preserveUnstructuredKeys[key] {
			res.Object[key] = value
		}

		// If we've determined that we're able to interface with conditions.Setter interface,
		// when dealing with the status patch, remove the status.conditions sub-field from the object.
		if isConditionsSetter && focus == statusPatch {
			// NOTE: Removing status.conditions changes the incoming object! This is safe because the condition patch
			// doesn't use the unstructured fields, and it runs before any other patch.
			//
			// If we want to be 100% safe, we could make a copy of the incoming object before modifying it, although
			// copies have a high cpu and high memory usage, therefore we intentionally choose to avoid extra copies
			// given that the ordering of operations and safety is handled internally by the patch helper.
			unstructured.RemoveNestedField(res.Object, "status", "conditions")
		}
	}

	return res
}
