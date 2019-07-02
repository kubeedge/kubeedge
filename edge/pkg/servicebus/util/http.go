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

// Package util
package util

import (
	"net/http"
)

// HTTPRequest is used structure used to unmarshal message content from clous
type HTTPRequest struct {
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

// HTTPResponse is HTTP request's response structure used to send response to cloud
type HTTPResponse struct {
	Header     http.Header `json:"header"`
	StatusCode int         `json:"status_code"`
	Body       []byte      `json:"body"`
}
