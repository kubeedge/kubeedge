package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/imdario/mergo"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceInfo struct {
	// Ordinal is the index of the template of this resource in
	// the manifetsts of EdgeApplication.
	Ordinal   int    `json:"ordinal"`
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (c *ResourceInfo) String() string {
	return fmt.Sprintf("%d, %s/%s, kind=%s, namespace=%s, name=%s", c.Ordinal, c.Group, c.Version, c.Kind, c.Namespace, c.Name)
}

type TemplateInfo struct {
	Ordinal  int
	Template *unstructured.Unstructured
}

// GetNodeGroupsByLabels can get all node groups matching these labels.
func GetNodesGroupsByLabels(ctx context.Context, c client.Client, matchLabels map[string]string) ([]appsv1alpha1.NodeGroup, error) {
	if matchLabels == nil {
		// Return empty when matchLabels is nil
		// Otherwise, it will select all node groups, it's not what we want
		return []appsv1alpha1.NodeGroup{}, nil
	}
	selector := labels.SelectorFromSet(labels.Set(matchLabels))
	nodeGroupList := &appsv1alpha1.NodeGroupList{}
	err := c.List(ctx, nodeGroupList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return nodeGroupList.Items, nil
}

func GetAllOverriders(ctx context.Context, edgeApp *appsv1alpha1.EdgeApplication, c client.Client) ([]overridemanager.OverriderInfo, error) {
	errs := []error{}
	sorted := map[string]overridemanager.OverriderInfo{}
	// get node group names by target node group labels
	for _, target := range edgeApp.Spec.WorkloadScope.TargetNodeGroupSelectors {
		groups, err := GetNodesGroupsByLabels(ctx, c, target.MatchLabels)
		if err != nil {
			klog.Errorf("failed to get node group by labels, %v", err)
			errs = append(errs, err)
			continue
		}
		for _, group := range groups {
			sorted[group.Name] = overridemanager.OverriderInfo{
				TargetNodeGroup: group.Name,
				Overriders:      &appsv1alpha1.Overriders{},
			}
		}
	}
	// get node group names by target node groups
	for index := range edgeApp.Spec.WorkloadScope.TargetNodeGroups {
		name := edgeApp.Spec.WorkloadScope.TargetNodeGroups[index].Name
		overriders := edgeApp.Spec.WorkloadScope.TargetNodeGroups[index].Overriders.DeepCopy()
		if nodegroup, exist := sorted[name]; exist {
			if err := mergo.Merge(&nodegroup.Overriders, overriders); err != nil {
				klog.Errorf("failed to merge overriders of nodegroup %s, %v", name, err)
				errs = append(errs, err)
			}
		} else {
			sorted[name] = overridemanager.OverriderInfo{
				TargetNodeGroup: edgeApp.Spec.WorkloadScope.TargetNodeGroups[index].Name,
				Overriders:      overriders,
			}
		}
	}
	// convert sorted to array
	infos := make([]overridemanager.OverriderInfo, 0, len(sorted))
	for _, overrider := range sorted {
		infos = append(infos, overrider)
	}
	return infos, errors.NewAggregate(errs)
}

func GetContainedResourceInfos(edgeApp *appsv1alpha1.EdgeApplication, yamlSerializer runtime.Serializer) ([]ResourceInfo, error) {
	tmplInfos, err := GetTemplatesInfosOfEdgeApp(edgeApp, yamlSerializer)
	if err != nil {
		return nil, fmt.Errorf("failed to get contained objs, %v", err)
	}
	infos := []ResourceInfo{}
	for _, tmplInfo := range tmplInfos {
		infos = append(infos, GetResourceInfoOfTemplateInfo(tmplInfo))
	}
	return infos, nil
}

func GetResourceInfoOfTemplateInfo(tmplInfo *TemplateInfo) ResourceInfo {
	tmpl := tmplInfo.Template
	gvk := tmpl.GroupVersionKind()
	info := ResourceInfo{
		Ordinal:   tmplInfo.Ordinal,
		Group:     gvk.Group,
		Version:   gvk.Version,
		Kind:      gvk.Kind,
		Namespace: tmpl.GetNamespace(),
		Name:      tmpl.GetName(),
	}
	return info
}

func GetTemplatesInfosOfEdgeApp(edgeApp *appsv1alpha1.EdgeApplication, yamlSerializer runtime.Serializer) ([]*TemplateInfo, error) {
	tmplInfos := []*TemplateInfo{}
	errs := []error{}
	for index, manifest := range edgeApp.Spec.WorkloadTemplate.Manifests {
		obj := &unstructured.Unstructured{}
		_, _, err := yamlSerializer.Decode(manifest.Raw, nil, obj)
		if err != nil {
			klog.Errorf("failed to decode manifest of edgeapp %s/%s, %v, manifest: %s",
				edgeApp.Namespace, edgeApp.Name, err, manifest)
			errs = append(errs, err)
			continue
		}
		tmplInfos = append(tmplInfos, &TemplateInfo{Ordinal: index, Template: obj})
	}
	return tmplInfos, errors.NewAggregate(errs)
}

func IsInitStatus(status *appsv1alpha1.ManifestStatus) bool {
	identifier := status.Identifier
	return identifier.Group == "" &&
		identifier.Version == "" &&
		identifier.Kind == "" &&
		identifier.Resource == "" &&
		identifier.Namespace == "" &&
		identifier.Name == ""
}

func IsIdentifierSameAsResourceInfo(identifier appsv1alpha1.ResourceIdentifier, info ResourceInfo) bool {
	return identifier.Group == info.Group &&
		identifier.Version == info.Version &&
		identifier.Kind == info.Kind &&
		identifier.Namespace == info.Namespace &&
		identifier.Name == info.Name
}
