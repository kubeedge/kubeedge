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

	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"
	. "github.com/kubeedge/kubeedge/edge/test/integration/utils/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AppHandler = "/pods"
)

//Run Test cases
var _ = Describe("Application deployment in edge_core Testing", func() {
	var UID string
	Context("Test application deployment and delete deployment", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			time.Sleep(2 * time.Second)
			common.PrintTestcaseNameandStatus()
		})

		It("TC_TEST_APP_DEPLOYMENT_1: Test application deployment in edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_2: Test List application deployment in edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			pods, err := GetPods(ctx.Cfg.EdgedEndpoint + AppHandler)
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
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[1], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[1], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_4: Test application deployment delete from edge_core", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			for i := 0; i < 2; i++ {
				UID = "deployment-app-" + edge.GetRandomString(10)
				IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[i], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
				Expect(IsAppDeployed).Should(BeTrue())
				CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
				time.Sleep(5 * time.Second)
			}
		})

		It("TC_TEST_APP_DEPLOYMENT_5: Test application deployment delete from edge_core", func() {
			var apps []string
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			for i := 0; i < 2; i++ {
				UID = "deployment-app-" + edge.GetRandomString(10)
				IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[i], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
				Expect(IsAppDeployed).Should(BeTrue())
				CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
				apps = append(apps, UID)
				time.Sleep(5 * time.Second)
			}
			for i, appname := range apps {
				IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, appname, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[i], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
				Expect(IsAppDeleted).Should(BeTrue())
				CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, appname)
			}
		})

		It("TC_TEST_APP_DEPLOYMENT_6: Test application deployment with restart policy : no restart", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyNever)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_7: Test application deployment with restrat policy : always", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyAlways)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_8: Test application deployment without liveness probe and service probe", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_9: Test application deployment with liveness probe ", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			httpact := v1.HTTPGetAction{Path: "/var/lib/edged", Scheme: "HTTP", Port: intstr.IntOrString{Type: intstr.Type(1), IntVal: 1884, StrVal: "1884"}}
			handler := v1.Handler{HTTPGet: &httpact}
			probe := v1.Probe{Handler: handler, TimeoutSeconds: 1, InitialDelaySeconds: 10, PeriodSeconds: 15}
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], LivenessProbe: &probe, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], LivenessProbe: &probe, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_10: Test application deployment with Service probe", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			httpact := v1.HTTPGetAction{Path: "/var/lib/edged", Scheme: "HTTP", Port: intstr.IntOrString{Type: intstr.Type(1), IntVal: 10255, StrVal: "10255"}}
			handler := v1.Handler{HTTPGet: &httpact}
			probe := v1.Probe{Handler: handler, TimeoutSeconds: 1, InitialDelaySeconds: 10, PeriodSeconds: 15}
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], ReadinessProbe: &probe, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], LivenessProbe: &probe, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_11: Test application deployment with resource memory limit", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			memory, err2 := resource.ParseQuantity("1024Mi")
			if err2 != nil {
				common.InfoV2("memory error")
			}
			limit := v1.ResourceList{v1.ResourceMemory: memory}
			r := v1.ResourceRequirements{Limits: limit}

			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_12: Test application deployment with resource cpu limit", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			cpu, err := resource.ParseQuantity("0.75")
			if err != nil {
				common.InfoV2("cpu resource parsing error")
			}
			limit := v1.ResourceList{v1.ResourceCPU: cpu}
			r := v1.ResourceRequirements{Limits: limit}

			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_13: Test application deployment with resource memory and cpu limit less than requested", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			cpu, err := resource.ParseQuantity("0.25")
			if err != nil {
				common.InfoV2("cpu error")
			}
			memory, err := resource.ParseQuantity("256M")
			if err != nil {
				common.InfoV2("memory error")
			}
			cpuReq, err := resource.ParseQuantity("0.50")
			memoReq, err := resource.ParseQuantity("512Mi")
			limit := v1.ResourceList{v1.ResourceCPU: cpu, v1.ResourceMemory: memory}
			request := v1.ResourceList{v1.ResourceCPU: cpuReq, v1.ResourceMemory: memoReq}
			r := v1.ResourceRequirements{Limits: limit, Requests: request}

			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_14: Test application deployment with requested and limit values of resource memory and cpu ", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			cpu, err := resource.ParseQuantity("0.75")
			if err != nil {
				common.InfoV2("cpu error")
			}
			memory, err2 := resource.ParseQuantity("1024Mi")
			if err2 != nil {
				common.InfoV2("memory error")
			}
			cpuReq, err := resource.ParseQuantity("0.25")
			memoReq, err := resource.ParseQuantity("512Mi")
			limit := v1.ResourceList{v1.ResourceCPU: cpu, v1.ResourceMemory: memory}
			request := v1.ResourceList{v1.ResourceCPU: cpuReq, v1.ResourceMemory: memoReq}
			r := v1.ResourceRequirements{Limits: limit, Requests: request}
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			time.Sleep(2 * time.Second)
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Resources: r, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_15: Test application deployment with container network configuration as host", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Ports: []v1.ContainerPort{}, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Ports: []v1.ContainerPort{}, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})

		It("TC_TEST_APP_DEPLOYMENT_16: Test application deployment with container network configuration as port mapping", func() {
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + edge.GetRandomString(10)
			port := []v1.ContainerPort{{HostPort: 10255, ContainerPort: 10255, Protocol: v1.ProtocolTCP, HostIP: "127.0.0.1"}}
			IsAppDeployed := HandleAddAndDeletePods(http.MethodPut, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Ports: port, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeployed).Should(BeTrue())
			CheckPodRunningState(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
			IsAppDeleted := HandleAddAndDeletePods(http.MethodDelete, ctx.Cfg.TestManager+AppHandler, UID, []v1.Container{{Name: UID, Image: ctx.Cfg.AppImageUrl[0], Ports: port, ImagePullPolicy: v1.PullIfNotPresent}}, v1.RestartPolicyOnFailure)
			Expect(IsAppDeleted).Should(BeTrue())
			CheckPodDeletion(ctx.Cfg.EdgedEndpoint+AppHandler, UID)
		})
	})
})
