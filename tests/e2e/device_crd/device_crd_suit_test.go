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

package device_crd

import (
	"bytes"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx          *utils.TestContext
	nodeSelector string
	NodeName     string
)
var (
	deviceCRDPath      = "../../../build/crds/devices/devices_v1alpha1_device.yaml"
	deviceModelCRDPath = "../../../build/crds/devices/devices_v1alpha1_devicemodel.yaml"
	crdHandler         = "/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions"
	deviceCRD          = "devices.devices.kubeedge.io"
	deviceModelCRD     = "devicemodels.devices.kubeedge.io"
)

//Function to run the Ginkgo Test
func TestEdgecoreAppDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		client := &http.Client{}
		utils.InfoV6("Before Suite Execution")
		ctx = utils.NewTestContext(utils.LoadConfig())
		NodeName = "integration-node-" + utils.GetRandomString(10)
		nodeSelector = "node-" + utils.GetRandomString(3)

		//Generate Cerificates for Edge and Cloud nodes copy to respective folders
		Expect(utils.GenerateCerts()).Should(BeNil())
		//Do the neccessary config changes in Cloud and Edge nodes
		Expect(utils.DeploySetup(ctx, NodeName, "deployment")).Should(BeNil())
		//Run ./edgecontroller binary
		Expect(utils.StartEdgeController()).Should(BeNil())
		//Register the Edge Node to Master
		Expect(utils.RegisterNodeToMaster(NodeName, ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, nodeSelector)).Should(BeNil())
		//Run ./edge_core after node registration
		Expect(utils.StartEdgeCore()).Should(BeNil())
		//Check node successfully registered or not
		Eventually(func() string {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, NodeName)
			utils.Info("Node Name: %v, Node Status: %v", NodeName, status)
			return status
		}, "60s", "4s").Should(Equal("Running"), "Node register to the k8s master is unsuccessfull !!")
		//Apply the CRDs
		filePath := path.Join(getpwd(), deviceModelCRDPath)
		deviceModelBody, err := ioutil.ReadFile(filePath)
		Expect(err).Should(BeNil())
		BodyBuf := bytes.NewReader(deviceModelBody)
		req, err := http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+crdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		filePath = path.Join(getpwd(), deviceCRDPath)
		deviceBody, err := ioutil.ReadFile(filePath)
		Expect(err).Should(BeNil())
		BodyBuf = bytes.NewReader(deviceBody)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+crdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		err = utils.MqttConnect()
		Expect(err).To(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
		//Deregister the edge node from master
		Expect(utils.DeRegisterNodeFromMaster(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, NodeName)).Should(BeNil())
		Eventually(func() int {
			statuscode := utils.CheckNodeDeleteStatus(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, NodeName)
			utils.Info("Node Name: %v, Node Statuscode: %v", NodeName, statuscode)
			return statuscode
		}, "60s", "4s").Should(Equal(http.StatusNotFound), "Node register to the k8s master is unsuccessfull !!")
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceModelCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		req, err = http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		//Run the Cleanup steps to kill edge_core and edgecontroller binaries
		Expect(utils.CleanUp("device_crd")).Should(BeNil())
		utils.Info("Cleanup is Successfull !!")
	})
	RunSpecs(t, "kubeedge Device Managemnet Suite")
}

func getpwd() string {
	_, file, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(file))
	Expect(err).Should(BeNil())
	return dir
}
