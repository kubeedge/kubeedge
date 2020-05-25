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

package common

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
)

//Function to get time in millisec
func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

//functiont to log the Ginkgo framework logs
func logf(level string, format string, args ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

//Funciton to log Filure logs
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logf("FAIL", msg)
	ginkgo.Fail(nowStamp()+": "+msg, 1)
}

//function for log level
func Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logf("INFO", msg)
}

//Function to print the test case name and status of execution
func PrintTestcaseNameandStatus() {
	var testdesc ginkgo.GinkgoTestDescription
	var Status string
	testdesc = ginkgo.CurrentGinkgoTestDescription()
	if testdesc.Failed {
		Status = "FAILED"
	} else {
		Status = "PASSED"
	}
	Infof("TestCase:%40s     Status=%s", testdesc.TestText, Status)
}
