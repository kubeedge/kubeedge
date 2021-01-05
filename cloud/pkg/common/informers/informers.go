package informers

import (
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	crdinformers "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

type newInformer func() cache.SharedIndexInformer

type KubeEdgeCustomeInformer interface {
	EdgeSitePod(nodeName string) cache.SharedIndexInformer
	EdgeNode() cache.SharedIndexInformer
}

type Manager interface {
	GetK8sInformerFactory() k8sinformer.SharedInformerFactory
	GetCRDInformerFactory() crdinformers.SharedInformerFactory
	KubeEdgeCustomeInformer
	Start(stopCh <-chan struct{})
}

var globalInformers Manager
var once sync.Once

func GetInformersManager() Manager {
	once.Do(func() {
		globalInformers = &informers{
			defaultResync:            0,
			keClient:                 client.GetKubeEdgeClient(),
			informers:                make(map[string]cache.SharedIndexInformer),
			startedInformers:         make(map[string]bool),
			crdSharedInformerFactory: crdinformers.NewSharedInformerFactory(client.GetKubeEdgeClient(), 0),
			k8sSharedInformerFactory: k8sinformer.NewSharedInformerFactory(client.GetKubeEdgeClient(), 0),
		}
	})
	return globalInformers
}

type informers struct {
	defaultResync            time.Duration
	keClient                 client.KubeEdgeClient
	lock                     sync.Mutex
	informers                map[string]cache.SharedIndexInformer
	startedInformers         map[string]bool
	crdSharedInformerFactory crdinformers.SharedInformerFactory
	k8sSharedInformerFactory k8sinformer.SharedInformerFactory
}

func (ifs *informers) GetK8sInformerFactory() k8sinformer.SharedInformerFactory {
	return ifs.k8sSharedInformerFactory
}

func (ifs *informers) GetCRDInformerFactory() crdinformers.SharedInformerFactory {
	return ifs.crdSharedInformerFactory
}

func (ifs *informers) EdgeSitePod(nodeName string) cache.SharedIndexInformer {
	return ifs.getInformer("edgesitepodinformer", func() cache.SharedIndexInformer {
		selector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "pods", v1.NamespaceAll, selector)
		return cache.NewSharedIndexInformer(lw, &v1.Pod{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) EdgeNode() cache.SharedIndexInformer {
	return ifs.getInformer("edgenodesinformer", func() cache.SharedIndexInformer {
		set := labels.Set{"node-role.kubernetes.io/edge": ""}
		selector := labels.SelectorFromSet(set)
		optionModifier := func(options *metav1.ListOptions) {
			options.LabelSelector = selector.String()
		}
		lw := cache.NewFilteredListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "nodes", v1.NamespaceAll, optionModifier)
		return cache.NewSharedIndexInformer(lw, &v1.Node{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Start(stopCh <-chan struct{}) {
	ifs.lock.Lock()
	defer ifs.lock.Unlock()

	for name, informer := range ifs.informers {
		if ifs.startedInformers[name] {
			klog.V(5).Infof("informer %s has being started, skip", name)
			continue
		}
		klog.V(5).Infof("start informer %s", name)
		go informer.Run(stopCh)
		ifs.startedInformers[name] = true
	}
	ifs.k8sSharedInformerFactory.Start(stopCh)
	ifs.crdSharedInformerFactory.Start(stopCh)
}

func (ifs *informers) getInformer(name string, newFunc newInformer) cache.SharedIndexInformer {
	ifs.lock.Lock()
	defer ifs.lock.Unlock()
	informer, exist := ifs.informers[name]
	if exist {
		return informer
	}
	informer = newFunc()
	ifs.informers[name] = informer
	return informer
}
