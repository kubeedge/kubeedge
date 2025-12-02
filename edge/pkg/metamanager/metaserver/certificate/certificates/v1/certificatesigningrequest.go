/*
Copyright 2024 The Kubernetes Authors.

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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/certificates/v1/fake/fake_node.go"
and made some variant
*/

package v1

import (
	"context"

	v1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	applyconfigurationscertificatesv1 "k8s.io/client-go/applyconfigurations/certificates/v1"
	fakecertificates "k8s.io/client-go/kubernetes/typed/certificates/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// CertificateSigningRequestsBridge implements CertificateSigningRequestInterface
type CertificateSigningRequestsBridge struct {
	fakecertificates.FakeCertificatesV1
	MetaClient client.CoreInterface
}

func (c *CertificateSigningRequestsBridge) Update(ctx context.Context, certificateSigningRequest *v1.CertificateSigningRequest, opts metav1.UpdateOptions) (*v1.CertificateSigningRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) UpdateStatus(ctx context.Context, certificateSigningRequest *v1.CertificateSigningRequest, opts metav1.UpdateOptions) (*v1.CertificateSigningRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) List(ctx context.Context, opts metav1.ListOptions) (*v1.CertificateSigningRequestList, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.CertificateSigningRequest, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) Apply(ctx context.Context, certificateSigningRequest *applyconfigurationscertificatesv1.CertificateSigningRequestApplyConfiguration, opts metav1.ApplyOptions) (result *v1.CertificateSigningRequest, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) ApplyStatus(ctx context.Context, certificateSigningRequest *applyconfigurationscertificatesv1.CertificateSigningRequestApplyConfiguration, opts metav1.ApplyOptions) (result *v1.CertificateSigningRequest, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) UpdateApproval(ctx context.Context, certificateSigningRequestName string, certificateSigningRequest *v1.CertificateSigningRequest, opts metav1.UpdateOptions) (*v1.CertificateSigningRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CertificateSigningRequestsBridge) Create(_ context.Context, certificateSigningRequest *v1.CertificateSigningRequest, _ metav1.CreateOptions) (result *v1.CertificateSigningRequest, err error) {
	return c.MetaClient.CertificateSigningRequests().Create(certificateSigningRequest)
}

func (c *CertificateSigningRequestsBridge) Get(_ context.Context, name string, _ metav1.GetOptions) (result *v1.CertificateSigningRequest, err error) {
	return c.MetaClient.CertificateSigningRequests().Get(name)
}
