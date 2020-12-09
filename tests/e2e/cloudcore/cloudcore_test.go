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

package cloudcore

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var SecretTestTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Secret test in cloudcore E2E", func() {
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	Context("Test that the secret", func() {
		BeforeEach(func() {
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = SecretTestTestTimerGroup.NewTestTimer(testDescription.TestText)
		})

		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
		})

		It("E2E_SECRET_VERIFICATION: Verify that the secret exists", func() {
			secrets, err := utils.GetSecrets(ctx.Cfg.K8SMasterForKubeEdge+constants.SecrettHandler, "")
			fmt.Print(secrets.Items)
			Expect(err).NotTo(BeNil())
			Expect(len(secrets.Items) > 1).NotTo(BeTrue())

			hasCaSecret := false
			hasCloudcoreSecret := false
			hasTokensecret := false

			for _, secret := range secrets.Items {
				if secret.Name == "casecret" {
					hasCaSecret = true
				} else if secret.Name == "cloudcoresecret" {
					hasCloudcoreSecret = true
				} else if secret.Name == "tokensecret" {
					hasTokensecret = true
				}

			}
			Expect(hasCaSecret).To(BeTrue())
			Expect(hasCloudcoreSecret).To(BeTrue())
			Expect(hasTokensecret).To(BeTrue())
		})
	})
})
