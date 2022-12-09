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

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller/config"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	crdfake "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned/fake"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
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

func TestDeleteObjectSyncs(t *testing.T) {
	tests := []struct {
		name          string
		ExpectedSyncs int
		ObjectSyncs   *v1alpha1.ObjectSync
	}{
		{
			name:          "test deleteObjectSyncs false",
			ObjectSyncs:   tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedSyncs: 0,
		},
	}
	for _, tt := range tests {
		t.Run("deleteObjectSyncs()", func(t *testing.T) {
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
			client := fake.NewSimpleClientset()
			informers := informers.NewSharedInformerFactory(client, 0)
			testController := newSyncController(true)
			testController.nodeLister = informers.Core().V1().Nodes().Lister()
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
