/*
Copyright 2024 The KubeEdge Authors.

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

package metaserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestNewApplication(t *testing.T) {
	cases := []struct {
		ctx         context.Context
		key         string
		verb        ApplicationVerb
		nodename    string
		subresource string
		option      interface{}
		reqBody     interface{}
	}{
		{
			ctx:         context.TODO(),
			key:         "key1",
			verb:        "GET",
			nodename:    "node1",
			subresource: "subresource1",
			option:      nil,
			reqBody:     nil,
		},
		{
			ctx:         context.TODO(),
			key:         "key2",
			verb:        "POST",
			nodename:    "node2",
			subresource: "subresource2",
			option:      []byte(`{"field-one":"value-one"}`),
			reqBody:     []byte(`{"field-one":"value-one"}`),
		},
	}

	for _, test := range cases {
		app, err := NewApplication(test.ctx, test.key, test.verb, test.nodename, test.subresource, test.option, test.reqBody)
		assert.NoError(t, err)
		assert.Equal(t, test.key, app.Key)
		assert.Equal(t, test.verb, app.Verb)
		assert.Equal(t, test.nodename, app.Nodename)
		assert.Equal(t, test.subresource, app.Subresource)
	}
}

func TestIdentifier(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult string
	}{
		{
			app: Application{
				Nodename:    "test-node-one",
				Key:         "group/version/resource/namespaces/name",
				Verb:        "GET",
				Option:      nil,
				ReqBody:     nil,
				Subresource: "pod",
			},
			stdResult: fmt.Sprintf("%x", sha256.Sum256([]byte("test-node-onegroup/version/resource/namespaces/nameGETpod"))),
		},
		{
			app: Application{
				Nodename:    "test-node-two",
				Key:         "group/version/resource/namespaces/name",
				Verb:        "POST",
				Option:      []byte(`{"foo":"bar"}`),
				ReqBody:     []byte(`{"baz":"qux"}`),
				Subresource: "pod",
			},
			stdResult: fmt.Sprintf("%x", sha256.Sum256([]byte("test-node-twogroup/version/resource/namespaces/namePOST{\"foo\":\"bar\"}{\"baz\":\"qux\"}pod"))),
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("Identifier for %v", test.app), func(t *testing.T) {
			id := test.app.Identifier()
			assert.Equal(t, test.stdResult, id)
		})
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult string
	}{
		{
			app: Application{
				Nodename: "test-node-one",
				Key:      "group/version/resource/namespaces/name",
				Verb:     "GET",
				Status:   "completed",
				Reason:   "test reason one",
			},
			stdResult: "(NodeName=test-node-one;Key=group/version/resource/namespaces/name;Verb=GET;Status=completed;Reason=test reason one)",
		},
		{
			app: Application{
				Nodename: "test-node-two",
				Key:      "group/version/resource/namespaces/name",
				Verb:     "POST",
				Status:   "pending",
				Reason:   "test reason two",
			},
			stdResult: "(NodeName=test-node-two;Key=group/version/resource/namespaces/name;Verb=POST;Status=pending;Reason=test reason two)",
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("String for %v", test.app), func(t *testing.T) {
			stringResult := test.app.String()
			assert.Equal(t, stringResult, test.stdResult)
		})
	}
}

func TestReqContent(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult []byte
	}{
		{
			app:       Application{ReqBody: []byte(`{"test":"data"}`)},
			stdResult: []byte(`{"test":"data"}`),
		},
		{
			app: Application{
				Nodename: "test-node",
				Key:      "group/version/resource/namespaces/name",
			},
			stdResult: nil,
		},
		{
			app:       Application{ReqBody: nil},
			stdResult: nil,
		},
	}

	for _, test := range cases {
		assert.Equal(t, test.app.ReqContent(), test.stdResult)
	}
}

func TestRespContent(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult []byte
	}{
		{
			app:       Application{RespBody: []byte(`{"test":"data"}`)},
			stdResult: []byte(`{"test":"data"}`),
		},
		{
			app: Application{
				Nodename: "test-node",
				Key:      "group/version/resource/namespaces/name",
			},
			stdResult: nil,
		},
		{
			app:       Application{RespBody: nil},
			stdResult: nil,
		},
	}

	for _, test := range cases {
		assert.Equal(t, test.app.RespContent(), test.stdResult)
	}
}

func TestOptionTo(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult map[string]string
	}{
		{
			app:       Application{Option: []byte(`{"field-one":"value-one"}`)},
			stdResult: map[string]string{"field-one": "value-one"},
		},
		{
			app:       Application{Option: []byte(`{"field-two":"value-two"}`)},
			stdResult: map[string]string{"field-two": "value-two"},
		},
	}

	for _, test := range cases {
		var result map[string]string
		err := test.app.OptionTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, test.stdResult, result)
	}
}

func TestReqBodyTo(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult map[string]string
	}{
		{
			app:       Application{ReqBody: []byte(`{"test-key":"test-value"}`)},
			stdResult: map[string]string{"test-key": "test-value"},
		},
	}
	for _, test := range cases {
		var result map[string]string
		err := test.app.ReqBodyTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, result, test.stdResult)
	}
}

func TestRespBodyTo(t *testing.T) {
	cases := []struct {
		app      Application
		expected map[string]string
	}{
		{
			app:      Application{RespBody: []byte(`{"test-key":"test-value"}`)},
			expected: map[string]string{"test-key": "test-value"},
		},
	}

	for _, test := range cases {
		var result map[string]string
		err := test.app.RespBodyTo(&result)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, result)
	}
}

func TestGVR(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult schema.GroupVersionResource
	}{
		{
			app: Application{
				Key: "/group/version/resource/namespaces/ns",
			},
			stdResult: schema.GroupVersionResource{
				Group:    "group",
				Version:  "version",
				Resource: "resource",
			},
		},
		{
			app: Application{
				Key: "/group-two/v2/resource/namespaces/ns-two",
			},
			stdResult: schema.GroupVersionResource{
				Group:    "group-two",
				Version:  "v2",
				Resource: "resource",
			},
		},
	}

	for _, test := range cases {
		t.Run("GVR", func(t *testing.T) {
			gvr := test.app.GVR()
			assert.Equal(t, test.stdResult, gvr)
		})
	}
}

func TestNamespace(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult string
	}{
		{
			app: Application{
				Key: "/group/version/resource/test-namespace-one/",
			},
			stdResult: "test-namespace-one",
		},
		{
			app: Application{
				Key: "/group-two/v2/resource/test-namespace-two/",
			},
			stdResult: "test-namespace-two",
		},
	}

	for _, test := range cases {
		t.Run("Namespace", func(t *testing.T) {
			ns := test.app.Namespace()
			assert.Equal(t, test.stdResult, ns)
		})
	}
}

func TestGetStatus(t *testing.T) {
	cases := []struct {
		app       Application
		stdResult ApplicationStatus
	}{
		{
			app:       Application{Status: PreApplying},
			stdResult: PreApplying,
		},
		{
			app:       Application{Status: Completed},
			stdResult: Completed,
		},
	}

	for _, test := range cases {
		t.Run("GetStatus", func(t *testing.T) {
			status := test.app.GetStatus()
			assert.Equal(t, test.stdResult, status)
		})
	}
}

func TestMsgToApplication(t *testing.T) {
	cases := []struct {
		msg       model.Message
		stdResult *Application
		hasError  bool
	}{
		{
			msg: model.Message{
				Content: []byte(`{"Key":"group/version/resource/namespaces/name","Verb":"GET","Nodename":"test-node"}`),
			},
			stdResult: &Application{
				Key:      "group/version/resource/namespaces/name",
				Verb:     "GET",
				Nodename: "test-node",
			},
			hasError: false,
		},
		{
			msg: model.Message{
				Content: []byte(`invalid-json`),
			},
			stdResult: nil,
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run("MsgToApplication", func(t *testing.T) {
			app, err := MsgToApplication(test.msg)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.stdResult, app)
			}
		})
	}
}

func TestMsgToApplications(t *testing.T) {
	cases := []struct {
		msg       model.Message
		stdResult map[string]Application
		hasError  bool
	}{
		{
			msg: model.Message{
				Content: []byte(`{"app1":{"Key":"group/version/resource/namespaces/name1","Verb":"GET","Nodename":"test-node1"},
				"app2":{"Key":"group/version/resource/namespaces/name2","Verb":"POST","Nodename":"test-node2"}}`),
			},
			stdResult: map[string]Application{
				"app1": {
					Key:      "group/version/resource/namespaces/name1",
					Verb:     "GET",
					Nodename: "test-node1",
				},
				"app2": {
					Key:      "group/version/resource/namespaces/name2",
					Verb:     "POST",
					Nodename: "test-node2",
				},
			},
			hasError: false,
		},
		{
			msg: model.Message{
				Content: []byte(`invalid-json`),
			},
			stdResult: nil,
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run("MsgToApplications", func(t *testing.T) {
			apps, err := MsgToApplications(test.msg)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.stdResult, apps)
			}
		})
	}
}
