package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	edgeinformers "github.com/kubeedge/api/client/informers/externalversions"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

func main() {
	ctx, _ := context.WithCancel(context.Background())
	client.InitKubeEdgeClient(&v1alpha1.KubeAPIConfig{
		KubeConfig: "_tmp/kant-test-1year-kubeconfig.yaml",
	}, false)
	sharedInformerFactory := edgeinformers.NewSharedInformerFactory(client.GetCRDClient(), 0)
	_, err := sharedInformerFactory.Operations().V1alpha2().NodeUpgradeJobs().Informer().
		AddEventHandler(new(EventHandler))
	if err != nil {
		klog.Fatalf("failed to add event handler of node upgrade job, err: %v", err)
	}
	// Call start after the add event handler
	sharedInformerFactory.Start(ctx.Done())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

type EventHandler struct{}

func (h *EventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	if isInInitialList {
		return
	}
	klog.Infof("add the node upgrade job %v", obj)
}

func (h *EventHandler) OnUpdate(oldObj, newObj interface{}) {
	klog.Infof("update the node upgrade job %+v", newObj)
}

func (h *EventHandler) OnDelete(obj interface{}) {
	klog.Infof("delete the node upgrade job %+v", obj)
}
