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

package deployment

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	nodeName     string
	nodeSelector string
	//context to load config and access across the package
	ctx *utils.TestContext
)

//Function to run the Ginkgo Test
func TestEdgecoreAppDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.Infof("Before Suite Execution")
		//cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(utils.LoadConfig())
		nodeName = "edge-node"
		nodeSelector = "test"

		err := utils.MqttConnect()
		Expect(err).To(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
	})

	RunSpecs(t, "kubeedge App Deploymet Suite")
}
