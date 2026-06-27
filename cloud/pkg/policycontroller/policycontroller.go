/*
Copyright 2024 The KubeEdge Authors.

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

// Options holds the optional settings for the policy controller manager.
// Every field is safe to leave at its zero value:
//   - LeaderElection=false disables leader election entirely (safe for keadm /
//     standalone deployments where in-cluster credentials are absent).
//   - LeaderElectionNamespace="" is only used when LeaderElection=true; an
//     explicit, non-empty value avoids the in-cluster namespace auto-detection
//     that would fail outside a Kubernetes Pod.
//   - HealthProbeBindAddress="" disables the health-probe HTTP listener.
type Options struct {
	// LeaderElection enables leader election so that at most one replica
	// runs the reconciliation loop at a time.  Should be true only when
	// CloudCore is deployed as a Kubernetes workload.
	LeaderElection bool

	// LeaderElectionNamespace is the namespace that holds the leader-election
	// Lease object.  Must be set explicitly when LeaderElection is true and
	// CloudCore is running outside a Kubernetes Pod (i.e. via keadm or as a
	// standalone binary), because controller-runtime cannot auto-detect the
	// namespace in that case.
	LeaderElectionNamespace string

	// HealthProbeBindAddress is the TCP address that the health-probe HTTP
	// server listens on (e.g. ":9002").  An empty string disables the
	// listener.
	HealthProbeBindAddress string
}

// policyController use beehive context message layer
type policyController struct {
	// kubeCfg is the REST client config used to build the controller-runtime
	// manager.  It is stored here so that Register() stays side-effect-free
	// and the manager is only constructed when Start() is called (i.e. when
	// the module is actually enabled).
	kubeCfg *rest.Config
	ctx     context.Context
}

var _ core.Module = (*policyController)(nil)

var accessScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(accessScheme))
	utilruntime.Must(policyv1alpha1.AddToScheme(accessScheme))
}

// newManager constructs and configures a controller-runtime manager according
// to opts but does NOT register any controllers.  It is the building block
// used by NewAccessRoleControllerManager and is also called directly by tests
// that need to verify manager-option handling without dialling a real API
// server (controller registration is what triggers the API-discovery round
// trip).
func newManager(kubeCfg *rest.Config, opts Options) (manager.Manager, error) {
	mgrOpts := controllerruntime.Options{
		Scheme: accessScheme,
		Metrics: controllerruntimemetrics.Options{
			SecureServing: false,
			BindAddress:   "0",
		}, // disable metrics
		HealthProbeBindAddress: opts.HealthProbeBindAddress,
	}

	if opts.LeaderElection {
		mgrOpts.LeaderElection          = true
		mgrOpts.LeaderElectionID        = "policy-controller.kubeedge.io"
		mgrOpts.LeaderElectionNamespace = opts.LeaderElectionNamespace
	}

	controllerManager, err := controllerruntime.NewManager(kubeCfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %w", err)
	}

	// healthz.Ping is the standard no-op checker provided by controller-runtime;
	// it always returns nil, signalling that the process is alive.
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
//
// Leader election and the health-probe listener are both opt-in via opts so
// that callers running outside a Kubernetes cluster (keadm, standalone binary)
// can pass LeaderElection=false and HealthProbeBindAddress="" without hitting
// the "not running in-cluster, please specify LeaderElectionNamespace" error
// that controller-runtime returns when it cannot auto-detect the namespace.
func NewAccessRoleControllerManager(ctx context.Context, kubeCfg *rest.Config, opts Options) (manager.Manager, error) {
	controllerManager, err := newManager(kubeCfg, opts)
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
// module with the Beehive runtime.  The controller-runtime manager is NOT
// constructed here; that happens in Start() which Beehive only calls when
// Enable() returns true.  This ensures that leader election and the
// health-probe listener are never activated when the RequireAuthorization
// feature gate is disabled, and that keadm / standalone deployments do not
// fail during registration.
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

// Start creates the controller-runtime manager and runs it.  Beehive only
// calls Start() when Enable() returns true, so all leader-election and
// health-probe machinery is activated only when the policy controller module
// is genuinely enabled.
//
// Leader election is enabled with LeaderElectionNamespace set to "kubeedge"
// (the CloudCore system namespace), which works for both in-cluster Pod
// deployments and standalone binaries without triggering the auto-detection
// path that requires the in-cluster service-account namespace file.
//
// mgr.Start blocks until the manager has stopped.
func (pc *policyController) Start() {
	mgr, err := NewAccessRoleControllerManager(pc.ctx, pc.kubeCfg, Options{
		LeaderElection:          true,
		LeaderElectionNamespace: "kubeedge",
		HealthProbeBindAddress:  ":9002",
	})
	if err != nil {
		klog.Fatalf("failed to create controller manager, %v", err)
	}
	// mgr.Start blocks until the manager has stopped
	if err := mgr.Start(pc.ctx); err != nil {
		klog.Fatalf("failed to start controller manager, %v", err)
	}
}
