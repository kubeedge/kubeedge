package overridemanager

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	apppsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

const (
	deploymentReplicasPath = "/spec/replicas"
)

type ReplicasOverrider struct{}

func (o *ReplicasOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	switch rawObj.GetKind() {
	case DeploymentKind:
		if overriders.Overriders.Replicas == nil {
			return nil
		}
		patch := overrideOption{
			Op:    string(apppsv1alpha1.OverriderOpReplace),
			Path:  deploymentReplicasPath,
			Value: *overriders.Overriders.Replicas,
		}
		if err := applyJSONPatch(rawObj, []overrideOption{patch}); err != nil {
			return fmt.Errorf("failed to apply replicas override on deployment %s/%s, %v",
				rawObj.GetNamespace(), rawObj.GetName(), err)
		}
		return nil

	default:
		return fmt.Errorf("failed to apply replicas override on obj %s/%s, gvk: %s unsupported",
			rawObj.GetNamespace(), rawObj.GetName(), rawObj.GroupVersionKind())
	}
}
