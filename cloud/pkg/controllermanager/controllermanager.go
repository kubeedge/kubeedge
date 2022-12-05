package controllermanager

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/statusmanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

var appsScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(appsScheme))
	utilruntime.Must(appsv1alpha1.AddToScheme(appsScheme))
}

func NewAppsControllerManager(ctx context.Context) (manager.Manager, error) {
	kubeCfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig, %v", err)
	}
	controllerManager, err := controllerruntime.NewManager(kubeCfg, controllerruntime.Options{
		Scheme: appsScheme,
		// TODO: leader election
		// TODO: /healthz

	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager, %v", err)
	}

	if err := setupControllers(ctx, controllerManager); err != nil {
		return nil, err
	}
	return controllerManager, nil
}

func setupControllers(ctx context.Context, mgr manager.Manager) error {
	Serializer := json.NewSerializerWithOptions(json.DefaultMetaFactory, appsScheme, appsScheme, json.SerializerOptions{Yaml: true})
	// TODO: add cacheReader for unstructured
	// This returned cli will directly acquire the unstructured objects from API Server which
	// have not be registered in the appsScheme. Currently, we only support deployment in
	// EdgeApplication, so there's no problem. We have to add cacheReader for unstructured
	// obj if we want to support more types, such as CRDs.
	cli := mgr.GetClient()
	nodeGroupController := &nodegroup.Controller{
		Client: cli,
	}

	edgeApplicationControllere := &edgeapplication.Controller{
		Client:        cli,
		Serializer:    Serializer,
		StatusManager: statusmanager.NewStatusManager(ctx, mgr, cli, Serializer),
		Overrider: &overridemanager.OverrideManager{
			Overriders: []overridemanager.Overrider{
				&overridemanager.NameOverrider{},
				&overridemanager.ReplicasOverrider{},
				&overridemanager.ImageOverrider{},
				&overridemanager.NodeSelectorOverrider{},
			},
		},
	}

	klog.Info("setup nodegroup controller")
	if err := nodeGroupController.SetupWithManager(ctx, mgr); err != nil {
		return fmt.Errorf("failed to setup nodegroup controller, %v", err)
	}
	if err := edgeApplicationControllere.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to setup edgeapplication controller, %v", err)
	}
	return nil
}
