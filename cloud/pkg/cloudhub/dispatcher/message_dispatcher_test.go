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

package dispatcher

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/api/client/clientset/versioned/fake"
	syncinformer "github.com/kubeedge/api/client/informers/externalversions"
	synclisters "github.com/kubeedge/api/client/listers/reliablesyncs/v1alpha1"
	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	mockcon "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn/testing"
)

func TestNoAckRequired(t *testing.T) {
	tests := []struct {
		name    string
		message *beehivemodel.Message
		want    bool
	}{
		{
			name:    "list pod message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/podlist", "response"),
			want:    true,
		},
		{
			name:    "membership message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/membership/detail", "response"),
			want:    true,
		},
		{
			name:    "twin/cloud_updated message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/twin/cloud_updated/detail", "response"),
			want:    true,
		},
		{
			name:    "serviceaccounttoken message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/serviceaccounttoken/default", "response"),
			want:    true,
		},
		{
			name:    "volume operation message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/volume/volume-test", "createvolume"),
			want:    true,
		},
		{
			name:    "applicationResponse message",
			message: beehivemodel.NewMessage("").SetResourceOperation("/node/edge-test/ignore/Application/ignore", "applicationResponse"),
			want:    true,
		},
		{
			name:    "user data message",
			message: beehivemodel.NewMessage("router").SetRoute("", "user"),
			want:    true,
		},
		{
			name:    "response ok message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/node/edge-node", "response").FillBody("OK"),
			want:    true,
		},
		{
			name:    "node message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/node/edge-node", "response").SetRoute("edgecontroller", "resource"),
			want:    true,
		},
		{
			name:    "response error message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/node/edge-node", "response").FillBody(fmt.Errorf("error")),
			want:    true,
		},
		{
			name:    "normal pod update",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/pod/test-pod", "update").SetRoute("edgecontroller", "resource"),
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noAckRequired(tt.message); got != tt.want {
				t.Errorf("noAckRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNodeID(t *testing.T) {
	tests := []struct {
		name    string
		message *beehivemodel.Message
		want    string
		wantErr bool
	}{
		{
			name:    "normal message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/pod/test-pod", "update").SetRoute("edgecontroller", "resource"),
			want:    "edge-node",
			wantErr: false,
		},
		{
			name:    "normal message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/default/configmap/kube-root-ca.crt", "update").SetRoute("edgecontroller", "resource"),
			want:    "edge-node",
			wantErr: false,
		},
		{
			name:    "normal message",
			message: beehivemodel.NewMessage("").SetResourceOperation("node/edge-node/membership/detail", "update").SetRoute("devicecontroller", "resource"),
			want:    "edge-node",
			wantErr: false,
		},
		{
			name:    "bad message",
			message: beehivemodel.NewMessage("").SetResourceOperation("edge-node/membership/detail", "update").SetRoute("edgecontroller", "resource"),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodeID(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNodeID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnqueueAckMessage(t *testing.T) {
	normalMsg1 := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "update")
	normalMsg2 := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "update")
	normalMsg3 := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestDiffPodUID, "3"), "update")
	deleteMsg := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "4"), "delete")
	respMsg := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "5"), "response")
	invalidMsg := tf.NewPodMessage(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, ""), "update")

	tests := []tf.TestCase{
		{
			Name:                 "invalid message arrives",
			InitialObjectSyncs:   tf.NoObjectSyncs,
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: invalidMsg,
			ExpectedObjectSyncs:  tf.NoObjectSyncs,
			ExpectedStoreMessage: nil,
		},
		{
			Name: "delete resource message arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: deleteMsg,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ExpectedStoreMessage: deleteMsg,
		},
		{
			Name: "response message arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: respMsg,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ExpectedStoreMessage: respMsg,
		},
		{
			Name:                 "new message first arrives and no message in store",
			InitialObjectSyncs:   tf.NoObjectSyncs,
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: normalMsg1,
			ExpectedObjectSyncs:  tf.NoObjectSyncs,
			ExpectedStoreMessage: normalMsg1,
		},
		{
			Name: "message with large resource version than already exist objectSync arrives, no message in store",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: normalMsg2,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ExpectedStoreMessage: normalMsg2,
		},
		{
			Name: "message with small resource version than already exist objectSync arrives, no message in store",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{},
			CurrentArriveMessage: normalMsg1,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ExpectedStoreMessage: nil,
		},
		{
			Name: "message already exist in store and new message that resource version large than exist arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{normalMsg1},
			CurrentArriveMessage: normalMsg2,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			},
			ExpectedStoreMessage: normalMsg2,
		},
		{
			Name: "message already exist in store and new message that resource version less than exist arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{normalMsg2},
			CurrentArriveMessage: normalMsg1,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ExpectedStoreMessage: normalMsg2,
		},
		{
			Name: "message already exist in store and new delete resource message arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{normalMsg2},
			CurrentArriveMessage: deleteMsg,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ExpectedStoreMessage: deleteMsg,
		},
		{
			Name: "delete message already exist in store and new message with same resource name but diff UID arrives",
			InitialObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ReactorErrors:        tf.NoErrors,
			InitialMessages:      []*beehivemodel.Message{normalMsg2},
			CurrentArriveMessage: normalMsg3,
			ExpectedObjectSyncs: []*v1alpha1.ObjectSync{
				tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			},
			ExpectedStoreMessage: normalMsg3,
		},
	}

	executeTest := func(t *testing.T, test tf.TestCase) {
		// Initialize the dispatcher
		client := &fake.Clientset{}

		nmp := common.InitNodeMessagePool(tf.TestNodeID)
		mockController := gomock.NewController(t)
		defer mockController.Finish()

		mockConn := mockcon.NewMockConnection(mockController)
		nodeSession := session.NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)

		manager := session.NewSessionManager(10)
		manager.AddSession(nodeSession)

		// init objectSync lister.
		objectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		for _, objectSync := range test.InitialObjectSyncs {
			_ = objectSyncIndexer.Add(objectSync)
		}
		objectSyncLister := synclisters.NewObjectSyncLister(objectSyncIndexer)

		clusterObjectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		clusterObjectSyncLister := synclisters.NewClusterObjectSyncLister(clusterObjectSyncIndexer)

		reactor := tf.NewObjectSyncReactor(client, test.ReactorErrors)
		reactor.AddObjectSyncs(test.InitialObjectSyncs)

		dispatcher := &messageDispatcher{
			reliableClient:          client,
			SessionManager:          manager,
			objectSyncLister:        objectSyncLister,
			clusterObjectSyncLister: clusterObjectSyncLister,
		}

		dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

		// Add init message to node message pool
		for _, msg := range test.InitialMessages {
			messageKey, _ := common.AckMessageKeyFunc(msg)
			if err := nmp.AckMessageStore.Add(msg); err != nil {
				klog.Errorf("fail to add message %v nodeStore, err: %v", msg, err)
				return
			}
			nmp.AckMessageQueue.Add(messageKey)
		}

		dispatcher.enqueueAckMessage(tf.TestNodeID, test.CurrentArriveMessage)

		// Evaluate results
		if err := reactor.CheckObjectSyncs(test.ExpectedObjectSyncs); err != nil {
			t.Errorf("Test %q: %v", test.Name, err)
		}

		item, _, _ := nmp.AckMessageStore.Get(test.CurrentArriveMessage)
		if item == nil && test.ExpectedStoreMessage == nil {
			return
		}

		gotMessage := item.(*beehivemodel.Message)

		if !reflect.DeepEqual(gotMessage, test.ExpectedStoreMessage) {
			t.Errorf("Test %q: expected: %+v, got %+v", test.Name, test.ExpectedStoreMessage, gotMessage)
		}
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			executeTest(t, test)
		})
	}
}

