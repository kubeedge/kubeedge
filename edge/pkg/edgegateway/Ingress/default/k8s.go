package defaults

import (
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// IngressClass indicates the class of the Ingress to use as filter
var IngressClass *networkingv1beta1.IngressClass


// MetaNamespaceKey knows how to make keys for API objects which implement meta.Interface.
func MetaNamespaceKey(obj interface{}) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Warning(err)
	}

	return key
}


// default path type is Prefix to not break existing definitions
var defaultPathType = networkingv1beta1.PathTypePrefix

// SetDefaultNginxPathType sets a default PathType when is not defined.
func SetDefaultNginxPathType(ing *networkingv1beta1.Ingress) {
	for _, rule := range ing.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for idx := range rule.IngressRuleValue.HTTP.Paths {
			p := &rule.IngressRuleValue.HTTP.Paths[idx]
			if p.PathType == nil {
				p.PathType = &defaultPathType
			}

			if *p.PathType == networkingv1beta1.PathTypeImplementationSpecific {
				p.PathType = &defaultPathType
			}
		}
	}
}
