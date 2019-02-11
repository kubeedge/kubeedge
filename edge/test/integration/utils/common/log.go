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

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
)

//Function to get time in millisec
func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

//functiont to log the Ginkgo framework logs
func logf(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

//Function to generate INFO logs
func Logf(format string, args ...interface{}) {
	logf("INFO", format, args...)
}

//Funciton to log Filure logs
func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logf("FAIL", msg)
	Fail(nowStamp()+": "+msg, 1)
}

//Function to log DEBUG logs
func Debug(format string, data []byte, err error) {
	if err == nil {
		glog.V(8).Infof(format, data)
	} else {
		glog.ErrorDepth(2, "http err:", err)
	}
}

//function for log level
func Info(format string, args ...interface{}) {
	glog.V(4).Infof(format, args...)
}

//function for log level
func InfoV2(format string, args ...interface{}) {
	glog.V(2).Infof(format, args...)
}

//function for log level
func InfoV6(format string, args ...interface{}) {
	glog.V(6).Infof(format, args...)
}

//Function to print the test case name and status of execution
func PrintTestcaseNameandStatus() {
	var testdesc GinkgoTestDescription
	var Status string
	testdesc = CurrentGinkgoTestDescription()
	if testdesc.Failed == true {
		Status = "FAILED"
	} else {
		Status = "PASSED"
	}
	InfoV6("TestCase:%40s     Status=%s", testdesc.TestText, Status)
}
