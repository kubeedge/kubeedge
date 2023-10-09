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
			name: "/healthz::get is not pass through path",
			path: "/healthz",
			verb: "get",
			want: false,
		}, {
			name: "/version::post is not pass through path",
			path: "/version",
			verb: "post",
			want: false,
		}, {
			name: "/version::get is pass through path",
			path: "/version",
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
