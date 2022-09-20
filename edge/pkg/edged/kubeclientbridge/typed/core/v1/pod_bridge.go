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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_pod.go"
and made some variant
*/

package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type PodsBridge struct {
	fakecorev1.FakePods
	ns         string
	MetaClient client.CoreInterface
}

func (c *PodsBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.Pod, err error) {
	return c.MetaClient.Pods(c.ns).Get(name)
}

func (c *PodsBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Pod, err error) {
	return c.MetaClient.Pods(c.ns).Patch(name, data)
}
