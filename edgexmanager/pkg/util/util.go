package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// AddAnnotations sets the desired annotations on the object and returns true if the annotations have changed.
func AddAnnotations(o metav1.Object, desired map[string]string) bool {
	if len(desired) == 0 {
		return false
	}
	annotations := o.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		o.SetAnnotations(annotations)
	}
	hasChanged := false
	for k, v := range desired {
		if cur, ok := annotations[k]; !ok || cur != v {
			annotations[k] = v
			hasChanged = true
		}
	}
	return hasChanged
}

// hasAnnotation returns true if the object has the specified annotation
func hasAnnotation(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[annotation]
	return ok
}

// HasOwnerRef returns true if the OwnerReference is already in the slice.
func HasOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) bool {
	return indexOwnerRef(ownerReferences, ref) > -1
}

// EnsureOwnerRef makes sure the slice contains the OwnerReference.
func EnsureOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) []metav1.OwnerReference {
	idx := indexOwnerRef(ownerReferences, ref)
	if idx == -1 {
		return append(ownerReferences, ref)
	}
	ownerReferences[idx] = ref
	return ownerReferences
}

// ReplaceOwnerRef re-parents an object from one OwnerReference to another
// It compares strictly based on UID to avoid reparenting across an intentional deletion: if an object is deleted
// and re-created with the same name and namespace, the only way to tell there was an in-progress deletion
// is by comparing the UIDs.
func ReplaceOwnerRef(ownerReferences []metav1.OwnerReference, source metav1.Object, target metav1.OwnerReference) []metav1.OwnerReference {
	fi := -1
	for index, r := range ownerReferences {
		if r.UID == source.GetUID() {
			fi = index
			ownerReferences[index] = target
			break
		}
	}
	if fi < 0 {
		ownerReferences = append(ownerReferences, target)
	}
	return ownerReferences
}

// RemoveOwnerRef returns the slice of owner references after removing the supplied owner ref
func RemoveOwnerRef(ownerReferences []metav1.OwnerReference, inputRef metav1.OwnerReference) []metav1.OwnerReference {
	if index := indexOwnerRef(ownerReferences, inputRef); index != -1 {
		return append(ownerReferences[:index], ownerReferences[index+1:]...)
	}
	return ownerReferences
}

// indexOwnerRef returns the index of the owner reference in the slice if found, or -1.
func indexOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) int {
	for index, r := range ownerReferences {
		if referSameObject(r, ref) {
			return index
		}
	}
	return -1
}

// PointsTo returns true if any of the owner references point to the given target
// Deprecated: Use IsOwnedByObject to cover differences in API version or backup/restore that changed UIDs.
func PointsTo(refs []metav1.OwnerReference, target *metav1.ObjectMeta) bool {
	for _, ref := range refs {
		if ref.UID == target.UID {
			return true
		}
	}
	return false
}

// IsOwnedByObject returns true if any of the owner references point to the given target.
func IsOwnedByObject(obj metav1.Object, target controllerutil.Object) bool {
	for _, ref := range obj.GetOwnerReferences() {
		ref := ref
		if refersTo(&ref, target) {
			return true
		}
	}
	return false
}

// IsControlledBy differs from metav1.IsControlledBy in that it checks the group (but not version), kind, and name vs uid.
func IsControlledBy(obj metav1.Object, owner controllerutil.Object) bool {
	controllerRef := metav1.GetControllerOfNoCopy(obj)
	if controllerRef == nil {
		return false
	}
	return refersTo(controllerRef, owner)
}

// Returns true if a and b point to the same object.
func referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && a.Kind == b.Kind && a.Name == b.Name
}

// Returns true if ref refers to obj.
func refersTo(ref *metav1.OwnerReference, obj controllerutil.Object) bool {
	refGv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return false
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	return refGv.Group == gvk.Group && ref.Kind == gvk.Kind && ref.Name == obj.GetName()
}
