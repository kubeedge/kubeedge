/*
Copyright 2026 The KubeEdge Authors.

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

package util

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	kjson "sigs.k8s.io/json"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

// DecodeStored decodes a meta_v2 value into obj and, for a pod, merges its podpatch from
// the meta table, as Get and List serve it.
func DecodeStored(codec runtime.Decoder, value string, obj *unstructured.Unstructured) error {
	if err := runtime.DecodeInto(codec, []byte(value), obj); err != nil {
		return err
	}
	if obj.GetKind() == "Pod" {
		return MergePatchedResource(obj, model.ResourceTypePodPatch)
	}
	return nil
}

// MergePatchedResource applies the patch of type resourceTypePatch stored in the meta table
// for obj, if any, to obj in place.
func MergePatchedResource(obj *unstructured.Unstructured, resourceTypePatch string) error {
	resKey := fmt.Sprintf("%s/%s/%s", obj.GetNamespace(), resourceTypePatch, obj.GetName())
	metas, err := dbclient.NewMetaService().QueryMeta("key", resKey)
	if err != nil {
		return err
	}
	if len(*metas) == 0 {
		return nil
	}
	schemaReferenceObj, err := scheme.Scheme.New(obj.GroupVersionKind())
	if err != nil {
		return fmt.Errorf("failed to build schema reference object, err: %+v", err)
	}
	patchMap := make(map[string]interface{})
	if err := kjson.UnmarshalCaseSensitivePreserveInts([]byte((*metas)[0]), &patchMap); err != nil {
		return err
	}
	patched, err := strategicpatch.StrategicMergeMapPatch(obj.Object, patchMap, schemaReferenceObj)
	if err != nil {
		return interpretStrategicMergePatchError(err)
	}
	obj.Object = patched
	return nil
}
