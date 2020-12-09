package httpserver

import "testing"

func TestSetNamespace(t *testing.T) {
	tests := []struct {
		name              string
		kubeedgeNamespace string
		want              string
	}{
		{
			name:              "null",
			kubeedgeNamespace: "",
			want:              "kubeedge",
		},
		{
			name:              "edge-system",
			kubeedgeNamespace: "edge-system",
			want:              "edge-system",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setKubeedgeNamespce(tt.kubeedgeNamespace); got != tt.want {
				t.Errorf("setKubeedgeNamespce() = %v, want %v", got, tt.want)
			}
		})
	}
}
