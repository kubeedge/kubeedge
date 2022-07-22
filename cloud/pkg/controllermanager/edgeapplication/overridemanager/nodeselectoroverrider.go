package overridemanager

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

type NodeSelectorOverrider struct{}

func (o *NodeSelectorOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	switch rawObj.GetKind() {
	case DeploymentKind:
		deploymentObj, err := ConvertToDeployment(rawObj)
		if err != nil {
			return fmt.Errorf("failed to convert Deployment from unstructured object: %v", err)
		}
		nodeGroupLabel := map[string]string{
			nodegroup.LabelBelongingTo: overriders.TargetNodeGroup,
		}
		deploymentObj.Spec.Template.Spec.NodeSelector = nodeGroupLabel
		unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploymentObj)
		if err != nil {
			return fmt.Errorf("failed to convert Deployment to unstructured object: %v", err)
		}
		rawObj.Object = unstructured
	default:
		return fmt.Errorf("cannot override nodeselector for obj of gvk %s", rawObj.GroupVersionKind())
	}
	return nil
}