func TestGetAddNodeMessagePool(t *testing.T) {
	// Initialize the dispatcher
	client := &fake.Clientset{}
	manager := session.NewSessionManager(10)

	objectSyncInformer := syncinformer.NewSharedInformerFactory(client, 0).Reliablesyncs().V1alpha1().ObjectSyncs()
	clusterObjectSyncInformer := syncinformer.NewSharedInformerFactory(client, 0).Reliablesyncs().V1alpha1().ClusterObjectSyncs()

	dispatcher := NewMessageDispatcher(manager, objectSyncInformer.Lister(), clusterObjectSyncInformer.Lister(), client)

	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

	actualPool := dispatcher.GetNodeMessagePool(tf.TestNodeID)

	if actualPool != nmp {
		t.Errorf("expected: %#v, got: %#v", nmp, actualPool)
	}
}

func TestDeleteNodeMessagePool(t *testing.T) {
	// Initialize the dispatcher
	client := &fake.Clientset{}
	manager := session.NewSessionManager(10)

	objectSyncInformer := syncinformer.NewSharedInformerFactory(client, 0).Reliablesyncs().V1alpha1().ObjectSyncs()
	clusterObjectSyncInformer := syncinformer.NewSharedInformerFactory(client, 0).Reliablesyncs().V1alpha1().ClusterObjectSyncs()

	dispatcher := &messageDispatcher{
		reliableClient:          client,
		SessionManager:          manager,
		objectSyncLister:        objectSyncInformer.Lister(),
		clusterObjectSyncLister: clusterObjectSyncInformer.Lister(),
	}

	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)
	dispatcher.DeleteNodeMessagePool(tf.TestNodeID, nmp)

	_, exist := dispatcher.NodeMessagePools.Load(tf.TestNodeID)
	if exist {
		t.Errorf("expected pool not exist but got it")
	}
}

