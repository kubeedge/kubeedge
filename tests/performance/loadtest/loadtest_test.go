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
package loadtest

import (
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	. "github.com/kubeedge/kubeedge/tests/performance/common"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"
)

var DeploymentTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

//RestartEdgeNodePodsToUseQuicProtocol function to switch the protocol and re-initialize the edgecore
func RestartEdgeNodePodsToUseQuicProtocol() error {
	var EdgeNodePods []string
	chconfigmapRet := make(chan error)
	for nodeName, conf := range NodeInfo {
		for range conf {
			cmd := exec.Command("bash", "-x", "scripts/update_configmap.sh", "create_edge_config", nodeName, quiccloudHubURL, conf[0])
			err := utils.PrintCombinedOutput(cmd)
			Expect(err).Should(BeNil())
			//Create ConfigMaps for Each EdgeNode created
			go utils.HandleConfigmap(chconfigmapRet, http.MethodPatch, ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler+"/"+conf[0], true)
			ret := <-chconfigmapRet
			Expect(ret).To(BeNil())
		}
	}

	pods, err := utils.GetPods(ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, "")
	Expect(err).To(BeNil())
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "edgecore-deployment") {
			//EdgeNodePodHost = pod.Spec.NodeName
			EdgeNodePods = append(EdgeNodePods, pod.Name)
		}
	}

	for i, _ := range EdgeNodePods {
		utils.DeletePods(ctx.Cfg.K8SMasterForProvisionEdgeNodes + AppHandler + "/" + EdgeNodePods[i])
	}

	Eventually(func() int {
		var count int
		for i, _ := range EdgeNodePods {
			status, statusCode := utils.GetPodState(ctx.Cfg.K8SMasterForProvisionEdgeNodes + AppHandler + "/" + EdgeNodePods[i])
			utils.InfoV2("PodName: %s status: %s StatusCode: %d", EdgeNodePods[i], status, statusCode)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "1200s", "4s").Should(Equal(len(EdgeNodePods)), "Delete Application deployment is Unsuccessfull, Pods are not deleted within the time")

	newpods, err := utils.GetPods(ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, "")
	Expect(err).To(BeNil())

	Eventually(func() int {
		var count int
		for _, pod := range newpods.Items {
			state, _ := utils.GetPodState(ctx.Cfg.K8SMasterForProvisionEdgeNodes + AppHandler + "/" + pod.Name)
			utils.InfoV2("PodName: %s PodStatus: %s", pod.Name, state)
			if state == "Running" {
				count++
			}
		}
		return count
	}, "1200s", "2s").Should(Equal(ctx.Cfg.NumOfNodes), "New Pods has not come to Running State")

	//Check All EdgeNode are in Running state
	Eventually(func() int {
		count := 0
		for edgenodeName, _ := range NodeInfo {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, edgenodeName)
			utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
			if status == "Running" {
				count++
			}
		}
		return count
	}, "60s", "2s").Should(Equal(ctx.Cfg.NumOfNodes), "Nodes register to the k8s master is unsuccessfull !!")

	return nil
}

func PullImageInAllEdgeNodes(appDeployments []string){
	var deploymentList v1.DeploymentList
	var podlist metav1.PodList
	for kubenode, val := range NodeInfo {
		UID := "edgecore-app-" + utils.GetRandomString(5)
		IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], val[1], val[0], 1)
		Expect(IsAppDeployed).Should(BeTrue())
		appDeployments = append(appDeployments, UID)
		err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
		Expect(err).To(BeNil())
		for _, deployment := range deploymentList.Items {
			if deployment.Name == UID {
				label := kubenode
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
				Expect(err).To(BeNil())
				break
			}
		}
		utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
	}
	//after pulling image to all edgenodes, delete the deployments on respective edgenodes
	for i, _ := range appDeployments {
		IsAppDeployed := utils.HandleDeployment(false, false, http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+"/"+appDeployments[i], "", ctx.Cfg.AppImageUrl[1], nodeSelector, "", 10)
		Expect(IsAppDeployed).Should(BeTrue())
	}
	podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
	Expect(err).To(BeNil())
	utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
	appDeployments = nil
}

