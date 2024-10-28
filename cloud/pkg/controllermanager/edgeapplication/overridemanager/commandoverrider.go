/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/overridemanager/commandoverride.go
- refactor commandOverrider as a struct that implements the Overrider interface
*/
package overridemanager

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/apps/v1alpha1"
)

type CommandOverrider struct{}

func (o *CommandOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	commandOverriders := overriders.Overriders.CommandOverriders
	for index := range commandOverriders {
		patches, err := buildCommandArgsPatches(CommandString, rawObj, &commandOverriders[index])
		if err != nil {
			return err
		}

		klog.V(4).Infof("Parsed JSON patches by commandOverriders(%+v): %+v", commandOverriders[index], patches)
		if err = applyJSONPatch(rawObj, patches); err != nil {
			return err
		}
	}

	return nil
}

// buildCommandArgsPatches build JSON patches for the resource object according to override declaration.
func buildCommandArgsPatches(target string, rawObj *unstructured.Unstructured, commandRunOverrider *v1alpha1.CommandArgsOverrider) ([]overrideOption, error) {
	switch rawObj.GetKind() {
	case PodKind:
		return buildCommandArgsPatchesWithPath(target, "spec/containers", rawObj, commandRunOverrider)
	case ReplicaSetKind:
		fallthrough
	case DeploymentKind:
		fallthrough
	case DaemonSetKind:
		fallthrough
	case JobKind:
		fallthrough
	case StatefulSetKind:
		return buildCommandArgsPatchesWithPath(target, "spec/template/spec/containers", rawObj, commandRunOverrider)
	}
	return nil, nil
}

func buildCommandArgsPatchesWithPath(target string, specContainersPath string, rawObj *unstructured.Unstructured, commandRunOverrider *v1alpha1.CommandArgsOverrider) ([]overrideOption, error) {
	patches := make([]overrideOption, 0)
	containers, ok, err := unstructured.NestedSlice(rawObj.Object, strings.Split(specContainersPath, pathSplit)...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieves path(%s) from rawObj, error: %v", specContainersPath, err)
	}
	if !ok || len(containers) == 0 {
		return nil, nil
	}
	klog.V(4).Infof("buildCommandArgsPatchesWithPath containers info (%+v)", containers)
	for index, container := range containers {
		if container.(map[string]interface{})["name"] == commandRunOverrider.ContainerName {
			commandArgsPath := fmt.Sprintf("/%s/%d/%s", specContainersPath, index, target)
			commandArgsValue := make([]string, 0)
			var patch overrideOption
			// if target is nil, to add new [target]
			if container.(map[string]interface{})[target] == nil {
				patch, _ = acquireAddOverrideOption(commandArgsPath, commandRunOverrider)
			} else {
				for _, val := range container.(map[string]interface{})[target].([]interface{}) {
					commandArgsValue = append(commandArgsValue, fmt.Sprintf("%s", val))
				}
				patch, _ = acquireReplaceOverrideOption(commandArgsPath, commandArgsValue, commandRunOverrider)
			}

			klog.V(4).Infof("[buildCommandArgsPatchesWithPath] containers patch info (%+v)", patch)
			patches = append(patches, patch)
		}
	}
	return patches, nil
}

func acquireAddOverrideOption(commandArgsPath string, commandOverrider *v1alpha1.CommandArgsOverrider) (overrideOption, error) {
	if !strings.HasPrefix(commandArgsPath, pathSplit) {
		return overrideOption{}, fmt.Errorf("internal error: [acquireCommandOverrideOption] commandRunPath should be start with / character")
	}
	newCommandArgs := overrideCommandArgs([]string{}, commandOverrider)
	return overrideOption{
		Op:    string(v1alpha1.OverriderOpAdd),
		Path:  commandArgsPath,
		Value: newCommandArgs,
	}, nil
}

func acquireReplaceOverrideOption(commandArgsPath string, commandArgsValue []string, commandOverrider *v1alpha1.CommandArgsOverrider) (overrideOption, error) {
	if !strings.HasPrefix(commandArgsPath, pathSplit) {
		return overrideOption{}, fmt.Errorf("internal error: [acquireCommandOverrideOption] commandRunPath should be start with / character")
	}
	newCommandArgs := overrideCommandArgs(commandArgsValue, commandOverrider)
	return overrideOption{
		Op:    string(v1alpha1.OverriderOpReplace),
		Path:  commandArgsPath,
		Value: newCommandArgs,
	}, nil
}

func overrideCommandArgs(curCommandArgs []string, commandArgsOverrider *v1alpha1.CommandArgsOverrider) []string {
	var newCommandArgs []string
	switch commandArgsOverrider.Operator {
	case v1alpha1.OverriderOpAdd:
		newCommandArgs = append(curCommandArgs, commandArgsOverrider.Value...)
	case v1alpha1.OverriderOpRemove:
		newCommandArgs = commandArgsRemove(curCommandArgs, commandArgsOverrider.Value)
	default:
		newCommandArgs = curCommandArgs
		klog.V(4).Infof("[overrideCommandArgs], op: %s , op not supported, ignored.", v1alpha1.OverriderOpRemove)
	}
	return newCommandArgs
}

func commandArgsRemove(curCommandArgs []string, removeValues []string) []string {
	newCommandArgs := make([]string, 0, len(curCommandArgs))
	currentSet := sets.NewString(removeValues...)
	for _, val := range curCommandArgs {
		if !currentSet.Has(val) {
			newCommandArgs = append(newCommandArgs, val)
		}
	}
	return newCommandArgs
}