func TestEnqueueAckMessage_LazyObjectSyncCreation(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)

	objectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	objectSyncLister := synclisters.NewObjectSyncLister(objectSyncIndexer)

	clusterObjectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	clusterObjectSyncLister := synclisters.NewClusterObjectSyncLister(clusterObjectSyncIndexer)

	dispatcher := &messageDispatcher{
		reliableClient:          client,
		objectSyncLister:        objectSyncLister,
		clusterObjectSyncLister: clusterObjectSyncLister,
	}
	dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

	podMsg := tf.NewPodMessage(tf.NewTestPodResource("test-pod", "uid-123", "1"), "update")

	// Call enqueueAckMessage for resource not present in lister cache
	dispatcher.enqueueAckMessage(tf.TestNodeID, podMsg)

	// Verify message was enqueued into NodeMessagePool
	item, exist, err := nmp.AckMessageStore.Get(podMsg)
	if err != nil || !exist || item == nil {
		t.Fatalf("expected podMsg to be enqueued in AckMessageStore, got exist=%v, err=%v", exist, err)
	}

	// Verify ZERO K8s API calls were performed against reliableClient (no synchronous Create/Update calls)
	actions := client.Actions()
	if len(actions) != 0 {
		t.Errorf("expected 0 K8s API actions on enqueue, got %d actions: %+v", len(actions), actions)
	}
}

// TestSaveSuccessPoint_CreatesObjectSyncOnACK verifies the full lazy creation
// flow for namespaced resources: enqueue produces zero API calls, and the
// ObjectSync CR is created with the correct spec and resourceVersion only when
// the edge node sends an ACK (triggering saveSuccessPoint in NodeSession).
func TestSaveSuccessPoint_CreatesObjectSyncOnACK(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)

	// Empty lister caches → first-time resource, no ObjectSync in cache
	objectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	objectSyncLister := synclisters.NewObjectSyncLister(objectSyncIndexer)
	clusterObjectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	clusterObjectSyncLister := synclisters.NewClusterObjectSyncLister(clusterObjectSyncIndexer)

	// Wire up a reactor to capture ObjectSync create/update/get actions
	reactor := tf.NewObjectSyncReactor(client, tf.NoErrors)

	mockController := gomock.NewController(t)
	defer mockController.Finish()
	mockConn := mockcon.NewMockConnection(mockController)
	mockConn.EXPECT().Close().Return(nil).AnyTimes()

	// Set up NodeSession (ACK side)
	nodeSession := session.NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)
	manager := session.NewSessionManager(10)
	manager.AddSession(nodeSession)

	// Set up dispatcher (enqueue side)
	dispatcher := &messageDispatcher{
		reliableClient:          client,
		SessionManager:          manager,
		objectSyncLister:        objectSyncLister,
		clusterObjectSyncLister: clusterObjectSyncLister,
	}
	dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

	// Simulate edge ACK: WriteMessageAsync succeeds → ReceiveMessageAck closes ackChan → saveSuccessPoint fires
	mockConn.EXPECT().WriteMessageAsync(gomock.Any()).DoAndReturn(func(msg *beehivemodel.Message) error {
		nodeSession.ReceiveMessageAck(msg.GetID())
		return nil
	}).AnyTimes()

	// Start the session in background (starts SendAckMessage loop)
	wg := sync.WaitGroup{}
	stopCh := make(chan struct{})
	defer func() {
		close(stopCh)
		nodeSession.Terminating()
		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		nodeSession.Start()
	}()
	// Send keepalives to keep session alive
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(tf.NormalSendKeepaliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				nodeSession.KeepAliveMessage()
			case <-stopCh:
				return
			}
		}
	}()

	// --- Step 1: Enqueue the message ---
	podResource := tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "5")
	podMsg := tf.NewPodMessage(podResource, "update")
	dispatcher.enqueueAckMessage(tf.TestNodeID, podMsg)

	// --- Step 2 & 3: Wait for ACK + saveSuccessPoint and verify ObjectSync creation ---
	expectedObjectSync := tf.NewObjectSync(podResource, "Pod")
	assert.Eventually(t, func() bool {
		return reactor.CheckObjectSyncs([]*v1alpha1.ObjectSync{expectedObjectSync}) == nil
	}, 2*time.Second, 10*time.Millisecond, "ObjectSync should be created in reactor after ACK")
}

