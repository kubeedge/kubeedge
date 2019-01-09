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
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
)

const ANSI_COLOR_LIGHT_CYAN = "\x1b[1;36m"
const ANSI_COLOR_LIGHT_RED = "\x1b[1;31m"
const ANSI_COLOR_LIGHT_RESET = "\x1b[0m"

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func logf(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

func Logf(format string, args ...interface{}) {
	logf("INFO", format, args...)
}

func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logf("FAIL", msg)
	Fail(nowStamp()+": "+msg, 1)
}

func debug(format string, data []byte, err error) {
	if err == nil {
		glog.V(8).Infof(format, data)
	} else {
		glog.ErrorDepth(2, "http err:", err)
	}
}

func DebugHttp(format string, v interface{}) {
	var data []byte
	var err error
	switch v.(type) {
	case *http.Request:
		data, err = httputil.DumpRequestOut(v.(*http.Request), true)
	case *http.Response:
		data, err = httputil.DumpResponse(v.(*http.Response), true)
	}
	debug(format, data, err)
}

func Info(format string, args ...interface{}) {
	glog.V(4).Infof(format, args...)
}

func InfoV2(format string, args ...interface{}) {
	glog.V(2).Infof(format, args...)
}

func InfoV6(format string, args ...interface{}) {
	glog.V(6).Infof(format, args...)
}

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
