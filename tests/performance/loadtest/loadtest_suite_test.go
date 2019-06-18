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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	. "github.com/kubeedge/kubeedge/tests/performance/common"
	"github.com/kubeedge/viaduct/pkg/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/api/core/v1"
)

//context to load config and access across the package
var (
	ctx               *utils.TestContext
	cfg               utils.Config
	nodeSelector      string
	cloudHubURL       string
	wsscloudHubURL    string
	quiccloudHubURL   string
	cloudCoreNodeName string
)

func TestEdgecoreK8sDeployment(t *testing.T) {
	var cloudCoreHostIP string
	var podlist metav1.PodList
	//var toTaint bool
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.InfoV6("Kubeedge deployment Load test Begin !!")
		cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(cfg)
		//apply label to all cluster nodes, use the selector to deploy all edgenodes to cluster nodes
		err := ApplyLabel(ctx.Cfg.K8SMasterForProvisionEdgeNodes + NodeHandler)
		Expect(err).Should(BeNil())
		//Create configMap for CloudCore
		CloudConfigMap = "cloudcore-configmap-" + utils.GetRandomString(5)
		CloudCoreDeployment = "cloudcore-deployment-" + utils.GetRandomString(5)
		//protocol to be used for test between edge and cloud
		if ctx.Cfg.Protocol == api.ProtocolTypeQuic {
			IsQuicProtocol = true
		} else {
			IsQuicProtocol = false
		}
		//Deploye cloudcore as a k8s resource to cluster-1
		err = HandleCloudDeployment(CloudConfigMap, CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge,
			ctx.Cfg.K8SMasterForKubeEdge+ConfigmapHandler, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, ctx.Cfg.CloudImageUrl, ctx.Cfg.NumOfNodes)
		Expect(err).Should(BeNil())
		time.Sleep(1 * time.Second)
		//Get the cloudCore pod Node name and IP
		podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
		Expect(err).To(BeNil())
		for _, pod := range podlist.Items {
			if strings.Contains(pod.Name, "cloudcore-deployment") {
				cloudCoreHostIP = pod.Status.HostIP
				cloudCoreNodeName = pod.Spec.NodeName
			}
			break
		}
		utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
		time.Sleep(5 * time.Second)
		//Create service for cloud
		err = utils.ExposeCloudService(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
		Expect(err).Should(BeNil())
		//Create a nodePort Service to access the cloud Service from the cluster nodes
		wsPort, quicPort := utils.GetServicePort(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
		wsNodePort := strconv.FormatInt(int64(wsPort), 10)
		quicNodePort := strconv.FormatInt(int64(quicPort), 10)
		quiccloudHubURL = cloudCoreHostIP + ":" + quicNodePort
		cloudHubURL = quiccloudHubURL
		wsscloudHubURL = "wss://" + cloudCoreHostIP + ":" + wsNodePort
		cloudHubURL = wsscloudHubURL

		//Deploye edgecore as a k8s resource to cluster-2
		podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler,
			ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, ctx.Cfg.NumOfNodes)

		//skip the pod scheduling in k8s node while kubeedge nodes are available to schedule
		ToTaint = true
		err = utils.TaintEdgeDeployedNode(ToTaint, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler+"/"+cloudCoreNodeName)
		Expect(err).Should(BeNil())
		ToTaint = false
	})
	AfterSuite(func() {
		By("Kubeedge deployment Load test End !!....!")

		DeleteEdgeDeployments(ctx.Cfg.K8SMasterForKubeEdge, ctx.Cfg.K8SMasterForProvisionEdgeNodes, ctx.Cfg.NumOfNodes)
		utils.CheckDeploymentPodDeleteState(ctx.Cfg.K8SMasterForProvisionEdgeNodes, podlist)
		//untaint Node
		err := utils.TaintEdgeDeployedNode(ToTaint, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler+"/"+cloudCoreNodeName)
		Expect(err).Should(BeNil())
		DeleteCloudDeployment(ctx.Cfg.K8SMasterForKubeEdge)

	})

	RunSpecs(t, "kubeedge Performace Load test Suite")
}
