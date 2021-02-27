package fakers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ runtime.ObjectConvertor = &fakeObjectConvertor{}

// fakeObjectConvertor do nothing
type fakeObjectConvertor struct{}

func (c *fakeObjectConvertor) Convert(in, out, context interface{}) error {
	return nil
}

func (c *fakeObjectConvertor) ConvertToVersion(in runtime.Object, _ runtime.GroupVersioner) (runtime.Object, error) {
	return in, nil
}

func (c *fakeObjectConvertor) ConvertFieldLabel(_ schema.GroupVersionKind, label, field string) (string, string, error) {
	return label, field, nil
}

func NewFakeObjectConvertor() runtime.ObjectConvertor {
	return &fakeObjectConvertor{}
}
