/*
Copyright 2023 The KubeEdge Authors.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/util"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

const ImagePrePull = "prepull"

func validateNode(node *v1.Node) bool {
	if !util.IsEdgeNode(node) {
		klog.Warningf("Node(%s) is not edge node", node.Name)
		return false
	}

	// if node is in NotReady state, cannot prepull images
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status != v1.ConditionTrue {
			klog.Warningf("Node(%s) is in NotReady state", node.Name)
			return false
		}
	}

	return true
}

// buildPrePullResource build prepull resource in msg send to edge node
func buildPrePullResource(imagePrePullName, nodeName string) string {
	resource := fmt.Sprintf("%s/%s/%s/%s", "node", nodeName, ImagePrePull, imagePrePullName)
	return resource
}

func parsePrePullresource(resource string) (string, string, error) {
	var nodeName, jobName string
	sli := strings.Split(resource, constants.ResourceSep)
	if len(sli) != 4 {
		return nodeName, jobName, fmt.Errorf("the resource %s is not the standard type", resource)
	}
	return sli[1], sli[3], nil
}

func patchImagePrePullStatus(crdClient crdClientset.Interface, imagePrePull *v1alpha1.ImagePrePullJob, status *v1alpha1.ImagePrePullStatus) error {
	oldValue := imagePrePull.DeepCopy()
	newValue := updateNodeImagePrePullStatus(oldValue, status)

	var completeFlag int
	var failedFlag bool
	newValue.Status.State = v1alpha1.PrePulling
	for _, statusValue := range newValue.Status.Status {
		if statusValue.State == v1alpha1.PrePullFailed {
			failedFlag = true
			completeFlag++
		}
		if statusValue.State == v1alpha1.PrePullSuccessful {
			completeFlag++
		}
	}
	if completeFlag == len(newValue.Status.Status) {
		if failedFlag {
			newValue.Status.State = v1alpha1.PrePullFailed
		} else {
			newValue.Status.State = v1alpha1.PrePullSuccessful
		}
	}

	oldData, err := json.Marshal(oldValue)
	if err != nil {
		return fmt.Errorf("failed to marshal the old ImagePrePullJob(%s): %v", oldValue.Name, err)
	}

	newData, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("failed to marshal the new ImagePrePullJob(%s): %v", newValue.Name, err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create a merge patch: %v", err)
	}

	_, err = crdClient.OperationsV1alpha1().ImagePrePullJobs().Patch(context.TODO(), newValue.Name, apimachineryType.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch update ImagePrePullJob status: %v", err)
	}

	return nil
}

func updateNodeImagePrePullStatus(imagePrePull *v1alpha1.ImagePrePullJob, status *v1alpha1.ImagePrePullStatus) *v1alpha1.ImagePrePullJob {
	// return value imageprepull cannot populate the input parameter old
	newValue := imagePrePull.DeepCopy()

	for index, nodeStatus := range newValue.Status.Status {
		if nodeStatus.NodeName == status.NodeName {
			newValue.Status.Status[index] = *status
			return newValue
		}
	}

	newValue.Status.Status = append(newValue.Status.Status, *status)
	return newValue
}
