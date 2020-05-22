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

package testsuite

import (
	"net/http"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func CreateDeploymentTest(replica int, deplName, nodeName, nodeSelector string, ctx *utils.TestContext) metav1.PodList {
	var deploymentList v1.DeploymentList
	var podlist metav1.PodList
	IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler, deplName, ctx.Cfg.AppImageURL[1], nodeSelector, "", replica)
	gomega.Expect(IsAppDeployed).Should(gomega.BeTrue())
	err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler)
	gomega.Expect(err).To(gomega.BeNil())

	time.Sleep(time.Second * 1)

	for _, deployment := range deploymentList.Items {
		if deployment.Name == deplName {
			label := nodeName
			podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
			gomega.Expect(err).To(gomega.BeNil())
			break
		}
	}
	utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, podlist, 240*time.Second)

	return podlist
}

func CreatePodTest(nodeName, podName string, ctx *utils.TestContext, pod *metav1.Pod) metav1.PodList {
	var podlist metav1.PodList
	IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podName, pod)
	gomega.Expect(IsAppDeployed).Should(gomega.BeTrue())
	label := nodeName

	time.Sleep(time.Second * 1)

	podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
	gomega.Expect(err).To(gomega.BeNil())
	utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, podlist, 240*time.Second)
	return podlist
}
