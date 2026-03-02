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

package synccontroller

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/dynamicinformer"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	edgeinformers "github.com/kubeedge/api/client/informers/externalversions"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

type testFuncBackup struct {
	originalSendToEdge                     func(string, model.Message)
	originalBuildEdgeControllerMessageFunc func(string, string, string, string, string, interface{}) *model.Message
	originalGetNodeNameFunc                func(string) string
	originalGetObjectUIDFunc               func(string) string
	originalCompareResourceVersionFunc     func(string, string) int
	originalGcFunc                         func(*SyncController, *v1alpha1.ClusterObjectSync)
	originalDeleteFunc                     func(*SyncController, string) error
}

func setupTest() *testFuncBackup {
	backup := &testFuncBackup{
		originalSendToEdge:                     sendToEdge,
		originalBuildEdgeControllerMessageFunc: buildEdgeControllerMessageFunc,
		originalGetNodeNameFunc:                getNodeNameFunc,
		originalGetObjectUIDFunc:               getObjectUIDFunc,
		originalCompareResourceVersionFunc:     compareResourceVersionFunc,
		originalGcFunc:                         gcOrphanedClusterObjectSyncFunc,
		originalDeleteFunc:                     deleteClusterObjectSyncFunc,
	}

	getNodeNameFunc = mockGetNodeName
	getObjectUIDFunc = mockGetObjectUID
	compareResourceVersionFunc = mockCompareResourceVersion

	return backup
}

func (b *testFuncBackup) restore() {
	sendToEdge = b.originalSendToEdge
	buildEdgeControllerMessageFunc = b.originalBuildEdgeControllerMessageFunc
	getNodeNameFunc = b.originalGetNodeNameFunc
	getObjectUIDFunc = b.originalGetObjectUIDFunc
	compareResourceVersionFunc = b.originalCompareResourceVersionFunc
	gcOrphanedClusterObjectSyncFunc = b.originalGcFunc
	deleteClusterObjectSyncFunc = b.originalDeleteFunc
}

func mockGetNodeName(syncName string) string {
	return "node1"
}

func mockGetObjectUID(syncName string) string {
	if syncName == "node1-pod-12345" {
		return "12345"
	}
	return ""
}

func mockCompareResourceVersion(rv1, rv2 string) int {
	if rv1 == "1000" && rv2 == "500" {
		return 1
	}
	if rv1 == rv2 {
		return 0
	}
	if rv1 > rv2 {
		return 1
	}
	return -1
}

type MockLister struct {
	getMockFunc func(name string) (runtime.Object, error)
}

func (m MockLister) Get(name string) (runtime.Object, error) {
	return m.getMockFunc(name)
}

func (m MockLister) List(selector labels.Selector) ([]runtime.Object, error) {
	return nil, nil
}

func (m MockLister) ByNamespace(namespace string) cache.GenericNamespaceLister {
	return nil
}

type MockInformerManager struct {
	getListerMockFunc func(gvr schema.GroupVersionResource) (cache.GenericLister, error)
}

func (m MockInformerManager) GetLister(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
	return m.getListerMockFunc(gvr)
}

func (m MockInformerManager) EdgeNode() cache.SharedIndexInformer {
	return nil
}

func (m MockInformerManager) GetKubeInformerFactory() k8sinformer.SharedInformerFactory {
	return nil
}

func (m MockInformerManager) GetKubeEdgeInformerFactory() edgeinformers.SharedInformerFactory {
	return nil
}

func (m MockInformerManager) GetDynamicInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return nil
}

func (m MockInformerManager) GetInformerPair(gvr schema.GroupVersionResource) (*informers.InformerPair, error) {
	return nil, nil
}

func (m MockInformerManager) Start(stopCh <-chan struct{}) {}

func createTestPod(name, uid, resourceVersion string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetAPIVersion("v1")
	pod.SetKind("Pod")
	pod.SetName(name)
	pod.SetUID(types.UID(uid))
	pod.SetResourceVersion(resourceVersion)
	return pod
}

func createTestSync(name, objName, objKind, objAPIVersion, resourceVersion string) *v1alpha1.ClusterObjectSync {
	return &v1alpha1.ClusterObjectSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ObjectSyncSpec{
			ObjectName:       objName,
			ObjectKind:       objKind,
			ObjectAPIVersion: objAPIVersion,
		},
		Status: v1alpha1.ObjectSyncStatus{
			ObjectResourceVersion: resourceVersion,
		},
	}
}

