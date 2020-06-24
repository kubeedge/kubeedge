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

package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLClient_HTTPDo(t *testing.T) {
	client, err := GetURLClient(nil)
	if err != nil {
		t.Errorf("GetURLClient error: %v", err)
	}

	ts := getMockServer(t)
	resp, err := client.HTTPDo("GET", ts.URL+"/test", http.Header{}, nil)
	if err != nil {
		t.Errorf("HTTPDo error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got error status code, resp is %v", resp)
	}
}

func getMockServer(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			if r.URL.EscapedPath() != "/test" {
				t.Errorf("path error: %s", r.URL.EscapedPath())
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))

	return ts
}
