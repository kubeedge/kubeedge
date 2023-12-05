/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/overridemanager/argsoverride.go
- refactor argsOverrider as a struct that implements the Overrider interface
*/
package overridemanager

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

type ArgsOverrider struct{}

func (o *ArgsOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	argsOverriders := overriders.Overriders.ArgsOverriders
	for index := range argsOverriders {
		patches, err := buildCommandArgsPatches(ArgsString, rawObj, &argsOverriders[index])
		if err != nil {
			return err
		}

		klog.V(4).Infof("Parsed JSON patches by argsOverriders(%+v): %+v", argsOverriders[index], patches)
		if err = applyJSONPatch(rawObj, patches); err != nil {
			return err
		}
	}

	return nil
}
