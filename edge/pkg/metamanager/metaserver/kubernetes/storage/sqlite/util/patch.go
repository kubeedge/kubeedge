/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
@CHANGELOG
KubeEdge Authors: To merge the patchBytes of StrategicMergePatchType into the original resource,
This file is derived from K8S apiserver code with reduced set of methods
Changes done are
1. Package util got some functions from "k8s.io/apiserver/pkg/endpoints/handlers/patch.go"
and made some variant
*/

package util

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/warning"
	kjson "sigs.k8s.io/json"
)

func StrategicPatchObject(
	requestContext context.Context,
	defaulter runtime.ObjectDefaulter,
	originalObject runtime.Object,
	patchBytes []byte,
	objToUpdate runtime.Object,
	schemaReferenceObj runtime.Object,
	validationDirective string,
) error {
	originalObjMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(originalObject)
	if err != nil {
		return err
	}

	patchMap := make(map[string]interface{})
	var strictErrs []error
	if validationDirective == metav1.FieldValidationWarn || validationDirective == metav1.FieldValidationStrict {
		strictErrs, err = kjson.UnmarshalStrict(patchBytes, &patchMap)
		if err != nil {
			return errors.NewBadRequest(err.Error())
		}
	} else {
		if err = kjson.UnmarshalCaseSensitivePreserveInts(patchBytes, &patchMap); err != nil {
			return errors.NewBadRequest(err.Error())
		}
	}

	return applyPatchToObject(requestContext, defaulter, originalObjMap, patchMap, objToUpdate, schemaReferenceObj, strictErrs, validationDirective)
}

// applyPatchToObject applies a strategic merge patch of <patchMap> to
// <originalMap> and stores the result in <objToUpdate>.
// NOTE: <objToUpdate> must be a versioned object.
func applyPatchToObject(
	requestContext context.Context,
	defaulter runtime.ObjectDefaulter,
	originalMap map[string]interface{},
	patchMap map[string]interface{},
	objToUpdate runtime.Object,
	schemaReferenceObj runtime.Object,
	strictErrs []error,
	validationDirective string,
) error {
	patchedObjMap, err := strategicpatch.StrategicMergeMapPatch(originalMap, patchMap, schemaReferenceObj)
	if err != nil {
		return interpretStrategicMergePatchError(err)
	}

	// Rather than serialize the patched map to JSON, then decode it to an object, we go directly from a map to an object
	converter := runtime.DefaultUnstructuredConverter
	returnUnknownFields := validationDirective == metav1.FieldValidationWarn || validationDirective == metav1.FieldValidationStrict
	if err := converter.FromUnstructuredWithValidation(patchedObjMap, objToUpdate, returnUnknownFields); err != nil {
		strictError, isStrictError := runtime.AsStrictDecodingError(err)
		switch {
		case !isStrictError:
			// disregard any sttrictErrs, because it's an incomplete
			// list of strict errors given that we don't know what fields were
			// unknown because StrategicMergeMapPatch failed.
			// Non-strict errors trump in this case.
			return errors.NewInvalid(schema.GroupKind{}, "", field.ErrorList{
				field.Invalid(field.NewPath("patch"), fmt.Sprintf("%+v", patchMap), err.Error()),
			})
		case validationDirective == metav1.FieldValidationWarn:
			addStrictDecodingWarnings(requestContext, append(strictErrs, strictError.Errors()...))
		default:
			strictDecodingError := runtime.NewStrictDecodingError(append(strictErrs, strictError.Errors()...))
			return errors.NewInvalid(schema.GroupKind{}, "", field.ErrorList{
				field.Invalid(field.NewPath("patch"), fmt.Sprintf("%+v", patchMap), strictDecodingError.Error()),
			})
		}
	} else if len(strictErrs) > 0 {
		switch {
		case validationDirective == metav1.FieldValidationWarn:
			addStrictDecodingWarnings(requestContext, strictErrs)
		default:
			return errors.NewInvalid(schema.GroupKind{}, "", field.ErrorList{
				field.Invalid(field.NewPath("patch"), fmt.Sprintf("%+v", patchMap), runtime.NewStrictDecodingError(strictErrs).Error()),
			})
		}
	}

	// Decoding from JSON to a versioned object would apply defaults, so we do the same here
	defaulter.Default(objToUpdate)

	return nil
}

// addStrictDecodingWarnings confirms that the error is a strict decoding error
// and if so adds a warning for each strict decoding violation.
func addStrictDecodingWarnings(requestContext context.Context, errs []error) {
	for _, e := range errs {
		yamlWarnings := parseYAMLWarnings(e.Error())
		for _, w := range yamlWarnings {
			warning.AddWarning(requestContext, "", w)
		}
	}
}

// parseYAMLWarnings takes the strict decoding errors from the yaml decoder's output
// and parses each individual warnings, or leaves the warning as is if
// it does not look like a yaml strict decoding error.
func parseYAMLWarnings(errString string) []string {
	var trimmedString string
	if trimmedShortString := strings.TrimPrefix(errString, shortPrefix); len(trimmedShortString) < len(errString) {
		trimmedString = trimmedShortString
	} else if trimmedLongString := strings.TrimPrefix(errString, longPrefix); len(trimmedLongString) < len(errString) {
		trimmedString = trimmedLongString
	} else {
		// not a yaml error, return as-is
		return []string{errString}
	}

	splitStrings := strings.Split(trimmedString, "\n")
	for i, s := range splitStrings {
		splitStrings[i] = strings.TrimSpace(s)
	}
	return splitStrings
}

// interpretStrategicMergePatchError interprets the error type and returns an error with appropriate HTTP code.
func interpretStrategicMergePatchError(err error) error {
	switch err {
	case mergepatch.ErrBadJSONDoc, mergepatch.ErrBadPatchFormatForPrimitiveList, mergepatch.ErrBadPatchFormatForRetainKeys, mergepatch.ErrBadPatchFormatForSetElementOrderList, mergepatch.ErrUnsupportedStrategicMergePatchFormat:
		return errors.NewBadRequest(err.Error())
	case mergepatch.ErrNoListOfLists, mergepatch.ErrPatchContentNotMatchRetainKeys:
		return errors.NewGenericServerResponse(http.StatusUnprocessableEntity, "", schema.GroupResource{}, "", err.Error(), 0, false)
	default:
		return err
	}
}

const (
	// shortPrefix is one possible beginning of yaml unmarshal strict errors.
	shortPrefix = "yaml: unmarshal errors:\n"
	// longPrefix is the other possible beginning of yaml unmarshal strict errors.
	longPrefix = "error converting YAML to JSON: yaml: unmarshal errors:\n"
)
