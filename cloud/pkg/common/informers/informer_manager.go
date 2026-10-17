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
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	edgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
	edgeinformers "github.com/kubeedge/api/client/informers/externalversions"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/common/constants"
)

var (
	// kubeScheme is built-in resources scheme for converting group,
	// version, and kind information to and from Go schemas
	kubeScheme = scheme.Scheme

	// kubeEdgeScheme is KubeEdge CRD resources scheme for converting
	// group, version, and kind information to and from Go schemas
	kubeEdgeScheme = edgescheme.Scheme

	// informerSyncTimeout bounds how long GetInformerPair waits for a new
	// informer's cache to sync. A resource cloudcore is not allowed to list
	// never syncs. It is kept below the 10s edge nodes wait for a metaserver
	// Application response, so a rejection still reaches the edge.
	informerSyncTimeout = 5 * time.Second
)

// InformerPair include informer and lister for resource
type InformerPair struct {
	// GenericLister is a lister that helps list generic resource.
	// All objects returned here must be treated as read-only.
	Lister cache.GenericLister

	// Informer provides add and get Indexers ability based on SharedInformer.
	Informer cache.SharedIndexInformer
}

type newInformer func() cache.SharedIndexInformer

type KubeEdgeCustomInformer interface {
	EdgeNode() cache.SharedIndexInformer
}

type Manager interface {
	// GetKubeInformerFactory return kubernetes built-in resources InformerFactory
	GetKubeInformerFactory() k8sinformer.SharedInformerFactory

	// GetKubeEdgeInformerFactory return KubeEdge CRD resources InformerFactory
	GetKubeEdgeInformerFactory() edgeinformers.SharedInformerFactory

	// GetDynamicInformerFactory return third-party CRD resources InformerFactory
	GetDynamicInformerFactory() dynamicinformer.DynamicSharedInformerFactory

	// KubeEdgeCustomInformer return custom informer
	KubeEdgeCustomInformer

	// Start start all InformerFactory
	Start(stopCh <-chan struct{})

	// GetInformerPair return InformerPair for the given GVR
	GetInformerPair(gvr schema.GroupVersionResource) (*InformerPair, error)

	// GetLister return cached lister for the given GVR
	GetLister(gvr schema.GroupVersionResource) (cache.GenericLister, error)
}

type informers struct {
	// defaultResync define default resync Period for custom informer
	defaultResync time.Duration

	// kubeClient kubernetes built-in resources client
	kubeClient kubernetes.Interface

	// stopCh is the stop channel to stop informers
	stopCh <-chan struct{}

	// mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources
	mapper meta.RESTMapper

	// lock protects to the informersByGVR and customInformers map
	lock sync.RWMutex

	// customInformers is the cache of informers that support custom filter
	customInformers map[string]cache.SharedIndexInformer

	// informersByGVR is the cache of informers keyed by GroupVersionResource
	informersByGVR map[schema.GroupVersionResource]*InformerPair

	// kubeInformerFactory provides shared informers for built-in resources
	// in all known API group versions.
	kubeInformerFactory k8sinformer.SharedInformerFactory

	// kubeEdgeInformerFactory provides shared informers for KubeEdge CRD
	// resources in all known API group versions.
	kubeEdgeInformerFactory edgeinformers.SharedInformerFactory

	// dynamicInformerFactory provides access to a shared informer and lister
	// for dynamic client. It is mainly used for third-party CRD resources.
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
}

var globalInformers Manager
var once sync.Once

func GetInformersManager() Manager {
	once.Do(func() {
		mapper, err := client.DefaultGetRestMapper()
		if err != nil {
			panic(fmt.Errorf("init rest mapper err: %v", err))
		}

		globalInformers = &informers{
			stopCh:                  beehiveContext.GetContext().Done(),
			defaultResync:           0,
			mapper:                  mapper,
			kubeClient:              client.GetKubeClient(),
			customInformers:         make(map[string]cache.SharedIndexInformer),
			informersByGVR:          make(map[schema.GroupVersionResource]*InformerPair),
			kubeEdgeInformerFactory: edgeinformers.NewSharedInformerFactory(client.GetCRDClient(), 0),
			kubeInformerFactory:     k8sinformer.NewSharedInformerFactory(client.GetKubeClient(), 0),
			dynamicInformerFactory:  dynamicinformer.NewFilteredDynamicSharedInformerFactory(client.GetDynamicClient(), 0, v1.NamespaceAll, nil),
		}
	})
	return globalInformers
}

func (ifs *informers) GetKubeInformerFactory() k8sinformer.SharedInformerFactory {
	return ifs.kubeInformerFactory
}

