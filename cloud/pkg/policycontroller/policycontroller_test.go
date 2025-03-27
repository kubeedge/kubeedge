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

	cfg := &rest.Config{Host: "https://localhost:8080"}

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

	_, _ = NewAccessRoleControllerManager(pc.ctx, cfg)

	moduleType := reflect.TypeOf((*core.Module)(nil)).Elem()
	if !reflect.TypeOf(pc).Implements(moduleType) {
		t.Error("policyController should implement core.Module")
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
	if funcType.NumIn() != 2 {
		t.Errorf("Expected NewAccessRoleControllerManager to take 2 arguments, got %d", funcType.NumIn())
	}

	if funcType.In(0).String() != contextTypeStr {
		t.Errorf("Expected first argument to be %s, got %s", contextTypeStr, funcType.In(0).String())
	}

	if funcType.In(1).String() != "*rest.Config" {
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

	managerField, exists := pcType.FieldByName("manager")
	if !exists {
		t.Error("Expected policyController to have manager field")
	} else if managerField.Type.String() != managerTypeStr {
		t.Errorf("Expected manager field to be %s, got %s", managerTypeStr, managerField.Type.String())
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
