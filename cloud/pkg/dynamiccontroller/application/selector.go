package application

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type LabelFieldSelector struct {
	Label labels.Selector
	Field fields.Selector
}

// MarshalJSON stores selectors by their canonical string forms. The concrete
// selector implementations keep their state in private fields, so encoding
// the interface values directly would otherwise lose the selector expression.
func (lf LabelFieldSelector) MarshalJSON() ([]byte, error) {
	label, field := "", ""
	if lf.Label != nil {
		label = lf.Label.String()
	}
	if lf.Field != nil {
		field = lf.Field.String()
	}
	return json.Marshal(struct {
		Label string `json:"labelSelector"`
		Field string `json:"fieldSelector"`
	}{Label: label, Field: field})
}

// UnmarshalJSON reconstructs selector implementations from their canonical
// expressions and rejects malformed input instead of silently broadening a
// selector to match every object.
func (lf *LabelFieldSelector) UnmarshalJSON(data []byte) error {
	var encoded struct {
		Label string `json:"labelSelector"`
		Field string `json:"fieldSelector"`
	}
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	label, err := labels.Parse(encoded.Label)
	if err != nil {
		return fmt.Errorf("parse labelSelector: %w", err)
	}
	field, err := fields.ParseSelector(encoded.Field)
	if err != nil {
		return fmt.Errorf("parse fieldSelector: %w", err)
	}
	lf.Label = label
	lf.Field = field
	return nil
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
