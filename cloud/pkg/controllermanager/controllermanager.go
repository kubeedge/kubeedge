package controllermanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodetask"
	"github.com/kubeedge/kubeedge/pkg/features"
)

var kubeedgeScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(kubeedgeScheme))
	utilruntime.Must(appsv1alpha1.AddToScheme(kubeedgeScheme))
	utilruntime.Must(operationsv1alpha2.AddToScheme(kubeedgeScheme))
}

type Controller interface {
	SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error
	reconcile.Reconciler
}

func NewControllerManager(ctx context.Context, kubeCfg *rest.Config, healthProbe string,
) (manager.Manager, error) {
	const nothingCheckName = "nothing"
	mgr, err := controllerruntime.NewManager(kubeCfg, controllerruntime.Options{
		Scheme:                 kubeedgeScheme,
		HealthProbeBindAddress: healthProbe,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager, err: %v", err)
	}

	if err := mgr.AddHealthzCheck(nothingCheckName, func(_ *http.Request) error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to add healthz check, err: %v", err)
	}
	if err := mgr.AddReadyzCheck(nothingCheckName, func(_ *http.Request) error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to add readyz check, err: %v", err)
	}

	che, err := newAndStartCache(ctx, kubeCfg)
	if err != nil {
		return nil, err
	}

	if err := setupControllers(ctx, mgr, che); err != nil {
		return nil, err
	}
	return mgr, nil
}

func newAndStartCache(ctx context.Context, kubeCfg *rest.Config,
) (che cache.Cache, err error) {
	che, err = cache.New(kubeCfg, cache.Options{
		// Register resources that need to be cached.
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Node{}: {},
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to create the cache, err: %v", err)
		return
	}
	go func() {
		err = che.Start(ctx)
	}()
	synced := che.WaitForCacheSync(ctx)
	if err != nil {
		err = fmt.Errorf("failed to start the cache, err: %v", err)
		return
	}
	if !synced {
		err = errors.New("could not sync the cache")
		return
	}
	return
}

func setupControllers(ctx context.Context, mgr manager.Manager, che cache.Cache) error {
	serializer := json.NewSerializerWithOptions(json.DefaultMetaFactory,
		kubeedgeScheme, kubeedgeScheme, json.SerializerOptions{Yaml: true})
	cli := mgr.GetClient()

	ctls := []Controller{
		nodegroup.NewController(cli),
		edgeapplication.NewController(ctx, cli, serializer, mgr),
	}
	if !features.DefaultFeatureGate.Enabled(features.DisableNodeTaskV1alpha2) {
		ctls = append(ctls, nodetask.NewImagePrePullJobController(cli, che))
		ctls = append(ctls, nodetask.NewConfigUpdateJobController(cli, che))
	} else {
		klog.V(1).Info("disabled the node task v1alpha2")
	}

	klog.V(1).Info("start setup controllers")
	for i := range ctls {
		ctl := ctls[i]
		if err := ctl.SetupWithManager(ctx, mgr); err != nil {
			return fmt.Errorf("failed to setup %T controller, err: %v", ctl, err)
		}
	}
	return nil
}
