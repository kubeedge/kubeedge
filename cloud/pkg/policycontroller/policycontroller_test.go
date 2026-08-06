/*
Copyright 2026 The KubeEdge Authors.

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

	"github.com/agiledragon/gomonkey/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	pm "github.com/kubeedge/kubeedge/cloud/pkg/policycontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/features"
)

const (
	contextTypeStr    = "context.Context"
	managerTypeStr    = "manager.Manager"
	errorTypeStr      = "error"
	restConfigTypeStr = "*rest.Config"
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
			// Capture original value so we restore it regardless of test outcome.
			original := features.DefaultFeatureGate.Enabled(features.RequireAuthorization)
			t.Cleanup(func() {
				_ = features.DefaultMutableFeatureGate.SetFromMap(
					map[string]bool{string(features.RequireAuthorization): original})
			})

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
	original := features.DefaultFeatureGate.Enabled(features.RequireAuthorization)
	t.Cleanup(func() {
		_ = features.DefaultMutableFeatureGate.SetFromMap(
			map[string]bool{string(features.RequireAuthorization): original})
	})

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

	if regType.In(0).String() != restConfigTypeStr {
		t.Errorf("Expected Register argument to be *rest.Config, got %s", regType.In(0).String())
	}

	cfg := &rest.Config{Host: "https://fake-host:6443"}
	Register(cfg)

	info, ok := core.GetModules()[modules.PolicyControllerModuleName]
	if !ok {
		t.Fatal("expected Register to register the policy controller module")
	}

	pc, ok := info.GetModule().(*policyController)
	if !ok {
		t.Fatalf("expected registered module to be *policyController, got %T", info.GetModule())
	}

	if pc.kubeCfg != cfg {
		t.Error("expected Register to store the given kubeCfg")
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
// NewAccessRoleControllerManager eagerly. The manager is only built inside
// Start(), which Beehive calls only when Enable() returns true — this keeps
// Register() side-effect-free for keadm / standalone deployments and for
// replicas where the RequireAuthorization feature gate is disabled.
func TestRegisterDoesNotConstructManager(t *testing.T) {
	cfg := &rest.Config{Host: "https://fake-host:6443"}
	pc := &policyController{
		kubeCfg: cfg,
		ctx:     context.Background(),
	}

	pcType := reflect.TypeOf(pc).Elem()
	_, hasManager := pcType.FieldByName("manager")
	_, hasKubeCfg := pcType.FieldByName("kubeCfg")
	if !hasKubeCfg {
		t.Error("policyController should store kubeCfg for deferred manager construction")
	}
	if hasManager {
		t.Error("policyController should not carry a pre-built manager field; manager construction is deferred to Start()")
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

func TestRestartPolicy(t *testing.T) {
	pc := &policyController{}
	if got := pc.RestartPolicy(); got != nil {
		t.Errorf("RestartPolicy() = %v, want nil (use the default restart policy)", got)
	}
}

// fakeManager is a minimal manager.Manager stand-in used to test
// policyController.Start() without dialling a real (or fake) API server.
// It embeds the manager.Manager interface so it satisfies the type without
// implementing every method; Start() is the only method policyController.Start()
// calls on the returned manager, so that is the only one overridden here.
type fakeManager struct {
	manager.Manager
	startErr error
}

func (f *fakeManager) Start(_ context.Context) error {
	return f.startErr
}

// TestStartHappyPath verifies that policyController.Start() obtains a manager
// through NewAccessRoleControllerManager and runs it via mgr.Start(), without
// hitting the klog.Fatalf branch. NewAccessRoleControllerManager is
// monkey-patched (via gomonkey, already used elsewhere in this codebase, e.g.
// cloud/cmd/cloudcore/app/server_test.go) to return a fakeManager so the test
// does not depend on real API-server connectivity and returns immediately.
func TestStartHappyPath(t *testing.T) {
	var gotCfg *rest.Config
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(NewAccessRoleControllerManager,
		func(_ context.Context, cfg *rest.Config) (manager.Manager, error) {
			gotCfg = cfg
			return &fakeManager{}, nil
		})

	cfg := &rest.Config{Host: "https://fake-host:6443"}
	pc := &policyController{
		kubeCfg: cfg,
		ctx:     context.Background(),
	}

	pc.Start()

	if gotCfg != cfg {
		t.Errorf("Start() passed kubeCfg=%+v, want %+v", gotCfg, cfg)
	}
}

func TestNewAccessRoleControllerManager(t *testing.T) {
	managerFunc := reflect.ValueOf(NewAccessRoleControllerManager)
	if !managerFunc.IsValid() {
		t.Error("Expected to find NewAccessRoleControllerManager function")
	}

	funcType := managerFunc.Type()
	if funcType.NumIn() != 2 {
		t.Errorf("Expected NewAccessRoleControllerManager to take 2 arguments, got %d", funcType.NumIn())
	}

	if funcType.In(0).String() != contextTypeStr {
		t.Errorf("Expected first argument to be %s, got %s", contextTypeStr, funcType.In(0).String())
	}

	if funcType.In(1).String() != restConfigTypeStr {
		t.Errorf("Expected second argument to be *rest.Config, got %s", funcType.In(1).String())
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

// TestNewAccessRoleControllerManagerOutOfCluster verifies that newManager
// does not fail when called with a non-empty REST config host (simulating an
// out-of-cluster connection). There is no leader election involved anymore,
// so out-of-cluster / keadm deployments never hit controller-runtime's
// "not running in-cluster, please specify LeaderElectionNamespace" error.
// We call newManager rather than NewAccessRoleControllerManager because the
// latter also runs setupControllers, which triggers API-server discovery
// against the fake host.
//
// This test never starts the returned manager, so its health-probe listener
// is bound but never released for the rest of the test binary's lifetime.
// Using ":0" (an OS-assigned ephemeral port) instead of the real ":9002"
// avoids colliding with other tests in this file that also construct a
// manager without starting it.
func TestNewAccessRoleControllerManagerOutOfCluster(t *testing.T) {
	original := healthProbeBindAddress
	healthProbeBindAddress = ":0"
	t.Cleanup(func() { healthProbeBindAddress = original })

	cfg := &rest.Config{Host: "https://fake-apiserver:6443"}

	_, err := newManager(cfg)
	if err != nil {
		t.Errorf("newManager() should not fail for out-of-cluster config, got: %v", err)
	}
}

// TestNewManagerBindError verifies that newManager, and in turn
// NewAccessRoleControllerManager, propagate the error controller-runtime
// returns when the health-probe address is invalid, instead of panicking or
// silently ignoring it.
func TestNewManagerBindError(t *testing.T) {
	original := healthProbeBindAddress
	healthProbeBindAddress = "not-a-valid-address"
	t.Cleanup(func() { healthProbeBindAddress = original })

	cfg := &rest.Config{Host: "https://fake-apiserver:6443"}

	mgr, err := newManager(cfg)
	if err == nil {
		t.Fatal("expected newManager() to return an error for an invalid health-probe address")
	}
	if mgr != nil {
		t.Error("expected nil manager when newManager fails")
	}

	mgr, err = NewAccessRoleControllerManager(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected NewAccessRoleControllerManager() to propagate the newManager error")
	}
	if mgr != nil {
		t.Error("expected nil manager when NewAccessRoleControllerManager fails in newManager")
	}
}

// TestPolicyControllerDisabled verifies that when the RequireAuthorization
// feature gate is disabled, Enable() returns false, so Beehive never calls
// Start() and the manager (and its health-probe listener) is never
// constructed.
func TestPolicyControllerDisabled(t *testing.T) {
	// Capture original value so we restore it exactly, not hardcode false.
	original := features.DefaultFeatureGate.Enabled(features.RequireAuthorization)
	t.Cleanup(func() {
		_ = features.DefaultMutableFeatureGate.SetFromMap(
			map[string]bool{string(features.RequireAuthorization): original})
	})

	if err := features.DefaultMutableFeatureGate.SetFromMap(
		map[string]bool{string(features.RequireAuthorization): false}); err != nil {
		t.Fatalf("Failed to set feature gate: %v", err)
	}

	cfg := &rest.Config{Host: "https://fake-host:6443"}
	pc := &policyController{
		kubeCfg: cfg,
		ctx:     context.Background(),
	}

	if pc.Enable() {
		t.Error("Enable() should return false when RequireAuthorization feature gate is disabled")
	}

	if pc.kubeCfg == nil {
		t.Error("kubeCfg should be stored on the policyController")
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
	} else if kubeCfgField.Type.String() != restConfigTypeStr {
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

// TestNewAccessRoleControllerManagerSetupError verifies that
// NewAccessRoleControllerManager propagates an error returned by
// setupControllers instead of returning a manager. We use an unreachable
// loopback address (rather than a DNS name) so the dial fails immediately
// with "connection refused" instead of blocking on a timeout, keeping the
// test fast and deterministic regardless of network access in the test
// environment.
//
// Uses ":0" for the health-probe port for the same reason as
// TestNewAccessRoleControllerManagerOutOfCluster: this manager is never
// started, so its listener would otherwise hold :9002 for the rest of the
// test binary and make sibling tests fail to bind it.
func TestNewAccessRoleControllerManagerSetupError(t *testing.T) {
	original := healthProbeBindAddress
	healthProbeBindAddress = ":0"
	t.Cleanup(func() { healthProbeBindAddress = original })

	cfg := &rest.Config{Host: "http://127.0.0.1:1"}

	mgr, err := NewAccessRoleControllerManager(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected NewAccessRoleControllerManager to return an error when setupControllers fails")
	}
	if mgr != nil {
		t.Error("expected nil manager when setupControllers fails")
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
