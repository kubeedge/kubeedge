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

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver"
	cloudcore "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgecore "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	edgesite "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgesite/v1alpha1"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
)

//GenerateCerts - Generates Cerificates for Edge and Cloud nodes copy to respective folders
func GenerateCerts() error {
	cmd := exec.Command("bash", "-x", "scripts/generate_cert.sh")
	return PrintCombinedOutput(cmd)
}

func StartCloudCore() error {
	catConfigCmd := exec.Command("sh", "-c", constants.CatCloudCoreConfigFile)
	Infof("===========> Executing: %s\n", strings.Join(catConfigCmd.Args, " "))
	bytes, _ := catConfigCmd.CombinedOutput()
	Infof("cloudcore, config:\n %v", string(bytes))

	//Run ./cloudcore binary
	cmd := exec.Command("sh", "-c", constants.RunCloudcore)
	if err := PrintCombinedOutput(cmd); err != nil {
		Errorf("start cloudcore error %v", err)
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckCloudcore)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatCloudcoreLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		lbytes, _ := catcmd.CombinedOutput()
		Errorf("cloudcore, log:\n %v", string(lbytes))
		Errorf("cloudcore start error %v", err)
		os.Exit(1)
	}
	return nil
}

func StartEdgeCore(master, nodeName string) error {
	token := getSecret(master)
	createEdgeCoreConfigFile(token, nodeName)

	catConfig := exec.Command("sh", "-c", constants.CatEdgeCoreConfigFile)
	Infof("===========> Executing: %s\n", strings.Join(catConfig.Args, " "))
	cbytes, _ := catConfig.CombinedOutput()
	Infof("edgecore config :\n %v", string(cbytes))

	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", constants.RunEdgecore)
	if err := PrintCombinedOutput(cmd); err != nil {
		Errorf("start edgecore error %v", err)
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckEdgecore)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatEdgecoreLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		Errorf("edgecore log:\n %v", string(bytes))
		Errorf("edgecore start error %v", err)
		os.Exit(1)
	}
	return nil
}

func getSecret(master string) string {
	secret := v1.Secret{}

	resp, err := SendHTTPRequest(http.MethodGet, master+"/api/v1/namespaces/kubeedge/secrets/tokensecret")
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Fatalf("HTTP Response reading has failed: %v", err)
		return ""
	}
	err = json.Unmarshal(contents, &secret)
	if err != nil {
		Fatalf("Unmarshal HTTP Response has failed: %v", err)
	}

	return string(secret.Data[httpserver.TokenDataName])
}

func StartEdgeSite() error {
	catConfig := exec.Command("sh", "-c", constants.CatEdgeSiteConfigFile)
	Infof("===========> Executing: %s\n", strings.Join(catConfig.Args, " "))
	cbytes, _ := catConfig.CombinedOutput()
	Infof("edgesite config:\n%v", string(cbytes))

	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", constants.RunEdgeSite)
	if err := PrintCombinedOutput(cmd); err != nil {
		Infof("start edgesite error %v", err)
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckEdgesite)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatEdgeSiteLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		Errorf("edgesite log:\n %v", string(bytes))
		Errorf("edgesite start error %v", err)
		os.Exit(1)
	}
	return nil
}

func DeploySetup(ctx *TestContext, nodeName, setupType string) error {
	// TODO change as constants or delete this function @kadisi
	switch setupType {
	case "deployment":
		createCloudCoreConfigFile(ctx.Cfg.KubeConfigPath)
	case "edgesite":
		createEdgeSiteConfigFile(ctx.Cfg.K8SMasterForKubeEdge, nodeName)
	}
	//Expect(err).Should(BeNil())
	time.Sleep(1 * time.Second)
	return nil
}

func CleanUp(setupType string) error {
	fmt.Println("**********************************", setupType)
	cmd := exec.Command("bash", "-x", "scripts/cleanup.sh", setupType)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func createCloudCoreConfigFile(kubeConfigPath string) {
	c := cloudcore.NewDefaultCloudCoreConfig()
	c.KubeAPIConfig.KubeConfig = kubeConfigPath
	c.KubeAPIConfig.Master = ""
	// TODO change ca file path @kadisi
	c.Modules.CloudHub.TLSCAFile = "/tmp/cloudcore/rootCA.crt"
	c.Modules.CloudHub.TLSCertFile = "/tmp/cloudcore/kubeedge.crt"
	c.Modules.CloudHub.TLSPrivateKeyFile = "/tmp/cloudcore/kubeedge.key"
	c.Modules.CloudStream.Enable = false

	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Marshal cloudcore config to yaml error %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(constants.CloudCoreConfigFile, data, os.ModePerm); err != nil {
		fmt.Printf("Create cloudcore config file %v error %v\n", constants.CloudCoreConfigFile, err)
		os.Exit(1)
	}
}

func createEdgeCoreConfigFile(token, nodeName string) {
	c := edgecore.NewDefaultEdgeCoreConfig()
	// TODO change ca file path @kadisi
	c.Modules.EdgeHub.TLSCAFile = "/tmp/edgecore/rootCA.crt"
	c.Modules.EdgeHub.TLSCertFile = "/tmp/edgecore/kubeedge.crt"
	c.Modules.EdgeHub.TLSPrivateKeyFile = "/tmp/edgecore/kubeedge.key"
	c.Modules.Edged.HostnameOverride = nodeName
	c.DataBase.DataSource = "/tmp/edgecore/edgecore.db"
	c.Modules.EventBus.MqttMode = edgecore.MqttModeInternal
	c.Modules.EdgeStream.Enable = false
	c.Modules.EdgeHub.Token = token

	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Marshal edgecore config to yaml error %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(constants.EdgeCoreConfigFile, data, os.ModePerm); err != nil {
		fmt.Printf("Create edgecore config file %v error %v\n", constants.EdgeCoreConfigFile, err)
		os.Exit(1)
	}
}

func createEdgeSiteConfigFile(kubeMaster, nodeName string) {
	c := edgesite.NewDefaultEdgeSiteConfig()
	// TODO change ca file path @kadisi
	c.Modules.Edged.HostnameOverride = nodeName
	c.KubeAPIConfig.Master = kubeMaster
	c.KubeAPIConfig.KubeConfig = ""
	c.DataBase.DataSource = "/tmp/edgesite/edgesite.db"

	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Marshal edgesite config to yaml error %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(constants.EdgeSiteConfigFile, data, os.ModePerm); err != nil {
		fmt.Printf("Create edgesite config file %v error %v\n", constants.EdgeSiteConfigFile, err)
		os.Exit(1)
	}
}