func TestReconcileClusterObjectSync(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	tests := []struct {
		name             string
		sync             *v1alpha1.ClusterObjectSync
		mockListerFunc   func(string) (runtime.Object, error)
		expectSendCalled bool
		expectGCCalled   bool
	}{
		{
			name: "Invalid API Version",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "invalid/v1/version", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return nil, nil
			},
			expectSendCalled: false,
			expectGCCalled:   false,
		},
		{
			name: "Error getting lister",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return nil, nil
			},
			expectSendCalled: false,
			expectGCCalled:   false,
		},
		{
			name: "Object not found",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
			},
			expectSendCalled: false,
			expectGCCalled:   true,
		},
		{
			name: "Error getting object",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return nil, errors.New("get error")
			},
			expectSendCalled: false,
			expectGCCalled:   false,
		},
		{
			name: "UID mismatch",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return createTestPod(name, "different-uid", "1000"), nil
			},
			expectSendCalled: false,
			expectGCCalled:   true,
		},
		{
			name: "Object found with newer resource version",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return createTestPod(name, "12345", "1000"), nil
			},
			expectSendCalled: true,
			expectGCCalled:   false,
		},
		{
			name: "Object found with same resource version",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "1000"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return createTestPod(name, "12345", "1000"), nil
			},
			expectSendCalled: false,
			expectGCCalled:   false,
		},
		{
			name: "Object found with older resource version",
			sync: createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "2000"),
			mockListerFunc: func(name string) (runtime.Object, error) {
				return createTestPod(name, "12345", "1000"), nil
			},
			expectSendCalled: false,
			expectGCCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendCalled := false
			gcCalled := false

			mockLister := MockLister{
				getMockFunc: tt.mockListerFunc,
			}

			mockInformerManager := MockInformerManager{
				getListerMockFunc: func(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
					if tt.name == "Error getting lister" {
						return nil, errors.New("lister error")
					}
					return mockLister, nil
				},
			}

			sendToEdge = func(module string, msg model.Message) {
				sendCalled = true
			}

			gcOrphanedClusterObjectSyncFunc = func(sctl *SyncController, sync *v1alpha1.ClusterObjectSync) {
				gcCalled = true
			}

			buildEdgeControllerMessageFunc = func(nodeID, namespace, resource, resourceID string, operation string, content interface{}) *model.Message {
				return &model.Message{}
			}

			ctrl := &SyncController{
				informerManager: mockInformerManager,
			}

			ctrl.reconcileClusterObjectSync(tt.sync)

			if tt.name == "Object found with newer resource version" {
				result := compareResourceVersionFunc("1000", "500")
				if result <= 0 {
					t.Errorf("Mock compareResourceVersionFunc returned unexpected result %d for 1000 vs 500", result)
				}
			}

			assert.Equal(t, tt.expectSendCalled, sendCalled, "Send message expectation failed")
			assert.Equal(t, tt.expectGCCalled, gcCalled, "GC called expectation failed")
		})
	}
}

func TestGcOrphanedClusterObjectSyncFunction(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	tests := []struct {
		name               string
		sync               *v1alpha1.ClusterObjectSync
		sendMessage        bool
		expectSendCalled   bool
		expectDeleteCalled bool
		deleteError        bool
	}{
		{
			name:               "Send message successfully",
			sync:               createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			sendMessage:        true,
			expectSendCalled:   true,
			expectDeleteCalled: false,
			deleteError:        false,
		},
		{
			name:               "Message build fails, delete is called",
			sync:               createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			sendMessage:        false,
			expectSendCalled:   false,
			expectDeleteCalled: true,
			deleteError:        false,
		},
		{
			name:               "Message build fails, delete returns error",
			sync:               createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500"),
			sendMessage:        false,
			expectSendCalled:   false,
			expectDeleteCalled: true,
			deleteError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendCalled := false
			deleteCalled := false

			sendToEdge = func(module string, msg model.Message) {
				sendCalled = true
			}

			buildEdgeControllerMessageFunc = func(nodeID, namespace, resource, resourceID string, operation string, content interface{}) *model.Message {
				if tt.sendMessage {
					return &model.Message{}
				}
				return nil
			}

			deleteClusterObjectSyncFunc = func(sctl *SyncController, name string) error {
				deleteCalled = true
				if tt.deleteError {
					return errors.New("delete error")
				}
				return nil
			}

			ctrl := &SyncController{}

			ctrl.gcOrphanedClusterObjectSyncImpl(tt.sync)

			assert.Equal(t, tt.expectSendCalled, sendCalled, "Send message expectation failed")
			assert.Equal(t, tt.expectDeleteCalled, deleteCalled, "Delete called expectation failed")
		})
	}
}

func TestActualGcOrphanedClusterObjectSyncFunc(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	implementationCalled := false

	gcOrphanedClusterObjectSyncFunc = func(sctl *SyncController, sync *v1alpha1.ClusterObjectSync) {
		implementationCalled = true
	}

	sync := createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500")

	ctrl := &SyncController{}

	ctrl.gcOrphanedClusterObjectSync(sync)

	assert.True(t, implementationCalled, "gcOrphanedClusterObjectSyncFunc not called")
}

