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
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx          *utils.TestContext
	cfg          utils.Config
	nodeSelector string
	NodeName     string
)
var (
	runController      = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/cloud; sudo nohup ./edgecontroller > edgecontroller.log 2>&1 &"
	runEdgecore        = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/edge/; sudo nohup ./edge_core > edge_core.log 2>&1 &"
	deviceCRDPath      = "../../../build/crds/devices/devices_v1alpha1_device.yaml"
	deviceModelCRDPath = "../../../build/crds/devices/devices_v1alpha1_devicemodel.yaml"
	NodeHandler        = "/api/v1/nodes"
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
		cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(cfg)
		NodeName = "integration-node-" + utils.GetRandomString(10)
		nodeSelector = "node-" + utils.GetRandomString(3)
		// Delete device model & device CRDs, if already existing
		req, err := http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceModelCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound).Should(Equal(true))
		req, err = http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound).Should(Equal(true))
		//Generate Certificates for Edge and Cloud nodes copy to respective folders
		cmd := exec.Command("bash", "-x", "scripts/generate_cert.sh")
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		//Do the neccessary config changes in Cloud and Edge nodes
		cmd = exec.Command("bash", "-x", "scripts/setup.sh", NodeName, ctx.Cfg.K8SMasterForKubeEdge)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		time.Sleep(1 * time.Second)
		//Run ./edgecontroller binary
		cmd = exec.Command("sh", "-c", runController)
		err = utils.PrintCombinedOutput(cmd)
		time.Sleep(5 * time.Second)
		Expect(err).Should(BeNil())
		//Register the Edge Node to Master
		err = utils.RegisterNodeToMaster(NodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, nodeSelector)
		Expect(err).Should(BeNil())
		//Run ./edge_core after node registration
		cmd = exec.Command("sh", "-c", runEdgecore)
		err = utils.PrintCombinedOutput(cmd)
		time.Sleep(5 * time.Second)
		//Check node successfully registered or not
		Eventually(func() string {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NodeName)
			utils.Info("Node Name: %v, Node Status: %v", NodeName, status)
			return status
		}, "60s", "4s").Should(Equal("Running"), "Node register to the k8s master is unsuccessfull !!")
		//Apply the CRDs
		filePath := path.Join(getpwd(), deviceModelCRDPath)
		deviceModelBody, err := ioutil.ReadFile(filePath)
		Expect(err).Should(BeNil())
		BodyBuf := bytes.NewReader(deviceModelBody)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+crdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
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
		err := utils.DeRegisterNodeFromMaster(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NodeName)
		Expect(err).Should(BeNil())
		Eventually(func() int {
			statuscode := utils.CheckNodeDeleteStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NodeName)
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
		cmd := exec.Command("bash", "-x", "scripts/cleanup.sh")
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		time.Sleep(2 * time.Second)
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
