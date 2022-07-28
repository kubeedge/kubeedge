/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package informers

import (
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic/dynamicinformer"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

type newInformer func() cache.SharedIndexInformer

type KubeEdgeCustomInformer interface {
	EdgeNode() cache.SharedIndexInformer
}

type Manager interface {
	GetK8sInformerFactory() k8sinformer.SharedInformerFactory
	GetCRDInformerFactory() crdinformers.SharedInformerFactory
	GetDynamicSharedInformerFactory() dynamicinformer.DynamicSharedInformerFactory
	KubeEdgeCustomInformer
	Start(stopCh <-chan struct{})
}

type informers struct {
	defaultResync                time.Duration
	keClient                     kubernetes.Interface
	lock                         sync.Mutex
	informers                    map[string]cache.SharedIndexInformer
	crdSharedInformerFactory     crdinformers.SharedInformerFactory
	k8sSharedInformerFactory     k8sinformer.SharedInformerFactory
	dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
}

var globalInformers Manager
var once sync.Once

func GetInformersManager() Manager {
	once.Do(func() {
		globalInformers = &informers{
			defaultResync:                0,
			keClient:                     client.GetKubeClient(),
			informers:                    make(map[string]cache.SharedIndexInformer),
			crdSharedInformerFactory:     crdinformers.NewSharedInformerFactory(client.GetCRDClient(), 0),
			k8sSharedInformerFactory:     k8sinformer.NewSharedInformerFactory(client.GetKubeClient(), 0),
			dynamicSharedInformerFactory: dynamicinformer.NewFilteredDynamicSharedInformerFactory(client.GetDynamicClient(), 0, v1.NamespaceAll, nil),
		}
	})
	return globalInformers
}

func (ifs *informers) GetK8sInformerFactory() k8sinformer.SharedInformerFactory {
	return ifs.k8sSharedInformerFactory
}

func (ifs *informers) GetCRDInformerFactory() crdinformers.SharedInformerFactory {
	return ifs.crdSharedInformerFactory
}

func (ifs *informers) GetDynamicSharedInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return ifs.dynamicSharedInformerFactory
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

//Note: please WaitForCache after getting an informer from factory, example:
//	informer := informerFactory.ForResource(gvr)
//	for gvr, cacheSync := range informerFactory.WaitForCacheSync(beehiveContext.Done()) {
//		if !cacheSync {
//			klog.Fatalf("unable to sync caches for: %s", gvr.String())
//		}
//	}

func (ifs *informers) Start(stopCh <-chan struct{}) {
	ifs.lock.Lock()
	defer ifs.lock.Unlock()

	for name, informer := range ifs.informers {
		klog.V(5).Infof("start informer %s", name)
		go informer.Run(stopCh)
	}
	ifs.k8sSharedInformerFactory.Start(stopCh)
	ifs.crdSharedInformerFactory.Start(stopCh)
	ifs.dynamicSharedInformerFactory.Start(stopCh)
}

// getInformer get an informer named "name" or store an informer got by "newFunc" as key "name"
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
