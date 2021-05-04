package common

import (
	"net"
	"reflect"
	"testing"

	"github.com/go-chassis/go-chassis/core/common"
)

func TestGetInterfaceIP(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    net.IP
		wantErr bool
	}{
		{
			name:    "TestGetInterfaceIP(): Case 1: Interface found",
			args:    args{name: "lo0"},
			want:    net.ParseIP("127.0.0.1"),
			wantErr: false,
		},
		{
			name:    "TestGetInterfaceIP(): Case 2: Interface not found",
			args:    args{name: "notfound"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInterfaceIP(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInterfaceIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInterfaceIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitServiceKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name          string
		args          args
		wantName      string
		wantNamespace string
	}{
		{
			name:          "TestSplitServiceKey(): Case 1: Test with name and namespace",
			args:          args{key: "key.test"},
			wantName:      "key",
			wantNamespace: "test",
		},
		{
			name:          "TestSplitServiceKey(): Case 2: Test with name",
			args:          args{key: "key"},
			wantName:      "key",
			wantNamespace: common.DefaultValue,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotNamespace := SplitServiceKey(tt.args.key)
			if gotName != tt.wantName {
				t.Errorf("SplitServiceKey() gotName = %v, want %v", gotName, tt.wantName)
				return
			}
			if gotNamespace != tt.wantNamespace {
				t.Errorf("SplitServiceKey() gotNamespace = %v, want %v", gotNamespace, tt.wantNamespace)
			}
		})
	}
}