func (ifs *informers) GetKubeEdgeInformerFactory() edgeinformers.SharedInformerFactory {
	return ifs.kubeEdgeInformerFactory
}

func (ifs *informers) GetDynamicInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return ifs.dynamicInformerFactory
}

func (ifs *informers) EdgeNode() cache.SharedIndexInformer {
	return ifs.getInformer("edgenodesinformer", func() cache.SharedIndexInformer {
		set := labels.Set{constants.EdgeNodeRoleKey: constants.EdgeNodeRoleValue}
		selector := labels.SelectorFromSet(set)
		optionModifier := func(options *metav1.ListOptions) {
			options.LabelSelector = selector.String()
		}
		lw := cache.NewFilteredListWatchFromClient(ifs.kubeClient.CoreV1().RESTClient(), "nodes", v1.NamespaceAll, optionModifier)
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

	for name, informer := range ifs.customInformers {
		klog.V(5).Infof("start informer %s", name)
		go informer.Run(stopCh)
	}
	ifs.kubeInformerFactory.Start(stopCh)
	ifs.kubeEdgeInformerFactory.Start(stopCh)
	ifs.dynamicInformerFactory.Start(stopCh)
}

// getInformer get an informer named "name" or store an informer got by "newFunc" as key "name"
func (ifs *informers) getInformer(name string, newFunc newInformer) cache.SharedIndexInformer {
	ifs.lock.Lock()
	defer ifs.lock.Unlock()
	informer, exist := ifs.customInformers[name]
	if exist {
		return informer
	}
	informer = newFunc()
	ifs.customInformers[name] = informer
	return informer
}

func (ifs *informers) GetLister(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
	informerPair, err := ifs.GetInformerPair(gvr)
	if err != nil {
		return nil, err
	}
	return informerPair.Lister, nil
}

func (ifs *informers) GetInformerPair(gvr schema.GroupVersionResource) (*InformerPair, error) {
	ifs.lock.Lock()
	informer, ok := ifs.informersByGVR[gvr]
	if ok {
		ifs.lock.Unlock()
		return informer, nil
	}
	genericInformer, err := ifs.forResource(gvr)
	ifs.lock.Unlock()
	if err != nil {
		return nil, err
	}

	informerPair := &InformerPair{
		Lister:   genericInformer.Lister(),
		Informer: genericInformer.Informer(),
	}

	if !informerPair.Informer.HasSynced() {
		klog.V(4).Infof("waiting for %s Informer to sync", gvr.String())
		// Wait for it to sync before returning the Informer so that folks don't read from a stale cache.
		// The wait is bounded and does not hold ifs.lock, so an informer that can never sync
		// does not block callers for other resources.
		ctx, cancel := context.WithTimeout(wait.ContextForChannel(ifs.stopCh), informerSyncTimeout)
		defer cancel()
		if !cache.WaitForCacheSync(ctx.Done(), informerPair.Informer.HasSynced) {
			return nil, fmt.Errorf("failed waiting for %s Informer to sync within %v", gvr.String(), informerSyncTimeout)
		}
	}

	// Only cache synced informers, so a later call for a resource that failed to sync waits again.
	ifs.lock.Lock()
	defer ifs.lock.Unlock()
	ifs.informersByGVR[gvr] = informerPair

	return informerPair, nil
}

func (ifs *informers) forResource(gvr schema.GroupVersionResource) (k8sinformer.GenericInformer, error) {
	gvk, err := ifs.mapper.KindFor(gvr)
	if err != nil {
		return nil, err
	}

	var genericInformer k8sinformer.GenericInformer

	switch {
	case kubeScheme.Recognizes(gvk):
		klog.V(4).Infof("built-in resource %s informer", gvr.String())

		genericInformer, err = ifs.kubeInformerFactory.ForResource(gvr)
		if err != nil {
			return nil, err
		}
		ifs.kubeInformerFactory.Start(ifs.stopCh)

	case kubeEdgeScheme.Recognizes(gvk):
		klog.V(4).Infof("KubeEdge CRD resource %s informer", gvr.String())

		genericInformer, err = ifs.kubeEdgeInformerFactory.ForResource(gvr)
		if err != nil {
			return nil, err
		}
		ifs.kubeEdgeInformerFactory.Start(ifs.stopCh)

	default:
		klog.V(4).Infof("Third-party resource %s informer", gvr.String())
		genericInformer = ifs.dynamicInformerFactory.ForResource(gvr)
		ifs.dynamicInformerFactory.Start(ifs.stopCh)
	}

	return genericInformer, nil
}
