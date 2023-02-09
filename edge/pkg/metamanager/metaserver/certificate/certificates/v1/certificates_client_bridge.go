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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/coordination/v1/fake/fake_coordination_client.go"
and made some variant
*/

package v1

import (
	v1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	fakev1 "k8s.io/client-go/kubernetes/typed/certificates/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// CertificatesV1Bridge is a CertificatesV1 bridge
type CertificatesV1Bridge struct {
	fakev1.FakeCertificatesV1
	MetaClient client.CoreInterface
}

func (c *CertificatesV1Bridge) CertificateSigningRequest(namespace string) v1.CertificateSigningRequestInterface {
	return &CertificateSigningRequestsBridge{fakev1.FakeCertificateSigningRequests{Fake: &c.FakeCertificatesV1}, c.MetaClient}
}
