/*
Copyright 2022 The Kubernetes Authors.

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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_secret.go"
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

// SecretBridge implements SecretInterface
type SecretBridge struct {
	fakecorev1.FakeCoreV1
	ns         string
	MetaClient client.CoreInterface
}

func (c *SecretBridge) Create(ctx context.Context, secret *corev1.Secret, opts metav1.CreateOptions) (*corev1.Secret, error) {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) Update(ctx context.Context, secret *corev1.Secret, opts metav1.UpdateOptions) (*corev1.Secret, error) {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) List(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Secret, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *SecretBridge) Apply(ctx context.Context, secret *appcorev1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Secret, err error) {
	//TODO implement me
	panic("implement me")
}

// Get takes name of the secret, and returns the corresponding secret object
func (c *SecretBridge) Get(_ context.Context, name string, _ metav1.GetOptions) (result *corev1.Secret, err error) {
	return c.MetaClient.Secrets(c.ns).Get(name)
}
