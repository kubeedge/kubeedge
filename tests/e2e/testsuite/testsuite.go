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

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"
	. "github.com/onsi/gomega"
)

func CreateDeploymentTest(replica int, deplName, nodeName, nodeSelector string, ctx *utils.TestContext) metav1.PodList {
	var deploymentList v1.DeploymentList
	var podlist metav1.PodList
	IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler, deplName, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)
	Expect(IsAppDeployed).Should(BeTrue())
	err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler)
	Expect(err).To(BeNil())
	for _, deployment := range deploymentList.Items {
		if deployment.Name == deplName {
			label := nodeName
			podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
			Expect(err).To(BeNil())
			break
		}
	}
	utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)

	return podlist
}

func CreatePodTest(nodeName, nodeSelector string, ctx *utils.TestContext)metav1.PodList{
	var podlist metav1.PodList
	//Generate the random string and assign as a UID
	UID := "pod-app-" + utils.GetRandomString(5)
	IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
	Expect(IsAppDeployed).Should(BeTrue())
	label := nodeName
	podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
	Expect(err).To(BeNil())
	utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
	return podlist
}
