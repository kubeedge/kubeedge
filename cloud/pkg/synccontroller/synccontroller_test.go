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

package synccontroller

import (
	"context"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	configv1alpha1 "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	crdfake "github.com/kubeedge/api/client/clientset/versioned/fake"
	crdinformers "github.com/kubeedge/api/client/informers/externalversions"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
)

func TestNewSyncControllerAndStartIt(t *testing.T) {
	tests := []struct {
		name       string
		controller *SyncController
		want       *SyncController
	}{
		{
			name: "New Sync controller",
			controller: &SyncController{
				enable: false,
			},
			want: &SyncController{
				enable: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testController := newSyncController(tt.controller.enable)
			if !reflect.DeepEqual(tt.want.enable, testController.Enable()) {
				t.Errorf("TestNewSyncController() = %v, want %v", testController.Enable(), tt.want.enable)
			}
			testController.informersSyncedFuncs = testController.informersSyncedFuncs[:0]
			testController.informersSyncedFuncs = append(testController.informersSyncedFuncs, func() bool {
				return true
			})
			testController.Start()
			time.Sleep(2 * time.Second)
			beehiveContext.Cancel()
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		controller *configv1alpha1.SyncController
		want       *configv1alpha1.SyncController
	}{
		{
			name: "Register Sync controller",
			controller: &configv1alpha1.SyncController{
				Enable: false,
			},
			want: &configv1alpha1.SyncController{
				Enable: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.controller)
			if !reflect.DeepEqual(tt.want.Enable, config.Config.SyncController.Enable) {
				t.Errorf("TestRegister() = %v, want %v", *config.Config.SyncController, *tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Run("SyncController.Name()", func(t *testing.T) {
		if got := (&SyncController{}).Name(); got != modules.SyncControllerModuleName {
			t.Errorf("SyncController.Name() returned unexpected result. got = %s, want = synccontroller", got)
		}
	})
}

func TestGroup(t *testing.T) {
	t.Run("SyncController.Group()", func(t *testing.T) {
		if got := (&SyncController{}).Group(); got != modules.SyncControllerModuleGroup {
			t.Errorf("SyncController.Group() returned unexpected result. got = %s, want = synccontroller", got)
		}
	})
}

func TestBuildObjectSyncName(t *testing.T) {
	t.Run("BuildObjectSyncName()", func(t *testing.T) {
		if got := BuildObjectSyncName("edge_node", "1234"); got != "edge_node.1234" {
			t.Errorf("BuildObjectSyncName() returned unexpected result. got = %s, want = edge_node.1234", got)
		}
	})
}

func TestGetNodeName(t *testing.T) {
	t.Run("getNodeName()", func(t *testing.T) {
		if got := getNodeName("edge-node.abcd-abcd"); got != "edge-node" {
			t.Errorf("getNodeName() returned unexpected result. got = %s, want = edge_node", got)
		}
		if got := getNodeName("edge.node.abcd-abcd"); got != "edge.node" {
			t.Errorf("getNodeName() returned unexpected result. got = %s, want = edge.node", got)
		}
	})
}

func TestGetObjectUID(t *testing.T) {
	t.Run("getObjectUID()", func(t *testing.T) {
		if got := getObjectUID("edge-node.abcd-abcd"); got != "abcd-abcd" {
			t.Errorf("getObjectUID() returned unexpected result. got = %s, want = abcd-abcd", got)
		}
	})
}

// newTestNodeLister returns a node lister that contains the given nodes.
func newTestNodeLister(t *testing.T, nodeNames ...string) corelisters.NodeLister {
	nodeInformer := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0).Core().V1().Nodes()
	for _, nodeName := range nodeNames {
		err := nodeInformer.Informer().GetIndexer().Add(&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		})
		if err != nil {
			t.Fatalf("failed to add node %s to the indexer: %v", nodeName, err)
		}
	}
	return nodeInformer.Lister()
}

func TestCheckObjectSync(t *testing.T) {
	tests := []struct {
		name           string
		nodeExists     bool
		missingFor     *time.Duration
		ExpectedResult bool
		expectTracked  bool
	}{
		{
			name:           "node exists",
			nodeExists:     true,
			ExpectedResult: false,
			expectTracked:  false,
		},
		{
			name:           "node registered again after being deleted",
			nodeExists:     true,
			missingFor:     durationPtr(time.Hour),
			ExpectedResult: false,
			expectTracked:  false,
		},
		{
			name:           "node just found missing",
			nodeExists:     false,
			ExpectedResult: false,
			expectTracked:  true,
		},
		{
			name:           "node missing within the retention",
			nodeExists:     false,
			missingFor:     durationPtr(time.Hour),
			ExpectedResult: false,
			expectTracked:  true,
		},
		{
			name:           "node missing for longer than the retention",
			nodeExists:     false,
			missingFor:     durationPtr(deletedNodeSyncsRetention + time.Minute),
			ExpectedResult: true,
			expectTracked:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testController := newSyncController(true)
			if tt.nodeExists {
				testController.nodeLister = newTestNodeLister(t, tf.TestNodeID)
			} else {
				testController.nodeLister = newTestNodeLister(t)
			}
			if tt.missingFor != nil {
				testController.nodeMissingSince = map[string]time.Time{tf.TestNodeID: time.Now().Add(-*tt.missingFor)}
			}

			got, err := testController.checkObjectSync(tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"))
			if err != nil {
				t.Errorf("checkObjectSync() returned unexpected error: %v", err)
			}
			if got != tt.ExpectedResult {
				t.Errorf("checkObjectSync() returned unexpected result. got = %v, want = %v", got, tt.ExpectedResult)
			}
			if _, tracked := testController.nodeMissingSince[tf.TestNodeID]; tracked != tt.expectTracked {
				t.Errorf("checkObjectSync() node tracked as missing = %v, want = %v", tracked, tt.expectTracked)
			}
		})
	}
}

func TestCheckClusterObjectSync(t *testing.T) {
	testController := newSyncController(true)
	testController.nodeLister = newTestNodeLister(t)
	sync := &v1alpha1.ClusterObjectSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: BuildObjectSyncName(tf.TestNodeID, tf.TestPodUID),
		},
	}

	got, err := testController.checkClusterObjectSync(sync)
	if err != nil || got {
		t.Errorf("checkClusterObjectSync() for a node that was just deleted = (%v, %v), want = (false, nil)", got, err)
	}

	testController.nodeMissingSince[tf.TestNodeID] = time.Now().Add(-deletedNodeSyncsRetention - time.Minute)
	got, err = testController.checkClusterObjectSync(sync)
	if err != nil || !got {
		t.Errorf("checkClusterObjectSync() for a node deleted longer than the retention = (%v, %v), want = (true, nil)", got, err)
	}
}

func TestForgetNodesWithoutSyncs(t *testing.T) {
	crdClient := crdfake.NewSimpleClientset()
	crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 0)
	err := crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Informer().GetIndexer().Add(
		tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"))
	if err != nil {
		t.Fatalf("failed to add ObjectSync to the indexer: %v", err)
	}
	err = crdInformers.Reliablesyncs().V1alpha1().ClusterObjectSyncs().Informer().GetIndexer().Add(&v1alpha1.ClusterObjectSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: BuildObjectSyncName("cluster-node", tf.TestPodUID),
		},
	})
	if err != nil {
		t.Fatalf("failed to add ClusterObjectSync to the indexer: %v", err)
	}

	testController := newSyncController(true)
	testController.objectSyncLister = crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Lister()
	testController.clusterObjectSyncLister = crdInformers.Reliablesyncs().V1alpha1().ClusterObjectSyncs().Lister()
	testController.nodeMissingSince = map[string]time.Time{
		tf.TestNodeID:  time.Now(),
		"cluster-node": time.Now(),
		"gone-node":    time.Now(),
	}

	testController.forgetNodesWithoutSyncs()

	want := []string{tf.TestNodeID, "cluster-node"}
	if len(testController.nodeMissingSince) != len(want) {
		t.Errorf("forgetNodesWithoutSyncs() kept %v, want %v", testController.nodeMissingSince, want)
	}
	for _, nodeName := range want {
		if _, ok := testController.nodeMissingSince[nodeName]; !ok {
			t.Errorf("forgetNodesWithoutSyncs() forgot node %s which still has sync records", nodeName)
		}
	}
}

