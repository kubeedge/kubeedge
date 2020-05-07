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

package bluetooth

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

//context to load config and access across the package
var (
	ctx          *utils.TestContext
	cfg          utils.Config
	nodeSelector string
	nodeName     string
)

const (
	mockHandler           = "/apis/devices.kubeedge.io/v1alpha1/namespaces/default/devicemodels"
	mockInstanceHandler   = "/apis/devices.kubeedge.io/v1alpha1/namespaces/default/devices"
	crdHandler            = "/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions"
	appHandler            = "/api/v1/namespaces/default/pods"
	nodeHandler           = "/api/v1/nodes"
	deploymentHandler     = "/apis/apps/v1/namespaces/default/deployments"
	deviceCRD             = "devices.devices.kubeedge.io"
	deviceModelCRD        = "devicemodels.devices.kubeedge.io"
	deviceModelPath       = "../../../../build/crds/devices/devices_v1alpha1_devicemodel.yaml"
	deviceInstancePath    = "../../../../build/crds/devices/devices_v1alpha1_device.yaml"
	devMockInstancePath   = "./crds/deviceinstance.yaml"
	devMockModelPath      = "./crds/devicemodel.yaml"
	makeFilePath          = "../../../../mappers/bluetooth_mapper/"
	sourceConfigPath      = "./configuration/config.yaml"
	destinationConfigPath = "../../../../mappers/bluetooth_mapper/configuration/config.yaml"
	deployPath            = "../../../../mappers/bluetooth_mapper/deployment.yaml"
)

type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

