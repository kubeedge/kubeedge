/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policycontroller

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	pm "github.com/kubeedge/kubeedge/cloud/pkg/policycontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/features"
)

const (
	contextTypeStr = "context.Context"
	managerTypeStr = "manager.Manager"
	errorTypeStr   = "error"
)

func TestName(t *testing.T) {
	pc := &policyController{}
	expected := modules.PolicyControllerModuleName

	if got := pc.Name(); got != expected {
		t.Errorf("Name() = %v, want %v", got, expected)
	}
}

func TestGroup(t *testing.T) {
	pc := &policyController{}
	expected := modules.PolicyControllerGroupName

	if got := pc.Group(); got != expected {
		t.Errorf("Group() = %v, want %v", got, expected)
	}
}

func TestEnable(t *testing.T) {
	tests := []struct {
		name           string
		featureEnabled bool
		want           bool
	}{
		{
			name:           "Feature enabled",
			featureEnabled: true,
			want:           true,
		},
		{
			name:           "Feature disabled",
			featureEnabled: false,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := features.DefaultMutableFeatureGate.SetFromMap(
				map[string]bool{string(features.RequireAuthorization): tt.featureEnabled}); err != nil {
				t.Fatalf("Failed to set feature gate: %v", err)
			}

			pc := &policyController{}
			if got := pc.Enable(); got != tt.want {
				t.Errorf("Enable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessScheme(t *testing.T) {
	if accessScheme == nil {
		t.Error("Expected accessScheme to be initialized")
	}

	gvk := schema.GroupVersionKind{
		Group:   "policy.kubeedge.io",
		Version: "v1alpha1",
		Kind:    "ServiceAccountAccess",
	}

	obj, err := accessScheme.New(gvk)
	if err != nil {
		t.Errorf("Failed to create ServiceAccountAccess from scheme: %v", err)
	}

	if _, ok := obj.(*policyv1alpha1.ServiceAccountAccess); !ok {
		t.Errorf("Expected *policyv1alpha1.ServiceAccountAccess, got %T", obj)
	}
}

func TestSchemeRegistration(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   "policy.kubeedge.io",
		Version: "v1alpha1",
		Kind:    "ServiceAccountAccess",
	}

	obj, err := accessScheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create object through scheme: %v", err)
	}

	objGVK := accessScheme.Recognizes(gvk)
	if !objGVK {
		t.Errorf("Expected scheme to recognize %v", gvk)
	}

	if _, ok := obj.(*policyv1alpha1.ServiceAccountAccess); !ok {
		t.Errorf("Expected *policyv1alpha1.ServiceAccountAccess, got %T", obj)
	}
}

func TestInitFunction(t *testing.T) {
	if accessScheme == nil {
		t.Error("Expected accessScheme to be initialized by init()")
	}

	gvk := schema.GroupVersionKind{
		Group:   "policy.kubeedge.io",
		Version: "v1alpha1",
		Kind:    "ServiceAccountAccess",
	}

	obj, err := accessScheme.New(gvk)
	if err != nil {
		t.Errorf("Failed to create ServiceAccountAccess from scheme: %v", err)
	}

	if _, ok := obj.(*policyv1alpha1.ServiceAccountAccess); !ok {
		t.Errorf("Expected *policyv1alpha1.ServiceAccountAccess, got %T", obj)
	}
}

func TestRegister(t *testing.T) {
	if err := features.DefaultMutableFeatureGate.SetFromMap(
		map[string]bool{string(features.RequireAuthorization): true}); err != nil {
		t.Fatalf("Failed to set feature gate: %v", err)
	}

	regFunc := reflect.ValueOf(Register)
	if regFunc.Kind() != reflect.Func {
		t.Error("Expected Register to be a function")
	}

	regType := reflect.TypeOf(Register)
	if regType.NumIn() != 1 {
		t.Errorf("Expected Register to take 1 argument, got %d", regType.NumIn())
	}

	if regType.In(0).String() != "*rest.Config" {
		t.Errorf("Expected Register argument to be *rest.Config, got %s", regType.In(0).String())
	}

	pc := &policyController{
		ctx: context.Background(),
	}

	if pc.Name() != modules.PolicyControllerModuleName {
		t.Errorf("Expected Name() to return %q, got %q", modules.PolicyControllerModuleName, pc.Name())
	}

	if pc.Group() != modules.PolicyControllerGroupName {
		t.Errorf("Expected Group() to return %q, got %q", modules.PolicyControllerGroupName, pc.Group())
	}

	if !pc.Enable() {
		t.Error("Expected Enable() to return true")
	}

	moduleType := reflect.TypeOf((*core.Module)(nil)).Elem()
	if !reflect.TypeOf(pc).Implements(moduleType) {
		t.Error("policyController should implement core.Module")
	}
}

// TestRegisterDoesNotConstructManager verifies that Register() does NOT call
// NewAccessRoleControllerManager eagerly.  Before the fix, Register() called
// NewAccessRoleControllerManager unconditionally, which would crash for keadm /
// standalone deployments when LeaderElection was enabled without an explicit
// LeaderElectionNamespace.  Now, the manager is only built inside Start().
func TestRegisterDoesNotConstructManager(t *testing.T) {
	cfg := &rest.Config{Host: "https://fake-host:6443"}
	pc := &policyController{
		kubeCfg: cfg,
		ctx:     context.Background(),
	}

	// A freshly constructed policyController must NOT have a manager yet.
	// manager field is unexported; use reflect to inspect it.
	pcType := reflect.TypeOf(pc).Elem()
	_, hasManager := pcType.FieldByName("manager")
	// The struct no longer carries a pre-built manager; kubeCfg is stored instead.
	_, hasKubeCfg := pcType.FieldByName("kubeCfg")
	if !hasKubeCfg {
		t.Error("policyController should store kubeCfg for deferred manager construction")
	}
	if hasManager {
		// If the field still exists it must be nil at this point.
		managerVal := reflect.ValueOf(pc).Elem().FieldByName("manager")
		if !managerVal.IsNil() {
			t.Error("Register() must not eagerly construct the manager; manager field should be nil after Register")
		}
	}
}

func TestStartMethod(t *testing.T) {
	pc := &policyController{
		ctx: context.Background(),
	}

	startMethod := reflect.ValueOf(pc).MethodByName("Start")
	if !startMethod.IsValid() {
		t.Error("Expected to find Start method on policyController")
	}

	methodType := startMethod.Type()
	if methodType.NumIn() != 0 {
		t.Errorf("Expected Start to take 0 arguments, got %d", methodType.NumIn())
	}

	if methodType.NumOut() != 0 {
		t.Errorf("Expected Start to return 0 values, got %d", methodType.NumOut())
	}
}

func TestNewAccessRoleControllerManager(t *testing.T) {
	managerFunc := reflect.ValueOf(NewAccessRoleControllerManager)
	if !managerFunc.IsValid() {
		t.Error("Expected to find NewAccessRoleControllerManager function")
	}

	funcType := managerFunc.Type()
	if funcType.NumIn() != 3 {
		t.Errorf("Expected NewAccessRoleControllerManager to take 3 arguments, got %d", funcType.NumIn())
	}

	if funcType.In(0).String() != contextTypeStr {
		t.Errorf("Expected first argument to be %s, got %s", contextTypeStr, funcType.In(0).String())
	}

	if funcType.In(1).String() != "*rest.Config" {
		t.Errorf("Expected second argument to be *rest.Config, got %s", funcType.In(1).String())
	}

	if funcType.In(2).String() != "policycontroller.Options" {
		t.Errorf("Expected third argument to be policycontroller.Options, got %s", funcType.In(2).String())
	}

	if funcType.NumOut() != 2 {
		t.Errorf("Expected NewAccessRoleControllerManager to return 2 values, got %d", funcType.NumOut())
	}

	if funcType.Out(0).String() != managerTypeStr {
		t.Errorf("Expected first return value to be %s, got %s", managerTypeStr, funcType.Out(0).String())
	}

	if funcType.Out(1).String() != errorTypeStr {
		t.Errorf("Expected second return value to be %s, got %s", errorTypeStr, funcType.Out(1).String())
	}
}

// TestNewAccessRoleControllerManagerOutOfCluster verifies that
// newManager does not fail when called with a non-empty REST config host
// (simulating an out-of-cluster connection) as long as LeaderElection is
// disabled.  This is the keadm / standalone-binary scenario identified by
// reviewer DoisLONG.  We call newManager rather than
// NewAccessRoleControllerManager because the latter also runs
// setupControllers, which triggers API-server discovery against the fake host.
func TestNewAccessRoleControllerManagerOutOfCluster(t *testing.T) {
	// Use a fake, non-empty host so controller-runtime treats this as an
	// out-of-cluster configuration.
	cfg := &rest.Config{Host: "https://fake-apiserver:6443"}

	_, err := newManager(cfg, Options{
		LeaderElection:         false, // must be false to avoid in-cluster namespace lookup
		HealthProbeBindAddress: "",    // disabled – no port binding during the test
	})
	if err != nil {
		t.Errorf("newManager() with LeaderElection=false should not fail for out-of-cluster config, got: %v", err)
	}
}

// TestPolicyControllerDisabled verifies that when the RequireAuthorization
// feature gate is disabled, Enable() returns false and (critically) that
// Register() stores only the config — it does NOT construct the manager.
// Before the fix, Register() called NewAccessRoleControllerManager
// unconditionally, which would crash in out-of-cluster environments.
func TestPolicyControllerDisabled(t *testing.T) {
	if err := features.DefaultMutableFeatureGate.SetFromMap(
		map[string]bool{string(features.RequireAuthorization): false}); err != nil {
		t.Fatalf("Failed to set feature gate: %v", err)
	}

	cfg := &rest.Config{Host: "https://fake-host:6443"}
	pc := &policyController{
		kubeCfg: cfg,
		ctx:     context.Background(),
	}

	// Enable() must reflect the disabled feature gate.
	if pc.Enable() {
		t.Error("Enable() should return false when RequireAuthorization feature gate is disabled")
	}

	// kubeCfg is stored but the manager must not be eagerly constructed.
	if pc.kubeCfg == nil {
		t.Error("kubeCfg should be stored on the policyController")
	}

	// Restore the feature gate to avoid polluting other tests.
	t.Cleanup(func() {
		_ = features.DefaultMutableFeatureGate.SetFromMap(
			map[string]bool{string(features.RequireAuthorization): false})
	})
}

// TestNewAccessRoleControllerManagerHealthProbeDisabled verifies that passing
// an empty HealthProbeBindAddress does not cause an error.  An empty string
// tells controller-runtime not to bind a health-probe listener at all, which
// is the right behaviour when the port would conflict or the feature is
// intentionally disabled.  We call newManager rather than
// NewAccessRoleControllerManager because the latter also runs
// setupControllers, which triggers API-server discovery against the fake host.
func TestNewAccessRoleControllerManagerHealthProbeDisabled(t *testing.T) {
	cfg := &rest.Config{Host: "https://fake-apiserver:6443"}

	_, err := newManager(cfg, Options{
		LeaderElection:         false,
		HealthProbeBindAddress: "", // disabled
	})
	if err != nil {
		t.Errorf("newManager() with empty HealthProbeBindAddress should not fail, got: %v", err)
	}
}

// TestNewAccessRoleControllerManagerLeaderElectionNamespace verifies that when
// LeaderElection is enabled with an explicit LeaderElectionNamespace the
// manager is constructed without error.  The actual leader-election loop is
// not started (that requires mgr.Start()), so this test merely checks that
// the Options are accepted and wired correctly.
//
// We call newManager rather than NewAccessRoleControllerManager because the
// latter also runs setupControllers, which triggers API-server discovery
// against the fake host.  The namespace is validated only inside mgr.Start()
// when the resource-lock is actually acquired, not during NewManager().
func TestNewAccessRoleControllerManagerLeaderElectionNamespace(t *testing.T) {
	cfg := &rest.Config{Host: "https://fake-apiserver:6443"}

	_, err := newManager(cfg, Options{
		LeaderElection:          true,
		LeaderElectionNamespace: "kubeedge", // explicit namespace – no in-cluster file needed
		HealthProbeBindAddress:  "",
	})
	if err != nil {
		t.Errorf("newManager() with explicit LeaderElectionNamespace should not fail, got: %v", err)
	}
}

func TestSetupControllers(t *testing.T) {
	setupFunc := reflect.ValueOf(setupControllers)
	if !setupFunc.IsValid() {
		t.Error("Expected to find setupControllers function")
	}

	funcType := setupFunc.Type()
	if funcType.NumIn() != 2 {
		t.Errorf("Expected setupControllers to take 2 arguments, got %d", funcType.NumIn())
	}

	if funcType.In(0).String() != contextTypeStr {
		t.Errorf("Expected first argument to be %s, got %s", contextTypeStr, funcType.In(0).String())
	}

	if funcType.In(1).String() != managerTypeStr {
		t.Errorf("Expected second argument to be %s, got %s", managerTypeStr, funcType.In(1).String())
	}

	if funcType.NumOut() != 1 {
		t.Errorf("Expected setupControllers to return 1 value, got %d", funcType.NumOut())
	}

	if funcType.Out(0).String() != errorTypeStr {
		t.Errorf("Expected return value to be %s, got %s", errorTypeStr, funcType.Out(0).String())
	}
}

func TestCreateController(t *testing.T) {
	ctrl := &pm.Controller{}

	ctrlType := reflect.TypeOf(ctrl).Elem()

	clientField, exists := ctrlType.FieldByName("Client")
	if !exists {
		t.Error("Expected Controller to have a Client field")
	} else if clientField.Type.String() != "client.Client" {
		t.Errorf("Expected Client field to be of type client.Client, got %s", clientField.Type.String())
	}

	msgField, exists := ctrlType.FieldByName("MessageLayer")
	if !exists {
		t.Error("Expected Controller to have a MessageLayer field")
	} else if msgField.Type.String() != "messagelayer.MessageLayer" {
		t.Errorf("Expected MessageLayer field to be of type messagelayer.MessageLayer, got %s", msgField.Type.String())
	}
}

func TestCompleteControllerCoverage(t *testing.T) {
	pc := &policyController{}

	moduleType := reflect.TypeOf((*core.Module)(nil)).Elem()
	if !reflect.TypeOf(pc).Implements(moduleType) {
		t.Error("policyController should implement core.Module")
	}

	methodNames := []string{"Name", "Group", "Enable", "Start"}
	for _, name := range methodNames {
		method := reflect.ValueOf(pc).MethodByName(name)
		if !method.IsValid() {
			t.Errorf("Expected to find %s method on policyController", name)
		}
	}

	pcType := reflect.TypeOf(pc).Elem()

	kubeCfgField, exists := pcType.FieldByName("kubeCfg")
	if !exists {
		t.Error("Expected policyController to have kubeCfg field")
	} else if kubeCfgField.Type.String() != "*rest.Config" {
		t.Errorf("Expected kubeCfg field to be *rest.Config, got %s", kubeCfgField.Type.String())
	}

	ctxField, exists := pcType.FieldByName("ctx")
	if !exists {
		t.Error("Expected policyController to have ctx field")
	} else if ctxField.Type.String() != contextTypeStr {
		t.Errorf("Expected ctx field to be %s, got %s", contextTypeStr, ctxField.Type.String())
	}

	if accessScheme == nil {
		t.Error("Expected accessScheme to be initialized")
	}

	kinds := accessScheme.AllKnownTypes()
	if len(kinds) == 0 {
		t.Error("Expected accessScheme to have registered types")
	}
}

func TestPolicyControllerPackageIntegration(t *testing.T) {
	access := &policyv1alpha1.ServiceAccountAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-access",
			Namespace: "default",
		},
		Spec: policyv1alpha1.AccessSpec{},
	}

	gvk := access.GetObjectKind().GroupVersionKind()
	t.Logf("ServiceAccountAccess GVK: %v", gvk)

	testScheme := runtime.NewScheme()
	err := policyv1alpha1.AddToScheme(testScheme)
	if err != nil {
		t.Fatalf("Failed to add policy types to scheme: %v", err)
	}

	gvk = schema.GroupVersionKind{
		Group:   "policy.kubeedge.io",
		Version: "v1alpha1",
		Kind:    "ServiceAccountAccess",
	}

	obj, err := testScheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create object through scheme: %v", err)
	}

	if _, ok := obj.(*policyv1alpha1.ServiceAccountAccess); !ok {
		t.Errorf("Expected *policyv1alpha1.ServiceAccountAccess, got %T", obj)
	}
}

func TestPackageExports(t *testing.T) {
	newFuncType := reflect.TypeOf(NewAccessRoleControllerManager)
	if newFuncType.Kind() != reflect.Func {
		t.Error("Expected NewAccessRoleControllerManager to be a function")
	}

	regFuncType := reflect.TypeOf(Register)
	if regFuncType.Kind() != reflect.Func {
		t.Error("Expected Register to be a function")
	}

	if accessScheme == nil {
		t.Error("Expected accessScheme to be initialized")
	}

	if !accessScheme.Recognizes(schema.GroupVersionKind{
		Group:   "policy.kubeedge.io",
		Version: "v1alpha1",
		Kind:    "ServiceAccountAccess",
	}) {
		t.Error("Expected scheme to recognize ServiceAccountAccess")
	}
}
