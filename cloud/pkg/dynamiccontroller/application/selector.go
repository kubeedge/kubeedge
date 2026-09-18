package application

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type LabelFieldSelector struct {
	Label labels.Selector
	Field fields.Selector
}

// labelFieldSelectorJSON is the wire representation of LabelFieldSelector.
// Both labels.Selector and fields.Selector are interfaces backed by
// unexported concrete types, so they cannot be marshaled/unmarshaled
// directly — this struct stores them as their string form instead.
type labelFieldSelectorJSON struct {
	Label string `json:"label"`
	Field string `json:"field"`
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

// MarshalJSON implements json.Marshaler by encoding Label and Field as
// their string representations.
func (lf LabelFieldSelector) MarshalJSON() ([]byte, error) {
	var labelStr, fieldStr string
	if lf.Label != nil {
		labelStr = lf.Label.String()
	}
	if lf.Field != nil {
		fieldStr = lf.Field.String()
	}
	aux := labelFieldSelectorJSON{
		Label: labelStr,
		Field: fieldStr,
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements json.Unmarshaler by parsing the stored label and field strings back into a labels.Selector and fields.Selector.
// Known limitation: labels.Nothing() and labels.Everything() both serialize to the empty string via .String(), so a Nothing() selector round-trips as Everything() after Marshal/Unmarshal.
// There is no exported way to distinguish them from outside the labels package.
// If a LabelFieldSelector is ever expected to carry Nothing() as a real, meaningful value, this round-trip loses that distinction.
func (lf *LabelFieldSelector) UnmarshalJSON(data []byte) error {
	var aux labelFieldSelectorJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	label, err := labels.Parse(aux.Label)
	if err != nil {
		return err
	}

	field, err := fields.ParseSelector(aux.Field)
	if err != nil {
		return err
	}

	lf.Label = label
	lf.Field = field
	return nil
}
