package passthrough

import "testing"

func TestIsPassThroughPath(t *testing.T) {
	tests := []struct {
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
		}, {
			name: "/version::get is pass through path",
			path: "/version",
			verb: "get",
			want: true,
		}, {
			name: "/healthz::put is not pass through path",
			path: "/healthz",
			verb: "put",
			want: false,
		}, {
			name: "/healthz::get is pass through path",
			path: "/healthz",
			verb: "get",
			want: true,
		}, {
			name: "/livez::patch is not pass through path",
			path: "/livez",
			verb: "patch",
			want: false,
		}, {
			name: "/livez::get is pass through path",
			path: "/livez",
			verb: "get",
			want: true,
		}, {
			name: "/readyz::delete is not pass through path",
			path: "/readyz",
			verb: "delete",
			want: false,
		}, {
			name: "/readyz::get is pass through path",
			path: "/readyz",
			verb: "get",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPassThroughPath(tt.path, tt.verb); got != tt.want {
				t.Errorf("IsPassThroughPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
