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

package deployment

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"

	operationsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var _ = Describe("Operation NodeUpgradeJob Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport SpecReport
	var edgeClientSet edgeclientset.Interface

	BeforeEach(func() {
		edgeClientSet = utils.NewKubeEdgeClient(ctx.Cfg.KubeConfigPath)
	})

	Context("Test operations: NodeUpgradeJob Creation and Deletion", func() {
		BeforeEach(func() {
			// Delete any pre-existing NodeUpgradeJobs
			list, err := utils.ListUpgradeNodeJob(edgeClientSet)
			Expect(err).To(BeNil())
			for _, upgrade := range list {
				err := utils.HandleNodeUpgradeJob(edgeClientSet, http.MethodDelete, nil, upgrade.Name)
				Expect(err).Should(BeNil())
			}

			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_NodeUpgradeJob_1: Create NodeUpgradeJob: upgrade succeed", func() {
			// create NodeUpgradeJob
			job := utils.NewNodeUpgradeJob()
			err := utils.HandleNodeUpgradeJob(edgeClientSet, http.MethodPost, job, "")
			Expect(err).Should(BeNil())

			expect := utils.NewNodeUpgradeJob()

			timeout := *expect.Spec.TimeoutSeconds
			pollErr := wait.Poll(10*time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
				list, err := utils.ListUpgradeNodeJob(edgeClientSet)
				if err != nil {
					return false, nil
				}
				// check whether upgrade result is successful
				if utils.CheckNodeUpgradeJobExists(list, expect, operationsv1alpha1.UpgradeSuccess) != nil {
					return false, nil
				}

				return true, nil
			})

			Expect(pollErr).To(BeNil())

			// delete NodeUpgradeJob
			err = utils.HandleNodeUpgradeJob(edgeClientSet, http.MethodDelete, nil, expect.Name)
			Expect(err).Should(BeNil())
		})

		It("E2E_CREATE_NodeUpgradeJob_2: Create NodeUpgradeJob: upgrade fail", func() {
			// create NodeUpgradeJob
			job := utils.NewNodeUpgradeJobWithWrongVersion()
			err := utils.HandleNodeUpgradeJob(edgeClientSet, http.MethodPost, job, "")
			Expect(err).Should(BeNil())

			expect := utils.NewNodeUpgradeJobWithWrongVersion()

			timeout := *expect.Spec.TimeoutSeconds
			pollErr := wait.Poll(10*time.Second, time.Duration(timeout+20)*time.Second, func() (bool, error) {
				list, err := utils.ListUpgradeNodeJob(edgeClientSet)
				if err != nil {
					return false, nil
				}
				// check whether upgrade result is failed
				if utils.CheckNodeUpgradeJobExists(list, expect, operationsv1alpha1.UpgradeFailedRollbackSuccess) != nil {
					return false, nil
				}

				return true, nil
			})

			Expect(pollErr).To(BeNil())

			// delete NodeUpgradeJob
			err = utils.HandleNodeUpgradeJob(edgeClientSet, http.MethodDelete, nil, expect.Name)
			Expect(err).Should(BeNil())
		})
	})
})
