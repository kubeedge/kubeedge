package fakers

import (
	"context"

	"k8s.io/apiserver/pkg/admission"
)

// alwaysAdmit is an implementation of admission.Interface which always says yes to an admit request.
type alwaysAdmit struct{}

var _ admission.MutationInterface = alwaysAdmit{}
var _ admission.ValidationInterface = alwaysAdmit{}

// Admit makes an admission decision based on the request attributes
func (alwaysAdmit) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) (err error) {
	return nil
}

// Validate makes an admission decision based on the request attributes.  It is NOT allowed to mutate.
func (alwaysAdmit) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) (err error) {
	return nil
}

// Handles returns true if this admission controller can handle the given operation
// where operation can be one of CREATE, UPDATE, DELETE, or CONNECT
func (alwaysAdmit) Handles(operation admission.Operation) bool {
	return true
}

// NewAlwaysAdmit creates a new always admit admission handler
func NewAlwaysAdmit() admission.Interface {
	return new(alwaysAdmit)
}
