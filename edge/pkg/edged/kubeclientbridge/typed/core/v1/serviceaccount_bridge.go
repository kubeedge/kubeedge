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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_serviceaccount.go"
and made some variant
*/

package v1

import (
	"context"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// ServiceAccountsBridge implements ServiceAccountInterface
type ServiceAccountsBridge struct {
	fakecorev1.FakeServiceAccounts
	ns         string
	MetaClient client.CoreInterface
}

// CreateToken takes the representation of a tokenRequest and creates it.  Returns the server's representation of the tokenRequest, and an error, if there is any.
func (c *ServiceAccountsBridge) CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *authenticationv1.TokenRequest, opts metav1.CreateOptions) (result *authenticationv1.TokenRequest, err error) {
	return c.MetaClient.ServiceAccountToken().GetServiceAccountToken(c.ns, serviceAccountName, tokenRequest)
}

func (c *ServiceAccountsBridge) Delete(ctx context.Context, podUID string, opts metav1.DeleteOptions) error {
	c.MetaClient.ServiceAccountToken().DeleteServiceAccountToken(types.UID(podUID))
	return nil
}

func (c *ServiceAccountsBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.ServiceAccount, err error) {
	return c.MetaClient.ServiceAccounts(c.ns).Get(name)
}
