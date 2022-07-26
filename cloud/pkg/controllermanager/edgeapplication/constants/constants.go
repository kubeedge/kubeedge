package constants

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	LastAppliedTemplateAnnotationKey    = "apps.kubeedge.io/last-applied-template"
	LastContainedResourcesAnnotationKey = "apps.kubeedge.io/last-contained-resources"
)

var OverriderTargetGVK = map[schema.GroupVersionKind]struct{}{
	DeploymentGVK: {},
}

var ServiceGVK = schema.GroupVersionKind{
	Version: "v1",
	Kind:    "Service",
}

var DeploymentGVK = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}
