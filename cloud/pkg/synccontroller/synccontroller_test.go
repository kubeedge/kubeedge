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
	"errors"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
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
				t.Errorf("TestNewSyncController() = %v, want %v", (*testController).Enable(), *tt.want)
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

func TestCheckObjectSync(t *testing.T) {
	tests := []struct {
		name           string
		ExpectedResult bool
		ObjectSyncs    *v1alpha1.ObjectSync
	}{
		{
			name:           "test checkObjectSync true",
			ObjectSyncs:    tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedResult: true,
		},
		{
			name:           "test checkObjectSync false",
			ObjectSyncs:    tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedResult: false,
		},
	}
	for _, tt := range tests {
		t.Run("checkObjectSync()", func(t *testing.T) {
			client := fake.NewSimpleClientset()
			informers := informers.NewSharedInformerFactory(client, 0)
			testController := newSyncController(true)
			testController.nodeLister = informers.Core().V1().Nodes().Lister()
			if tt.ExpectedResult == false {
				err := informers.Core().V1().Nodes().Informer().GetIndexer().Add(&v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: tf.TestNodeID,
					},
				})
				if err != nil {
					t.Errorf("checkObjectSync() failed when create node, error: %v", err)
				}
			}
			got, _ := testController.checkObjectSync(tt.ObjectSyncs)
			if got != tt.ExpectedResult {
				t.Errorf("checkObjectSync() returned unexpected result. got = %v, want = %v", got, tt.ExpectedResult)
			}
		})
	}
}

// newGCTestController builds a SyncController wired to fake clients, with the
// given ObjectSync present in both the lister cache and the API, and an empty
// node lister (so the sync's node is treated as deleted).
func newGCTestController(t *testing.T, objectSync *v1alpha1.ObjectSync) *SyncController {
	t.Helper()
	crdClient := crdfake.NewSimpleClientset()
	crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 0)
	if err := crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Informer().GetIndexer().Add(objectSync); err != nil {
		t.Fatalf("failed to add ObjectSync to indexer: %v", err)
	}
	if _, err := crdClient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).Create(context.Background(), objectSync, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create ObjectSync: %v", err)
	}

	// A ClusterObjectSync for the same node, so the cluster-scoped GC path is exercised too.
	clusterObjectSync := &v1alpha1.ClusterObjectSync{ObjectMeta: metav1.ObjectMeta{Name: objectSync.Name}}
	if err := crdInformers.Reliablesyncs().V1alpha1().ClusterObjectSyncs().Informer().GetIndexer().Add(clusterObjectSync); err != nil {
		t.Fatalf("failed to add ClusterObjectSync to indexer: %v", err)
	}
	if _, err := crdClient.ReliablesyncsV1alpha1().ClusterObjectSyncs().Create(context.Background(), clusterObjectSync, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create ClusterObjectSync: %v", err)
	}

	client := fake.NewSimpleClientset()
	kubeInformers := informers.NewSharedInformerFactory(client, 0)

	testController := newSyncController(true)
	testController.crdclient = crdClient
	testController.nodeLister = kubeInformers.Core().V1().Nodes().Lister()
	testController.objectSyncLister = crdInformers.Reliablesyncs().V1alpha1().ObjectSyncs().Lister()
	testController.clusterObjectSyncLister = crdInformers.Reliablesyncs().V1alpha1().ClusterObjectSyncs().Lister()
	return testController
}

func TestGCNodeSyncs(t *testing.T) {
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")
	testController := newGCTestController(t, objectSync)

	before, _ := testController.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
	if len(before.Items) != 1 {
		t.Fatalf("gcNodeSyncs() setup: got %d ObjectSyncs, want 1", len(before.Items))
	}

	if err := testController.gcNodeSyncs(getNodeName(objectSync.Name)); err != nil {
		t.Fatalf("gcNodeSyncs() returned error: %v", err)
	}

	got, _ := testController.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
	if len(got.Items) != 0 {
		t.Errorf("gcNodeSyncs() got %d ObjectSyncs, want 0", len(got.Items))
	}
	gotCluster, _ := testController.crdclient.ReliablesyncsV1alpha1().ClusterObjectSyncs().List(context.Background(), metav1.ListOptions{})
	if len(gotCluster.Items) != 0 {
		t.Errorf("gcNodeSyncs() got %d ClusterObjectSyncs, want 0", len(gotCluster.Items))
	}
}

