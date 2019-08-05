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

package edgecore

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx          *utils.TestContext
	NodeName     string
	NodeSelector string
)
var (
	deviceCRDPath      = "../../../build/crds/devices/devices_v1alpha1_device.yaml"
	deviceModelCRDPath = "../../../build/crds/devices/devices_v1alpha1_devicemodel.yaml"
	deviceCRD          = "devices.devices.kubeedge.io"
	deviceModelCRD     = "devicemodels.devices.kubeedge.io"
)

var CloudCoreDeployment, CloudConfigMap string

//Function to run the Ginkgo Test
func TestEdgecoreDeviceDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		client := &http.Client{}
		var cloudCoreHostIP string
		utils.Infof("Before Suite Execution")
		ctx = utils.NewTestContext(utils.LoadConfig())
		//Apply the CRDs
		filePath := path.Join(getpwd(), deviceModelCRDPath)
		deviceModelBody, err := ioutil.ReadFile(filePath)
		Expect(err).Should(BeNil())
		BodyBuf := bytes.NewReader(deviceModelBody)
		req, err := http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.CrdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		filePath = path.Join(getpwd(), deviceCRDPath)
		deviceBody, err := ioutil.ReadFile(filePath)
		Expect(err).Should(BeNil())
		BodyBuf = bytes.NewReader(deviceBody)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+constants.CrdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))

		NodeName = "e2e-node-" + utils.GetRandomString(10)
		NodeSelector = "node-" + utils.GetRandomString(3)
		CloudConfigMap = "cloudcore-configmap-" + utils.GetRandomString(5)
		CloudCoreDeployment = "cloudcore-deployment-" + utils.GetRandomString(5)
		//Deploy cloudcore as a k8s resource to cluster
		err = utils.HandleCloudDeployment(CloudConfigMap, CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge,
			ctx.Cfg.K8SMasterForKubeEdge+constants.ConfigmapHandler, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler, ctx.Cfg.CloudImageUrl, 10)
		Expect(err).Should(BeNil())
		time.Sleep(1 * time.Second)
		//Get the cloudCore IP
		podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, "")
		Expect(err).To(BeNil())
		for _, pod := range podlist.Items {
			if strings.Contains(pod.Name, "cloudcore-deployment") {
				cloudCoreHostIP = pod.Status.HostIP
				break
			}
		}
		utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
		time.Sleep(3 * time.Second)

		//Create service for cloud
		err = utils.ExposeCloudService(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+constants.ServiceHandler)
		Expect(err).Should(BeNil())
		//Create a nodePort Service to access the cloud Service from the cluster nodes
		wsPort, _ := utils.GetServicePort(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+constants.ServiceHandler)
		wsNodePort := strconv.FormatInt(int64(wsPort), 10)
		wsscloudHubURL := "wss://" + cloudCoreHostIP + ":" + wsNodePort
		cloudHubURL := wsscloudHubURL

		//Deploy edgecore as a k8s resource to cluster
		utils.CreateConfigMapforEdgeCore(cloudHubURL, ctx.Cfg.K8SMasterForKubeEdge+constants.ConfigmapHandler, ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, NodeName, NodeSelector)
		utils.HandleEdgeCorePodDeployment(ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, NodeName)
		time.Sleep(1 * time.Second)
		err = utils.MqttConnect()
		Expect(err).To(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+constants.CrdHandler+"/"+deviceModelCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		req, err = http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+constants.CrdHandler+"/"+deviceCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))

		utils.DeleteCloudDeployment(ctx.Cfg.K8SMasterForKubeEdge, CloudCoreDeployment, CloudConfigMap)
		utils.DeleteEdgeDeployments(ctx.Cfg.K8SMasterForKubeEdge)
		//Run the Cleanup steps to kill edgecore and cloudcore binaries
		Expect(utils.CleanUp("edgecore")).Should(BeNil())
		utils.Infof("Cleanup is Successful !!")
	})
	RunSpecs(t, "kubeedge Device Management Suite")
}

func getpwd() string {
	_, file, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(file))
	Expect(err).Should(BeNil())
	return dir
}
