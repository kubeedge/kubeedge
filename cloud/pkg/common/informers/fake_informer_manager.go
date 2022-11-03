/*
Copyright 2022 The KubeEdge Authors.

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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	kubeedgefake "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned/fake"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

type fakeManager struct {
	lock                    sync.RWMutex
	dynamicClient           dynamic.Interface
	kubeClient              kubernetes.Interface
	kubeEdgeClient          crdClientset.Interface
	kubeInformerFactory     k8sinformer.SharedInformerFactory
	kubeEdgeInformerFactory crdinformers.SharedInformerFactory
	dynamicInformerFactory  dynamicinformer.DynamicSharedInformerFactory
	informersByGVR          map[schema.GroupVersionResource]*InformerPair
}

func NewFakeInformerManager() Manager {
	dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
	kubeClient := kubefake.NewSimpleClientset()
	kubeEdgeClient := kubeedgefake.NewSimpleClientset()

	return &fakeManager{
		dynamicClient:           dynamicClient,
		kubeClient:              kubeClient,
		kubeEdgeClient:          kubeEdgeClient,
		informersByGVR:          make(map[schema.GroupVersionResource]*InformerPair),
		kubeEdgeInformerFactory: crdinformers.NewSharedInformerFactory(kubeEdgeClient, 0),
		kubeInformerFactory:     k8sinformer.NewSharedInformerFactory(kubeClient, 0),
		dynamicInformerFactory:  dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, 0, v1.NamespaceAll, nil),
	}
}

// GetKubeInformerFactory return kubernetes built-in resources InformerFactory
func (fm *fakeManager) GetKubeInformerFactory() k8sinformer.SharedInformerFactory {
	return fm.kubeInformerFactory
}

// GetKubeEdgeInformerFactory return KubeEdge CRD resources InformerFactory
func (fm *fakeManager) GetKubeEdgeInformerFactory() crdinformers.SharedInformerFactory {
	return fm.kubeEdgeInformerFactory
}

// GetDynamicInformerFactory return third-party CRD resources InformerFactory
func (fm *fakeManager) GetDynamicInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return fm.dynamicInformerFactory
}

// Start start all InformerFactory
func (fm *fakeManager) Start(stopCh <-chan struct{}) {
}

// GetInformerPair return InformerPair for the given GVR
func (fm *fakeManager) GetInformerPair(gvr schema.GroupVersionResource) (*InformerPair, error) {
	switch gvr {
	case schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}:
		fm.lock.Lock()
		defer fm.lock.Unlock()

		informer, ok := fm.informersByGVR[gvr]
		if ok {
			return informer, nil
		}

		podInformer, err := fm.kubeInformerFactory.ForResource(gvr)
		if err != nil {
			return nil, err
		}
		fm.informersByGVR[gvr] = &InformerPair{
			Lister:   podInformer.Lister(),
			Informer: podInformer.Informer(),
		}

		return fm.informersByGVR[gvr], nil

	default:
		return nil, nil
	}
}

// GetLister return cached lister for the given GVR
func (fm *fakeManager) GetLister(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
	informerPair, err := fm.GetInformerPair(gvr)
	if err != nil {
		return nil, err
	}
	return informerPair.Lister, nil
}

func (fm *fakeManager) EdgeNode() cache.SharedIndexInformer {
	klog.Errorf("Not implemented")
	return nil
}
