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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/coordination/v1/fake/fake_lease.go"
and made some variant
*/

package v1

import (
	"context"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakecoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// LeaseBridge implements LeaseInterface
type LeaseBridge struct {
	fakecoordinationv1.FakeLeases
	ns         string
	MetaClient client.CoreInterface
}

func (c *LeaseBridge) Create(ctx context.Context, lease *coordinationv1.Lease, opts metav1.CreateOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Create(lease)
}

func (c *LeaseBridge) Update(ctx context.Context, lease *coordinationv1.Lease, opts metav1.UpdateOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Update(lease)
}

func (c *LeaseBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Get(name)
}
