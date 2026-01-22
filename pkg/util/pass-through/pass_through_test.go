package passthrough

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPassThroughPath(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		name string
		path string
		verb string
		want bool
	}{
		{
			name: "/version::post is not pass through path",
			path: "/version",
			verb: "post",
			want: false,
		},
		{
			name: "/version::get is pass through path",
			path: "/version",
			verb: "get",
			want: true,
		},
		{
			name: "/healthz::put is not pass through path",
			path: "/healthz",
			verb: "put",
			want: false,
		},
		{
			name: "/healthz::get is pass through path",
			path: "/healthz",
			verb: "get",
			want: true,
		},
		{
			name: "/livez::patch is not pass through path",
			path: "/livez",
			verb: "patch",
			want: false,
		},
		{
			name: "/livez::get is pass through path",
			path: "/livez",
			verb: "get",
			want: true,
		},
		{
			name: "/readyz::delete is not pass through path",
			path: "/readyz",
			verb: "delete",
			want: false,
		},
		{
			name: "/readyz::get is pass through path",
			path: "/readyz",
			verb: "get",
			want: true,
		},
		{
			name: "/api::get is pass through path",
			path: "/api",
			verb: "get",
			want: true,
		},
		{
			name: "/apis::get is pass through path",
			path: "/apis",
			verb: "get",
			want: true,
		},
		{
			name: "/api/v1/namespaces/kube-system/pods::get is not pass through path",
			path: "/api/v1/namespaces/kube-system/pods",
			verb: "get",
			want: false,
		},
		{
			name: "/api/v1::get is pass through path",
			path: "/api/v1",
			verb: "get",
			want: true,
		},
		{
			name: "/apis/discovery.k8s.io/v1::get is pass through path",
			path: "/apis/discovery.k8s.io/v1",
			verb: "get",
			want: true,
		},
		{
			name: "/version/::get is pass through path",
			path: "/version/",
			verb: "get",
			want: true,
		},
		{
			name: "/openapi/v2::get is pass through path",
			path: "/openapi/v2",
			verb: "get",
			want: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			assert.Equal(IsPassThroughPath(testcase.path, testcase.verb), testcase.want, "Path: %s Verb: %s", testcase.path, testcase.verb)
		})
	}
}
