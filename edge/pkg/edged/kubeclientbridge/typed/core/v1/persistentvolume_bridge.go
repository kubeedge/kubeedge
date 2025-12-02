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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_persistentvolume.go"
and made some variant
*/

package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// PersistentVolumesBridge implements PersistentVolumeInterface
type PersistentVolumesBridge struct {
	fakecorev1.FakeCoreV1
	MetaClient client.CoreInterface
}

func (c *PersistentVolumesBridge) Create(ctx context.Context, persistentVolume *corev1.PersistentVolume, opts metav1.CreateOptions) (*corev1.PersistentVolume, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) Update(ctx context.Context, persistentVolume *corev1.PersistentVolume, opts metav1.UpdateOptions) (*corev1.PersistentVolume, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) UpdateStatus(ctx context.Context, persistentVolume *corev1.PersistentVolume, opts metav1.UpdateOptions) (*corev1.PersistentVolume, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PersistentVolumeList, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.PersistentVolume, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) Apply(ctx context.Context, persistentVolume *appcorev1.PersistentVolumeApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.PersistentVolume, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *PersistentVolumesBridge) ApplyStatus(ctx context.Context, persistentVolume *appcorev1.PersistentVolumeApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.PersistentVolume, err error) {
	//TODO implement me
	panic("implement me")
}

// Get takes name of the persistentVolume, and returns the corresponding persistentVolume object, and an error if there is any.
func (c *PersistentVolumesBridge) Get(_ context.Context, name string, options metav1.GetOptions) (result *corev1.PersistentVolume, err error) {
	return c.MetaClient.PersistentVolumes().Get(name, options)
}
