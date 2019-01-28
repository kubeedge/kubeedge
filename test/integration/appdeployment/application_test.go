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

package application_test

import (
	"net/http"
	"time"

	"github.com/kubeedge/kubeedge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/test/integration/utils/edge"
	. "github.com/kubeedge/kubeedge/test/integration/utils/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//Run Test cases
var _ = Describe("Application deployment in edge_core Testing", func() {
	var UID string
	Context("Test application deployment and delete deployment", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[0])
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+"/pods", UID)
			time.Sleep(2 * time.Second)
			common.PrintTestcaseNameandStatus()
		})

		It("TC_TEST_APP_DEPLOYMENT_1: Test application deployment in edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[0])
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+"/pods", UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_2: Test List application deployment in edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[0])
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+"/pods", UID)
			pods, err := GetPods(ctx.Cfg.EdgedEndpoint + "/pods")
			Expect(err).To(BeNil())
			common.Info("Get pods from Edged is Successfull !!")
			for index := range pods.Items {
				pod := &pods.Items[index]
				common.InfoV2("PodName: %s PodStatus: %s", pod.Name, pod.Status.Phase)
			}
		})

		It("TC_TEST_APP_DEPLOYMENT_3: Test application deployment delete from edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[1])
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+"/pods", UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[1])
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+"/pods", UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_4: Test application deployment delete from edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			for i := 0; i < 2; i++ {
				UID = "deployment-app-" + edge.GetRandomString(10)
				IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[i])
				Expect(IsAppDeployed).Should(BeTrue())
				CheckPodRunningState(ctx.Cfg.EdgedEndpoint+"/pods", UID)
				time.Sleep(5 * time.Second)
			}
		})

		It("TC_TEST_APP_DEPLOYMENT_5: Test application deployment delete from edge_core", func() {
			var apps []string
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			for i := 0; i < 2; i++ {
				UID = "deployment-app-" + edge.GetRandomString(10)
				IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+"/apps", UID, ctx.Cfg.AppImageUrl[i])
				Expect(IsAppDeployed).Should(BeTrue())
				CheckPodRunningState(ctx.Cfg.EdgedEndpoint+"/pods", UID)
				apps = append(apps, UID)
				time.Sleep(5 * time.Second)
			}
			for i, appname := range apps {
				IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+"/apps", appname, ctx.Cfg.AppImageUrl[i])
				Expect(IsAppDeleted).Should(BeTrue())
				CheckPodDeletion(ctx.Cfg.EdgedEndpoint+"/pods", appname)
			}
		})

	})
})