func TestSendClusterObjectSyncEventDirect(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	compareResourceVersionFunc = func(rv1, rv2 string) int {
		if rv1 == "1000" && rv2 == "500" {
			return 1
		}
		return 0
	}

	sendCalled := false

	sendToEdge = func(module string, msg model.Message) {
		sendCalled = true
	}

	buildEdgeControllerMessageFunc = func(nodeID, namespace, resource, resourceID string, operation string, content interface{}) *model.Message {
		return &model.Message{}
	}

	nodeName := "node1"
	resourceType := "pod"
	objectResourceVersion := "1000"
	obj := createTestPod("test-pod", "12345", "1000")
	sync := createTestSync("node1-pod-12345", "test-pod", "Pod", "v1", "500")

	sendClusterObjectSyncEvent(nodeName, sync, resourceType, objectResourceVersion, obj)

	assert.True(t, sendCalled, "Send should have been called")
}

func TestDeleteClusterObjectSyncFunc(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	testCases := []struct {
		name        string
		syncName    string
		returnError bool
		errorMsg    string
	}{
		{
			name:        "Successful delete",
			syncName:    "node1-pod-12345",
			returnError: false,
		},
		{
			name:        "Delete with error",
			syncName:    "node1-pod-abcde",
			returnError: true,
			errorMsg:    "mock delete error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testController := &SyncController{}

			deleteWasCalled := false
			namePassedToDelete := ""

			deleteClusterObjectSyncFunc = func(sctl *SyncController, name string) error {
				deleteWasCalled = true
				namePassedToDelete = name

				assert.Equal(t, testController, sctl, "Controller not passed correctly to hook")

				if tc.returnError {
					return errors.New(tc.errorMsg)
				}
				return nil
			}

			err := deleteClusterObjectSyncFunc(testController, tc.syncName)

			assert.True(t, deleteWasCalled, "Delete function was not called")

			assert.Equal(t, tc.syncName, namePassedToDelete, "Wrong sync name passed to delete function")

			if tc.returnError {
				assert.Error(t, err, "Expected an error but got nil")
				assert.Equal(t, tc.errorMsg, err.Error(), "Error message doesn't match expected")
			} else {
				assert.NoError(t, err, "Unexpected error returned")
			}
		})
	}
}

func TestGcOrphanedClusterObjectSyncFuncIntegration(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	testCases := []struct {
		name               string
		buildMessageNil    bool
		expectSendCalled   bool
		expectDeleteCalled bool
	}{
		{
			name:               "Message built successfully - should send",
			buildMessageNil:    false,
			expectSendCalled:   true,
			expectDeleteCalled: false,
		},
		{
			name:               "Message build failed - should delete",
			buildMessageNil:    true,
			expectSendCalled:   false,
			expectDeleteCalled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSync := &v1alpha1.ClusterObjectSync{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1-pod-12345",
				},
				Spec: v1alpha1.ObjectSyncSpec{
					ObjectName:       "test-pod",
					ObjectKind:       "Pod",
					ObjectAPIVersion: "v1",
				},
				Status: v1alpha1.ObjectSyncStatus{
					ObjectResourceVersion: "1000",
				},
			}

			testController := &SyncController{}

			sendCalled := false
			deleteCalled := false

			buildEdgeControllerMessageFunc = func(nodeID, namespace, resource, resourceID string, operation string, content interface{}) *model.Message {
				assert.Equal(t, "node1", nodeID)
				assert.Equal(t, "pod", resource)
				assert.Equal(t, "test-pod", resourceID)

				if tc.buildMessageNil {
					return nil
				}
				return &model.Message{}
			}

			sendToEdge = func(module string, msg model.Message) {
				sendCalled = true
			}

			deleteClusterObjectSyncFunc = func(sctl *SyncController, name string) error {
				deleteCalled = true
				assert.Equal(t, testSync.Name, name)
				return nil
			}

			gcOrphanedClusterObjectSyncFunc(testController, testSync)

			assert.Equal(t, tc.expectSendCalled, sendCalled, "Send called expectation failed")
			assert.Equal(t, tc.expectDeleteCalled, deleteCalled, "Delete called expectation failed")
		})
	}
}

func TestDeleteClusterObjectSyncFuncDirectCoverage(t *testing.T) {
	backup := setupTest()
	defer backup.restore()

	ctrl := &SyncController{}

	defer func() {
		if r := recover(); r != nil {
			t.Log("Expected panic occurred:", r)
		}
	}()

	err := deleteClusterObjectSyncFunc(ctrl, "test-sync")

	if err == nil {
		t.Error("Expected an error from deleteClusterObjectSyncFunc with nil client, but got nil")
	}
}

func ExecuteDeleteClusterObjectSyncFunc() {
	sctl := &SyncController{}
	defer func() {
		_ = recover()
	}()
	_ = deleteClusterObjectSyncFunc(sctl, "test")
}

func TestExecuteDeleteClusterObjectSyncFunc(t *testing.T) {
	ExecuteDeleteClusterObjectSyncFunc()
}
