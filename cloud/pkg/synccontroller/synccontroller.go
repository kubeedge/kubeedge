package synccontroller

import (
	"context"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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
	// maxRetries is the number of times a node's sync-record garbage collection is
	// retried before it is dropped and left to the periodic backstop.
	maxRetries = 5
	// nodeGCResyncPeriod is how often all sync records are scanned to re-enqueue
	// nodes that are gone but still have sync records (e.g. a delete that exhausted
	// its retries, or a node deletion missed while the controller was down).
	nodeGCResyncPeriod = 5 * time.Minute
	// nodeGCQueueName names the workqueue used for node sync-record GC.
	nodeGCQueueName = "synccontroller_node_gc"
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

	// nodeGCQueue holds the names of deleted nodes whose ObjectSync and
	// ClusterObjectSync records need to be garbage collected. A workqueue keeps
	// the GC off the node informer goroutine, coalesces duplicate node deletions,
	// and provides cancellable, rate-limited retries.
	nodeGCQueue workqueue.RateLimitingInterface
}

var _ core.Module = (*SyncController)(nil)

func newSyncController(enable bool) *SyncController {
	var sctl = &SyncController{
		enable:          enable,
		crdclient:       keclient.GetCRDClient(),
		kubeclient:      keclient.GetDynamicClient(),
		informerManager: informers.GetInformersManager(),
		nodeGCQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeGCQueueName),
	}
	// informer factory
	k8sInformerFactory := informers.GetInformersManager().GetKubeInformerFactory()
	crdInformerFactory := informers.GetInformersManager().GetKubeEdgeInformerFactory()

	objectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ObjectSyncs()
	clusterObjectSyncsInformer := crdInformerFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	nodesInformer := k8sInformerFactory.Core().V1().Nodes()
	_, err := nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: sctl.enqueueNodeForGC,
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

func (sctl *SyncController) RestartPolicy() *core.ModuleRestartPolicy {
	return nil
}

// Start controller
func (sctl *SyncController) Start() {
	if !cache.WaitForCacheSync(beehiveContext.Done(), sctl.informersSyncedFuncs...) {
		klog.Errorf("unable to sync caches for sync controller")
		return
	}

	// Shut the GC queue down when the module stops so the worker goroutine exits.
	go func() {
		<-beehiveContext.Done()
		sctl.nodeGCQueue.ShutDown()
	}()

	// Garbage collect sync records for deleted nodes off the informer goroutine.
	go wait.Until(sctl.runNodeGCWorker, time.Second, beehiveContext.Done())
	// Periodic backstop: re-enqueue nodes that are gone but still have sync records,
	// so retry-exhausted or missed deletions are eventually reclaimed. Its first run
	// also cleans up records left behind while the controller was down.
	go wait.Until(sctl.enqueueOrphanedNodeSyncs, nodeGCResyncPeriod, beehiveContext.Done())

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

// enqueueNodeForGC queues a deleted node's name so its sync records are garbage
// collected by the worker instead of on the informer goroutine.
func (sctl *SyncController) enqueueNodeForGC(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("cannot convert to Node, unexpected object type: %T", obj)
			return
		}
		node, ok = tombstone.Obj.(*v1.Node)
		if !ok {
			klog.Errorf("cannot convert tombstone to Node, unexpected object type: %T", tombstone.Obj)
			return
		}
	}
	sctl.nodeGCQueue.Add(node.Name)
}

func (sctl *SyncController) runNodeGCWorker() {
	for sctl.processNextNodeGC() {
	}
}

func (sctl *SyncController) processNextNodeGC() bool {
	key, quit := sctl.nodeGCQueue.Get()
	if quit {
		return false
	}
	defer sctl.nodeGCQueue.Done(key)

	nodeName := key.(string)
	if err := sctl.gcNodeSyncs(nodeName); err != nil {
		if sctl.nodeGCQueue.NumRequeues(key) < maxRetries {
			klog.Warningf("retrying GC of sync records for node %s: %v", nodeName, err)
			sctl.nodeGCQueue.AddRateLimited(key)
			return true
		}
		klog.Errorf("dropping GC of sync records for node %s after %d retries: %v", nodeName, maxRetries, err)
	}
	sctl.nodeGCQueue.Forget(key)
	return true
}

// gcNodeSyncs deletes the ObjectSync and ClusterObjectSync records that belong to
// nodeName once that node no longer exists.
func (sctl *SyncController) gcNodeSyncs(nodeName string) error {
	var errs []error

	objectSyncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, sync := range objectSyncs {
		if getNodeName(sync.Name) != nodeName {
			continue
		}
		isGarbage, err := sctl.checkObjectSync(sync)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !isGarbage {
			continue
		}
		klog.Infof("ObjectSync %s will be deleted since node %s has been deleted", sync.Name, nodeName)
		if err := sctl.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(sync.Namespace).Delete(context.TODO(), sync.Name, *metav1.NewDeleteOptions(0)); err != nil && !errors.IsNotFound(err) {
			errs = append(errs, err)
		}
	}

	clusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, sync := range clusterObjectSyncs {
		if getNodeName(sync.Name) != nodeName {
			continue
		}
		isGarbage, err := sctl.checkClusterObjectSync(sync)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !isGarbage {
			continue
		}
		klog.Infof("ClusterObjectSync %s will be deleted since node %s has been deleted", sync.Name, nodeName)
		if err := sctl.crdclient.ReliablesyncsV1alpha1().ClusterObjectSyncs().Delete(context.TODO(), sync.Name, *metav1.NewDeleteOptions(0)); err != nil && !errors.IsNotFound(err) {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// enqueueOrphanedNodeSyncs scans all sync records and enqueues the nodes that no
// longer exist so their records are garbage collected. It is the periodic backstop
// for deletions that were missed or that exhausted their retries.
func (sctl *SyncController) enqueueOrphanedNodeSyncs() {
	nodeNames := make(map[string]struct{})

	objectSyncs, err := sctl.objectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ObjectSyncs for node GC: %v", err)
		return
	}
	for _, sync := range objectSyncs {
		nodeNames[getNodeName(sync.Name)] = struct{}{}
	}

	clusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ClusterObjectSyncs for node GC: %v", err)
		return
	}
	for _, sync := range clusterObjectSyncs {
		nodeNames[getNodeName(sync.Name)] = struct{}{}
	}

	for nodeName := range nodeNames {
		if _, err := sctl.nodeLister.Get(nodeName); errors.IsNotFound(err) {
			sctl.nodeGCQueue.Add(nodeName)
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
