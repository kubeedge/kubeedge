package synccontroller

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	reliablesyncslisters "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// SyncController use beehive context message layer
type SyncController struct {
	enable bool
	//client
	crdclient crdClientset.Interface
	// lister
	nodeLister              corelisters.NodeLister
	objectSyncLister        reliablesyncslisters.ObjectSyncLister
	clusterObjectSyncLister reliablesyncslisters.ClusterObjectSyncLister

	kubeclient dynamic.Interface

	informersSyncedFuncs []cache.InformerSynced
}

var _ core.Module = (*SyncController)(nil)

func newSyncController(enable bool) *SyncController {
	var sctl = &SyncController{
		enable:     enable,
		crdclient:  keclient.GetCRDClient(),
		kubeclient: keclient.GetDynamicClient(),
	}
	// informer factory
	k8sInformerFactory := informers.GetInformersManager().GetK8sInformerFactory()
	crdInformerFactory := informers.GetInformersManager().GetCRDInformerFactory()

	objectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ObjectSyncs()
	clusterObjectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	nodesInformer := k8sInformerFactory.Core().V1().Nodes()
	nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sctl.deleteObjectSyncs()
		},
	})
	// lister
	sctl.nodeLister = nodesInformer.Lister()

	sctl.objectSyncLister = objectSyncsInformer.Lister()
	sctl.clusterObjectSyncLister = clusterObjectSyncsInformer.Lister()
	// InformerSynced
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, objectSyncsInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, clusterObjectSyncsInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, nodesInformer.Informer().HasSynced)

	return sctl
}

func Register(ec *configv1alpha1.SyncController) {
	config.InitConfigure(ec)
	core.Register(newSyncController(ec.Enable))
}

// Name of controller
func (sctl *SyncController) Name() string {
	return modules.SyncControllerModuleName
}

// Group of controller
func (sctl *SyncController) Group() string {
	return modules.SyncControllerModuleGroup
}

// Group of controller
func (sctl *SyncController) Enable() bool {
	return sctl.enable
}

// Start controller
func (sctl *SyncController) Start() {
	if !cache.WaitForCacheSync(beehiveContext.Done(), sctl.informersSyncedFuncs...) {
		klog.Errorf("unable to sync caches for sync controller")
		return
	}

	sctl.deleteObjectSyncs() //check outdate sync before start to reconcile
	go wait.Until(sctl.reconcile, 5*time.Second, beehiveContext.Done())
}

func (sctl *SyncController) reconcile() {
	allClusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ClusterObjectSyncs: %v", err)
	}
	sctl.manageClusterObjectSync(allClusterObjectSyncs)

	allObjectSyncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ObjectSyncs: %v", err)
	}
	sctl.manageObjectSync(allObjectSyncs)
}

// Compare the cluster scope objects that have been persisted to the edge with the cluster scope objects in K8s,
// and generate update and delete events to the edge
func (sctl *SyncController) manageClusterObjectSync(syncs []*v1alpha1.ClusterObjectSync) {
	// TODO: Handle cluster scope resource
}

// Compare the namespace scope objects that have been persisted to the edge with the namespace scope objects in K8s,
// and generate update and delete events to the edge
func (sctl *SyncController) manageObjectSync(syncs []*v1alpha1.ObjectSync) {
	for _, sync := range syncs {
		sctl.manageObject(sync)
	}
}

func (sctl *SyncController) deleteObjectSyncs() {
	syncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ObjectSyncs: %v", err)
	}
	for _, sync := range syncs {
		nodeName := getNodeName(sync.Name)
		isGarbage, err := sctl.checkObjectSync(sync)
		if err != nil {
			klog.Errorf("failed to check ObjectSync outdated, %s", err)
		}
		if isGarbage {
			klog.Infof("ObjectSync %s will be deleted since node %s has been deleted", sync.Name, nodeName)
			err = sctl.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(sync.Namespace).Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0))
			if err != nil {
				klog.Errorf("failed to delete objectSync %s for edgenode %s, err: %v", sync.Name, nodeName, err)
			}
		}
	}
}

// checkObjectSync checks whether objectSync is outdated
func (sctl *SyncController) checkObjectSync(sync *v1alpha1.ObjectSync) (bool, error) {
	nodeName := getNodeName(sync.Name)
	_, err := sctl.nodeLister.Get(nodeName)
	if errors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

// BuildObjectSyncName builds the name of objectSync/clusterObjectSync
func BuildObjectSyncName(nodeName, UID string) string {
	return nodeName + "." + UID
}

func getNodeName(syncName string) string {
	tmps := strings.Split(syncName, ".")
	return strings.Join(tmps[:len(tmps)-1], ".")
}

func getObjectUID(syncName string) string {
	tmps := strings.Split(syncName, ".")
	return tmps[len(tmps)-1]
}
