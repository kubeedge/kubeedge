package class

import (
	def "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
	networking "k8s.io/api/networking/v1beta1"
)

const (
	// IngressKey picks a specific "class" for the Ingress.
	// The controller only processes Ingresses with this annotation either
	// unset, or set to either the configured value or the empty string.
	IngressKey = "kubernetes.io/ingress.class"
)

var (
	// DefaultClass defines the default class used in the nginx ingress controller
	DefaultClass = "nginx"

	// IngressClass sets the runtime ingress class to use
	// An empty string means accept all ingresses without
	// annotation and the ones configured with class nginx
	IngressClass = "nginx"
)

// IsValid returns true if the given Ingress specify the ingress.class
// annotation or IngressClassName resource for Kubernetes >= v1.18
func IsValid(ing *networking.Ingress) bool {
	// 1. with annotation or IngressClass
	ingress, ok := ing.GetAnnotations()[IngressKey]
	if !ok && ing.Spec.IngressClassName != nil {
		ingress = *ing.Spec.IngressClassName
	}

	// empty ingress and IngressClass equal default
	if len(ingress) == 0 && IngressClass == DefaultClass {
		return true
	}

	// k8s > v1.18.
	// Processing may be redundant because k8s.IngressClass is obtained by IngressClass
	// 3. without annotation and IngressClass. Check IngressClass
	if def.IngressClass != nil {
		return ingress == def.IngressClass.Name
	}

	// 4. with IngressClass
	return ingress == IngressClass
}