func TestGCNodeSyncsKeepsLiveNode(t *testing.T) {
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")
	testController := newGCTestController(t, objectSync)

	// The node still exists (e.g. it was recreated), so its sync records must be kept.
	nodeName := getNodeName(objectSync.Name)
	client := fake.NewSimpleClientset()
	kubeInformers := informers.NewSharedInformerFactory(client, 0)
	if err := kubeInformers.Core().V1().Nodes().Informer().GetIndexer().Add(&v1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}); err != nil {
		t.Fatalf("failed to add node: %v", err)
	}
	testController.nodeLister = kubeInformers.Core().V1().Nodes().Lister()

	if err := testController.gcNodeSyncs(nodeName); err != nil {
		t.Fatalf("gcNodeSyncs() returned error: %v", err)
	}
	got, _ := testController.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
	if len(got.Items) != 1 {
		t.Errorf("gcNodeSyncs() deleted syncs for a live node; got %d ObjectSyncs, want 1", len(got.Items))
	}
}

func TestProcessNextNodeGC(t *testing.T) {
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")
	testController := newGCTestController(t, objectSync)

	testController.nodeGCQueue.Add(getNodeName(objectSync.Name))
	if cont := testController.processNextNodeGC(); !cont {
		t.Errorf("processNextNodeGC() = false, want true")
	}
	got, _ := testController.crdclient.ReliablesyncsV1alpha1().ObjectSyncs(tf.TestNamespace).List(context.Background(), metav1.ListOptions{})
	if len(got.Items) != 0 {
		t.Errorf("processNextNodeGC() left %d ObjectSyncs, want 0", len(got.Items))
	}

	testController.nodeGCQueue.ShutDown()
	if cont := testController.processNextNodeGC(); cont {
		t.Errorf("processNextNodeGC() after shutdown = true, want false")
	}
}

func TestEnqueueOrphanedNodeSyncs(t *testing.T) {
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")
	testController := newGCTestController(t, objectSync)

	testController.enqueueOrphanedNodeSyncs()
	if got := testController.nodeGCQueue.Len(); got != 1 {
		t.Errorf("enqueueOrphanedNodeSyncs() queue len = %d, want 1", got)
	}
}

func TestEnqueueNodeForGC(t *testing.T) {
	testController := newSyncController(true)

	testController.enqueueNodeForGC(&v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}})
	testController.enqueueNodeForGC(cache.DeletedFinalStateUnknown{Obj: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-b"}}})
	testController.enqueueNodeForGC("not-a-node")
	testController.enqueueNodeForGC(cache.DeletedFinalStateUnknown{Obj: "not-a-node"})

	if got := testController.nodeGCQueue.Len(); got != 2 {
		t.Errorf("enqueueNodeForGC() queue len = %d, want 2", got)
	}
}

func TestProcessNextNodeGCRetriesOnError(t *testing.T) {
	objectSync := tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod")
	testController := newGCTestController(t, objectSync)

	// Force ObjectSync deletes to fail so gcNodeSyncs returns an error.
	fakeCRD := testController.crdclient.(*crdfake.Clientset)
	fakeCRD.PrependReactor("delete", "objectsyncs", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("simulated delete failure"))
	})
	fakeCRD.PrependReactor("delete", "clusterobjectsyncs", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("simulated cluster delete failure"))
	})

	nodeName := getNodeName(objectSync.Name)
	testController.nodeGCQueue.Add(nodeName)
	if cont := testController.processNextNodeGC(); !cont {
		t.Errorf("processNextNodeGC() = false, want true")
	}
	if got := testController.nodeGCQueue.NumRequeues(nodeName); got == 0 {
		t.Errorf("processNextNodeGC() did not requeue after a failed GC")
	}
}
