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
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/apps/v1alpha1"
)

type ResourcesOverrider struct{}

func (o *ResourcesOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	resourcesOverriders := overriders.Overriders.ResourcesOverriders
	for index := range resourcesOverriders {
		patches, err := buildResourcesPatches(rawObj, &resourcesOverriders[index])
		if err != nil {
			return err
		}

		klog.V(4).Infof("Parsed JSON patches by ResourcesOverrider(%+v): %+v", resourcesOverriders[index], patches)
		if err = applyJSONPatch(rawObj, patches); err != nil {
			return err
		}
	}

	return nil
}

// buildResourcesPatches build JSON patches for the resource object according to override declaration.
func buildResourcesPatches(rawObj *unstructured.Unstructured, resourcesOverrider *v1alpha1.ResourcesOverrider) ([]overrideOption, error) {
	switch rawObj.GetKind() {
	case PodKind:
		return buildResourcesPatchesWithPath("spec/containers", rawObj, resourcesOverrider)
	case ReplicaSetKind:
		fallthrough
	case DeploymentKind:
		fallthrough
	case DaemonSetKind:
		fallthrough
	case JobKind:
		fallthrough
	case StatefulSetKind:
		return buildResourcesPatchesWithPath("spec/template/spec/containers", rawObj, resourcesOverrider)
	}
	return nil, nil
}

func buildResourcesPatchesWithPath(specContainersPath string, rawObj *unstructured.Unstructured, resourcesOverrider *v1alpha1.ResourcesOverrider) ([]overrideOption, error) {
	patches := make([]overrideOption, 0)
	containers, ok, err := unstructured.NestedSlice(rawObj.Object, strings.Split(specContainersPath, pathSplit)...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve path(%s) from rawObj, error: %v", specContainersPath, err)
	}
	if !ok || len(containers) == 0 {
		return nil, nil
	}
	klog.V(4).Infof("buildResourcesPatchesWithPath containers info (%+v)", containers)
	for index, container := range containers {
		if container.(map[string]interface{})["name"] == resourcesOverrider.ContainerName {
			resourcesPath := fmt.Sprintf("/%s/%d/resources", specContainersPath, index)
			resourcesValue := resourcesOverrider.Value

			// Acquire selective override patches based on non-zero fields in resourcesValue
			patchList, err := acquireOverride(resourcesPath, resourcesValue)
			if err != nil {
				return nil, err
			}

			klog.V(4).Infof("[buildResourcesPatchesWithPath] containers patch info (%+v)", patchList)
			patches = append(patches, patchList...)
		}
	}
	return patches, nil
}

// acquireOverrideOptions for adding selective resources override.
func acquireOverride(resourcesPath string, resourcesValue corev1.ResourceRequirements) ([]overrideOption, error) {
	if !strings.HasPrefix(resourcesPath, pathSplit) {
		return nil, errors.New("internal error: [acquireOverride] resourcesPath should start with / character")
	}

	patches := []overrideOption{}

	// Handle Requests
	for resourceName, quantity := range resourcesValue.Requests {
		// If quantity is non-zero, add a patch for this specific request
		if !quantity.IsZero() {
			requestPath := fmt.Sprintf("%s/requests/%s", resourcesPath, resourceName)
			patches = append(patches, overrideOption{
				Op:    string(v1alpha1.OverriderOpReplace),
				Path:  requestPath,
				Value: quantity.String(),
			})
		}
	}

	// Handle Limits
	for resourceName, quantity := range resourcesValue.Limits {
		// If quantity is non-zero, add a patch for this specific limit
		if !quantity.IsZero() {
			limitPath := fmt.Sprintf("%s/limits/%s", resourcesPath, resourceName)
			patches = append(patches, overrideOption{
				Op:    string(v1alpha1.OverriderOpReplace),
				Path:  limitPath,
				Value: quantity.String(),
			})
		}
	}

	return patches, nil
}
