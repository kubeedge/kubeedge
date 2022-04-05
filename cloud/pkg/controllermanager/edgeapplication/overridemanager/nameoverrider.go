package overridemanager

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NameOverrider struct{}

func (o *NameOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	// TODO: consider how to override if oldName is empty
	oldName := rawObj.GetName()
	newName := fmt.Sprintf("%s-%s", oldName, overriders.TargetNodeGroup)
	rawObj.SetName(newName)
	return nil
}
