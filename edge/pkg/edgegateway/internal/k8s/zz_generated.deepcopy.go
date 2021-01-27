package k8s

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *PodInfo) DeepCopyInto(out *PodInfo) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PodInfo) DeepCopyObject() runtime.Object {
	out := PodInfo{}
	in.DeepCopyInto(&out)

	return &out
}

