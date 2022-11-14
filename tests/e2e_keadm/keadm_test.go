/*
Copyright 2019 The KubeEdge Authors.

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

package keadm

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	. "github.com/kubeedge/kubeedge/tests/e2e/testsuite"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DeploymentTestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Application deployment test in keadm E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport GinkgoTestDescription

	var clientSet clientset.Interface

	BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
	})

	Context("Test application deployment using Pod spec", func() {
		BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			labelSelector := labels.SelectorFromSet(constants.KubeEdgeE2ELabel)
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())

			for _, pod := range podList.Items {
				err = utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				Expect(err).To(BeNil())
			}

			utils.CheckPodDeleteState(clientSet, podList)

			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_POD_DEPLOYMENT: Create a pod and check the pod is coming up correctly", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, ctx.Cfg.AppImageURL[0])

			CreatePodTest(clientSet, pod)
		})
	})
})
