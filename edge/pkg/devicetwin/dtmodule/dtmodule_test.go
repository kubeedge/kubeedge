/*
Copyright 2018 The KubeEdge Authors.

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

package dtmodule_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	. "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmanager"
	. "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
)

var _ = Describe("Dtmodule", func() {
	Describe("Unit Test for dtmodule.go", func() {
		testChan := make(chan interface{})
		var dtContext *dtcontext.DTContext
		var dtModule DTModule
		Context("Testing for InitWorker", func() {
			It("NameMemModule", func() {
				dtModule.Name = "MemModule"
				dtModule.InitWorker(testChan, testChan, testChan, dtContext)
				want := MemWorker{Group: "MemModule",
					Worker: Worker{ReceiverChan: testChan,
						ConfirmChan:   testChan,
						HeartBeatChan: testChan,
						DTContexts:    dtContext}}
				Expect(dtModule.Worker).To(Equal(want))
			})
			It("NameTwinModule", func() {
				dtModule.Name = "TwinModule"
				dtModule.InitWorker(testChan, testChan, testChan, dtContext)
				want := TwinWorker{Group: "TwinModule",
					Worker: Worker{ReceiverChan: testChan,
						ConfirmChan:   testChan,
						HeartBeatChan: testChan,
						DTContexts:    dtContext}}
				Expect(dtModule.Worker).To(Equal(want))
			})
			It("NameDeviceModule", func() {
				dtModule.Name = "DeviceModule"
				dtModule.InitWorker(testChan, testChan, testChan, dtContext)
				want := DeviceWorker{Group: "DeviceModule",
					Worker: Worker{ReceiverChan: testChan,
						ConfirmChan:   testChan,
						HeartBeatChan: testChan,
						DTContexts:    dtContext}}
				Expect(dtModule.Worker).To(Equal(want))
			})
			It("NameCommModule", func() {
				dtModule.Name = "CommModule"
				dtModule.InitWorker(testChan, testChan, testChan, dtContext)
				want := CommWorker{Group: "CommModule",
					Worker: Worker{ReceiverChan: testChan,
						ConfirmChan:   testChan,
						HeartBeatChan: testChan,
						DTContexts:    dtContext}}
				Expect(dtModule.Worker).To(Equal(want))
			})

		})
	})
})
