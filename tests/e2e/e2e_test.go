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

package e2e

import (
	"flag"
	"os"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/spf13/pflag"

	_ "github.com/kubeedge/kubeedge/tests/e2e/apps"
	_ "github.com/kubeedge/kubeedge/tests/e2e/device"
	_ "github.com/kubeedge/kubeedge/tests/e2e/rule"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func TestMain(m *testing.M) {
	utils.CopyFlags(utils.Flags, flag.CommandLine)
	utils.RegisterFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	os.Exit(m.Run())
}

// Function to run the Ginkgo Test
func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	var _ = ginkgo.BeforeSuite(func() {
		utils.Infof("Before Suite Execution")

		err := utils.MqttConnect()
		gomega.Expect(err).To(gomega.BeNil())
	})
	ginkgo.AfterSuite(func() {
		ginkgo.By("After Suite Execution....!")
	})

	ginkgo.RunSpecs(t, "kubeedge e2e Suite")
}
