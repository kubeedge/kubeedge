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

package jsonpatch

import (
	"encoding/json"

	"k8s.io/klog/v2"
)

const (
	OpAdd     Operation = "add"
	OpRemove  Operation = "remove"
	OpReplace Operation = "replace"
)

type Operation string

// Items maps the json expression of jsonpatch.
type Items []Item

// New returns a new Items.
func New() Items {
	return make(Items, 0)
}

// Add adds a operation of jsonpatch .
func (items Items) Add(op Operation, path string, value any) Items {
	item := newItem(op, path)
	if err := item.setValue(value); err != nil {
		// There are usually no errors here.
		klog.Warningf("failed to set value: %v", err)
		return items
	}
	return append(items, item)
}

// ToJSON returns the json buffers.
func (items Items) ToJSON() ([]byte, error) {
	return json.Marshal(items)
}

// Item is a operation of jsonpatch.
type Item struct {
	Op    Operation `json:"op"`
	Path  string    `json:"path"`
	Value string    `json:"value,omitempty"`
}

// newItem returns a new Item.
func newItem(op Operation, path string) Item {
	return Item{
		Op:   op,
		Path: path,
	}
}

// setValue sets a object value to jsonpath. It will convert the structure into a json string.
func (o *Item) setValue(value any) error {
	if value == nil {
		return nil
	}
	switch value := value.(type) {
	case string:
		o.Value = value
	default:
		bff, err := json.Marshal(value)
		if err != nil {
			return err
		}
		o.Value = string(bff)
	}
	return nil
}
