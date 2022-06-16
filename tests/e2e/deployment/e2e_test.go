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
	"flag"
	"fmt"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	e2ereporters "k8s.io/kubernetes/test/e2e/reporters"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	nodeName     string
	nodeSelector string
	// context to load config and access across the package
	ctx *utils.TestContext
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
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.Infof("Before Suite Execution")
		ctx = utils.NewTestContext(utils.LoadConfig())
		nodeName = "edge-node"
		nodeSelector = "test"

		err := utils.MqttConnect()
		Expect(err).To(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
	})

	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []Reporter
	if utils.LoadConfig().ReportDir != "" {
		if err := os.MkdirAll(utils.LoadConfig().ReportDir, 0755); err != nil {
			klog.Errorf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(utils.LoadConfig().ReportDir, fmt.Sprintf("junit_%v.xml", utils.LoadConfig().ReportPrefix))))
		}
	}

	// Stream the progress to stdout and optionally a URL accepting progress updates.
	r = append(r, e2ereporters.NewProgressReporter(utils.LoadConfig().ProgressReportURL))

	// The DetailsRepoerter will output details about every test (name, files, lines, etc) which helps
	// when documenting our tests.
	if len(utils.LoadConfig().SpecSummaryOutput) > 0 {
		r = append(r, e2ereporters.NewDetailsReporterFile(utils.LoadConfig().SpecSummaryOutput))
	}

	RunSpecsWithDefaultAndCustomReporters(t, "kubeedge e2e suite", r)
}