func TestNodeDeletedAgainStartsNewRetention(t *testing.T) {
	testController := newSyncController(true)
	testController.nodeLister = newTestNodeLister(t)
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: tf.TestNodeID}}
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")

	// The node was deleted longer than the retention ago.
	testController.deleteNode(node)
	testController.nodeMissingSince[tf.TestNodeID] = time.Now().Add(-deletedNodeSyncsRetention - time.Minute)

	// The node registers again, which ends its retention period.
	testController.addNode(node)
	if _, tracked := testController.nodeMissingSince[tf.TestNodeID]; tracked {
		t.Errorf("addNode() kept tracking node %s as missing", tf.TestNodeID)
	}

	// The node is deleted again before the next periodic GC (the delete event of the
	// informer can also carry a tombstone), so its sync records get a new retention period.
	testController.deleteNode(cache.DeletedFinalStateUnknown{Key: tf.TestNodeID, Obj: node})
	got, err := testController.checkObjectSync(objectSync)
	if err != nil || got {
		t.Errorf("checkObjectSync() after the node was deleted again = (%v, %v), want = (false, nil)", got, err)
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func TestDeleteObjectSyncs(t *testing.T) {
	tests := []struct {
		name          string
		nodeExists    bool
		missingFor    *time.Duration
		ExpectedSyncs int
		ObjectSyncs   *v1alpha1.ObjectSync
	}{
		{
			name:          "keep the syncs of an existing node",
			nodeExists:    true,
			ObjectSyncs:   tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedSyncs: 1,
		},
		{
			name:          "keep the syncs of a node that was just deleted",
			ObjectSyncs:   tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedSyncs: 1,
		},
		{
			name:          "keep the syncs of a node deleted within the retention",
			missingFor:    durationPtr(time.Hour),
			ObjectSyncs:   tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedSyncs: 1,
		},
		{
			name:          "delete the syncs of a node deleted for longer than the retention",
			missingFor:    durationPtr(deletedNodeSyncsRetention + time.Minute),
			ObjectSyncs:   tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedSyncs: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crdClient := crdfake.NewSimpleClientset()
			crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 0)
			err := crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Informer().GetIndexer().Add(tt.ObjectSyncs)
			if err != nil {
				t.Errorf("deleteObjectSyncs() failed when create ObjectSyncs indexer, error: %v", err)
			}
			_, err = crdClient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).Create(context.Background(), tt.ObjectSyncs, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("deleteObjectSyncs() failed when create ObjectSyncs, error: %v", err)
			}
			testController := newSyncController(true)
			if tt.nodeExists {
				testController.nodeLister = newTestNodeLister(t, tf.TestNodeID)
			} else {
				testController.nodeLister = newTestNodeLister(t)
			}
			if tt.missingFor != nil {
				testController.nodeMissingSince = map[string]time.Time{tf.TestNodeID: time.Now().Add(-*tt.missingFor)}
			}
			testController.crdclient = crdClient
			testController.objectSyncLister = crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Lister()
			syncs, _ := crdClient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
			if len(syncs.Items) != 1 {
				t.Errorf("deleteObjectSyncs() list returned unexpected result. got = %v, want = %v", len(syncs.Items), 1)
			}
			testController.deleteObjectSyncs()
			got, _ := crdClient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
			if len(got.Items) != tt.ExpectedSyncs {
				t.Errorf("deleteObjectSyncs() list returned unexpected result. got = %v, want = %v", len(got.Items), tt.ExpectedSyncs)
			}
		})
	}
}
