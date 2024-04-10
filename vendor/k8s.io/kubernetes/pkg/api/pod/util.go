/*
Copyright 2015 The Kubernetes Authors.

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

package pod

import (
	"strings"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metavalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/helper"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"
	"k8s.io/kubernetes/pkg/features"
)

// ContainerType signifies container type
type ContainerType int

const (
	// Containers is for normal containers
	Containers ContainerType = 1 << iota
	// InitContainers is for init containers
	InitContainers
	// EphemeralContainers is for ephemeral containers
	EphemeralContainers
)

// AllContainers specifies that all containers be visited
const AllContainers ContainerType = (InitContainers | Containers | EphemeralContainers)

// AllFeatureEnabledContainers returns a ContainerType mask which includes all container
// types except for the ones guarded by feature gate.
func AllFeatureEnabledContainers() ContainerType {
	return AllContainers
}

// ContainerVisitor is called with each container spec, and returns true
// if visiting should continue.
type ContainerVisitor func(container *api.Container, containerType ContainerType) (shouldContinue bool)

// VisitContainers invokes the visitor function with a pointer to every container
// spec in the given pod spec with type set in mask. If visitor returns false,
// visiting is short-circuited. VisitContainers returns true if visiting completes,
// false if visiting was short-circuited.
func VisitContainers(podSpec *api.PodSpec, mask ContainerType, visitor ContainerVisitor) bool {
	if mask&InitContainers != 0 {
		for i := range podSpec.InitContainers {
			if !visitor(&podSpec.InitContainers[i], InitContainers) {
				return false
			}
		}
	}
	if mask&Containers != 0 {
		for i := range podSpec.Containers {
			if !visitor(&podSpec.Containers[i], Containers) {
				return false
			}
		}
	}
	if mask&EphemeralContainers != 0 {
		for i := range podSpec.EphemeralContainers {
			if !visitor((*api.Container)(&podSpec.EphemeralContainers[i].EphemeralContainerCommon), EphemeralContainers) {
				return false
			}
		}
	}
	return true
}

// Visitor is called with each object name, and returns true if visiting should continue
type Visitor func(name string) (shouldContinue bool)

func skipEmptyNames(visitor Visitor) Visitor {
	return func(name string) bool {
		if len(name) == 0 {
			// continue visiting
			return true
		}
		// delegate to visitor
		return visitor(name)
	}
}

// VisitPodSecretNames invokes the visitor function with the name of every secret
// referenced by the pod spec. If visitor returns false, visiting is short-circuited.
// Transitive references (e.g. pod -> pvc -> pv -> secret) are not visited.
// Returns true if visiting completed, false if visiting was short-circuited.
func VisitPodSecretNames(pod *api.Pod, visitor Visitor, containerType ContainerType) bool {
	visitor = skipEmptyNames(visitor)
	for _, reference := range pod.Spec.ImagePullSecrets {
		if !visitor(reference.Name) {
			return false
		}
	}
	VisitContainers(&pod.Spec, containerType, func(c *api.Container, containerType ContainerType) bool {
		return visitContainerSecretNames(c, visitor)
	})
	var source *api.VolumeSource
	for i := range pod.Spec.Volumes {
		source = &pod.Spec.Volumes[i].VolumeSource
		switch {
		case source.AzureFile != nil:
			if len(source.AzureFile.SecretName) > 0 && !visitor(source.AzureFile.SecretName) {
				return false
			}
		case source.CephFS != nil:
			if source.CephFS.SecretRef != nil && !visitor(source.CephFS.SecretRef.Name) {
				return false
			}
		case source.Cinder != nil:
			if source.Cinder.SecretRef != nil && !visitor(source.Cinder.SecretRef.Name) {
				return false
			}
		case source.FlexVolume != nil:
			if source.FlexVolume.SecretRef != nil && !visitor(source.FlexVolume.SecretRef.Name) {
				return false
			}
		case source.Projected != nil:
			for j := range source.Projected.Sources {
				if source.Projected.Sources[j].Secret != nil {
					if !visitor(source.Projected.Sources[j].Secret.Name) {
						return false
					}
				}
			}
		case source.RBD != nil:
			if source.RBD.SecretRef != nil && !visitor(source.RBD.SecretRef.Name) {
				return false
			}
		case source.Secret != nil:
			if !visitor(source.Secret.SecretName) {
				return false
			}
		case source.ScaleIO != nil:
			if source.ScaleIO.SecretRef != nil && !visitor(source.ScaleIO.SecretRef.Name) {
				return false
			}
		case source.ISCSI != nil:
			if source.ISCSI.SecretRef != nil && !visitor(source.ISCSI.SecretRef.Name) {
				return false
			}
		case source.StorageOS != nil:
			if source.StorageOS.SecretRef != nil && !visitor(source.StorageOS.SecretRef.Name) {
				return false
			}
		case source.CSI != nil:
			if source.CSI.NodePublishSecretRef != nil && !visitor(source.CSI.NodePublishSecretRef.Name) {
				return false
			}
		}
	}
	return true
}

func visitContainerSecretNames(container *api.Container, visitor Visitor) bool {
	for _, env := range container.EnvFrom {
		if env.SecretRef != nil {
			if !visitor(env.SecretRef.Name) {
				return false
			}
		}
	}
	for _, envVar := range container.Env {
		if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
			if !visitor(envVar.ValueFrom.SecretKeyRef.Name) {
				return false
			}
		}
	}
	return true
}

// VisitPodConfigmapNames invokes the visitor function with the name of every configmap
// referenced by the pod spec. If visitor returns false, visiting is short-circuited.
// Transitive references (e.g. pod -> pvc -> pv -> secret) are not visited.
// Returns true if visiting completed, false if visiting was short-circuited.
func VisitPodConfigmapNames(pod *api.Pod, visitor Visitor, containerType ContainerType) bool {
	visitor = skipEmptyNames(visitor)
	VisitContainers(&pod.Spec, containerType, func(c *api.Container, containerType ContainerType) bool {
		return visitContainerConfigmapNames(c, visitor)
	})
	var source *api.VolumeSource
	for i := range pod.Spec.Volumes {
		source = &pod.Spec.Volumes[i].VolumeSource
		switch {
		case source.Projected != nil:
			for j := range source.Projected.Sources {
				if source.Projected.Sources[j].ConfigMap != nil {
					if !visitor(source.Projected.Sources[j].ConfigMap.Name) {
						return false
					}
				}
			}
		case source.ConfigMap != nil:
			if !visitor(source.ConfigMap.Name) {
				return false
			}
		}
	}
	return true
}

func visitContainerConfigmapNames(container *api.Container, visitor Visitor) bool {
	for _, env := range container.EnvFrom {
		if env.ConfigMapRef != nil {
			if !visitor(env.ConfigMapRef.Name) {
				return false
			}
		}
	}
	for _, envVar := range container.Env {
		if envVar.ValueFrom != nil && envVar.ValueFrom.ConfigMapKeyRef != nil {
			if !visitor(envVar.ValueFrom.ConfigMapKeyRef.Name) {
				return false
			}
		}
	}
	return true
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *api.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status api.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == api.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status api.PodStatus) *api.PodCondition {
	_, condition := GetPodCondition(&status, api.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *api.PodStatus, conditionType api.PodConditionType) (int, *api.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// UpdatePodCondition updates existing pod condition or creates a new one. Sets LastTransitionTime to now if the
// status has changed.
// Returns true if pod condition has changed or has been added.
func UpdatePodCondition(status *api.PodStatus, condition *api.PodCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this pod condition.
	conditionIndex, oldCondition := GetPodCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastProbeTime.Equal(&oldCondition.LastProbeTime) &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

func checkContainerUseIndivisibleHugePagesValues(container api.Container) bool {
	for resourceName, quantity := range container.Resources.Limits {
		if helper.IsHugePageResourceName(resourceName) {
			if !helper.IsHugePageResourceValueDivisible(resourceName, quantity) {
				return true
			}
		}
	}

	for resourceName, quantity := range container.Resources.Requests {
		if helper.IsHugePageResourceName(resourceName) {
			if !helper.IsHugePageResourceValueDivisible(resourceName, quantity) {
				return true
			}
		}
	}

	return false
}

// usesIndivisibleHugePagesValues returns true if the one of the containers uses non-integer multiple
// of huge page unit size
func usesIndivisibleHugePagesValues(podSpec *api.PodSpec) bool {
	foundIndivisibleHugePagesValue := false
	VisitContainers(podSpec, AllContainers, func(c *api.Container, containerType ContainerType) bool {
		if checkContainerUseIndivisibleHugePagesValues(*c) {
			foundIndivisibleHugePagesValue = true
		}
		return !foundIndivisibleHugePagesValue // continue visiting if we haven't seen an invalid value yet
	})

	if foundIndivisibleHugePagesValue {
		return true
	}

	for resourceName, quantity := range podSpec.Overhead {
		if helper.IsHugePageResourceName(resourceName) {
			if !helper.IsHugePageResourceValueDivisible(resourceName, quantity) {
				return true
			}
		}
	}

	return false
}

// hasInvalidTopologySpreadConstraintLabelSelector return true if spec.TopologySpreadConstraints have any entry with invalid labelSelector
func hasInvalidTopologySpreadConstraintLabelSelector(spec *api.PodSpec) bool {
	for _, constraint := range spec.TopologySpreadConstraints {
		errs := metavalidation.ValidateLabelSelector(constraint.LabelSelector, metavalidation.LabelSelectorValidationOptions{AllowInvalidLabelValueInSelector: false}, nil)
		if len(errs) != 0 {
			return true
		}
	}
	return false
}

// GetValidationOptionsFromPodSpecAndMeta returns validation options based on pod specs and metadata
func GetValidationOptionsFromPodSpecAndMeta(podSpec, oldPodSpec *api.PodSpec, podMeta, oldPodMeta *metav1.ObjectMeta) apivalidation.PodValidationOptions {
	// default pod validation options based on feature gate
	opts := apivalidation.PodValidationOptions{
		AllowInvalidPodDeletionCost: !utilfeature.DefaultFeatureGate.Enabled(features.PodDeletionCost),
		// Allow pod spec to use status.hostIPs in downward API if feature is enabled
		AllowHostIPsField: utilfeature.DefaultFeatureGate.Enabled(features.PodHostIPs),
		// Do not allow pod spec to use non-integer multiple of huge page unit size default
		AllowIndivisibleHugePagesValues:                   false,
		AllowInvalidLabelValueInSelector:                  false,
		AllowInvalidTopologySpreadConstraintLabelSelector: false,
		AllowMutableNodeSelectorAndNodeAffinity:           utilfeature.DefaultFeatureGate.Enabled(features.PodSchedulingReadiness),
	}

	if oldPodSpec != nil {
		// if old spec has status.hostIPs downwardAPI set, we must allow it
		opts.AllowHostIPsField = opts.AllowHostIPsField || hasUsedDownwardAPIFieldPathWithPodSpec(oldPodSpec, "status.hostIPs")

		// if old spec used non-integer multiple of huge page unit size, we must allow it
		opts.AllowIndivisibleHugePagesValues = usesIndivisibleHugePagesValues(oldPodSpec)

		opts.AllowInvalidLabelValueInSelector = hasInvalidLabelValueInAffinitySelector(oldPodSpec)
		// if old spec has invalid labelSelector in topologySpreadConstraint, we must allow it
		opts.AllowInvalidTopologySpreadConstraintLabelSelector = hasInvalidTopologySpreadConstraintLabelSelector(oldPodSpec)
	}
	if oldPodMeta != nil && !opts.AllowInvalidPodDeletionCost {
		// This is an update, so validate only if the existing object was valid.
		_, err := helper.GetDeletionCostFromPodAnnotations(oldPodMeta.Annotations)
		opts.AllowInvalidPodDeletionCost = err != nil
	}

	return opts
}

func hasUsedDownwardAPIFieldPathWithPodSpec(podSpec *api.PodSpec, fieldPath string) bool {
	if podSpec == nil {
		return false
	}
	for _, vol := range podSpec.Volumes {
		if hasUsedDownwardAPIFieldPathWithVolume(&vol, fieldPath) {
			return true
		}
	}
	for _, c := range podSpec.InitContainers {
		if hasUsedDownwardAPIFieldPathWithContainer(&c, fieldPath) {
			return true
		}
	}
	for _, c := range podSpec.Containers {
		if hasUsedDownwardAPIFieldPathWithContainer(&c, fieldPath) {
			return true
		}
	}
	return false
}

func hasUsedDownwardAPIFieldPathWithVolume(volume *api.Volume, fieldPath string) bool {
	if volume == nil || volume.DownwardAPI == nil {
		return false
	}
	for _, file := range volume.DownwardAPI.Items {
		if file.FieldRef != nil &&
			file.FieldRef.FieldPath == fieldPath {
			return true
		}
	}
	return false
}

func hasUsedDownwardAPIFieldPathWithContainer(container *api.Container, fieldPath string) bool {
	if container == nil {
		return false
	}
	for _, env := range container.Env {
		if env.ValueFrom != nil &&
			env.ValueFrom.FieldRef != nil &&
			env.ValueFrom.FieldRef.FieldPath == fieldPath {
			return true
		}
	}
	return false
}

// GetValidationOptionsFromPodTemplate will return pod validation options for specified template.
func GetValidationOptionsFromPodTemplate(podTemplate, oldPodTemplate *api.PodTemplateSpec) apivalidation.PodValidationOptions {
	var newPodSpec, oldPodSpec *api.PodSpec
	var newPodMeta, oldPodMeta *metav1.ObjectMeta
	// we have to be careful about nil pointers here
	// replication controller in particular is prone to passing nil
	if podTemplate != nil {
		newPodSpec = &podTemplate.Spec
		newPodMeta = &podTemplate.ObjectMeta
	}
	if oldPodTemplate != nil {
		oldPodSpec = &oldPodTemplate.Spec
		oldPodMeta = &oldPodTemplate.ObjectMeta
	}
	return GetValidationOptionsFromPodSpecAndMeta(newPodSpec, oldPodSpec, newPodMeta, oldPodMeta)
}

// DropDisabledTemplateFields removes disabled fields from the pod template metadata and spec.
// This should be called from PrepareForCreate/PrepareForUpdate for all resources containing a PodTemplateSpec
func DropDisabledTemplateFields(podTemplate, oldPodTemplate *api.PodTemplateSpec) {
	var (
		podSpec           *api.PodSpec
		podAnnotations    map[string]string
		oldPodSpec        *api.PodSpec
		oldPodAnnotations map[string]string
	)
	if podTemplate != nil {
		podSpec = &podTemplate.Spec
		podAnnotations = podTemplate.Annotations
	}
	if oldPodTemplate != nil {
		oldPodSpec = &oldPodTemplate.Spec
		oldPodAnnotations = oldPodTemplate.Annotations
	}
	dropDisabledFields(podSpec, podAnnotations, oldPodSpec, oldPodAnnotations)
}

// DropDisabledPodFields removes disabled fields from the pod metadata and spec.
// This should be called from PrepareForCreate/PrepareForUpdate for all resources containing a Pod
func DropDisabledPodFields(pod, oldPod *api.Pod) {
	var (
		podSpec           *api.PodSpec
		podStatus         *api.PodStatus
		podAnnotations    map[string]string
		oldPodSpec        *api.PodSpec
		oldPodStatus      *api.PodStatus
		oldPodAnnotations map[string]string
	)
	if pod != nil {
		podSpec = &pod.Spec
		podStatus = &pod.Status
		podAnnotations = pod.Annotations
	}
	if oldPod != nil {
		oldPodSpec = &oldPod.Spec
		oldPodStatus = &oldPod.Status
		oldPodAnnotations = oldPod.Annotations
	}
	dropDisabledFields(podSpec, podAnnotations, oldPodSpec, oldPodAnnotations)
	dropDisabledPodStatusFields(podStatus, oldPodStatus, podSpec, oldPodSpec)
}

// dropDisabledFields removes disabled fields from the pod metadata and spec.
func dropDisabledFields(
	podSpec *api.PodSpec, podAnnotations map[string]string,
	oldPodSpec *api.PodSpec, oldPodAnnotations map[string]string,
) {
	// the new spec must always be non-nil
	if podSpec == nil {
		podSpec = &api.PodSpec{}
	}

	if !utilfeature.DefaultFeatureGate.Enabled(features.AppArmor) && !appArmorInUse(oldPodAnnotations) {
		for k := range podAnnotations {
			if strings.HasPrefix(k, v1.AppArmorBetaContainerAnnotationKeyPrefix) {
				delete(podAnnotations, k)
			}
		}
	}

	// If the feature is disabled and not in use, drop the hostUsers field.
	if !utilfeature.DefaultFeatureGate.Enabled(features.UserNamespacesSupport) && !hostUsersInUse(oldPodSpec) {
		// Drop the field in podSpec only if SecurityContext is not nil.
		// If it is nil, there is no need to set hostUsers=nil (it will be nil too).
		if podSpec.SecurityContext != nil {
			podSpec.SecurityContext.HostUsers = nil
		}
	}

	// If the feature is disabled and not in use, drop the schedulingGates field.
	if !utilfeature.DefaultFeatureGate.Enabled(features.PodSchedulingReadiness) && !schedulingGatesInUse(oldPodSpec) {
		podSpec.SchedulingGates = nil
	}

	dropDisabledProcMountField(podSpec, oldPodSpec)

	dropDisabledTopologySpreadConstraintsFields(podSpec, oldPodSpec)
	dropDisabledNodeInclusionPolicyFields(podSpec, oldPodSpec)
	dropDisabledMatchLabelKeysField(podSpec, oldPodSpec)
	dropDisabledDynamicResourceAllocationFields(podSpec, oldPodSpec)

	if !utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) && !inPlacePodVerticalScalingInUse(oldPodSpec) {
		// Drop ResizePolicy fields. Don't drop updates to Resources field as template.spec.resources
		// field is mutable for certain controllers. Let ValidatePodUpdate handle it.
		for i := range podSpec.Containers {
			podSpec.Containers[i].ResizePolicy = nil
		}
		for i := range podSpec.InitContainers {
			podSpec.InitContainers[i].ResizePolicy = nil
		}
		for i := range podSpec.EphemeralContainers {
			podSpec.EphemeralContainers[i].ResizePolicy = nil
		}
	}

	if !utilfeature.DefaultFeatureGate.Enabled(features.SidecarContainers) && !restartableInitContainersInUse(oldPodSpec) {
		// Drop the RestartPolicy field of init containers.
		for i := range podSpec.InitContainers {
			podSpec.InitContainers[i].RestartPolicy = nil
		}
		// For other types of containers, validateContainers will handle them.
	}
}

// dropDisabledPodStatusFields removes disabled fields from the pod status
func dropDisabledPodStatusFields(podStatus, oldPodStatus *api.PodStatus, podSpec, oldPodSpec *api.PodSpec) {
	// the new status is always be non-nil
	if podStatus == nil {
		podStatus = &api.PodStatus{}
	}

	if !utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) && !inPlacePodVerticalScalingInUse(oldPodSpec) {
		// Drop Resize, AllocatedResources, and Resources fields
		dropResourcesFields := func(csl []api.ContainerStatus) {
			for i := range csl {
				csl[i].AllocatedResources = nil
				csl[i].Resources = nil
			}
		}
		dropResourcesFields(podStatus.ContainerStatuses)
		dropResourcesFields(podStatus.InitContainerStatuses)
		dropResourcesFields(podStatus.EphemeralContainerStatuses)
		podStatus.Resize = ""
	}

	if !utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) && !dynamicResourceAllocationInUse(oldPodSpec) {
		podStatus.ResourceClaimStatuses = nil
	}

	// drop HostIPs to empty (disable PodHostIPs).
	if !utilfeature.DefaultFeatureGate.Enabled(features.PodHostIPs) && !hostIPsInUse(oldPodStatus) {
		podStatus.HostIPs = nil
	}
}

func hostIPsInUse(podStatus *api.PodStatus) bool {
	if podStatus == nil {
		return false
	}
	return len(podStatus.HostIPs) > 0
}

// dropDisabledDynamicResourceAllocationFields removes pod claim references from
// container specs and pod-level resource claims unless they are already used
// by the old pod spec.
func dropDisabledDynamicResourceAllocationFields(podSpec, oldPodSpec *api.PodSpec) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) && !dynamicResourceAllocationInUse(oldPodSpec) {
		dropResourceClaimRequests(podSpec.Containers)
		dropResourceClaimRequests(podSpec.InitContainers)
		dropEphemeralResourceClaimRequests(podSpec.EphemeralContainers)
		podSpec.ResourceClaims = nil
	}

	// Setting VolumeClaimTemplate.Resources.Claims should have been
	// treated as an error by validation when extending
	// ResourceRequirements in 1.26. Now we can only accept it and drop the
	// field.
	for i := range podSpec.Volumes {
		if podSpec.Volumes[i].Ephemeral != nil && podSpec.Volumes[i].Ephemeral.VolumeClaimTemplate != nil {
			podSpec.Volumes[i].Ephemeral.VolumeClaimTemplate.Spec.Resources.Claims = nil
		}
	}
}

func dynamicResourceAllocationInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}

	// We only need to check this field because the containers cannot have
	// resource requirements entries for claims without a corresponding
	// entry at the pod spec level.
	return len(podSpec.ResourceClaims) > 0
}

func dropResourceClaimRequests(containers []api.Container) {
	for i := range containers {
		containers[i].Resources.Claims = nil
	}
}

func dropEphemeralResourceClaimRequests(containers []api.EphemeralContainer) {
	for i := range containers {
		containers[i].Resources.Claims = nil
	}
}

// dropDisabledTopologySpreadConstraintsFields removes disabled fields from PodSpec related
// to TopologySpreadConstraints only if it is not already used by the old spec.
func dropDisabledTopologySpreadConstraintsFields(podSpec, oldPodSpec *api.PodSpec) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.MinDomainsInPodTopologySpread) &&
		!minDomainsInUse(oldPodSpec) &&
		podSpec != nil {
		for i := range podSpec.TopologySpreadConstraints {
			podSpec.TopologySpreadConstraints[i].MinDomains = nil
		}
	}
}

// minDomainsInUse returns true if the pod spec is non-nil
// and has non-nil MinDomains field in TopologySpreadConstraints.
func minDomainsInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}

	for _, c := range podSpec.TopologySpreadConstraints {
		if c.MinDomains != nil {
			return true
		}
	}
	return false
}

// dropDisabledProcMountField removes disabled fields from PodSpec related
// to ProcMount only if it is not already used by the old spec
func dropDisabledProcMountField(podSpec, oldPodSpec *api.PodSpec) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.ProcMountType) && !procMountInUse(oldPodSpec) {
		defaultProcMount := api.DefaultProcMount
		VisitContainers(podSpec, AllContainers, func(c *api.Container, containerType ContainerType) bool {
			if c.SecurityContext != nil && c.SecurityContext.ProcMount != nil {
				// The ProcMount field was improperly forced to non-nil in 1.12.
				// If the feature is disabled, and the existing object is not using any non-default values, and the ProcMount field is present in the incoming object, force to the default value.
				// Note: we cannot force the field to nil when the feature is disabled because it causes a diff against previously persisted data.
				c.SecurityContext.ProcMount = &defaultProcMount
			}
			return true
		})
	}
}

// dropDisabledNodeInclusionPolicyFields removes disabled fields from PodSpec related
// to NodeInclusionPolicy only if it is not used by the old spec.
func dropDisabledNodeInclusionPolicyFields(podSpec, oldPodSpec *api.PodSpec) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.NodeInclusionPolicyInPodTopologySpread) && podSpec != nil {
		if !nodeTaintsPolicyInUse(oldPodSpec) {
			for i := range podSpec.TopologySpreadConstraints {
				podSpec.TopologySpreadConstraints[i].NodeTaintsPolicy = nil
			}
		}
		if !nodeAffinityPolicyInUse(oldPodSpec) {
			for i := range podSpec.TopologySpreadConstraints {
				podSpec.TopologySpreadConstraints[i].NodeAffinityPolicy = nil
			}
		}
	}
}

// dropDisabledMatchLabelKeysField removes disabled fields from PodSpec related
// to MatchLabelKeys only if it is not already used by the old spec.
func dropDisabledMatchLabelKeysField(podSpec, oldPodSpec *api.PodSpec) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.MatchLabelKeysInPodTopologySpread) && !matchLabelKeysInUse(oldPodSpec) {
		for i := range podSpec.TopologySpreadConstraints {
			podSpec.TopologySpreadConstraints[i].MatchLabelKeys = nil
		}
	}
}

// matchLabelKeysInUse returns true if the pod spec is non-nil
// and has MatchLabelKeys field set in TopologySpreadConstraints.
func matchLabelKeysInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}

	for _, c := range podSpec.TopologySpreadConstraints {
		if len(c.MatchLabelKeys) > 0 {
			return true
		}
	}
	return false
}

// nodeAffinityPolicyInUse returns true if the pod spec is non-nil and has NodeAffinityPolicy field set
// in TopologySpreadConstraints
func nodeAffinityPolicyInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	for _, c := range podSpec.TopologySpreadConstraints {
		if c.NodeAffinityPolicy != nil {
			return true
		}
	}
	return false
}

// nodeTaintsPolicyInUse returns true if the pod spec is non-nil and has NodeTaintsPolicy field set
// in TopologySpreadConstraints
func nodeTaintsPolicyInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	for _, c := range podSpec.TopologySpreadConstraints {
		if c.NodeTaintsPolicy != nil {
			return true
		}
	}
	return false
}

// hostUsersInUse returns true if the pod spec has spec.hostUsers field set.
func hostUsersInUse(podSpec *api.PodSpec) bool {
	if podSpec != nil && podSpec.SecurityContext != nil && podSpec.SecurityContext.HostUsers != nil {
		return true
	}

	return false
}

// inPlacePodVerticalScalingInUse returns true if pod spec is non-nil and ResizePolicy is set
func inPlacePodVerticalScalingInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	var inUse bool
	VisitContainers(podSpec, Containers, func(c *api.Container, containerType ContainerType) bool {
		if len(c.ResizePolicy) > 0 {
			inUse = true
			return false
		}
		return true
	})
	return inUse
}

// procMountInUse returns true if the pod spec is non-nil and has a SecurityContext's ProcMount field set to a non-default value
func procMountInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}

	var inUse bool
	VisitContainers(podSpec, AllContainers, func(c *api.Container, containerType ContainerType) bool {
		if c.SecurityContext == nil || c.SecurityContext.ProcMount == nil {
			return true
		}
		if *c.SecurityContext.ProcMount != api.DefaultProcMount {
			inUse = true
			return false
		}
		return true
	})

	return inUse
}

// appArmorInUse returns true if the pod has apparmor related information
func appArmorInUse(podAnnotations map[string]string) bool {
	for k := range podAnnotations {
		if strings.HasPrefix(k, v1.AppArmorBetaContainerAnnotationKeyPrefix) {
			return true
		}
	}
	return false
}

// schedulingGatesInUse returns true if the pod spec is non-nil and it has SchedulingGates field set.
func schedulingGatesInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	return len(podSpec.SchedulingGates) != 0
}

// restartableInitContainersInUse returns true if the pod spec is non-nil and
// it has any init container with ContainerRestartPolicyAlways.
func restartableInitContainersInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	var inUse bool
	VisitContainers(podSpec, InitContainers, func(c *api.Container, containerType ContainerType) bool {
		if c.RestartPolicy != nil && *c.RestartPolicy == api.ContainerRestartPolicyAlways {
			inUse = true
			return false
		}
		return true
	})
	return inUse
}

func hasInvalidLabelValueInAffinitySelector(spec *api.PodSpec) bool {
	if spec.Affinity != nil {
		if spec.Affinity.PodAffinity != nil {
			for _, term := range spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				allErrs := apivalidation.ValidatePodAffinityTermSelector(term, false, nil)
				if len(allErrs) != 0 {
					return true
				}
			}
			for _, term := range spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				allErrs := apivalidation.ValidatePodAffinityTermSelector(term.PodAffinityTerm, false, nil)
				if len(allErrs) != 0 {
					return true
				}
			}
		}
		if spec.Affinity.PodAntiAffinity != nil {
			for _, term := range spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				allErrs := apivalidation.ValidatePodAffinityTermSelector(term, false, nil)
				if len(allErrs) != 0 {
					return true
				}
			}
			for _, term := range spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				allErrs := apivalidation.ValidatePodAffinityTermSelector(term.PodAffinityTerm, false, nil)
				if len(allErrs) != 0 {
					return true
				}
			}
		}
	}
	return false
}

func MarkPodProposedForResize(oldPod, newPod *api.Pod) {
	for i, c := range newPod.Spec.Containers {
		if c.Resources.Requests == nil {
			continue
		}
		if cmp.Equal(oldPod.Spec.Containers[i].Resources, c.Resources) {
			continue
		}
		findContainerStatus := func(css []api.ContainerStatus, cName string) (api.ContainerStatus, bool) {
			for i := range css {
				if css[i].Name == cName {
					return css[i], true
				}
			}
			return api.ContainerStatus{}, false
		}
		if cs, ok := findContainerStatus(newPod.Status.ContainerStatuses, c.Name); ok {
			if !cmp.Equal(c.Resources.Requests, cs.AllocatedResources) {
				newPod.Status.Resize = api.PodResizeStatusProposed
				break
			}
		}
	}
}
