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
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
)

//GenerateCerts - Generates Cerificates for Edge and Cloud nodes copy to respective folders
func GenerateCerts() error {
	cmd := exec.Command("bash", "-x", "scripts/generate_cert.sh")
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	return nil
}

func StartCloudCore() error {
	//Run ./cloudcore binary
	cmd := exec.Command("sh", "-c", constants.RunCloudcore)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckCloudcore)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatCloudcoreLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		Errorf("Failed to run cloudcore, error log:\n %v", string(bytes))
		os.Exit(1)
	}
	return nil
}

func StartEdgeCore() error {
	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", constants.RunEdgecore)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckEdgecore)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatEdgecoreLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		Errorf("Failed to run edgecore, error log:\n %v", string(bytes))
		os.Exit(1)
	}
	return nil
}

func StartEdgeSite() error {
	//Run ./edgecore after node registration
	cmd := exec.Command("sh", "-c", constants.RunEdgeSite)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
	}
	//Expect(err).Should(BeNil())
	time.Sleep(5 * time.Second)

	checkcmd := exec.Command("sh", "-c", constants.CheckEdgesite)
	if err := PrintCombinedOutput(checkcmd); err != nil {
		catcmd := exec.Command("sh", "-c", constants.CatEdgeSiteLog)
		Infof("===========> Executing: %s\n", strings.Join(catcmd.Args, " "))
		bytes, _ := catcmd.CombinedOutput()
		Errorf("Failed to run edgesite, error log:\n %v", string(bytes))
		os.Exit(1)
	}
	return nil
}

func DeploySetup(ctx *TestContext, nodeName, setupType string) error {
	//Do the neccessary config changes in Cloud and Edge nodes
	cmd := exec.Command("bash", "-x", "scripts/setup.sh", setupType, nodeName, ctx.Cfg.K8SMasterForKubeEdge)
	if err := PrintCombinedOutput(cmd); err != nil {
		return err
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
