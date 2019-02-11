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
	"testing"

	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx *edge.TestContext
	cfg edge.Config
)

//Function to run the Ginkgo Test
func TestEdgecoreAppDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	//var UID string
	var _ = BeforeSuite(func() {
		common.InfoV6("Before Suite Execution")
		cfg = edge.LoadConfig()
		ctx = edge.NewTestContext(cfg)
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
	})

	RunSpecs(t, "kubeedge App Deploymet Suite")
}
