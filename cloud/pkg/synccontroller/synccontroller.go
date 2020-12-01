package synccontroller

import (
	"context"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	crdinformerfactory "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions"
	deviceinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/devices/v1alpha2"
	syncinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/reliablesyncs/v1alpha1"
	devicelister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/devices/v1alpha2"
	synclister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// SyncController use beehive context message layer
type SyncController struct {
	enable bool

	// informer
	podInformer               coreinformers.PodInformer
	configMapInformer         coreinformers.ConfigMapInformer
	secretInformer            coreinformers.SecretInformer
	serviceInformer           coreinformers.ServiceInformer
	endpointInformer          coreinformers.EndpointsInformer
	nodeInformer              coreinformers.NodeInformer
	deviceInformer            deviceinformer.DeviceInformer
	clusterObjectSyncInformer syncinformer.ClusterObjectSyncInformer
	objectSyncInformer        syncinformer.ObjectSyncInformer

	// synced
	podSynced               cache.InformerSynced
	configMapSynced         cache.InformerSynced
	secretSynced            cache.InformerSynced
	serviceSynced           cache.InformerSynced
	endpointSynced          cache.InformerSynced
	nodeSynced              cache.InformerSynced
	deviceSynced            cache.InformerSynced
	clusterObjectSyncSynced cache.InformerSynced
	objectSyncSynced        cache.InformerSynced

	// lister
	podLister               corelisters.PodLister
	configMapLister         corelisters.ConfigMapLister
	secretLister            corelisters.SecretLister
	serviceLister           corelisters.ServiceLister
	endpointLister          corelisters.EndpointsLister
	nodeLister              corelisters.NodeLister
	deviceLister            devicelister.DeviceLister
	clusterObjectSyncLister synclister.ClusterObjectSyncLister
	objectSyncLister        synclister.ObjectSyncLister

	// client
	crdClient *versioned.Clientset
}

func newSyncController(enable bool) *SyncController {
	config, err := buildConfig()
	if err != nil {
		klog.Errorf("Failed to build config, err: %v", err)
		os.Exit(1)
	}
	kubeClient := kubernetes.NewForConfigOrDie(config)
	crdClient := versioned.NewForConfigOrDie(config)

	kubeSharedInformers := informers.NewSharedInformerFactory(kubeClient, 0)
	crdFactory := crdinformerfactory.NewSharedInformerFactory(crdClient, 0)

	podInformer := kubeSharedInformers.Core().V1().Pods()
	configMapInformer := kubeSharedInformers.Core().V1().ConfigMaps()
	secretInformer := kubeSharedInformers.Core().V1().Secrets()
	serviceInformer := kubeSharedInformers.Core().V1().Services()
	endpointInformer := kubeSharedInformers.Core().V1().Endpoints()
	nodeInformer := kubeSharedInformers.Core().V1().Nodes()
	deviceInformer := crdFactory.Devices().V1alpha2().Devices()
	clusterObjectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	objectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ObjectSyncs()

	sctl := &SyncController{
		enable: enable,

		podInformer:               podInformer,
		configMapInformer:         configMapInformer,
		secretInformer:            secretInformer,
		serviceInformer:           serviceInformer,
		endpointInformer:          endpointInformer,
		nodeInformer:              nodeInformer,
		deviceInformer:            deviceInformer,
		clusterObjectSyncInformer: clusterObjectSyncInformer,
		objectSyncInformer:        objectSyncInformer,

		podSynced:               podInformer.Informer().HasSynced,
		configMapSynced:         configMapInformer.Informer().HasSynced,
		secretSynced:            secretInformer.Informer().HasSynced,
		serviceSynced:           serviceInformer.Informer().HasSynced,
		endpointSynced:          endpointInformer.Informer().HasSynced,
		nodeSynced:              nodeInformer.Informer().HasSynced,
		deviceSynced:            deviceInformer.Informer().HasSynced,
		clusterObjectSyncSynced: clusterObjectSyncInformer.Informer().HasSynced,
		objectSyncSynced:        objectSyncInformer.Informer().HasSynced,

		podLister:               podInformer.Lister(),
		configMapLister:         configMapInformer.Lister(),
		secretLister:            secretInformer.Lister(),
		serviceLister:           serviceInformer.Lister(),
		endpointLister:          endpointInformer.Lister(),
		nodeLister:              nodeInformer.Lister(),
		clusterObjectSyncLister: clusterObjectSyncInformer.Lister(),
		objectSyncLister:        objectSyncInformer.Lister(),

		crdClient: crdClient,
	}

	sctl.nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sctl.deleteObjectSyncs()
		},
	})

	return sctl
}

func Register(ec *configv1alpha1.SyncController, kubeAPIConfig *configv1alpha1.KubeAPIConfig) {
	config.InitConfigure(ec, kubeAPIConfig)
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
	go sctl.podInformer.Informer().Run(beehiveContext.Done())
	go sctl.configMapInformer.Informer().Run(beehiveContext.Done())
	go sctl.secretInformer.Informer().Run(beehiveContext.Done())
	go sctl.serviceInformer.Informer().Run(beehiveContext.Done())
	go sctl.endpointInformer.Informer().Run(beehiveContext.Done())
	go sctl.nodeInformer.Informer().Run(beehiveContext.Done())

	go sctl.deviceInformer.Informer().Run(beehiveContext.Done())
	go sctl.clusterObjectSyncInformer.Informer().Run(beehiveContext.Done())
	go sctl.objectSyncInformer.Informer().Run(beehiveContext.Done())

	if !cache.WaitForCacheSync(beehiveContext.Done(),
		sctl.podSynced,
		sctl.configMapSynced,
		sctl.secretSynced,
		sctl.serviceSynced,
		sctl.endpointSynced,
		sctl.nodeSynced,
		sctl.deviceSynced,
		sctl.clusterObjectSyncSynced,
		sctl.objectSyncSynced,
	) {
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
			err = sctl.crdClient.ReliablesyncsV1alpha1().ObjectSyncs(sync.Namespace).Delete(context.Background(), sync.Name, *metav1.NewDeleteOptions(0))
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

// build Config from flags
func buildConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Config.KubeAPIConfig.Master,
		config.Config.KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(config.Config.KubeAPIConfig.QPS)
	kubeConfig.Burst = int(config.Config.KubeAPIConfig.Burst)
	kubeConfig.ContentType = "application/json"

	return kubeConfig, nil
}
