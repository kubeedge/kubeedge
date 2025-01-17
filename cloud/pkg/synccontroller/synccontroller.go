package synccontroller

import (
	"context"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	reliablesyncslisters "github.com/kubeedge/api/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
)

const (
	// maxRetries is the number of times trying to delete ObjectSyncs and ClusterObjectSyncs.
	maxRetries       = 5
	deleteSyncsDelay = 1 * time.Second
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

	informerManager informers.Manager
}

var _ core.Module = (*SyncController)(nil)

func newSyncController(enable bool) *SyncController {
	var sctl = &SyncController{
		enable:          enable,
		crdclient:       keclient.GetCRDClient(),
		kubeclient:      keclient.GetDynamicClient(),
		informerManager: informers.GetInformersManager(),
	}
	// informer factory
	k8sInformerFactory := informers.GetInformersManager().GetKubeInformerFactory()
	crdInformerFactory := informers.GetInformersManager().GetKubeEdgeInformerFactory()

	objectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ObjectSyncs()
	clusterObjectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	nodesInformer := k8sInformerFactory.Core().V1().Nodes()
	_, err := nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sctl.deleteObjectSyncs()
			sctl.deleteClusterObjectSyncs()
		},
	})
	if err != nil {
		klog.Fatalf("new synccontroller failed, add event handler err: %v", err)
	}
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

// Enable of controller
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
	sctl.deleteClusterObjectSyncs()

	go wait.Until(sctl.reconcileObjectSyncs, 5*time.Second, beehiveContext.Done())

	go wait.Until(sctl.reconcileClusterObjectSyncs, 5*time.Second, beehiveContext.Done())
}

// reconcileObjectSyncs compare the version of the resource that has been sent to the
// edge recorded in objectSync with the version of the resource in k8s and generate a
// corresponding event to send to the edge according to the comparison result
func (sctl *SyncController) reconcileObjectSyncs() {
	allObjectSyncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ObjectSyncs: %v", err)
	}

	for _, sync := range allObjectSyncs {
		sctl.reconcileObjectSync(sync)
	}
}

// reconcileClusterObjectSyncs compare the version of the resource that has been sent
// to the edge recorded in ClusterObjectSync with the version of the resource in k8s and
// generate a corresponding event to send to the edge according to the comparison result
func (sctl *SyncController) reconcileClusterObjectSyncs() {
	allClusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ClusterObjectSyncs: %v", err)
	}

	for _, sync := range allClusterObjectSyncs {
		sctl.reconcileClusterObjectSync(sync)
	}
}

func (sctl *SyncController) deleteObjectSyncs() {
	syncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ObjectSyncs: %v", err)
	}
	for _, sync := range syncs {
		// If an error occurs while deleting ObjectSyncs, will retry.
		err = retry.Do(
			func() error {
				nodeName := getNodeName(sync.Name)
				isGarbage, err := sctl.checkObjectSync(sync)
				if err != nil {
					klog.Warningf("failed to check ObjectSync outdated, %s", err)
					return err
				}
				if isGarbage {
					klog.Infof("ObjectSync %s will be deleted since node %s has been deleted", sync.Name, nodeName)
					err = sctl.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(sync.Namespace).Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0))
					if err != nil {
						klog.Warningf("failed to delete objectSync %s for edgenode %s, err: %v", sync.Name, nodeName, err)
						return err
					}
				}
				return nil
			},
			retry.Delay(deleteSyncsDelay),
			retry.Attempts(maxRetries),
		)
		if err != nil {
			klog.Errorf("failed to delete objectSync %s, err: %v", sync.Name, err)
		}
	}
}

func (sctl *SyncController) deleteClusterObjectSyncs() {
	syncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the clusterObjectSync: %v", err)
	}
	for _, sync := range syncs {
		// If an error occurs while deleting ClusterObjectSyncs, will retry.
		err = retry.Do(
			func() error {
				nodeName := getNodeName(sync.Name)
				isGarbage, err := sctl.checkClusterObjectSync(sync)
				if err != nil {
					klog.Warningf("failed to check ClusterObjectSync outdated, %s", err)
					return err
				}
				if isGarbage {
					klog.Infof("ClusterObjectSync %s will be deleted since node %s has been deleted", sync.Name, nodeName)
					err = sctl.crdclient.ReliablesyncsV1alpha1().ClusterObjectSyncs().Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0))
					if err != nil {
						klog.Warningf("failed to delete ClusterObjectSync %s for edgenode %s, err: %v", sync.Name, nodeName, err)
						return err
					}
				}
				return nil
			},
			retry.Delay(deleteSyncsDelay),
			retry.Attempts(maxRetries),
		)
		if err != nil {
			klog.Errorf("failed to delete ClusterObjectSync %s, err: %v", sync.Name, err)
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

// checkClusterObjectSync checks whether ClusterObjectSync is outdated
func (sctl *SyncController) checkClusterObjectSync(sync *v1alpha1.ClusterObjectSync) (bool, error) {
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
