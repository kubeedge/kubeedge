package synccontroller

import (
	"context"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/listers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// SyncController use beehive context message layer
type SyncController struct {
	enable bool
}

func newSyncController(enable bool) *SyncController {
	sctl := &SyncController{
		enable: enable,
	}
	// declared used informer
	_ = informers.GetGlobalInformers().Pod()
	_ = informers.GetGlobalInformers().ConfigMap()
	_ = informers.GetGlobalInformers().Secrets()
	_ = informers.GetGlobalInformers().Service()
	_ = informers.GetGlobalInformers().Endpoints()
	_ = informers.GetGlobalInformers().Node()
	_ = informers.GetGlobalInformers().Device()
	_ = informers.GetGlobalInformers().ClusterObjectSync()
	_ = informers.GetGlobalInformers().ObjectSync()
	informers.GetGlobalInformers().Node().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sctl.deleteObjectSyncs()
		},
	})

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
	if !cache.WaitForCacheSync(beehiveContext.Done(),
		informers.GetGlobalInformers().Pod().HasSynced,
		informers.GetGlobalInformers().ConfigMap().HasSynced,
		informers.GetGlobalInformers().Secrets().HasSynced,
		informers.GetGlobalInformers().Service().HasSynced,
		informers.GetGlobalInformers().Endpoints().HasSynced,
		informers.GetGlobalInformers().Node().HasSynced,
		informers.GetGlobalInformers().Device().HasSynced,
		informers.GetGlobalInformers().ClusterObjectSync().HasSynced,
		informers.GetGlobalInformers().ObjectSync().HasSynced,
	) {
		klog.Errorf("unable to sync caches for sync controller")
		return
	}

	go wait.Until(sctl.reconcile, 5*time.Second, beehiveContext.Done())

	sctl.deleteObjectSyncs()
}

func (sctl *SyncController) reconcile() {
	allClusterObjectSyncs, err := listers.GetListers().ClusterObjectSyncLister().List(labels.Everything())
	if err != nil {
		klog.Errorf("Filed to list all the ClusterObjectSyncs: %v", err)
	}
	sctl.manageClusterObjectSync(allClusterObjectSyncs)

	allObjectSyncs, err := listers.GetListers().ObjectSyncLister().List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all the ObjectSyncs: %v", err)
	}
	sctl.manageObjectSync(allObjectSyncs)

	sctl.manageCreateFailedObject()
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
		switch sync.Spec.ObjectKind {
		case model.ResourceTypePod:
			sctl.managePod(sync)
		case model.ResourceTypeConfigmap:
			sctl.manageConfigMap(sync)
		case model.ResourceTypeSecret:
			sctl.manageSecret(sync)
		case commonconst.ResourceTypeService:
			sctl.manageService(sync)
		case commonconst.ResourceTypeEndpoints:
			sctl.manageEndpoint(sync)
		// TODO: add device here
		default:
			klog.Errorf("Unsupported object kind: %v", sync.Spec.ObjectKind)
		}
	}
}

func (sctl *SyncController) deleteObjectSyncs() {
	syncs, err := listers.GetListers().ObjectSyncLister().List(labels.Everything())
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
			err = keclient.GetKubeEdgeClient().ObjectSync(sync.Namespace).Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0))
			if err != nil {
				klog.Errorf("failed to delete objectSync %s for edgenode %s, err: %v", sync.Name, nodeName, err)
			}
		}
	}
}

// checkObjectSync checks whether objectSync is outdated
func (sctl *SyncController) checkObjectSync(sync *v1alpha1.ObjectSync) (bool, error) {
	nodeName := getNodeName(sync.Name)
	_, err := listers.GetListers().NodeLister().Get(nodeName)
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

func isFromEdgeNode(nodes []*v1.Node, nodeName string) bool {
	for _, node := range nodes {
		if node.Name == nodeName {
			return true
		}
	}
	return false
}
