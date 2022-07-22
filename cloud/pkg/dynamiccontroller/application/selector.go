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
