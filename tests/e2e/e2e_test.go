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
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ereporters "k8s.io/kubernetes/test/e2e/reporters"

	_ "github.com/kubeedge/kubeedge/tests/e2e/apps"
	_ "github.com/kubeedge/kubeedge/tests/e2e/device"
	_ "github.com/kubeedge/kubeedge/tests/e2e/rule"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func TestMain(m *testing.M) {
	utils.CopyFlags(utils.Flags, flag.CommandLine)
	utils.RegisterFlags(flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	framework.AfterReadingAllFlags(&framework.TestContext)
	os.Exit(m.Run())
}

// Function to run the Ginkgo Test
func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	var _ = ginkgo.BeforeSuite(func() {
		utils.Infof("Before Suite Execution")

		if utils.LoadConfig().TestDevice {
			err := utils.MqttConnect()
			gomega.Expect(err).To(gomega.BeNil())
		}
	})

	ginkgo.AfterSuite(func() {
		ginkgo.By("After Suite Execution....!")
	})

	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			klog.Errorf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v.xml", framework.TestContext.ReportPrefix))))
		}
	}

	// Stream the progress to stdout and optionally a URL accepting progress updates.
	r = append(r, e2ereporters.NewProgressReporter(framework.TestContext.ProgressReportURL))

	// The DetailsRepoerter will output details about every test (name, files, lines, etc) which helps
	// when documenting our tests.
	if len(framework.TestContext.SpecSummaryOutput) > 0 {
		r = append(r, e2ereporters.NewDetailsReporterFile(framework.TestContext.SpecSummaryOutput))
	}

	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "kubeedge e2e suite", r)
}
