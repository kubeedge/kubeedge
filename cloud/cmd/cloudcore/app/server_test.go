package app

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/policycontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/router"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager"
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

// registerModules now threads c.Modules.IptablesManager into both edgecontroller.Register
// and cloudstream.Register (needed so their shouldUseEdgeTunnelIP() checks know whether
// iptablesManager is running in internal or external mode). Patch out every sub-module's
// Register call so this only exercises the wiring in registerModules itself, not the real
// module construction (several of them, e.g. cloudhub/policycontroller, need a live cluster).
func TestRegisterModules(t *testing.T) {
	var gotEdgeControllerIptablesMgr *v1alpha1.IptablesManager
	var gotCloudStreamIptablesMgr *v1alpha1.IptablesManager

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(cloudhub.Register, func(*v1alpha1.CloudHub) {})
	patches.ApplyFunc(edgecontroller.Register, func(_ *v1alpha1.EdgeController, iptablesMgr *v1alpha1.IptablesManager) {
		gotEdgeControllerIptablesMgr = iptablesMgr
	})
	patches.ApplyFunc(devicecontroller.Register, func(*v1alpha1.DeviceController) {})
	patches.ApplyFunc(taskmanager.Register, func(*v1alpha1.TaskManager) {})
	patches.ApplyFunc(synccontroller.Register, func(*v1alpha1.SyncController) {})
	patches.ApplyFunc(cloudstream.Register, func(_ *v1alpha1.CloudStream, _ *v1alpha1.CommonConfig, iptablesMgr *v1alpha1.IptablesManager) {
		gotCloudStreamIptablesMgr = iptablesMgr
	})
	patches.ApplyFunc(router.Register, func(*v1alpha1.Router) {})
	patches.ApplyFunc(dynamiccontroller.Register, func(*v1alpha1.DynamicController, bool) {})
	patches.ApplyFunc(policycontroller.Register, func(*rest.Config) {})

	iptablesMgr := &v1alpha1.IptablesManager{Mode: v1alpha1.ExternalMode}
	config := &v1alpha1.CloudCoreConfig{
		CommonConfig: &v1alpha1.CommonConfig{},
		Modules: &v1alpha1.Modules{
			CloudHub:          &v1alpha1.CloudHub{},
			EdgeController:    &v1alpha1.EdgeController{},
			DeviceController:  &v1alpha1.DeviceController{},
			TaskManager:       &v1alpha1.TaskManager{},
			SyncController:    &v1alpha1.SyncController{},
			CloudStream:       &v1alpha1.CloudStream{},
			Router:            &v1alpha1.Router{},
			DynamicController: &v1alpha1.DynamicController{},
			IptablesManager:   iptablesMgr,
		},
	}

	registerModules(config)

	assert.Same(t, iptablesMgr, gotEdgeControllerIptablesMgr)
	assert.Same(t, iptablesMgr, gotCloudStreamIptablesMgr)
}

func TestRegisterModules_NilIptablesManager(t *testing.T) {
	var gotEdgeControllerIptablesMgr *v1alpha1.IptablesManager
	var gotCloudStreamIptablesMgr *v1alpha1.IptablesManager

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(cloudhub.Register, func(*v1alpha1.CloudHub) {})
	patches.ApplyFunc(edgecontroller.Register, func(_ *v1alpha1.EdgeController, iptablesMgr *v1alpha1.IptablesManager) {
		gotEdgeControllerIptablesMgr = iptablesMgr
	})
	patches.ApplyFunc(devicecontroller.Register, func(*v1alpha1.DeviceController) {})
	patches.ApplyFunc(taskmanager.Register, func(*v1alpha1.TaskManager) {})
	patches.ApplyFunc(synccontroller.Register, func(*v1alpha1.SyncController) {})
	patches.ApplyFunc(cloudstream.Register, func(_ *v1alpha1.CloudStream, _ *v1alpha1.CommonConfig, iptablesMgr *v1alpha1.IptablesManager) {
		gotCloudStreamIptablesMgr = iptablesMgr
	})
	patches.ApplyFunc(router.Register, func(*v1alpha1.Router) {})
	patches.ApplyFunc(dynamiccontroller.Register, func(*v1alpha1.DynamicController, bool) {})
	patches.ApplyFunc(policycontroller.Register, func(*rest.Config) {})

	config := &v1alpha1.CloudCoreConfig{
		CommonConfig: &v1alpha1.CommonConfig{},
		Modules: &v1alpha1.Modules{
			CloudHub:          &v1alpha1.CloudHub{},
			EdgeController:    &v1alpha1.EdgeController{},
			DeviceController:  &v1alpha1.DeviceController{},
			TaskManager:       &v1alpha1.TaskManager{},
			SyncController:    &v1alpha1.SyncController{},
			CloudStream:       &v1alpha1.CloudStream{},
			Router:            &v1alpha1.Router{},
			DynamicController: &v1alpha1.DynamicController{},
			IptablesManager:   nil,
		},
	}

	registerModules(config)

	assert.Nil(t, gotEdgeControllerIptablesMgr)
	assert.Nil(t, gotCloudStreamIptablesMgr)
}
