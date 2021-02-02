package application

import (
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	return lf.Labels().Matches(set) && lf.Labels().Matches(set2)
}

func (lf *LabelFieldSelector) MatchObj(obj runtime.Object) bool {
	objLabels, objFields, err := util.UnstructuredAttr(obj)
	if err != nil {
		return false
	}
	return lf.Match(objLabels, objFields)

}

/*
func (lf *LabelFieldSelector) MarshalJSON() ([]byte, error) {
	bytes,err := json.Marshal(lf.String())
	if err !=nil{
		return []byte{},err
	}
	return bytes, nil
	//return []byte("\""+lf.String()+"\""), nil
}
func (lf *LabelFieldSelector) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b,&s);err !=nil{
		return err
	}
	//s = strings.TrimPrefix(s,"\"")
	//s = strings.TrimSuffix(s,"\"")
	slice := strings.Split(s, ";")
	objLabels, err := labels.Parse(slice[0])
	if err != nil {
		return err
	}
	lf.Label = objLabels
	objFileds, err := fields.ParseSelector(slice[1])
	if err != nil {
		return err
	}
	lf.Field = objFileds
	return nil
}
*/
