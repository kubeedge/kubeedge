/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/overridemanager/imageoverride.go
- refactor imageOverrider as a struct that implements the Overrider interface
*/
package overridemanager

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/groupingcontroller/edgeapplication/overridemanager/imageparser"
	groupingv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/grouping/v1alpha1"
)

const (
	pathSplit         = "/"
	podSpecPrefix     = "/spec"
	podTemplatePrefix = "/spec/template/spec"
)

type ImageOverrider struct{}

var _ Overrider = &ImageOverrider{}

func (o *ImageOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	imageOverriders := overriders.Overriders.ImageOverrider
	for index := range imageOverriders {
		patches, err := buildPatches(rawObj, &imageOverriders[index])
		if err != nil {
			klog.Errorf("Build patches with imageOverrides err: %v", err)
			return err
		}

		klog.V(4).Infof("Parsed JSON patches by imageOverriders(%+v): %+v", imageOverriders[index], patches)
		if err = applyJSONPatch(rawObj, patches); err != nil {
			return err
		}
	}

	return nil
}

// buildPatches parse JSON patches from resource object by imageOverriders
func buildPatches(rawObj *unstructured.Unstructured, imageOverrider *groupingv1alpha1.ImageOverrider) ([]overrideOption, error) {
	if imageOverrider.Predicate == nil {
		return buildPatchesWithEmptyPredicate(rawObj, imageOverrider)
	}

	return buildPatchesWithPredicate(rawObj, imageOverrider)
}

func buildPatchesWithEmptyPredicate(rawObj *unstructured.Unstructured, imageOverrider *groupingv1alpha1.ImageOverrider) ([]overrideOption, error) {
	switch rawObj.GetKind() {
	case PodKind:
		podObj, err := ConvertToPod(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Pod from unstructured object: %v", err)
		}
		return extractPatchesBy(podObj.Spec, podSpecPrefix, imageOverrider)
	case ReplicaSetKind:
		replicaSetObj, err := ConvertToReplicaSet(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ReplicaSet from unstructured object: %v", err)
		}
		return extractPatchesBy(replicaSetObj.Spec.Template.Spec, podTemplatePrefix, imageOverrider)
	case DeploymentKind:
		deploymentObj, err := ConvertToDeployment(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Deployment from unstructured object: %v", err)
		}
		return extractPatchesBy(deploymentObj.Spec.Template.Spec, podTemplatePrefix, imageOverrider)
	case DaemonSetKind:
		daemonSetObj, err := ConvertToDaemonSet(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DaemonSet from unstructured object: %v", err)
		}
		return extractPatchesBy(daemonSetObj.Spec.Template.Spec, podTemplatePrefix, imageOverrider)
	case StatefulSetKind:
		statefulSetObj, err := ConvertToStatefulSet(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert StatefulSet from unstructured object: %v", err)
		}
		return extractPatchesBy(statefulSetObj.Spec.Template.Spec, podTemplatePrefix, imageOverrider)
	case JobKind:
		jobObj, err := ConvertToJob(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Job from unstructured object: %v", err)
		}
		return extractPatchesBy(jobObj.Spec.Template.Spec, podTemplatePrefix, imageOverrider)
	}

	return nil, nil
}

func extractPatchesBy(podSpec corev1.PodSpec, prefixPath string, imageOverrider *groupingv1alpha1.ImageOverrider) ([]overrideOption, error) {
	patches := make([]overrideOption, 0)

	for containerIndex, container := range podSpec.Containers {
		patch, err := acquireOverrideOption(spliceImagePath(prefixPath, containerIndex), container.Image, imageOverrider)
		if err != nil {
			return nil, err
		}

		patches = append(patches, patch)
	}

	return patches, nil
}

func spliceImagePath(prefixPath string, containerIndex int) string {
	return fmt.Sprintf("%s/containers/%d/image", prefixPath, containerIndex)
}

