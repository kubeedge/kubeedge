package fakers

import "k8s.io/apimachinery/pkg/runtime"

var _ runtime.ObjectDefaulter = &fakeObjectDefaulter{}

// fakeObjectDefaulter do nothing
type fakeObjectDefaulter struct{}

func (d *fakeObjectDefaulter) Default(in runtime.Object) {}

func NewFakeObjectDefaulter() runtime.ObjectDefaulter {
	return &fakeObjectDefaulter{}
}
