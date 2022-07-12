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

package application

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// TODO: how to solve json marshal unmashal problem against labels.Selector or fields.Selector?
type LabelFieldSelector struct {
	Label labels.Selector
	Field fields.Selector
}

func NewSelector(ls string, fs string) LabelFieldSelector {
	label, _ := labels.Parse(ls)
	field := fields.ParseSelectorOrDie(fs)
	lf := LabelFieldSelector{
		Label: label,
		Field: field,
	}
	return lf
}

func (lf *LabelFieldSelector) Labels() labels.Selector {
	return lf.Label
}

func (lf *LabelFieldSelector) Fields() fields.Selector {
	return lf.Field
}

func (lf *LabelFieldSelector) String() string {
	var ret string
	if lf.Label != nil {
		ret += lf.Label.String()
	}
	ret += ";"
	if lf.Field != nil {
		ret += lf.Field.String()
	}
	return ret
}

func (lf *LabelFieldSelector) Match(set labels.Set, set2 fields.Set) bool {
	return lf.Labels().Matches(set) && lf.Fields().Matches(set2)
}

func (lf *LabelFieldSelector) MatchObj(obj runtime.Object) bool {
	objLabels, objFields, err := util.UnstructuredAttr(obj)
	if err != nil {
		return false
	}
	return lf.Match(objLabels, objFields)
}