//Run Test cases
var _ = Describe("Application deployment test in Perfronace test EdgeNodes", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	var podlist metav1.PodList
	var appDeployments []string

	Context("Pull images to all KubeEdge nodes", func() {
		FIt("PULL_IMAGE_ALL_KUBEEDGE_NODES: Pull image to all KubeEdge edge nodes", func() {
			PullImageInAllEdgeNodes(appDeployments)
		})
	})

	Context("Test application deployment on Kubeedge EdgeNodes Through Websocket", func() {
		BeforeEach(func() {
			testDescription = CurrentGinkgoTestDescription()
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+"/"+UID, "", ctx.Cfg.AppImageUrl[1], nodeSelector, "", 10)
			Expect(IsAppDeployed).Should(BeTrue())

			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			podlist = metav1.PodList{}
		})

		Measure("WSS_MEASURE_PERF_LOADTEST_NODES_10: Create 10 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 10
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)

		Measure("WSS_MEASURE_PERF_LOADTEST_NODES_20: Create 20 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 20
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)

		Measure("WSS_MEASURE_PERF_LOADTEST_NODES_50: Create 50 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 50
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)
		Measure("WSS_MEASURE_PERF_NODETEST_NODES_75: Create 75 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 75
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)
		Measure("WSS_MEASURE_PERF_NODETEST_NODES_100: Create 100 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 100
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
	})

	Context("stress test on single Kubeedge EdgeNode", func() {
		var appDeployments []string
		BeforeEach(func() {
			testDescription = CurrentGinkgoTestDescription()
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			for i, _ := range appDeployments {
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+"/"+appDeployments[i], "", ctx.Cfg.AppImageUrl[1], nodeSelector, "", 10)
				Expect(IsAppDeployed).Should(BeTrue())
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			podlist = metav1.PodList{}
			appDeployments = nil
		})

		It("QUIC_APP_DEPLOYMENT_1: Switch to Quic and pull the image in all KubeEdge nodes", func() {
			err := RestartEdgeNodePodsToUseQuicProtocol()
			Expect(err).To(BeNil())
			PullImageInAllEdgeNodes(appDeployments)
		})

		FMeasure("MEASURE_PERF_NODETEST_SINGLE_NODE_1: Create 100 application Deployments, Measure Pod Running time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			var err error
			var nodeSelector string
			var nodeName string
			for key, val := range NodeInfo {
				nodeSelector = val[1]
				nodeName = key
				break
			}
			b.Time("MEASURE_PERF_NODETEST_SINGLE_NODE_1", func() {
				replica := 1
				for i := 0; i < 100; i++ {
					//Generate the random string and assign as a UID
					UID = "edgecore-app-" + utils.GetRandomString(5)
					appDeployments = append(appDeployments, UID)
					go utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler,
						UID, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)
				}
				time.Sleep(10 * time.Second)
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, nodeName)
				Expect(err).To(BeNil())
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})
		}, 5)

		Measure("MEASURE_PERF_NODETEST_SINGLE_NODE_1: Create 100 application Deployments while each deployment have 100 replica, Measure Pod Running time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			var err error
			var nodeSelector string
			//var nodeName string
			b.Time("MEASURE_PERF_NODETEST_SINGLE_NODE_1", func() {
				replica := 10
				for _, val := range NodeInfo {
					nodeSelector = val[1]
					//Generate the random string and assign as a UID
					UID = "edgecore-app-" + utils.GetRandomString(5)
					appDeployments = append(appDeployments, UID)
					go utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler,
						UID, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)

				}
				time.Sleep(10 * time.Second)
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
				Expect(err).To(BeNil())
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})
		}, 1)
	})

	Context("Test application deployment on Kubeedge EdgeNodes with Quic Protocol", func() {
		var appDeployments []string
		BeforeEach(func() {
			testDescription = CurrentGinkgoTestDescription()
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+"/"+UID, "", ctx.Cfg.AppImageUrl[1], nodeSelector, "", 10)
			Expect(IsAppDeployed).Should(BeTrue())

			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			podlist = metav1.PodList{}
		})

		It("QUIC_APP_DEPLOYMENT_1: Switch to Quic and pull the image in all KubeEdge nodes", func() {
			err := RestartEdgeNodePodsToUseQuicProtocol()
			Expect(err).To(BeNil())
			PullImageInAllEdgeNodes(appDeployments)
		})

		Measure("QUIC_MEASURE_PERF_NODETEST_NODES_1: Create 1 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {

			runtime := b.Time("QUIC_MEASURE_PERF_NODETEST_NODES_1", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 1
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)

		Measure("QUIC_MEASURE_PERF_NODETEST_NODES_10: Create 10 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			runtime := b.Time("QUIC_MEASURE_PERF_NODETEST_NODES_10", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 10
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)

		Measure("QUIC_MEASURE_PERF_NODETEST_NODES_50: Create 50 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			runtime := b.Time("QUIC_MEASURE_PERF_NODETEST_NODES_50", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 50
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})

			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)
		Measure("QUIC_MEASURE_PERF_NODETEST_NODES_75: Create 75 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			runtime := b.Time("QUIC_MEASURE_PERF_NODETEST_NODES_75", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 75
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})
			glog.Infof("Runtime stats: %+v", runtime)

		}, 5)
		Measure("QUIC_MEASURE_PERF_NODETEST_NODES_100: Create 100 KubeEdge Node Deployment, Measure time for application comes into Running state", func(b Benchmarker) {
			runtime := b.Time("QUIC_MEASURE_PERF_NODETEST_NODES_100", func() {
				var deploymentList v1.DeploymentList
				podlist = metav1.PodList{}
				replica := 100
				//Generate the random string and assign as a UID
				UID = "edgecore-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], "", "", replica)
				Expect(IsAppDeployed).Should(BeTrue())
				err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
				Expect(err).To(BeNil())
				for _, deployment := range deploymentList.Items {
					if deployment.Name == UID {
						//label := nodeName
						time.Sleep(2 * time.Second)
						podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
						Expect(err).To(BeNil())
						break
					}
				}
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
	})
})
