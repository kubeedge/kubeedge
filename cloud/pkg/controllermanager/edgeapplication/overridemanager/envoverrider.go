/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package overridemanager

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/apps/v1alpha1"
)

type EnvOverrider struct{}

func (o *EnvOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	envOverriders := overriders.Overriders.EnvOverriders
	for index := range envOverriders {
		patches, err := buildEnvPatches(rawObj, &envOverriders[index])
		if err != nil {
			return err
		}

		klog.V(4).Infof("Parsed JSON patches by EnvOverrider(%+v): %+v", envOverriders[index], patches)
		if err = applyJSONPatch(rawObj, patches); err != nil {
			return err
		}
	}

	return nil
}

// buildEnvPatches build JSON patches for the resource object according to EnvOverrider declaration.
func buildEnvPatches(rawObj *unstructured.Unstructured, envOverrider *v1alpha1.EnvOverrider) ([]overrideOption, error) {
	switch rawObj.GetKind() {
	case PodKind:
		return buildEnvPatchesWithPath("spec/containers", rawObj, envOverrider)
	case ReplicaSetKind:
		fallthrough
	case DeploymentKind:
		fallthrough
	case DaemonSetKind:
		fallthrough
	case JobKind:
		fallthrough
	case StatefulSetKind:
		return buildEnvPatchesWithPath("spec/template/spec/containers", rawObj, envOverrider)
	}
	return nil, nil
}

func buildEnvPatchesWithPath(specContainersPath string, rawObj *unstructured.Unstructured, envOverrider *v1alpha1.EnvOverrider) ([]overrideOption, error) {
	patches := make([]overrideOption, 0)
	containers, ok, err := unstructured.NestedSlice(rawObj.Object, strings.Split(specContainersPath, pathSplit)...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve path(%s) from rawObj, error: %v", specContainersPath, err)
	}
	if !ok || len(containers) == 0 {
		return nil, nil
	}
	klog.V(4).Infof("buildEnvPatchesWithPath containers info (%+v)", containers)
	for index, container := range containers {
		if container.(map[string]interface{})["name"] == envOverrider.ContainerName {
			envPath := fmt.Sprintf("/%s/%d/env", specContainersPath, index)
			envValue := make([]corev1.EnvVar, 0)
			var patch overrideOption
			// if env is nil, to add new [env]
			if container.(map[string]interface{})["env"] == nil {
				patch, _ = acquireAddEnvOverrideOption(envPath, envOverrider)
			} else {
				env, ok := container.(map[string]interface{})["env"].([]interface{})
				if !ok {
					return nil, fmt.Errorf("failed to retrieve env from container")
				}
				for _, val := range env {
					envVar, err := convertToEnvVar(val)
					if err != nil {
						return nil, err
					}
					envValue = append(envValue, *envVar)
				}
				patch, _ = acquireReplaceEnvOverrideOption(envPath, envValue, envOverrider)
			}

			klog.V(4).Infof("[buildEnvPatchesWithPath] containers patch info (%+v)", patch)
			patches = append(patches, patch)
		}
	}
	return patches, nil
}

func acquireAddEnvOverrideOption(envPath string, envOverrider *v1alpha1.EnvOverrider) (overrideOption, error) {
	if !strings.HasPrefix(envPath, pathSplit) {
		return overrideOption{}, fmt.Errorf("internal error: [acquireAddEnvOverrideOption] envPath should start with / character")
	}
	newEnv, err := overrideEnv([]corev1.EnvVar{}, envOverrider)
	if err != nil {
		return overrideOption{}, err
	}
	return overrideOption{
		Op:    string(v1alpha1.OverriderOpAdd),
		Path:  envPath,
		Value: newEnv,
	}, nil
}

func acquireReplaceEnvOverrideOption(envPath string, envValue []corev1.EnvVar, envOverrider *v1alpha1.EnvOverrider) (overrideOption, error) {
	if !strings.HasPrefix(envPath, pathSplit) {
		return overrideOption{}, fmt.Errorf("internal error: [acquireReplaceEnvOverrideOption] envPath should start with / character")
	}

	newEnv, err := overrideEnv(envValue, envOverrider)
	if err != nil {
		return overrideOption{}, err
	}

	return overrideOption{
		Op:    string(v1alpha1.OverriderOpReplace),
		Path:  envPath,
		Value: newEnv,
	}, nil
}

func overrideEnv(curEnv []corev1.EnvVar, envOverrider *v1alpha1.EnvOverrider) ([]corev1.EnvVar, error) {
	var newEnv []corev1.EnvVar
	switch envOverrider.Operator {
	case v1alpha1.OverriderOpAdd:
		newEnv = append(curEnv, envOverrider.Value...)
	case v1alpha1.OverriderOpRemove:
		newEnv = envRemove(curEnv, envOverrider.Value)
	case v1alpha1.OverriderOpReplace:
		newEnv = replaceEnv(curEnv, envOverrider.Value)
	default:
		newEnv = curEnv
		klog.V(4).Infof("[overrideEnv], op: %s , op not supported, ignored.", envOverrider.Operator)
	}
	return newEnv, nil
}