func buildPatchesWithPredicate(rawObj *unstructured.Unstructured, imageOverrider *groupingv1alpha1.ImageOverrider) ([]overrideOption, error) {
	patches := make([]overrideOption, 0)

	imageValue, err := obtainImageValue(rawObj, imageOverrider.Predicate.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain imageValue with predicate path(%s), error: %v", imageOverrider.Predicate.Path, err)
	}

	patch, err := acquireOverrideOption(imageOverrider.Predicate.Path, imageValue, imageOverrider)
	if err != nil {
		return nil, err
	}

	patches = append(patches, patch)
	return patches, nil
}

func obtainImageValue(rawObj *unstructured.Unstructured, predicatePath string) (string, error) {
	pathSegments := strings.Split(strings.Trim(predicatePath, pathSplit), pathSplit)
	imageValue := ""
	currentObj := rawObj.Object
	ok := false
	for index := 0; index < len(pathSegments)-1; index++ {
		switch currentObj[pathSegments[index]].(type) {
		case map[string]interface{}:
			currentObj = currentObj[pathSegments[index]].(map[string]interface{})
		case []interface{}:
			tmpSlice := currentObj[pathSegments[index]].([]interface{})
			sliceIndex, err := strconv.ParseInt(pathSegments[index+1], 10, 32)
			if err != nil {
				return "", fmt.Errorf("path(%s) of rawObj's is not number", pathSegments[index+1])
			}
			currentObj = tmpSlice[sliceIndex].(map[string]interface{})
			index++
		default:
			return "", fmt.Errorf("path(%s) of rawObj's type is not map[string]interface{} and []interface{}", pathSegments[index])
		}
	}

	imageValue, ok = currentObj[pathSegments[len(pathSegments)-1]].(string)
	if !ok {
		return "", fmt.Errorf("failed to convert path(%s) to string", pathSegments[len(pathSegments)-1])
	}

	return imageValue, nil
}

func acquireOverrideOption(imagePath, curImage string, imageOverrider *groupingv1alpha1.ImageOverrider) (overrideOption, error) {
	if !strings.HasPrefix(imagePath, pathSplit) {
		return overrideOption{}, fmt.Errorf("imagePath should be start with / character")
	}

	newImage, err := overrideImage(curImage, imageOverrider)
	if err != nil {
		return overrideOption{}, err
	}

	return overrideOption{
		Op:    string(groupingv1alpha1.OverriderOpReplace),
		Path:  imagePath,
		Value: newImage,
	}, nil
}

func overrideImage(curImage string, imageOverrider *groupingv1alpha1.ImageOverrider) (string, error) {
	imageComponent, err := imageparser.Parse(curImage)
	if err != nil {
		return "", fmt.Errorf("failed to parse image value(%s), error: %v", curImage, err)
	}

	switch imageOverrider.Component {
	case groupingv1alpha1.Registry:
		switch imageOverrider.Operator {
		case groupingv1alpha1.OverriderOpAdd:
			imageComponent.SetHostname(imageComponent.Hostname() + imageOverrider.Value)
		case groupingv1alpha1.OverriderOpReplace:
			imageComponent.SetHostname(imageOverrider.Value)
		case groupingv1alpha1.OverriderOpRemove:
			imageComponent.RemoveHostname()
		}
		return imageComponent.String(), nil
	case groupingv1alpha1.Repository:
		switch imageOverrider.Operator {
		case groupingv1alpha1.OverriderOpAdd:
			imageComponent.SetRepository(imageComponent.Repository() + imageOverrider.Value)
		case groupingv1alpha1.OverriderOpReplace:
			imageComponent.SetRepository(imageOverrider.Value)
		case groupingv1alpha1.OverriderOpRemove:
			imageComponent.RemoveRepository()
		}
		return imageComponent.String(), nil
	case groupingv1alpha1.Tag:
		switch imageOverrider.Operator {
		case groupingv1alpha1.OverriderOpAdd:
			imageComponent.SetTagOrDigest(imageComponent.TagOrDigest() + imageOverrider.Value)
		case groupingv1alpha1.OverriderOpReplace:
			imageComponent.SetTagOrDigest(imageOverrider.Value)
		case groupingv1alpha1.OverriderOpRemove:
			imageComponent.RemoveTagOrDigest()
		}
		return imageComponent.String(), nil
	}

	// should never reach to here
	return "", fmt.Errorf("unsupported image component(%s)", imageOverrider.Component)
}
