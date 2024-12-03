/*
Copyright 2025 The KubeEdge Authors.

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

package common

import (
	"encoding/json"
	"io"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/cmd/get"
)

type ExtPrintFlags struct {
	PrintFlags *get.PrintFlags
}

func (exPrintFlags *ExtPrintFlags) PrintToTable(obj runtime.Object, isAllNamespace bool, w io.Writer) error {
	if isAllNamespace {
		if err := exPrintFlags.PrintFlags.EnsureWithNamespace(); err != nil {
			return err
		}
	}

	printer, err := exPrintFlags.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}
	return printer.PrintObj(obj, w)
}

func (exPrintFlags *ExtPrintFlags) PrintToJSONYaml(objectList []runtime.Object) error {
	list := v1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}
	var obj runtime.Object
	if len(objectList) != 1 {
		for _, info := range objectList {
			if info == nil {
				continue
			}
			o := info.DeepCopyObject()
			list.Items = append(list.Items, runtime.RawExtension{Object: o})
		}

		listData, err := json.Marshal(list)
		if err != nil {
			return err
		}

		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, listData)
		if err != nil {
			return err
		}
		obj = converted
	} else {
		obj = objectList[0]
	}
	return exPrintFlags.printGeneric(obj)
}

func (exPrintFlags *ExtPrintFlags) printGeneric(obj runtime.Object) error {
	printer, err := exPrintFlags.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}
	if meta.IsListType(obj) {
		items, err := meta.ExtractList(obj)
		if err != nil {
			return err
		}

		// take the items and create a new list for display
		list := &unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"kind":       "List",
				"apiVersion": "v1",
				"metadata":   map[string]interface{}{},
			},
		}
		if listMeta, err := meta.ListAccessor(obj); err == nil {
			list.Object["metadata"] = map[string]interface{}{
				"selfLink":        listMeta.GetSelfLink(),
				"resourceVersion": listMeta.GetResourceVersion(),
			}
		}

		for _, item := range items {
			list.Items = append(list.Items, *item.(*unstructured.Unstructured))
		}
		if err := printer.PrintObj(list, os.Stdout); err != nil {
			return err
		}
	} else {
		var value map[string]interface{}
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		if err := printer.PrintObj(&unstructured.Unstructured{Object: value}, os.Stdout); err != nil {
			return err
		}
	}
	return nil
}
