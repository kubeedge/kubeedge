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
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	. "github.com/kubeedge/kubeedge/tests/e2e/testsuite"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DeploymentTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Application deployment test in keadm E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	Context("Test application deployment using Pod spec", func() {
		BeforeEach(func() {
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var podlist corev1.PodList
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
			Expect(err).To(BeNil())
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_POD_DEPLOYMENT: Create a pod and check the pod is coming up correctly", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeName)

			CreatePodTest(nodeName, podName, ctx, pod)
		})
	})
})

func NewPodObj(podName, imgURL, nodeName string) *corev1.Pod {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{"app": "nginx"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: imgURL,
				},
			},
			NodeName: nodeName,
		},
	}
	return &pod
}
