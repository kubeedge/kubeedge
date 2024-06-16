package metaserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIdentifier(t *testing.T) {
	cases := []struct {
		app      Application
		expected string
	}{
		{
			app: Application{
				Key:         "group/version/resource/namespaces/name",
				Verb:        "GET",
				Nodename:    "test-node",
				Subresource: "status",
				Option:      nil,
				ReqBody:     nil,
			},
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte("test-nodegroup/version/resource/namespaces/nameGETstatus"))),
		},
		{
			app: Application{
				Key:         "group/version/resource/namespaces/name",
				Verb:        "POST",
				Nodename:    "test-node",
				Subresource: "status",
				Option:      []byte(`{"foo":"bar"}`),
				ReqBody:     []byte(`{"baz":"qux"}`),
			},
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte("test-nodegroup/version/resource/namespaces/namePOSTstatus{\"foo\":\"bar\"}{\"baz\":\"qux\"}"))),
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("Identifier for %v", test.app), func(t *testing.T) {
			id := test.app.Identifier()
			assert.Equal(t, test.expected, id)
		})
	}
}

func TestOptionTo(t *testing.T) {
	type TestOption struct {
		Foo string `json:"foo"`
	}

	cases := []struct {
		app      Application
		expected TestOption
	}{
		{
			app: Application{
				Option: []byte(`{"foo":"bar"}`),
			},
			expected: TestOption{Foo: "bar"},
		},
		{
			app: Application{
				Option: []byte(`{"foo":"baz"}`),
			},
			expected: TestOption{Foo: "baz"},
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("OptionTo for %v", test.app), func(t *testing.T) {
			var opt TestOption
			err := test.app.OptionTo(&opt)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, opt)
		})
	}
}

func TestReqBodyTo(t *testing.T) {
	type TestReqBody struct {
		Baz string `json:"baz"`
	}

	cases := []struct {
		app      Application
		expected TestReqBody
	}{
		{
			app: Application{
				ReqBody: []byte(`{"baz":"qux"}`),
			},
			expected: TestReqBody{Baz: "qux"},
		},
		{
			app: Application{
				ReqBody: []byte(`{"baz":"quux"}`),
			},
			expected: TestReqBody{Baz: "quux"},
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("ReqBodyTo for %v", test.app), func(t *testing.T) {
			var body TestReqBody
			err := test.app.ReqBodyTo(&body)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, body)
		})
	}
}

func TestRespBodyTo(t *testing.T) {
	type TestRespBody struct {
		Quux string `json:"quux"`
	}

	cases := []struct {
		app      Application
		expected TestRespBody
	}{
		{
			app: Application{
				RespBody: []byte(`{"quux":"corge"}`),
			},
			expected: TestRespBody{Quux: "corge"},
		},
		{
			app: Application{
				RespBody: []byte(`{"quux":"grault"}`),
			},
			expected: TestRespBody{Quux: "grault"},
		},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("RespBodyTo for %v", test.app), func(t *testing.T) {
			var body TestRespBody
			err := test.app.RespBodyTo(&body)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, body)
		})
	}
}
