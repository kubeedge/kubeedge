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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	controllerruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	pm "github.com/kubeedge/kubeedge/cloud/pkg/policycontroller/manager"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
)

// healthProbeBindAddress is the TCP address that the policy-controller
// manager's health-probe HTTP server listens on.
//
// There is no leader election here: every CloudCore replica runs its own
// policy controller and its own health endpoint. Each CloudCore instance
// only holds live CloudHub sessions for the edge nodes routed to it (session
// affinity is handled by the Service in front of CloudCore), so a single
// elected leader would be unable to deliver policy updates to edge nodes
// connected to the other replicas. Instead, every replica reconciles, and
// Controller.send2Edge (in the manager package) narrows delivery to the
// edge nodes actually managed by the local instance.
//
// This is a var (not a const) purely so tests that construct a real manager
// without starting it can point it at an OS-assigned ephemeral port (":0")
// instead of colliding with each other on :9002 within the same test binary.
var healthProbeBindAddress = ":9002"

// policyController use beehive context message layer
type policyController struct {
	// kubeCfg is the REST client config used to build the controller-runtime
	// manager. It is stored here so that Register() stays side-effect-free and
	// the manager is only constructed when Start() is called, i.e. when the
	// module is actually enabled.
	kubeCfg *rest.Config
	ctx     context.Context
}

var _ core.Module = (*policyController)(nil)

var accessScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(accessScheme))
	utilruntime.Must(policyv1alpha1.AddToScheme(accessScheme))
}

// newManager constructs and configures a controller-runtime manager but does
// NOT register any controllers. It is the building block used by
// NewAccessRoleControllerManager and is also called directly by tests that
// need to verify manager construction without dialling a real API server
// (controller registration is what triggers the API-discovery round trip).
func newManager(kubeCfg *rest.Config) (manager.Manager, error) {
	controllerManager, err := controllerruntime.NewManager(kubeCfg, controllerruntime.Options{
		Scheme: accessScheme,
		Metrics: controllerruntimemetrics.Options{
			SecureServing: false,
			BindAddress:   "0",
		}, // disable metrics
		HealthProbeBindAddress: healthProbeBindAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %w", err)
	}

	// healthz.Ping is the standard no-op checker provided by controller-runtime;
	// it always returns nil, signalling that the process is alive. This is a
	// process-level liveness signal only — it does not verify cache sync,
	// controller startup, or CloudHub session state.
	if err := controllerManager.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add healthz check: %w", err)
	}
	if err := controllerManager.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add readyz check: %w", err)
	}

	return controllerManager, nil
}

// NewAccessRoleControllerManager creates a controller-runtime manager for the
// policy controller and registers all controllers with it.
func NewAccessRoleControllerManager(ctx context.Context, kubeCfg *rest.Config) (manager.Manager, error) {
	controllerManager, err := newManager(kubeCfg)
	if err != nil {
		return nil, err
	}
	if err := setupControllers(ctx, controllerManager); err != nil {
		return nil, err
	}
	return controllerManager, nil
}

func setupControllers(ctx context.Context, mgr manager.Manager) error {
	// mgr.GetClient() will directly acquire the unstructured objects from API Server which
	// have not be registered in the accessScheme.
	pc := &pm.Controller{
		Client:       mgr.GetClient(),
		Reader:       mgr.GetAPIReader(),
		MessageLayer: messagelayer.PolicyControllerMessageLayer(),
	}

	klog.Info("setup policy controller")
	if err := pc.SetupWithManager(ctx, mgr); err != nil {
		return fmt.Errorf("failed to setup policy controller: %w", err)
	}
	return nil
}

// Register stores the REST config on the policyController and registers the
// module with the Beehive runtime. The controller-runtime manager is NOT
// constructed here; that happens in Start(), which Beehive only calls when
// Enable() returns true. This ensures the health-probe listener is never
// bound when the RequireAuthorization feature gate is disabled.
//
// Every CloudCore replica calls Register and runs the policy controller —
// see the healthProbeBindAddress doc comment for why there is no leader
// election.
func Register(kubeCfg *rest.Config) {
	pc := &policyController{
		kubeCfg: kubeCfg,
		ctx:     beehiveContext.GetContext(),
	}
	core.Register(pc)
}

// Name of controller
func (pc *policyController) Name() string {
	return modules.PolicyControllerModuleName
}

// Group of controller
func (pc *policyController) Group() string {
	return modules.PolicyControllerGroupName
}

// Enable indicates whether enable this module
func (pc *policyController) Enable() bool {
	return kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization)
}

// RestartPolicy returns nil to use the default restart policy.
func (pc *policyController) RestartPolicy() *core.ModuleRestartPolicy {
	return nil
}

// Start creates the controller-runtime manager and runs it. Beehive only
// calls Start() when Enable() returns true, so the health-probe listener is
// only bound when the policy controller module is genuinely enabled.
//
// mgr.Start blocks until the manager has stopped.
func (pc *policyController) Start() {
	mgr, err := NewAccessRoleControllerManager(pc.ctx, pc.kubeCfg)
	if err != nil {
		klog.Fatalf("failed to create controller manager, %v", err)
	}
	if err := mgr.Start(pc.ctx); err != nil {
		klog.Fatalf("failed to start controller manager, %v", err)
	}
}
