package overridemanager

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NameOverrider struct{}

func (o *NameOverrider) ApplyOverrides(rawObj *unstructured.Unstructured, overriders OverriderInfo) error {
	oldName := rawObj.GetName()
	if overriders.TargetNodeGroup != "" {
		newName := fmt.Sprintf("%s-%s", oldName, overriders.TargetNodeGroup)
		rawObj.SetName(newName)
	}
	if len(overriders.TargetNodeLabelSelector.MatchLabels) > 0 {
		suffix := CreateSuffixFromLabels(overriders.TargetNodeLabelSelector.MatchLabels)
		newName := fmt.Sprintf("%s-%s", oldName, suffix)
		rawObj.SetName(newName)
	}

	return nil
}
func CreateSuffixFromLabels(labels map[string]string) string {
	// Sort keys for deterministic hash
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create hash from labels
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(labels[k]))
	}

	return fmt.Sprintf("ls-%x", h.Sum(nil)[:4]) // ls prefix indicates LabelSelector
}
