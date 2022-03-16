package image

import (
	"reflect"
	"testing"
)

func TestEdgeSet(t *testing.T) {
	type args struct {
		imageRepository string
		version         string
	}
	tests := []struct {
		name string
		args args
		want Set
	}{
		{
			name: "repo nil, ver not nil",
			args: args{
				imageRepository: "",
				version:         "v1.9.1",
			},
			want: Set{
				EdgeCore:  "kubeedge/installation-package:v1.9.1",
				EdgeMQTT:  "eclipse-mosquitto:1.6.15",
				EdgePause: "kubeedge/pause:3.1",
			},
		},
		{
			name: "repo nil, ver nil",
			args: args{
				imageRepository: "",
				version:         "",
			},
			want: Set{
				EdgeCore:  "kubeedge/installation-package",
				EdgeMQTT:  "eclipse-mosquitto:1.6.15",
				EdgePause: "kubeedge/pause:3.1",
			},
		},
		{
			name: "repo not nil, ver not nil",
			args: args{
				imageRepository: "kubeedge-test",
				version:         "v1.9.1",
			},
			want: Set{
				EdgeCore:  "kubeedge-test/installation-package:v1.9.1",
				EdgeMQTT:  "kubeedge-test/eclipse-mosquitto:1.6.15",
				EdgePause: "kubeedge-test/pause:3.1",
			},
		},
		{
			name: "repo not nil, ver nil",
			args: args{
				imageRepository: "kubeedge-test",
				version:         "",
			},
			want: Set{
				EdgeCore:  "kubeedge-test/installation-package",
				EdgeMQTT:  "kubeedge-test/eclipse-mosquitto:1.6.15",
				EdgePause: "kubeedge-test/pause:3.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EdgeSet(tt.args.imageRepository, tt.args.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EdgeSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
