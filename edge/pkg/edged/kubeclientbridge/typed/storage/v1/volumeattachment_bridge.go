/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To make a bridge between kubeclient and metaclient,
This file is derived from K8S client-go code with reduced set of methods
Changes done are
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/storage/v1/fake/fake_volumeattachment.go"
and made some variant
*/

package v1

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/applyconfigurations/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// VolumeAttachmentsBridge implements PersistentVolumeInterface
type VolumeAttachmentsBridge struct {
	fakestoragev1.FakeStorageV1
	MetaClient client.CoreInterface
}

func (c *VolumeAttachmentsBridge) Create(ctx context.Context, volumeAttachment *storagev1.VolumeAttachment, opts metav1.CreateOptions) (*storagev1.VolumeAttachment, error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) Update(ctx context.Context, volumeAttachment *storagev1.VolumeAttachment, opts metav1.UpdateOptions) (*storagev1.VolumeAttachment, error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) UpdateStatus(ctx context.Context, volumeAttachment *storagev1.VolumeAttachment, opts metav1.UpdateOptions) (*storagev1.VolumeAttachment, error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) List(ctx context.Context, opts metav1.ListOptions) (*storagev1.VolumeAttachmentList, error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *storagev1.VolumeAttachment, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) Apply(ctx context.Context, volumeAttachment *v1.VolumeAttachmentApplyConfiguration, opts metav1.ApplyOptions) (result *storagev1.VolumeAttachment, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *VolumeAttachmentsBridge) ApplyStatus(ctx context.Context, volumeAttachment *v1.VolumeAttachmentApplyConfiguration, opts metav1.ApplyOptions) (result *storagev1.VolumeAttachment, err error) {
	//TODO implement me
	panic("implement me")
}

// Get takes name of the persistentVolume, and returns the corresponding persistentVolume object
func (c *VolumeAttachmentsBridge) Get(_ context.Context, name string, options metav1.GetOptions) (result *storagev1.VolumeAttachment, err error) {
	return c.MetaClient.VolumeAttachments(metav1.NamespaceDefault).Get(name, options)
}
