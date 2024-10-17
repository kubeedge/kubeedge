package overridemanager

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

type NodeSelectorOverrider struct{}

func (o *NodeSelectorOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	deploymentObj, err := ConvertToDeployment(rawObj)
	if err != nil {
		return fmt.Errorf("failed to convert Deployment from unstructured object: %v", err)
	}
	switch rawObj.GetKind() {
	case "Deployment":
		if overriders.TargetNodeGroup != "" {
			nodeGroupLabel := map[string]string{
				nodegroup.LabelBelongingTo: overriders.TargetNodeGroup,
			}
			deploymentObj.Spec.Template.Spec.NodeSelector = nodeGroupLabel
		}
		if len(overriders.TargetNodeLabelSelector.MatchLabels) > 0 {
			nodeAffinity := &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{},
						},
					},
				},
			}

			for key, value := range overriders.TargetNodeLabelSelector.MatchLabels {
				nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
					append(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions,
						corev1.NodeSelectorRequirement{
							Key:      key,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{value},
						})
			}

			if deploymentObj.Spec.Template.Spec.Affinity == nil {
				deploymentObj.Spec.Template.Spec.Affinity = &corev1.Affinity{}
			}
			deploymentObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		}
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploymentObj)
		if err != nil {
			return err
		}
		rawObj.Object = unstructuredObj

	default:
		return fmt.Errorf("cannot override nodeselector for obj of gvk %s", rawObj.GroupVersionKind())
	}

	return nil
}
