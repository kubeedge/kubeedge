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

package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type mockVolumeAttachmentsInterface struct {
	volumeAttachment *storagev1.VolumeAttachment
	err              error
}

func (f *mockVolumeAttachmentsInterface) Create(*storagev1.VolumeAttachment) (*storagev1.VolumeAttachment, error) {
	return nil, nil
}

func (f *mockVolumeAttachmentsInterface) Update(*storagev1.VolumeAttachment) error {
	return nil
}

func (f *mockVolumeAttachmentsInterface) Delete(_ string) error {
	return nil
}

func (f *mockVolumeAttachmentsInterface) Get(_ string, _ metav1.GetOptions) (*storagev1.VolumeAttachment, error) {
	return f.volumeAttachment, f.err
}

type mockMetaClient struct {
	volumeAttachments client.VolumeAttachmentsInterface
}

func (f *mockMetaClient) VolumeAttachments(_ string) client.VolumeAttachmentsInterface {
	return f.volumeAttachments
}
func (f *mockMetaClient) Events(string) client.EventsInterface                     { return nil }
func (f *mockMetaClient) Pods(string) client.PodsInterface                         { return nil }
func (f *mockMetaClient) PodStatus(string) client.PodStatusInterface               { return nil }
func (f *mockMetaClient) ConfigMaps(string) client.ConfigMapsInterface             { return nil }
func (f *mockMetaClient) Nodes(string) client.NodesInterface                       { return nil }
func (f *mockMetaClient) NodeStatus(string) client.NodeStatusInterface             { return nil }
func (f *mockMetaClient) Secrets(string) client.SecretsInterface                   { return nil }
func (f *mockMetaClient) ServiceAccountToken() client.ServiceAccountTokenInterface { return nil }
func (f *mockMetaClient) ServiceAccounts(string) client.ServiceAccountInterface    { return nil }
func (f *mockMetaClient) PersistentVolumes() client.PersistentVolumesInterface     { return nil }
func (f *mockMetaClient) PersistentVolumeClaims(string) client.PersistentVolumeClaimsInterface {
	return nil
}
func (f *mockMetaClient) Leases(string) client.LeasesInterface { return nil }
func (f *mockMetaClient) CertificateSigningRequests() client.CertificateSigningRequestInterface {
	return nil
}

func TestGet(t *testing.T) {
	assert := assert.New(t)

	expectedVolumeAttachment := &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-volume-attachment",
		},
	}

	mockVolumeAttachments := &mockVolumeAttachmentsInterface{
		volumeAttachment: expectedVolumeAttachment,
		err:              nil,
	}

	mockMetaClient := &mockMetaClient{
		volumeAttachments: mockVolumeAttachments,
	}

	volumeAttachmentsBridge := &VolumeAttachmentsBridge{
		FakeVolumeAttachments: fakestoragev1.FakeVolumeAttachments{},
		MetaClient:            mockMetaClient,
	}

	result, err := volumeAttachmentsBridge.Get(context.Background(), "test-volume-attachment", metav1.GetOptions{})

	assert.NoError(err)
	assert.Equal(expectedVolumeAttachment, result)
}
