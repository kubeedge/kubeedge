/*
Copyright 2025 The KubeEdge Authors.

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

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	routerv1 "github.com/kubeedge/api/apis/rules/v1"
	fakeclientset "github.com/kubeedge/api/client/clientset/versioned/fake"
	crdinformers "github.com/kubeedge/api/client/informers/externalversions"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/manager"
	commonconstants "github.com/kubeedge/kubeedge/common/constants"
)

func createTestNodeAndPod(client kubernetes.Interface, nodeName, podName, namespace string) error {
	// Create edge node
	edgeNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				commonconstants.EdgeNodeRoleKey: commonconstants.EdgeNodeRoleValue,
			},
		},
	}

	// Create pod scheduled on the edge node
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}

	// Create the node in the fake client
	_, err := client.CoreV1().Nodes().Create(context.Background(), edgeNode, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create the pod in the fake client
	_, err = client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

type TestMessageLayer struct {
	SendMessages    []model.Message
	ReceiveMessages []model.Message
}

func (tml *TestMessageLayer) Send(message model.Message) error {
	tml.SendMessages = append(tml.SendMessages, message)
	return nil
}

func (tml *TestMessageLayer) Receive() (model.Message, error) {
	if len(tml.ReceiveMessages) > 0 {
		msg := tml.ReceiveMessages[0]
		tml.ReceiveMessages = tml.ReceiveMessages[1:]
		return msg, nil
	}
	return model.Message{}, errors.New("no messages available")
}

func (tml *TestMessageLayer) Response(message model.Message) error {
	return nil
}

func TestDownstreamController_Start(t *testing.T) {
	// Create a done channel for beehive context
	doneChan := make(chan struct{})
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock beehiveContext.Done to return the channel
	patches.ApplyFunc(beehiveContext.Done, func() <-chan struct{} {
		return doneChan
	})

	// Close the done channel after test to stop all goroutines
	defer close(doneChan)

	dc := &DownstreamController{
		podManager:           &manager.PodManager{},
		configmapManager:     &manager.ConfigMapManager{},
		secretManager:        &manager.SecretManager{},
		nodeManager:          &manager.NodesManager{},
		rulesManager:         &manager.RuleManager{},
		ruleEndpointsManager: &manager.RuleEndpointManager{},
	}

	err := dc.Start()
	assert.NoError(t, err)
}

func TestDownstreamController_initLocating(t *testing.T) {
	client := fake.NewSimpleClientset()

	err := createTestNodeAndPod(client, "edge-node-1", "test-pod", "default")
	assert.NoError(t, err)

	dc := &DownstreamController{
		kubeClient: client,
		lc:         &manager.LocationCache{},
	}

	err = dc.initLocating()
	assert.NoError(t, err)

	isEdgeNode := dc.lc.IsEdgeNode("edge-node-1")
	assert.True(t, isEdgeNode)
}

func MockPod(name, namespace, nodeName, resourceVersion string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func MockConfigMap(name, namespace, resourceVersion string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
	}
}

func MockSecret(name, namespace, resourceVersion string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
	}
}

func MockNode(name, resourceVersion string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: resourceVersion,
		},
	}
}

func MockRule(name, resourceVersion string) *routerv1.Rule {
	return &routerv1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: resourceVersion,
		},
	}
}

func MockRuleEndpoint(name, resourceVersion string) *routerv1.RuleEndpoint {
	return &routerv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: resourceVersion,
		},
	}
}

type DirectSharedIndexInformer struct {
	registrations []cache.ResourceEventHandlerRegistration
}

func (d *DirectSharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	reg := &DirectResourceEventHandlerRegistration{}
	d.registrations = append(d.registrations, reg)
	return reg, nil
}

func (d *DirectSharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	reg := &DirectResourceEventHandlerRegistration{}
	d.registrations = append(d.registrations, reg)
	return reg, nil
}

func (d *DirectSharedIndexInformer) RemoveEventHandler(handle cache.ResourceEventHandlerRegistration) error {
	return nil
}

func (d *DirectSharedIndexInformer) GetStore() cache.Store {
	return &DirectStore{}
}

func (d *DirectSharedIndexInformer) GetController() cache.Controller {
	return &DirectController{}
}

func (d *DirectSharedIndexInformer) Run(stopCh <-chan struct{}) {
}

func (d *DirectSharedIndexInformer) HasSynced() bool {
	return true
}

func (d *DirectSharedIndexInformer) LastSyncResourceVersion() string {
	return "1"
}

func (d *DirectSharedIndexInformer) IsStopped() bool {
	return false
}

func (d *DirectSharedIndexInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	return nil
}

func (d *DirectSharedIndexInformer) SetTransform(transformer cache.TransformFunc) error {
	return nil
}

func (d *DirectSharedIndexInformer) AddIndexers(indexers cache.Indexers) error {
	return nil
}

func (d *DirectSharedIndexInformer) GetIndexer() cache.Indexer {
	return &DirectIndexer{}
}

type DirectResourceEventHandlerRegistration struct{}

func (d *DirectResourceEventHandlerRegistration) HasSynced() bool {
	return true
}

type DirectController struct{}

func (d *DirectController) Run(stopCh <-chan struct{}) {
}

func (d *DirectController) HasSynced() bool {
	return true
}

func (d *DirectController) LastSyncResourceVersion() string {
	return "1"
}

type DirectStore struct{}

func (d *DirectStore) Add(obj interface{}) error {
	return nil
}

func (d *DirectStore) Update(obj interface{}) error {
	return nil
}

func (d *DirectStore) Delete(obj interface{}) error {
	return nil
}

func (d *DirectStore) List() []interface{} {
	return []interface{}{}
}

func (d *DirectStore) ListKeys() []string {
	return []string{}
}

func (d *DirectStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (d *DirectStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (d *DirectStore) Replace(items []interface{}, resourceVersion string) error {
	return nil
}

func (d *DirectStore) Resync() error {
	return nil
}

type DirectIndexer struct {
	DirectStore
}

func (d *DirectIndexer) Index(indexName string, obj interface{}) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (d *DirectIndexer) IndexKeys(indexName, indexedValue string) ([]string, error) {
	return []string{}, nil
}

func (d *DirectIndexer) ListIndexFuncValues(indexName string) []string {
	return []string{}
}

func (d *DirectIndexer) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (d *DirectIndexer) GetIndexers() cache.Indexers {
	return cache.Indexers{}
}

func (d *DirectIndexer) AddIndexers(newIndexers cache.Indexers) error {
	return nil
}

type DirectKubeEdgeCustomInformer struct{}

func (d *DirectKubeEdgeCustomInformer) EdgeNode() cache.SharedIndexInformer {
	return &DirectSharedIndexInformer{}
}

type MockPodManager struct {
	EventChannel chan watch.Event
}

func (mpm *MockPodManager) Events() chan watch.Event {
	return mpm.EventChannel
}

func TestNewDownstreamController(t *testing.T) {
	doneChan := make(chan struct{})
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(beehiveContext.Done, func() <-chan struct{} {
		return doneChan
	})

	defer close(doneChan)

	fakeClient := fake.NewSimpleClientset()

	err := createTestNodeAndPod(fakeClient, "edge-node-1", "test-pod", "default")
	assert.NoError(t, err)

	messageLayer := &TestMessageLayer{
		SendMessages:    make([]model.Message, 0),
		ReceiveMessages: make([]model.Message, 0),
	}

	patches.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
		return fakeClient
	})

	patches.ApplyFunc(messagelayer.EdgeControllerMessageLayer, func() messagelayer.MessageLayer {
		return messageLayer
	})

	config := &v1alpha1.EdgeController{
		Buffer: &v1alpha1.EdgeControllerBuffer{
			ConfigMapEvent:     100,
			SecretEvent:        100,
			PodEvent:           100,
			RulesEvent:         100,
			RuleEndpointsEvent: 100,
		},
	}

	k8sInformerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	crdScheme := runtime.NewScheme()
	err = routerv1.AddToScheme(crdScheme)
	assert.NoError(t, err)

	crdClientset := fakeclientset.NewSimpleClientset()
	crdInformerFactory := crdinformers.NewSharedInformerFactory(crdClientset, 0)

	directKeInformer := &DirectKubeEdgeCustomInformer{}

	podInformer := k8sInformerFactory.Core().V1().Pods()
	podManager, err := manager.NewPodManager(config, podInformer.Informer())
	assert.NoError(t, err)

	configMapInformer := k8sInformerFactory.Core().V1().ConfigMaps()
	configMapManager, err := manager.NewConfigMapManager(config, configMapInformer.Informer())
	assert.NoError(t, err)

	secretInformer := k8sInformerFactory.Core().V1().Secrets()
	secretManager, err := manager.NewSecretManager(config, secretInformer.Informer())
	assert.NoError(t, err)

	nodeInformer := directKeInformer.EdgeNode()
	nodesManager, err := manager.NewNodesManager(nodeInformer)
	assert.NoError(t, err)

	rulesInformer := crdInformerFactory.Rules().V1().Rules().Informer()
	rulesManager, err := manager.NewRuleManager(config, rulesInformer)
	assert.NoError(t, err)

	ruleEndpointsInformer := crdInformerFactory.Rules().V1().RuleEndpoints().Informer()
	ruleEndpointsManager, err := manager.NewRuleEndpointManager(config, ruleEndpointsInformer)
	assert.NoError(t, err)

	lc := &manager.LocationCache{}

	dc := &DownstreamController{
		kubeClient:           fakeClient,
		podManager:           podManager,
		configmapManager:     configMapManager,
		secretManager:        secretManager,
		nodeManager:          nodesManager,
		rulesManager:         rulesManager,
		ruleEndpointsManager: ruleEndpointsManager,
		messageLayer:         messageLayer,
		lc:                   lc,
		podLister:            podInformer.Lister(),
	}

	err = dc.initLocating()
	assert.NoError(t, err)

	isEdgeNode := dc.lc.IsEdgeNode("edge-node-1")
	assert.True(t, isEdgeNode, "Edge node should be registered in location cache")
}
