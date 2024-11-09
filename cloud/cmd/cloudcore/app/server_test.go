package app

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util"
)

func TestNegotiateTunnelPort(t *testing.T) {
	type testCase struct {
		isConfigExits bool
		isPortExist   bool
		isPortUsed    bool
	}
	cases := testCase{}
	var cm = v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        modules.TunnelPort,
			Namespace:   constants.SystemNamespace,
			Annotations: map[string]string{},
		},
	}
	hostnameOverride := util.GetHostname()
	localIP, _ := util.GetLocalIP(hostnameOverride)
	patch := gomonkey.NewPatches()
	defer patch.Reset()
	patch.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
		if cases.isConfigExits {
			record := "{}"
			if cases.isPortExist {
				record = "{\"ipTunnelPort\":{\"" + localIP + "\":10351},\"port\":{\"10351\":true}}"
			} else if cases.isPortUsed {
				record = "{\"ipTunnelPort\":{\"127.0.0.1\":10351},\"port\":{\"10351\":true}}"
			}
			cm.ObjectMeta.Annotations[modules.TunnelPortRecordAnnotationKey] = record
			return fakekube.NewSimpleClientset(&cm)
		}
		return fakekube.NewSimpleClientset()
	})

	tests := []struct {
		name    string
		cases   testCase
		want    int
		wantErr bool
	}{
		{
			name:    "config not exits",
			want:    10351,
			wantErr: false,
		},
		{
			name:    "port record exits",
			cases:   testCase{isConfigExits: true, isPortExist: true},
			want:    10351,
			wantErr: false,
		},
		{
			name:    "port used",
			cases:   testCase{isConfigExits: true, isPortUsed: true},
			want:    10352,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases = tt.cases
			got, err := NegotiateTunnelPort()
			if (err != nil) != tt.wantErr {
				t.Errorf("NegotiateTunnelPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("NegotiateTunnelPort() got = %v, want %v", *got, tt.want)
			}
		})
	}
}
