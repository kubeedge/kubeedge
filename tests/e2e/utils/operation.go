/*
Copyright 2022 The KubeEdge Authors.

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

package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonconstants "github.com/kubeedge/kubeedge/common/constants"
	operationsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

func ListUpgradeNodeJob(c edgeclientset.Interface) ([]operationsv1alpha1.NodeUpgradeJob, error) {
	upgrade, err := c.OperationsV1alpha1().NodeUpgradeJobs().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return upgrade.Items, nil
}

func CheckNodeUpgradeJobExists(upgrades []operationsv1alpha1.NodeUpgradeJob,
	expectedUpgrade *operationsv1alpha1.NodeUpgradeJob,
	result operationsv1alpha1.UpgradeResult) error {
	exist := false
	for _, upgrade := range upgrades {
		if upgrade.Name == expectedUpgrade.Name {
			exist = true
			if !reflect.DeepEqual(upgrade.Spec, expectedUpgrade.Spec) {
				return errors.New("not match with expected")
			}

			// check whether status is what we expected
			if upgrade.Status.State != operationsv1alpha1.Completed {
				return fmt.Errorf("upgrade total state %v is not completed", upgrade.Status.State)
			}
			for _, s := range upgrade.Status.Status {
				if s.State != operationsv1alpha1.Completed || s.History.Result != result {
					return fmt.Errorf("unexpected state: %v, result: %v, expect result is %v", s.State, s.History.Result, result)
				}
			}
			break
		}
	}
	if !exist {
		return errors.New("the requested NodeUpgradeJob is not found")
	}

	return nil
}

// HandleNodeUpgradeJob to handle NodeUpgradeJob.
func HandleNodeUpgradeJob(c edgeclientset.Interface, operation string, job *operationsv1alpha1.NodeUpgradeJob, UID string) error {
	switch operation {
	case http.MethodPost:
		_, err := c.OperationsV1alpha1().NodeUpgradeJobs().Create(context.TODO(), job, metav1.CreateOptions{})
		return err
	case http.MethodDelete:
		err := c.OperationsV1alpha1().NodeUpgradeJobs().Delete(context.TODO(), UID, metav1.DeleteOptions{})
		return err
	}

	return nil
}

func NewNodeUpgradeJob() *operationsv1alpha1.NodeUpgradeJob {
	var timeout uint32 = 120
	upgrade := operationsv1alpha1.NodeUpgradeJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeUpgradeJob",
			APIVersion: "operations.kubeedge.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-node-upgrade-job",
		},
		Spec: operationsv1alpha1.NodeUpgradeJobSpec{
			Version:        "v1.12.0",
			UpgradeTool:    "dry-run",
			TimeoutSeconds: &timeout,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					commonconstants.EdgeNodeRoleKey: commonconstants.EdgeNodeRoleValue,
				},
			},
		},
	}
	return &upgrade
}

func NewNodeUpgradeJobWithWrongVersion() *operationsv1alpha1.NodeUpgradeJob {
	var timeout uint32 = 30
	upgrade := operationsv1alpha1.NodeUpgradeJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeUpgradeJob",
			APIVersion: "operations.kubeedge.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-node-upgrade-job",
		},
		Spec: operationsv1alpha1.NodeUpgradeJobSpec{
			// this version is not exist, so upgrade will be failed
			Version:        "v1.0.111",
			UpgradeTool:    "dry-run",
			TimeoutSeconds: &timeout,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					commonconstants.EdgeNodeRoleKey: commonconstants.EdgeNodeRoleValue,
				},
			},
		},
	}
	return &upgrade
}
