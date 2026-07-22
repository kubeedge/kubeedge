package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/policycontroller"
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

// TestRegisterPolicyController verifies that registerPolicyController()
// selects the deployment mode and lease namespace based on the outcome of
// rest.InClusterConfig(): in-cluster (Pod) environments must enable leader
// election with the "kubeedge" lease namespace, while keadm/standalone
// environments must disable it. rest.InClusterConfig and
// policycontroller.RegisterWithOptions are monkey-patched so the test does
// not depend on the actual environment it runs in or perform a real module
// registration.
func TestRegisterPolicyController(t *testing.T) {
	tests := []struct {
		name          string
		inClusterErr  error
		wantMode      policycontroller.DeploymentMode
		wantNamespace string
	}{
		{
			name:          "in-cluster environment enables leader election",
			inClusterErr:  nil,
			wantMode:      policycontroller.DeploymentModeInCluster,
			wantNamespace: "kubeedge",
		},
		{
			name:          "standalone environment disables leader election",
			inClusterErr:  errors.New("not running in-cluster"),
			wantMode:      policycontroller.DeploymentModeStandalone,
			wantNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMode policycontroller.DeploymentMode
			var gotNamespace string

			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFunc(rest.InClusterConfig, func() (*rest.Config, error) {
				if tt.inClusterErr != nil {
					return nil, tt.inClusterErr
				}
				return &rest.Config{}, nil
			})
			patches.ApplyFunc(policycontroller.RegisterWithOptions,
				func(_ *rest.Config, mode policycontroller.DeploymentMode, leaseNamespace string) {
					gotMode = mode
					gotNamespace = leaseNamespace
				})

			registerPolicyController()

			if gotMode != tt.wantMode {
				t.Errorf("registerPolicyController() mode = %v, want %v", gotMode, tt.wantMode)
			}
			if gotNamespace != tt.wantNamespace {
				t.Errorf("registerPolicyController() leaseNamespace = %q, want %q", gotNamespace, tt.wantNamespace)
			}
		})
	}
}
