package synccontroller

import (
	"os"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	crdinformerfactory "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions"
	deviceinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/devices/v1alpha1"
	syncinformer "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions/reliablesyncs/v1alpha1"
	devicelister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/devices/v1alpha1"
	synclister "github.com/kubeedge/kubeedge/cloud/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
)

// SyncController use beehive context message layer
type SyncController struct {
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
}

func newSyncController() *SyncController {
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
	deviceInformer := crdFactory.Devices().V1alpha1().Devices()
	clusterObjectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	objectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ObjectSyncs()

	sc := &SyncController{
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

		podLister:       podInformer.Lister(),
		configMapLister: configMapInformer.Lister(),
		secretLister:    secretInformer.Lister(),
		serviceLister:   serviceInformer.Lister(),
		endpointLister:  endpointInformer.Lister(),
		nodeLister:      nodeInformer.Lister(),
	}

	return sc
}

func Register() {
	core.Register(newSyncController())
}

// Name of controller
func (sctl *SyncController) Name() string {
	return SyncControllerModuleName
}

// Group of controller
func (sctl *SyncController) Group() string {
	return SyncControllerModuleGroup
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
}

func (sctl *SyncController) reconcile() {
	allClusterObjectSyncs, err := sctl.clusterObjectSyncLister.List(nil)
	if err != nil {
		klog.Errorf("Filed to list all the ClusterObjectSyncs: %v", err)
	}
	sctl.manageClusterObjectSync(allClusterObjectSyncs)

	allObjectSyncs, err := sctl.objectSyncLister.List(nil)
	if err != nil {
		klog.Errorf("Filed to list all the ObjectSyncs: %v", err)
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
		case PodKind:
			sctl.managePod(sync)
		case ConfigMapKind:
			sctl.manageConfigMap(sync)
		case SecretKind:
			sctl.manageSecret(sync)
		case ServiceKind:
			sctl.manageService(sync)
		case EndpointKind:
			sctl.manageEndpoint(sync)
		case DeviceKind:
			sctl.manageDevice(sync)
		default:
			klog.Errorf("Unsupported object kind： %v", sync.Spec.ObjectKind)
		}
	}
}

func buildObjectSyncName(nodeName, UID string) string {
	return nodeName + "/" + UID
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
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Get().KubeMaster, config.Get().KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = config.Get().KubeQPS
	kubeConfig.Burst = config.Get().KubeBurst
	kubeConfig.ContentType = config.Get().KubeContentType

	return kubeConfig, nil
}
