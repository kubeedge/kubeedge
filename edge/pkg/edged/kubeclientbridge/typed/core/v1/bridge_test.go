/*
Copyright 2026 The KubeEdge Authors.

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

package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type mockPodsInterface struct {
	pod *corev1.Pod
	err error
}

func (m *mockPodsInterface) Create(pod *corev1.Pod) (*corev1.Pod, error) {
	return pod, m.err
}

func (m *mockPodsInterface) Update(pod *corev1.Pod) error {
	return m.err
}

func (m *mockPodsInterface) Delete(name string, opts metav1.DeleteOptions) error {
	return m.err
}

func (m *mockPodsInterface) Get(name string) (*corev1.Pod, error) {
	return m.pod, m.err
}

func (m *mockPodsInterface) Patch(name string, ptData []byte) (*corev1.Pod, error) {
	return m.pod, m.err
}

type mockNodesInterface struct {
	node *corev1.Node
	err  error
}

func (m *mockNodesInterface) Create(node *corev1.Node) (*corev1.Node, error) {
	return node, m.err
}

func (m *mockNodesInterface) Update(node *corev1.Node) error {
	return m.err
}

func (m *mockNodesInterface) Delete(name string) error {
	return m.err
}

func (m *mockNodesInterface) Get(name string) (*corev1.Node, error) {
	return m.node, m.err
}

func (m *mockNodesInterface) Patch(name string, ptData []byte) (*corev1.Node, error) {
	return m.node, m.err
}

type mockConfigMapsInterface struct {
	configMap *corev1.ConfigMap
	err       error
}

func (m *mockConfigMapsInterface) Create(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return cm, m.err
}

func (m *mockConfigMapsInterface) Update(cm *corev1.ConfigMap) error {
	return m.err
}

func (m *mockConfigMapsInterface) Delete(name string) error {
	return m.err
}

func (m *mockConfigMapsInterface) Get(name string) (*corev1.ConfigMap, error) {
	return m.configMap, m.err
}

type mockSecretsInterface struct {
	secret *corev1.Secret
	err    error
}

func (m *mockSecretsInterface) Create(s *corev1.Secret) (*corev1.Secret, error) {
	return s, m.err
}

func (m *mockSecretsInterface) Update(s *corev1.Secret) error {
	return m.err
}

func (m *mockSecretsInterface) Delete(name string) error {
	return m.err
}

func (m *mockSecretsInterface) Get(name string) (*corev1.Secret, error) {
	return m.secret, m.err
}

type mockPersistentVolumesInterface struct {
	pv  *corev1.PersistentVolume
	err error
}

func (m *mockPersistentVolumesInterface) Create(s *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	return s, m.err
}

func (m *mockPersistentVolumesInterface) Update(s *corev1.PersistentVolume) error {
	return m.err
}

func (m *mockPersistentVolumesInterface) Delete(name string) error {
	return m.err
}

func (m *mockPersistentVolumesInterface) Get(name string, options metav1.GetOptions) (*corev1.PersistentVolume, error) {
	return m.pv, m.err
}

type mockPersistentVolumeClaimsInterface struct {
	pvc *corev1.PersistentVolumeClaim
	err error
}

func (m *mockPersistentVolumeClaimsInterface) Create(s *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	return s, m.err
}

func (m *mockPersistentVolumeClaimsInterface) Update(s *corev1.PersistentVolumeClaim) error {
	return m.err
}

func (m *mockPersistentVolumeClaimsInterface) Delete(name string) error {
	return m.err
}

func (m *mockPersistentVolumeClaimsInterface) Get(name string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
	return m.pvc, m.err
}

type mockServiceAccountInterface struct {
	sa  *corev1.ServiceAccount
	err error
}

func (m *mockServiceAccountInterface) Create(s *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	return s, m.err
}

func (m *mockServiceAccountInterface) Update(s *corev1.ServiceAccount) error {
	return m.err
}

func (m *mockServiceAccountInterface) Delete(name string) error {
	return m.err
}

func (m *mockServiceAccountInterface) Get(name string) (*corev1.ServiceAccount, error) {
	return m.sa, m.err
}

type mockEventsInterface struct {
	event *corev1.Event
	err   error
}

func (m *mockEventsInterface) Create(s *corev1.Event, opts metav1.CreateOptions) (*corev1.Event, error) {
	return s, m.err
}

func (m *mockEventsInterface) Update(s *corev1.Event, opts metav1.UpdateOptions) (*corev1.Event, error) {
	return s, m.err
}

func (m *mockEventsInterface) Delete(name string, opts metav1.DeleteOptions) error {
	return m.err
}

func (m *mockEventsInterface) Get(name string, opts metav1.GetOptions) (*corev1.Event, error) {
	return m.event, m.err
}

func (m *mockEventsInterface) Apply(event *corev1apply.EventApplyConfiguration, opts metav1.ApplyOptions) (*corev1.Event, error) {
	return m.event, m.err
}

func (m *mockEventsInterface) Patch(name string, ptData types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*corev1.Event, error) {
	return m.event, m.err
}

func (m *mockEventsInterface) CreateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	return event, m.err
}

func (m *mockEventsInterface) UpdateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	return event, m.err
}

func (m *mockEventsInterface) PatchWithEventNamespace(event *corev1.Event, data []byte) (*corev1.Event, error) {
	return event, m.err
}

type mockCoreMetaClient struct {
	pods                   client.PodsInterface
	nodes                  client.NodesInterface
	configMaps             client.ConfigMapsInterface
	secrets                client.SecretsInterface
	persistentVolumes      client.PersistentVolumesInterface
	persistentVolumeClaims client.PersistentVolumeClaimsInterface
	serviceAccounts        client.ServiceAccountInterface
	events                 client.EventsInterface
}

func (m *mockCoreMetaClient) Pods(ns string) client.PodsInterface { return m.pods }
func (m *mockCoreMetaClient) Nodes(ns string) client.NodesInterface { return m.nodes }
func (m *mockCoreMetaClient) ConfigMaps(ns string) client.ConfigMapsInterface {
	return m.configMaps
}
func (m *mockCoreMetaClient) Secrets(ns string) client.SecretsInterface { return m.secrets }
func (m *mockCoreMetaClient) PersistentVolumes() client.PersistentVolumesInterface {
	return m.persistentVolumes
}
func (m *mockCoreMetaClient) PersistentVolumeClaims(ns string) client.PersistentVolumeClaimsInterface {
	return m.persistentVolumeClaims
}
func (m *mockCoreMetaClient) ServiceAccounts(ns string) client.ServiceAccountInterface {
	return m.serviceAccounts
}
func (m *mockCoreMetaClient) Events(ns string) client.EventsInterface { return m.events }
func (m *mockCoreMetaClient) NodeStatus(ns string) client.NodeStatusInterface  { return nil }
func (m *mockCoreMetaClient) PodStatus(ns string) client.PodStatusInterface    { return nil }
func (m *mockCoreMetaClient) VolumeAttachments(ns string) client.VolumeAttachmentsInterface {
	return nil
}
func (m *mockCoreMetaClient) ServiceAccountToken() client.ServiceAccountTokenInterface { return nil }
func (m *mockCoreMetaClient) Leases(ns string) client.LeasesInterface                 { return nil }
func (m *mockCoreMetaClient) CertificateSigningRequests() client.CertificateSigningRequestInterface {
	return nil
}

func TestPodsBridge(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
	mockMetaClient := &mockCoreMetaClient{
		pods: &mockPodsInterface{pod: pod},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &PodsBridge{
		PodInterface: fake.Pods("default"),
		ns:           "default",
		MetaClient:   mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-pod", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, pod, got)

	got, err = bridge.Create(context.Background(), pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, pod, got)

	got, err = bridge.Patch(context.Background(), "test-pod", types.StrategicMergePatchType, []byte(""), metav1.PatchOptions{})
	assert.NoError(t, err)
	assert.Equal(t, pod, got)

	err = bridge.Delete(context.Background(), "test-pod", metav1.DeleteOptions{})
	assert.NoError(t, err)
}

func TestNodesBridge(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}}
	mockMetaClient := &mockCoreMetaClient{
		nodes: &mockNodesInterface{node: node},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &NodesBridge{
		NodeInterface: fake.Nodes(),
		MetaClient:    mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-node", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, node, got)

	got, err = bridge.Create(context.Background(), node, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, node, got)

	got, err = bridge.Update(context.Background(), node, metav1.UpdateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, node, got)

	got, err = bridge.Patch(context.Background(), "test-node", types.StrategicMergePatchType, []byte(""), metav1.PatchOptions{})
	assert.NoError(t, err)
	assert.Equal(t, node, got)
}

func TestConfigMapBridge(t *testing.T) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-cm"}}
	mockMetaClient := &mockCoreMetaClient{
		configMaps: &mockConfigMapsInterface{configMap: cm},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &ConfigMapBridge{
		ConfigMapInterface: fake.ConfigMaps("default"),
		ns:                 "default",
		MetaClient:         mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-cm", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, cm, got)
}

func TestSecretBridge(t *testing.T) {
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-s"}}
	mockMetaClient := &mockCoreMetaClient{
		secrets: &mockSecretsInterface{secret: s},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &SecretBridge{
		SecretInterface: fake.Secrets("default"),
		ns:              "default",
		MetaClient:      mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-s", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, s, got)
}

func TestPersistentVolumeBridge(t *testing.T) {
	pv := &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "test-pv"}}
	mockMetaClient := &mockCoreMetaClient{
		persistentVolumes: &mockPersistentVolumesInterface{pv: pv},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &PersistentVolumesBridge{
		PersistentVolumeInterface: fake.PersistentVolumes(),
		MetaClient:                mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-pv", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, pv, got)
}

func TestPersistentVolumeClaimBridge(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-pvc"}}
	mockMetaClient := &mockCoreMetaClient{
		persistentVolumeClaims: &mockPersistentVolumeClaimsInterface{pvc: pvc},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &PersistentVolumeClaimsBridge{
		PersistentVolumeClaimInterface: fake.PersistentVolumeClaims("default"),
		ns:                             "default",
		MetaClient:                     mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-pvc", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, pvc, got)
}

func TestServiceAccountBridge(t *testing.T) {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "test-sa"}}
	mockMetaClient := &mockCoreMetaClient{
		serviceAccounts: &mockServiceAccountInterface{sa: sa},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &ServiceAccountsBridge{
		ServiceAccountInterface: fake.ServiceAccounts("default"),
		ns:                      "default",
		MetaClient:              mockMetaClient,
	}

	got, err := bridge.Get(context.Background(), "test-sa", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, sa, got)
}

func TestEventBridge(t *testing.T) {
	event := &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "test-event"}}
	mockMetaClient := &mockCoreMetaClient{
		events: &mockEventsInterface{event: event},
	}
	fake := fakecorev1.FakeCoreV1{}
	bridge := &EventsBridge{
		EventInterface: fake.Events("default"),
		ns:             "default",
		MetaClient:     mockMetaClient,
	}

	got, err := bridge.CreateWithEventNamespace(event)
	assert.NoError(t, err)
	assert.Equal(t, event, got)

	got, err = bridge.PatchWithEventNamespace(event, []byte(""))
	assert.NoError(t, err)
	assert.Equal(t, event, got)
}
