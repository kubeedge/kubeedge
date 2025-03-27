/*
Copyright 2024 The KubeEdge Authors.

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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/beehive/pkg/core/model"
	messagelayer "github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	edgeapi "github.com/kubeedge/kubeedge/common/types"

	authenticationv1 "k8s.io/api/authentication/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	defaultNamespace = "default"
	testNamespace    = "test-namespace"
	defaultNodeID    = "node-id"
)

type MockMessageLayer struct {
	ReceivedMessages []model.Message
	ResponseMessages []model.Message
	SendMessages     []model.Message
}

func (m *MockMessageLayer) Send(message model.Message) error {
	m.SendMessages = append(m.SendMessages, message)
	return nil
}

func (m *MockMessageLayer) Receive() (model.Message, error) {
	if len(m.ReceivedMessages) > 0 {
		msg := m.ReceivedMessages[0]
		m.ReceivedMessages = m.ReceivedMessages[1:]
		return msg, nil
	}
	return model.Message{}, errors.New("no messages")
}

func (m *MockMessageLayer) Response(message model.Message) error {
	m.ResponseMessages = append(m.ResponseMessages, message)
	return nil
}

var defaultConf = v1alpha1.NewDefaultCloudCoreConfig()
var UC *UpstreamController
var mockMessageLayer *MockMessageLayer

func ToInt64(i int64) *int64 {
	return &i
}

func ToString(s string) *string {
	return &s
}

func ToInt32(i int32) *int32 {
	return &i
}

type MockCRDClient struct {
	RuleObj *rulesv1.Rule
}

func (m *MockCRDClient) AppsV1alpha1() interface{}    { return nil }
func (m *MockCRDClient) CoreV1alpha1() interface{}    { return nil }
func (m *MockCRDClient) DevicesV1alpha2() interface{} { return nil }
func (m *MockCRDClient) DevicesV1alpha1() interface{} { return nil }

type MockRulesV1Interface struct {
	rule *rulesv1.Rule
}

func (m *MockRulesV1Interface) RESTClient() interface{} { return nil }

type MockRulesInterface struct {
	rule *rulesv1.Rule
}

func (m *MockRulesInterface) Create(ctx context.Context, rule *rulesv1.Rule, opts metav1.CreateOptions) (*rulesv1.Rule, error) {
	return m.rule, nil
}

func (m *MockRulesInterface) Update(ctx context.Context, rule *rulesv1.Rule, opts metav1.UpdateOptions) (*rulesv1.Rule, error) {
	return m.rule, nil
}

func (m *MockRulesInterface) UpdateStatus(ctx context.Context, rule *rulesv1.Rule, opts metav1.UpdateOptions) (*rulesv1.Rule, error) {
	return m.rule, nil
}

func (m *MockRulesInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return nil
}

func (m *MockRulesInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return nil
}

func (m *MockRulesInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*rulesv1.Rule, error) {
	return m.rule, nil
}

func (m *MockRulesInterface) List(ctx context.Context, opts metav1.ListOptions) (*rulesv1.RuleList, error) {
	return &rulesv1.RuleList{}, nil
}

func (m *MockRulesInterface) Watch(ctx context.Context, opts metav1.ListOptions) (interface{}, error) {
	return nil, nil
}

func (m *MockRulesInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*rulesv1.Rule, error) {
	return m.rule, nil
}

func setupTest(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	factory := informers.NewSharedInformerFactory(kubeClient, 0)

	factory.Core().V1().Nodes().Informer().GetStore()
	factory.Core().V1().Pods().Informer().GetStore()
	factory.Core().V1().Secrets().Informer().GetStore()
	factory.Core().V1().ConfigMaps().Informer().GetStore()
	factory.Coordination().V1().Leases().Informer().GetStore()

	var err error
	UC, err = NewUpstreamController(defaultConf.Modules.EdgeController, factory)
	if err != nil {
		t.Fatalf("Failed to create UpstreamController: %v", err)
	}

	UC.kubeClient = kubeClient

	mockMessageLayer = &MockMessageLayer{
		ReceivedMessages: []model.Message{},
		ResponseMessages: []model.Message{},
		SendMessages:     []model.Message{},
	}
	UC.messageLayer = mockMessageLayer

	UC.eventChan = make(chan model.Message, 20)
	UC.nodeStatusChan = make(chan model.Message, 20)
	UC.podStatusChan = make(chan model.Message, 20)
	UC.configMapChan = make(chan model.Message, 20)
	UC.secretChan = make(chan model.Message, 20)
	UC.createNodeChan = make(chan model.Message, 20)
	UC.podDeleteChan = make(chan model.Message, 20)
	UC.patchNodeChan = make(chan model.Message, 20)
	UC.patchPodChan = make(chan model.Message, 20)
	UC.certificasesSigningRequestChan = make(chan model.Message, 20)
	UC.serviceAccountTokenChan = make(chan model.Message, 20)
	UC.createLeaseChan = make(chan model.Message, 20)
	UC.queryLeaseChan = make(chan model.Message, 20)
	UC.ruleStatusChan = make(chan model.Message, 20)
	UC.createPodChan = make(chan model.Message, 20)
	UC.persistentVolumeChan = make(chan model.Message, 20)
	UC.persistentVolumeClaimChan = make(chan model.Message, 20)
	UC.volumeAttachmentChan = make(chan model.Message, 20)

	defaultNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: defaultNamespace},
	}
	_, err = kubeClient.CoreV1().Namespaces().Create(context.Background(), defaultNs, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("Failed to create default namespace: %v", err)
	}

	go UC.processEvent()
	go UC.updateNodeStatus()
	go UC.updatePodStatus()
	go UC.registerNode()
	go UC.deletePod()
	go UC.patchNode()
	go UC.patchPod()
	go UC.processCSR()
	go UC.processServiceAccountToken()
	go UC.createOrUpdateLease()
	go UC.queryLease()
	go UC.updateRuleStatus()
	go UC.createPod()
	go UC.querySecret()
	go UC.queryConfigMap()
	go UC.queryPersistentVolume()
	go UC.queryPersistentVolumeClaim()
	go UC.queryVolumeAttachment()
}

var Events = []*corev1.Event{
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "InsertEvent", Namespace: ""},
		Reason:     "insert",
		Message:    "Insert from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "UpdateEvent", Namespace: ""},
		Reason:     "update",
		Message:    "Update from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "PatchEvent", Namespace: ""},
		Reason:     "insert",
		Message:    "Preparation: Insert from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "PatchEvent", Namespace: ""},
		Reason:     "patch",
		Message:    "Patch from BIT-CCS group to Kubeedge team",
	},
}

func TestQueryConfigMap(t *testing.T) {
	setupTest(t)

	cmName := "test-configmap"
	namespace := defaultNamespace

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"test-key": "test-value",
		},
	}

	_, err := UC.kubeClient.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test configmap: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, namespace, model.ResourceTypeConfigmap, cmName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-query-configmap"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
	}

	var receivedResp bool
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(200 * time.Millisecond)
			for _, resp := range mockMessageLayer.ResponseMessages {
				if resp.GetParentID() == msg.GetID() {
					receivedResp = true
					return
				}
			}
		}
	}()

	UC.configMapChan <- msg
	time.Sleep(1000 * time.Millisecond)

	if !receivedResp {
		t.Errorf("Did not receive configmap query response")
	}
}

func TestQuerySecret(t *testing.T) {
	setupTest(t)

	secretName := "test-secret"
	namespace := defaultNamespace

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
	}

	_, err := UC.kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test secret: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, namespace, model.ResourceTypeSecret, secretName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-query-secret"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
	}

	UC.secretChan <- msg
	time.Sleep(1000 * time.Millisecond)

	found := false
	for _, respMsg := range mockMessageLayer.ResponseMessages {
		if respMsg.GetParentID() == msg.GetID() {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected response message for node patch was not sent")
	}
}

func TestDeletePod(t *testing.T) {
	setupTest(t)

	podName := "test-delete-pod"
	podNamespace := testNamespace
	podUID := types.UID("pod-uid-delete-123")

	_, _ = UC.kubeClient.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: podNamespace}}, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
			UID:       podUID,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	_, err := UC.kubeClient.CoreV1().Pods(podNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	nodeID := defaultNodeID
	resource := "node/" + nodeID + "/" + podNamespace + "/" + model.ResourceTypePod + "/" + podName

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-delete-pod"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.DeleteOperation,
		},
		Content: string(podUID),
	}

	UC.podDeleteChan <- msg
	time.Sleep(500 * time.Millisecond)

	_, err = UC.kubeClient.CoreV1().Pods(podNamespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("Pod should have been deleted but still exists")
	}

	found := false
	for _, respMsg := range mockMessageLayer.ResponseMessages {
		if respMsg.GetOperation() == model.ResponseOperation && respMsg.GetParentID() == msg.GetID() {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected response message for pod deletion was not sent")
	}
}

func TestPatchPod(t *testing.T) {
	setupTest(t)

	podName := "test-patch-pod"
	podNamespace := testNamespace

	_, _ = UC.kubeClient.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: podNamespace}}, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	_, err := UC.kubeClient.CoreV1().Pods(podNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	patchData := []byte(`{"status":{"phase":"Running"}}`)

	nodeID := defaultNodeID
	resource := "node/" + nodeID + "/" + podNamespace + "/" + model.ResourceTypePodPatch + "/" + podName

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-patch-pod"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.PatchOperation,
		},
		Content: string(patchData),
	}

	UC.patchPodChan <- msg
	time.Sleep(500 * time.Millisecond)

	found := false
	for _, respMsg := range mockMessageLayer.ResponseMessages {
		if respMsg.GetOperation() == model.ResponseOperation && respMsg.GetParentID() == msg.GetID() {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected response message for pod patch was not sent")
	}
}

func TestMain(m *testing.M) {
	defaultConf.Modules.EdgeController.Enable = true
	m.Run()
}

func TestDispatchMessage(t *testing.T) {
	setupTest(t)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-dispatch-podStatus"},
		Router: model.MessageRoute{
			Resource:  "node/node1/default/podstatus/test-pod",
			Operation: model.UpdateOperation,
		},
	}

	resourceType, err := messagelayer.GetResourceType(msg)
	if err != nil {
		t.Fatalf("Failed to get resource type: %v", err)
	}

	if resourceType != model.ResourceTypePodStatus {
		t.Errorf("Got incorrect resource type: %s, expected: %s", resourceType, model.ResourceTypePodStatus)
	}

	typeTests := []struct {
		resource string
		expected string
	}{
		{"node/node1/default/event/test-event", model.ResourceTypeEvent},
		{"node/node1/default/configmap/test-cm", model.ResourceTypeConfigmap},
		{"node/node1/default/secret/test-secret", model.ResourceTypeSecret},
		{"node/node1/default/node/test-node", model.ResourceTypeNode},
		{"node/node1/default/nodestatus/test-node", model.ResourceTypeNodeStatus},
		{"node/node1/default/nodepatch/test-node", model.ResourceTypeNodePatch},
		{"node/node1/default/podpatch/test-pod", model.ResourceTypePodPatch},
	}

	for _, tt := range typeTests {
		testMsg := model.Message{
			Router: model.MessageRoute{
				Resource: tt.resource,
			},
		}

		gotType, err := messagelayer.GetResourceType(testMsg)
		if err != nil {
			t.Errorf("Failed to get resource type for %s: %v", tt.resource, err)
			continue
		}

		if gotType != tt.expected {
			t.Errorf("For resource %s: got type %s, expected %s", tt.resource, gotType, tt.expected)
		}
	}
}

func TestPodStatusErrorPaths(t *testing.T) {
	setupTest(t)

	podName := "test-pod-err"
	podNamespace := testNamespace
	podUID := types.UID("pod-uid-err-123")
	wrongUID := types.UID("wrong-uid-456")

	_, _ = UC.kubeClient.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: podNamespace}}, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
			UID:       podUID,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	_, err := UC.kubeClient.CoreV1().Pods(podNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	// Test case 1: Pod not found
	podStatus1 := edgeapi.PodStatusRequest{
		Name: "non-existent-pod",
		UID:  podUID,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	podStatusData1, _ := json.Marshal(podStatus1)

	msg1 := model.Message{
		Header: model.MessageHeader{ID: "test-pod-status-not-found"},
		Router: model.MessageRoute{
			Resource:  fmt.Sprintf("node/node-id/%s/%s/%s", podNamespace, model.ResourceTypePodStatus, "non-existent-pod"),
			Operation: model.UpdateOperation,
		},
		Content: string(podStatusData1),
	}

	UC.podStatusChan <- msg1
	time.Sleep(500 * time.Millisecond)

	// Test case 2: UID mismatch
	podStatus2 := edgeapi.PodStatusRequest{
		Name: podName,
		UID:  wrongUID,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	podStatusData2, _ := json.Marshal(podStatus2)

	msg2 := model.Message{
		Header: model.MessageHeader{ID: "test-pod-status-uid-mismatch"},
		Router: model.MessageRoute{
			Resource:  fmt.Sprintf("node/node-id/%s/%s/%s", podNamespace, model.ResourceTypePodStatus, podName),
			Operation: model.UpdateOperation,
		},
		Content: string(podStatusData2),
	}

	UC.podStatusChan <- msg2
	time.Sleep(500 * time.Millisecond)

	// Test case 3: Pod with deletionTimestamp and terminal phase
	pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	_, err = UC.kubeClient.CoreV1().Pods(podNamespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update pod with deletion timestamp: %v", err)
	}

	podStatus3 := edgeapi.PodStatusRequest{
		Name: podName,
		UID:  podUID,
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test-container",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   0,
							FinishedAt: metav1.Now(),
						},
					},
				},
			},
		},
	}

	podStatusData3, _ := json.Marshal(podStatus3)

	msg3 := model.Message{
		Header: model.MessageHeader{ID: "test-pod-status-terminating"},
		Router: model.MessageRoute{
			Resource:  fmt.Sprintf("node/node-id/%s/%s/%s", podNamespace, model.ResourceTypePodStatus, podName),
			Operation: model.UpdateOperation,
		},
		Content: string(podStatusData3),
	}

	UC.podStatusChan <- msg3
	time.Sleep(500 * time.Millisecond)
}

func TestSortedContainerStatuses(t *testing.T) {
	containerStatuses := SortedContainerStatuses{
		{Name: "c"},
		{Name: "a"},
		{Name: "b"},
	}

	if containerStatuses.Len() != 3 {
		t.Errorf("SortedContainerStatuses.Len() = %d, want 3", containerStatuses.Len())
	}

	if !containerStatuses.Less(1, 0) {
		t.Errorf("SortedContainerStatuses.Less(1, 0) = false, want true")
	}

	if !containerStatuses.Less(1, 2) {
		t.Errorf("SortedContainerStatuses.Less(1, 2) = false, want true")
	}

	if containerStatuses.Less(0, 1) {
		t.Errorf("SortedContainerStatuses.Less(0, 1) = true, want false")
	}

	containerStatuses.Swap(0, 1)
	if containerStatuses[0].Name != "a" || containerStatuses[1].Name != "c" {
		t.Errorf("SortedContainerStatuses.Swap(0, 1) failed, got %v", containerStatuses)
	}

	sort.Sort(containerStatuses)
	if containerStatuses[0].Name != "a" || containerStatuses[1].Name != "b" || containerStatuses[2].Name != "c" {
		t.Errorf("sort.Sort(containerStatuses) failed, got %v", containerStatuses)
	}
}

func TestSortInitContainerStatuses(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "init1"},
				{Name: "init2"},
				{Name: "init3"},
			},
		},
	}

	statuses := []corev1.ContainerStatus{
		{Name: "init3"},
		{Name: "init1"},
		{Name: "init2"},
	}

	SortInitContainerStatuses(pod, statuses)

	if statuses[0].Name != "init1" || statuses[1].Name != "init2" || statuses[2].Name != "init3" {
		t.Errorf("SortInitContainerStatuses failed, got %v", statuses)
	}
}

func TestPersistentVolumeOperations(t *testing.T) {
	setupTest(t)

	pvName := "test-pv"

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/test-path",
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}

	_, err := UC.kubeClient.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test PV: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, "default", "persistentvolume", pvName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-query-pv"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
	}

	UC.persistentVolumeChan <- msg
	time.Sleep(1000 * time.Millisecond)
}

func TestVolumeAttachmentOperations(t *testing.T) {
	setupTest(t)

	attachmentName := "test-attachment"
	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, "default", "volumeattachment", attachmentName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-query-attachment"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
	}

	UC.volumeAttachmentChan <- msg
	time.Sleep(1000 * time.Millisecond)
}

func TestServiceAccountToken(t *testing.T) {
	setupTest(t)

	saName := "test-serviceaccount"
	namespace := defaultNamespace

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
		},
	}

	_, err := UC.kubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test service account: %v", err)
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ToInt64(3600),
		},
	}

	tokenRequestData, err := json.Marshal(tokenRequest)
	if err != nil {
		t.Fatalf("Failed to marshal token request: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, namespace, model.ResourceTypeServiceAccountToken, saName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-sa-token"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
		Content: string(tokenRequestData),
	}

	UC.serviceAccountTokenChan <- msg
	time.Sleep(1000 * time.Millisecond)
}

func TestUpdateNodeStatus(t *testing.T) {
	setupTest(t)

	nodeName := "test-status-node"
	nodeID := "node-status-id"

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}

	_, err := UC.kubeClient.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	nodeStatus := &edgeapi.NodeStatusRequest{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:               corev1.NodeReady,
					Status:             corev1.ConditionTrue,
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	nodeStatusData, err := json.Marshal(nodeStatus)
	if err != nil {
		t.Fatalf("Failed to marshal node status: %v", err)
	}

	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, "default", model.ResourceTypeNodeStatus, nodeName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-update-node-status"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.UpdateOperation,
		},
		Content: string(nodeStatusData),
	}

	UC.nodeStatusChan <- msg
	time.Sleep(1000 * time.Millisecond)
}

func TestProcessCSR(t *testing.T) {
	setupTest(t)

	csrName := "test-csr"

	csr := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: csrName,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request: []byte(base64.StdEncoding.EncodeToString([]byte("test-csr-data"))),
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageServerAuth,
			},
		},
	}

	csrData, err := json.Marshal(csr)
	if err != nil {
		t.Fatalf("Failed to marshal CSR: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, "default", model.ResourceTypeCSR, csrName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-create-csr"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.InsertOperation,
		},
		Content: string(csrData),
	}

	UC.certificasesSigningRequestChan <- msg
	time.Sleep(1000 * time.Millisecond)
}

func TestLeaseOperations(t *testing.T) {
	setupTest(t)

	leaseName := "test-lease"
	namespace := defaultNamespace

	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       ToString("test-holder"),
			LeaseDurationSeconds: ToInt32(60),
		},
	}

	leaseData, err := json.Marshal(lease)
	if err != nil {
		t.Fatalf("Failed to marshal lease: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, namespace, model.ResourceTypeLease, leaseName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-create-lease"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.InsertOperation,
		},
		Content: string(leaseData),
	}

	UC.createLeaseChan <- msg
	time.Sleep(1000 * time.Millisecond)

	queryMsg := model.Message{
		Header: model.MessageHeader{ID: "test-query-lease"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.QueryOperation,
		},
	}

	UC.queryLeaseChan <- queryMsg
	time.Sleep(1000 * time.Millisecond)
}

func TestCreatePod(t *testing.T) {
	setupTest(t)

	podName := "test-create-pod"
	namespace := defaultNamespace

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	podData, err := json.Marshal(pod)
	if err != nil {
		t.Fatalf("Failed to marshal pod: %v", err)
	}

	nodeID := defaultNodeID
	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, namespace, model.ResourceTypePod, podName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-create-pod"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.InsertOperation,
		},
		Content: string(podData),
	}

	UC.createPodChan <- msg
	time.Sleep(1000 * time.Millisecond)

	createdPod, err := UC.kubeClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get created pod: %v", err)
	} else if createdPod.Name != podName {
		t.Errorf("Pod name mismatch, expected %s, got %s", podName, createdPod.Name)
	}
}

func TestEventReport(t *testing.T) {
	setupTest(t)

	singleEvent := &corev1.Event{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "InsertEvent", Namespace: "default"},
		Reason:     "insert",
		Message:    "Insert from BIT-CCS group to Kubeedge team",
	}

	eventData, err := json.Marshal(singleEvent)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	resource := "node/node1/default/" + model.ResourceTypeEvent + "/InsertEvent"

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-event-insert"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.InsertOperation,
		},
		Content: string(eventData),
	}

	UC.eventChan <- msg
	time.Sleep(200 * time.Millisecond)

	result, err := UC.kubeClient.CoreV1().Events("default").Get(context.Background(), "InsertEvent", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get created event: %v", err)
	} else if result.Name != "InsertEvent" {
		t.Errorf("Event name mismatch, expected InsertEvent, got %s", result.Name)
	}
}

func TestCreateNode(t *testing.T) {
	setupTest(t)

	nodeName := "test-node"
	nodeID := "node-id-123"

	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"test-label":         "test-value",
				"kubernetes.io/role": "edge",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:               corev1.NodeReady,
					Status:             corev1.ConditionTrue,
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	nodeData, err := json.Marshal(testNode)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	resource := fmt.Sprintf("node/%s/%s/%s/%s", nodeID, "default", model.ResourceTypeNode, nodeName)

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-create-node"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.InsertOperation,
		},
		Content: string(nodeData),
	}

	UC.createNodeChan <- msg
	time.Sleep(1000 * time.Millisecond)

	createdNode, err := UC.kubeClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created node: %v", err)
	}

	if createdNode.Name != nodeName {
		t.Errorf("Node name mismatch, expected %s, got %s", nodeName, createdNode.Name)
	}

	found := false
	for _, respMsg := range mockMessageLayer.ResponseMessages {
		if respMsg.GetParentID() == msg.GetID() {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected response message for node creation was not sent")
	}
}

func TestUpdatePodStatus(t *testing.T) {
	setupTest(t)

	podName := "test-pod"
	podNamespace := testNamespace
	podUID := types.UID("pod-uid-123")

	_, _ = UC.kubeClient.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: podNamespace}}, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
			UID:       podUID,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	_, err := UC.kubeClient.CoreV1().Pods(podNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	podStatus := edgeapi.PodStatusRequest{
		Name: podName,
		UID:  podUID,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test-container",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
					Ready: true,
				},
			},
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	podStatusData, err := json.Marshal(podStatus)
	if err != nil {
		t.Fatalf("Failed to marshal pod status: %v", err)
	}

	nodeID := defaultNodeID
	resource := "node/" + nodeID + "/" + podNamespace + "/" + model.ResourceTypePodStatus + "/" + podName

	msg := model.Message{
		Header: model.MessageHeader{ID: "test-update-pod-status"},
		Router: model.MessageRoute{
			Resource:  resource,
			Operation: model.UpdateOperation,
		},
		Content: string(podStatusData),
	}

	UC.podStatusChan <- msg
	time.Sleep(500 * time.Millisecond)

	updatedPod, err := UC.kubeClient.CoreV1().Pods(podNamespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	if updatedPod.Status.Phase != corev1.PodRunning {
		t.Errorf("Pod phase mismatch, expected %s, got %s", corev1.PodRunning, updatedPod.Status.Phase)
	}
}
