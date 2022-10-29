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

func TestCloudSet(t *testing.T) {
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
				version:         "v1.12.0",
			},
			want: Set{
				CloudAdmission:         "kubeedge/admission:v1.12.0",
				CloudCloudcore:         "kubeedge/cloudcore:v1.12.0",
				CloudIptablesManager:   "kubeedge/iptables-manager:v1.12.0",
				CloudControllerManager: "kubeedge/controller-manager:v1.12.0",
			},
		},
		{
			name: "repo nil, ver nil",
			args: args{
				imageRepository: "",
				version:         "",
			},
			want: Set{
				CloudAdmission:         "kubeedge/admission",
				CloudCloudcore:         "kubeedge/cloudcore",
				CloudIptablesManager:   "kubeedge/iptables-manager",
				CloudControllerManager: "kubeedge/controller-manager",
			},
		},
		{
			name: "repo not nil, ver not nil",
			args: args{
				imageRepository: "kubeedge-test",
				version:         "v1.12.0",
			},
			want: Set{
				CloudAdmission:         "kubeedge-test/admission:v1.12.0",
				CloudCloudcore:         "kubeedge-test/cloudcore:v1.12.0",
				CloudIptablesManager:   "kubeedge-test/iptables-manager:v1.12.0",
				CloudControllerManager: "kubeedge-test/controller-manager:v1.12.0",
			},
		},
		{
			name: "repo not nil, ver nil",
			args: args{
				imageRepository: "kubeedge-test",
				version:         "",
			},
			want: Set{
				CloudAdmission:         "kubeedge-test/admission",
				CloudCloudcore:         "kubeedge-test/cloudcore",
				CloudIptablesManager:   "kubeedge-test/iptables-manager",
				CloudControllerManager: "kubeedge-test/controller-manager",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CloudSet(tt.args.imageRepository, tt.args.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloudSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_Get(t *testing.T) {
	tests := []struct {
		name string
		s    Set
		args string
		want string
	}{
		{
			name: "get cloudcore image",
			s:    Set{"cloudcore": "kubeedge-test/cloudcore:1.12.0"},
			args: "cloudcore",
			want: "kubeedge-test/cloudcore:1.12.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Get(tt.args); got != tt.want {
				t.Errorf("Set.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSet_List(t *testing.T) {
	tests := []struct {
		name string
		s    Set
		want []string
	}{
		{
			name: "test list",
			s:    Set{"cloudcore": "kubeedge-test/cloudcore:v1.12.0", "admission": "kubeedge-test/admission:v1.12.0", "controller-manager": "kubeedge-test/controller-manager:v1.12.0", "iptables-manager": "kubeedge-test/iptables-manager:v1.12.0"},
			want: []string{"kubeedge-test/cloudcore:v1.12.0", "kubeedge-test/admission:v1.12.0", "kubeedge-test/controller-manager:v1.12.0", "kubeedge-test/iptables-manager:v1.12.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.List()
			// we don't care about array sequence, so convert slice to map and compare it
			gotMap := make(map[string]string)
			for _, v := range got {
				gotMap[v] = ""
			}
			wantMap := make(map[string]string)
			for _, v := range tt.want {
				wantMap[v] = ""
			}
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("Set.List() = %v, but want %v", got, tt.want)
			}
		})
	}
}

func TestSet_Merge(t *testing.T) {
	tests := []struct {
		name string
		s    Set
		args Set
		want Set
	}{
		{
			name: "no overlapping keys",
			s:    Set{"kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/admission": "v1.12.0"},
			args: Set{"kubeedge-test/iptables-manager": "v1.12.0", "kubeedge-test/controller-manager": "v1.12.0"},
			want: Set{"kubeedge-test/admission": "v1.12.0", "kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/controller-manager": "v1.12.0", "kubeedge-test/iptables-manager": "v1.12.0"},
		},
		{
			name: "all no overlapping keys",
			s:    Set{"kubeedge-test/cloudcore": "v1.9.1", "kubeedge-test/admission": "v1.9.1"},
			args: Set{"kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/admission": "v1.12.0"},
			want: Set{"kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/admission": "v1.12.0"},
		},
		{
			name: "partially overlapping keys",
			s:    Set{"kubeedge-test/cloudcore": "v1.9.1", "kubeedge-test/admission": "v1.9.1", "kubeedge-test/iptables-manager": "v1.12.0"},
			args: Set{"kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/admission": "v1.12.0"},
			want: Set{"kubeedge-test/cloudcore": "v1.12.0", "kubeedge-test/admission": "v1.12.0", "kubeedge-test/iptables-manager": "v1.12.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Merge(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Set.Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
