package informers

import (
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

type newInformer func() cache.SharedIndexInformer

type Informers interface {
	EdgeSitePod(nodeName string) cache.SharedIndexInformer
	Pod() cache.SharedIndexInformer
	ConfigMap() cache.SharedIndexInformer
	Secrets() cache.SharedIndexInformer
	Service() cache.SharedIndexInformer
	Endpoints() cache.SharedIndexInformer
	Node() cache.SharedIndexInformer
	EdgeNode() cache.SharedIndexInformer
	ClusterObjectSync() cache.SharedIndexInformer
	ObjectSync() cache.SharedIndexInformer
	Device() cache.SharedIndexInformer
	Start(stopCh <-chan struct{})
}

var globalInformers Informers
var once sync.Once

func GetGlobalInformers() Informers {
	once.Do(func() {
		globalInformers = &informers{
			defaultResync:    0,
			keClient:         client.GetKubeEdgeClient(),
			informers:        make(map[string]cache.SharedIndexInformer),
			startedInformers: make(map[string]bool),
		}
	})
	return globalInformers
}

type informers struct {
	defaultResync    time.Duration
	keClient         client.KubeEdgeClient
	lock             sync.Mutex
	informers        map[string]cache.SharedIndexInformer
	startedInformers map[string]bool
}

func (ifs *informers) EdgeSitePod(nodeName string) cache.SharedIndexInformer {
	return ifs.getInformer("edgesitepodinformer", func() cache.SharedIndexInformer {
		selector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "pods", v1.NamespaceAll, selector)
		return cache.NewSharedIndexInformer(lw, &v1.Pod{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Pod() cache.SharedIndexInformer {
	return ifs.getInformer("podinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "pods", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.Pod{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) ConfigMap() cache.SharedIndexInformer {
	return ifs.getInformer("configmapinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "configmaps", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.ConfigMap{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Secrets() cache.SharedIndexInformer {
	return ifs.getInformer("secretsinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "secrets", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.Secret{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Service() cache.SharedIndexInformer {
	return ifs.getInformer("serviceinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "services", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.Service{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Endpoints() cache.SharedIndexInformer {
	return ifs.getInformer("endpointsinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "endpoints", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.Endpoints{}, ifs.defaultResync, cache.Indexers{})
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

func (ifs *informers) Node() cache.SharedIndexInformer {
	return ifs.getInformer("nodesinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.CoreV1().RESTClient(), "nodes", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1.Node{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) ClusterObjectSync() cache.SharedIndexInformer {
	return ifs.getInformer("clusterbojectsyncinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.ReliablesyncsRestClient(), "clusterobjectsyncs", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1alpha1.ClusterObjectSync{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) ObjectSync() cache.SharedIndexInformer {
	return ifs.getInformer("objectsyncinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.ReliablesyncsRestClient(), "objectsyncs", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1alpha1.ObjectSync{}, ifs.defaultResync, cache.Indexers{})
	})
}

func (ifs *informers) Device() cache.SharedIndexInformer {
	return ifs.getInformer("devicesinformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(ifs.keClient.DevicesRestClient(), "devices", v1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &v1alpha2.Device{}, ifs.defaultResync, cache.Indexers{})
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