func replaceEnv(curEnv []corev1.EnvVar, replaceValues []corev1.EnvVar) []corev1.EnvVar {
	newEnv := make([]corev1.EnvVar, 0, len(curEnv))
	currentMap := make(map[string]corev1.EnvVar)

	// Populate current map with existing environment variables
	for _, envVar := range curEnv {
		currentMap[envVar.Name] = envVar
	}

	// Replace or add new environment variables
	for _, replaceVar := range replaceValues {
		currentMap[replaceVar.Name] = replaceVar
	}

	// Convert map back to slice
	for _, envVar := range currentMap {
		newEnv = append(newEnv, envVar)
	}

	return newEnv
}

func envRemove(curEnv []corev1.EnvVar, removeValues []corev1.EnvVar) []corev1.EnvVar {
	newEnv := make([]corev1.EnvVar, 0, len(curEnv))
	currentSet := sets.NewString()
	for _, val := range removeValues {
		currentSet.Insert(val.Name)
	}
	for _, envVar := range curEnv {
		if !currentSet.Has(envVar.Name) {
			newEnv = append(newEnv, envVar)
		}
	}
	return newEnv
}

func convertToEnvVar(value interface{}) (*corev1.EnvVar, error) {
	envMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert env value to map[string]interface{}")
	}
	name, ok := envMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to convert env value name to string")
	}
	var env corev1.EnvVar
	env.Name = name
	if value, ok := envMap["value"].(string); ok {
		env.Value = value
	}
	if value, ok := envMap["valueFrom"].(map[string]interface{}); ok {
		valueFrom, err := convertToEnvVarSource(value)
		if err != nil {
			return nil, err
		}
		env.ValueFrom = &valueFrom
	}
	return &env, nil
}

func convertToEnvVarSource(value map[string]interface{}) (corev1.EnvVarSource, error) {
	var envVarSource corev1.EnvVarSource
	var err error
	if envVarSource.FieldRef, err = processFieldRef(value); err != nil {
		return envVarSource, err
	}
	if envVarSource.ResourceFieldRef, err = processResourceFieldRef(value); err != nil {
		return envVarSource, err
	}
	if envVarSource.ConfigMapKeyRef, err = processConfigMapKeyRef(value); err != nil {
		return envVarSource, err
	}
	if envVarSource.SecretKeyRef, err = processSecretKeyRef(value); err != nil {
		return envVarSource, err
	}

	return envVarSource, nil
}

func processFieldRef(value map[string]interface{}) (*corev1.ObjectFieldSelector, error) {
	fp, ffOK, err := unstructured.NestedString(value, "fieldRef", "fieldPath")
	if err != nil {
		return nil, err
	}
	av, faOK, err := unstructured.NestedString(value, "fieldRef", "apiVersion")
	if err != nil {
		return nil, err
	}
	if ffOK && faOK {
		return &corev1.ObjectFieldSelector{FieldPath: fp, APIVersion: av}, nil
	}
	return nil, nil
}

func processResourceFieldRef(value map[string]interface{}) (*corev1.ResourceFieldSelector, error) {
	r, rrOK, err := unstructured.NestedString(value, "resourceFieldRef", "resource")
	if err != nil {
		return nil, err
	}
	c, rcOK, err := unstructured.NestedString(value, "resourceFieldRef", "containerName")
	if err != nil {
		return nil, err
	}
	divisor, rdOK, err := unstructured.NestedString(value, "resourceFieldRef", "divisor")
	if err != nil {
		return nil, err
	}

	if rrOK && rcOK && rdOK {
		return &corev1.ResourceFieldSelector{
			ContainerName: c,
			Resource:      r,
			Divisor:       resource.MustParse(divisor),
		}, nil
	}
	return nil, nil
}

func processConfigMapKeyRef(value map[string]interface{}) (*corev1.ConfigMapKeySelector, error) {
	name, cnOK, err := unstructured.NestedString(value, "configMapKeyRef", "name")
	if err != nil {
		return nil, err
	}
	key, ckOK, err := unstructured.NestedString(value, "configMapKeyRef", "key")
	if err != nil {
		return nil, err
	}
	if cnOK && ckOK {
		return &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: name},
			Key:                  key,
		}, nil
	}
	return nil, nil
}

func processSecretKeyRef(value map[string]interface{}) (*corev1.SecretKeySelector, error) {
	name, snOK, err := unstructured.NestedString(value, "secretKeyRef", "name")
	if err != nil {
		return nil, err
	}
	key, skOK, err := unstructured.NestedString(value, "secretKeyRef", "key")
	if err != nil {
		return nil, err
	}

	if snOK && skOK {
		return &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: name},
			Key:                  key,
		}, nil
	}
	return nil, nil
}