// Testing the basic bluetooth mapper functionalities
func TestMapperCharacteristics(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.Infof("Before Suite Execution")
		cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(cfg)
		t := time.Now()
		nodeName = "edge-node-bluetooth"
		nodeSelector = "node-" + utils.GetRandomString(3)

		//Generate Cerificates for Edge and Cloud nodes copy to respective folders
		Expect(utils.GenerateCerts()).Should(BeNil())
		//Do the necessary config changes in Cloud and Edge nodes
		Expect(utils.DeploySetup(ctx, nodeName, "deployment")).Should(BeNil())

		//Apply CRD for devicemodel
		curPath := getpwd()
		file := path.Join(curPath, deviceModelPath)
		body, err := ioutil.ReadFile(file)
		Expect(err).Should(BeNil())
		client := &http.Client{}
		BodyBuf := bytes.NewReader(body)
		req, err := http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+crdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).To(BeNil())
		utils.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))

		//Apply CRD for deviceinstance
		curPath = getpwd()
		file = path.Join(curPath, deviceInstancePath)
		body, err = ioutil.ReadFile(file)
		Expect(err).Should(BeNil())
		client = &http.Client{}
		BodyBuf = bytes.NewReader(body)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+crdHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).To(BeNil())
		utils.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))

		//Run ./cloudcore binary
		Expect(utils.StartCloudCore()).Should(BeNil())

		//Register the Edge Node to Master
		err = utils.RegisterNodeToMaster(nodeName, ctx.Cfg.K8SMasterForKubeEdge+nodeHandler, nodeSelector)
		Expect(err).Should(BeNil())

		//Run ./edgecore after node registration
		Expect(utils.StartEdgeCore(ctx.Cfg.K8SMasterForKubeEdge, nodeName)).Should(BeNil())

		//Check node successfully registered or not
		Eventually(func() string {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+nodeHandler, nodeName)
			utils.Infof("Node Name: %v, Node Status: %v", nodeName, status)
			return status
		}, "60s", "4s").Should(Equal("Running"), "Node register to the k8s master is unsuccessful !!")

		// Adding label to node
		utils.ApplyLabelToNode(ctx.Cfg.K8SMasterForKubeEdge+nodeHandler+"/"+nodeName, "bluetooth", "true")

		// Changing the config yaml of bluetooth mapper
		t = time.Now()
		pwd := getpwd()
		sourcePath := path.Join(pwd, sourceConfigPath)
		destinationPath := path.Join(pwd, destinationConfigPath)
		cmd := exec.Command("cp", sourcePath, destinationPath)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())

		//Building bluetooth mapper
		curPath = getpwd()
		newPath := path.Join(curPath, makeFilePath)
		os.Chdir(newPath)
		cmd = exec.Command("make", "bluetooth_mapper_image")
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())

		//dockertag
		tagname := cfg.DockerHubUserName + "/bluetooth_mapper:v1.0"
		cmd = exec.Command("docker", "tag", "bluetooth_mapper:v1.0", tagname)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())

		//docker login
		cmd = exec.Command("docker", "login", "-u", cfg.DockerHubUserName, "-p", cfg.DockerHubPassword)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())

		//docker push
		cmd = exec.Command("docker", "push", tagname)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())

		//apply CRD for mock devicemodel
		curPath = getpwd()
		file = path.Join(curPath, devMockModelPath)
		body, err = ioutil.ReadFile(file)
		Expect(err).Should(BeNil())
		client = &http.Client{}
		BodyBuf = bytes.NewReader(body)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+mockHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		utils.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))

		//apply CRD for mock deviceinstance
		curPath = getpwd()
		file = path.Join(curPath, devMockInstancePath)
		body, err = ioutil.ReadFile(file)
		Expect(err).Should(BeNil())
		client = &http.Client{}
		BodyBuf = bytes.NewReader(body)
		req, err = http.NewRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+mockInstanceHandler, BodyBuf)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		utils.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))

		//updating deployment file with edgenode name and dockerhubusername
		curPath = getpwd()
		newPath = path.Join(curPath, "../../")
		os.Chdir(newPath)
		cmd = exec.Command("bash", "-x", "scripts/bluetoothconfig.sh", ctx.Cfg.DockerHubUserName, nodeName)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
	})

	AfterSuite(func() {
		By("After Suite Execution....!")
		//Delete Deployment
		var podlist metav1.PodList
		var deploymentList v1.DeploymentList
		var UID string = "bluetooth-device-mapper-deployment"
		err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+deploymentHandler)
		Expect(err).To(BeNil())
		for _, deployment := range deploymentList.Items {
			if deployment.Name == UID {
				label := nodeName
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+appHandler, label)
				Expect(err).To(BeNil())
				StatusCode := utils.DeleteDeployment(ctx.Cfg.K8SMasterForKubeEdge+deploymentHandler, deployment.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
		}
		utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+appHandler, podlist)

		// Delete mock device instances created
		var deviceList v1alpha1.DeviceList
		deviceInstanceList, err := utils.GetDevice(&deviceList, ctx.Cfg.K8SMasterForKubeEdge+mockInstanceHandler, nil)
		Expect(err).To(BeNil())
		for _, device := range deviceInstanceList {
			IsDeviceDeleted, statusCode := utils.HandleDeviceInstance(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+mockInstanceHandler, nodeName, "/"+device.Name, "")
			Expect(IsDeviceDeleted).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
		}

		// Delete mock device model created
		var deviceModelList v1alpha1.DeviceModelList
		list, err := utils.GetDeviceModel(&deviceModelList, ctx.Cfg.K8SMasterForKubeEdge+mockHandler, nil)
		Expect(err).To(BeNil())
		for _, model := range list {
			IsDeviceModelDeleted, statusCode := utils.HandleDeviceModel(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+mockHandler, "/"+model.Name, "")
			Expect(IsDeviceModelDeleted).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
		}

		//Deleting the created devicemodel and deviceinstance
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err := client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		req, err = http.NewRequest(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+crdHandler+"/"+deviceModelCRD, nil)
		Expect(err).Should(BeNil())
		req.Header.Set("Content-Type", "application/yaml")
		resp, err = client.Do(req)
		Expect(err).Should(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))

		//Deregister the edge node from master
		err = utils.DeRegisterNodeFromMaster(ctx.Cfg.K8SMasterForKubeEdge+nodeHandler, nodeName)
		Expect(err).Should(BeNil())
		Eventually(func() int {
			statuscode := utils.CheckNodeDeleteStatus(ctx.Cfg.K8SMasterForKubeEdge+nodeHandler, nodeName)
			utils.Infof("Node Name: %v, Node Statuscode: %v", nodeName, statuscode)
			return statuscode
		}, "60s", "4s").Should(Equal(http.StatusNotFound), "Node register to the k8s master is unsuccessful !!")

		Expect(utils.CleanUp("deployment")).Should(BeNil())
		time.Sleep(2 * time.Second)

		utils.Infof("Cleanup is Successful !!")
	})
	RunSpecs(t, "Kubeedge Mapper Test Suite")
}

// This function is used to obtain the present working directory
func getpwd() string {
	_, file, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(file))
	Expect(err).Should(BeNil())
	return dir
}