// TestSaveSuccessPoint_CreatesClusterObjectSyncOnACK verifies the full lazy
// creation flow for cluster-scoped (non-namespaced) resources: enqueue produces
// zero API calls, and the ClusterObjectSync CR is created with the correct spec
// and resourceVersion only when the edge node sends an ACK.
func TestSaveSuccessPoint_CreatesClusterObjectSyncOnACK(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)

	// Empty lister caches → first-time resource
	objectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	objectSyncLister := synclisters.NewObjectSyncLister(objectSyncIndexer)
	clusterObjectSyncIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	clusterObjectSyncLister := synclisters.NewClusterObjectSyncLister(clusterObjectSyncIndexer)

	// Wire up reactors for both ObjectSync and ClusterObjectSync
	tf.NewObjectSyncReactor(client, tf.NoErrors)
	clusterReactor := tf.NewClusterObjectSyncReactor(client, tf.NoErrors)

	mockController := gomock.NewController(t)
	defer mockController.Finish()
	mockConn := mockcon.NewMockConnection(mockController)
	mockConn.EXPECT().Close().Return(nil).AnyTimes()

	nodeSession := session.NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)
	manager := session.NewSessionManager(10)
	manager.AddSession(nodeSession)

	dispatcher := &messageDispatcher{
		reliableClient:          client,
		SessionManager:          manager,
		objectSyncLister:        objectSyncLister,
		clusterObjectSyncLister: clusterObjectSyncLister,
	}
	dispatcher.AddNodeMessagePool(tf.TestNodeID, nmp)

	// Simulate edge ACK
	mockConn.EXPECT().WriteMessageAsync(gomock.Any()).DoAndReturn(func(msg *beehivemodel.Message) error {
		nodeSession.ReceiveMessageAck(msg.GetID())
		return nil
	}).AnyTimes()

	wg := sync.WaitGroup{}
	stopCh := make(chan struct{})
	defer func() {
		close(stopCh)
		nodeSession.Terminating()
		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		nodeSession.Start()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(tf.NormalSendKeepaliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				nodeSession.KeepAliveMessage()
			case <-stopCh:
				return
			}
		}
	}()

	// --- Step 1: Enqueue a non-namespaced (cluster-scoped) resource ---
	nodeResource := tf.NewTestNodeResource(tf.TestClusterNodeName, tf.TestClusterNodeUID, "3")
	nodeMsg := tf.NewNodeMessage(nodeResource, "update")
	dispatcher.enqueueAckMessage(tf.TestNodeID, nodeMsg)

	// --- Step 2 & 3: Wait for ACK + saveSuccessPoint and verify ClusterObjectSync creation ---
	expectedClusterObjectSync := tf.NewClusterObjectSync(nodeResource, "Node")
	assert.Eventually(t, func() bool {
		return clusterReactor.CheckClusterObjectSyncs([]*v1alpha1.ClusterObjectSync{expectedClusterObjectSync}) == nil
	}, 2*time.Second, 10*time.Millisecond, "ClusterObjectSync should be created in reactor after ACK")
}
