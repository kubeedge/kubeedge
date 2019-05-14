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
package utils

import (
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

//Test context struct
type TestContext struct {
	Cfg Config
}

//NewTestContext function to build testcontext with provided config.
func NewTestContext(cfg Config) *TestContext {
	return &TestContext{
		Cfg: cfg,
	}
}

//SendHttpRequest Function to prepare the http req and send
func SendHttpRequest(method, reqApi string) (error, *http.Response) {
	var body io.Reader
	var resp *http.Response

	client := &http.Client{}
	req, err := http.NewRequest(method, reqApi, body)
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return err, resp
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err = client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return err, resp
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return nil, resp
}

//MapLabels function add label selector
func MapLabels(ls map[string]string) string {
	selector := make([]string, 0, len(ls))
	for key, value := range ls {
		selector = append(selector, key+"="+value)
	}
	sort.StringSlice(selector).Sort()
	return url.QueryEscape(strings.Join(selector, ","))
}
