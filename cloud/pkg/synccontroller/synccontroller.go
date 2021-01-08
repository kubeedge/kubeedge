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
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	devicesv1alpha2listers "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/devices/v1alpha2"
	reliablesyncsv1alpha1listers "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// SyncController use beehive context message layer
type SyncController struct {
	enable bool
	//client
	crdclient crdClientset.Interface
	// lister
	podLister               corev1listers.PodLister
	configMapLister         corev1listers.ConfigMapLister
	secretLister            corev1listers.SecretLister
	seviceLister            corev1listers.ServiceLister
	endpointsLister         corev1listers.EndpointsLister
	nodeLister              corev1listers.NodeLister
	objectSyncLister        reliablesyncsv1alpha1listers.ObjectSyncLister
	clusterObjectSyncLister reliablesyncsv1alpha1listers.ClusterObjectSyncLister
	deviceLister            devicesv1alpha2listers.DeviceLister
	informersSyncedFuncs    []cache.InformerSynced
}

func newSyncController(enable bool) *SyncController {
	var sctl = &SyncController{
		enable:    enable,
		crdclient: keclient.GetKubeEdgeClient(),
	}
	// informer factory
	k8sInformerFactory := informers.GetInformersManager().GetK8sInformerFactory()
	crdInformerFactory := informers.GetInformersManager().GetCRDInformerFactory()
	// informer
	podInformer := k8sInformerFactory.Core().V1().Pods()
	configMapInformer := k8sInformerFactory.Core().V1().ConfigMaps()
	secretInformer := k8sInformerFactory.Core().V1().Secrets()
	serviceInformer := k8sInformerFactory.Core().V1().Services()
	endpointInformer := k8sInformerFactory.Core().V1().Endpoints()
	devicesInformer := crdInformerFactory.Devices().V1alpha2().Devices()
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
	sctl.podLister = podInformer.Lister()
	sctl.configMapLister = configMapInformer.Lister()
	sctl.secretLister = secretInformer.Lister()
	sctl.seviceLister = serviceInformer.Lister()
	sctl.endpointsLister = endpointInformer.Lister()
	sctl.deviceLister = devicesInformer.Lister()
	sctl.objectSyncLister = objectSyncsInformer.Lister()
	sctl.clusterObjectSyncLister = clusterObjectSyncsInformer.Lister()
	// InformerSynced
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, podInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, configMapInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, secretInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, serviceInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, endpointInformer.Informer().HasSynced)
	sctl.informersSyncedFuncs = append(sctl.informersSyncedFuncs, devicesInformer.Informer().HasSynced)
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

	go wait.Until(sctl.reconcile, 5*time.Second, beehiveContext.Done())

	sctl.deleteObjectSyncs()
}

func (sctl *SyncController) reconcile() {
	allClusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Filed to list all the ClusterObjectSyncs: %v", err)
	}
	sctl.manageClusterObjectSync(allClusterObjectSyncs)

	allObjectSyncs, err := sctl.objectSyncLister.List(labels.Everything())
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

func isFromEdgeNode(nodes []*v1.Node, nodeName string) bool {
	for _, node := range nodes {
		if node.Name == nodeName {
			return true
		}
	}
	return false
}
