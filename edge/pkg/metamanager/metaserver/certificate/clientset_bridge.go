/*
Copyright The Kubernetes Authors.

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
1. Package kubeclientbridge got some functions from "k8s.io/client-go/kubernetes/fake/clientset_generated.go"
and made some variant
*/

package certificate

import (
	clientset "k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	certificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	fakecertificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	kecertificates "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/certificate/certificates/v1"
)

// NewSimpleClientset is new interface
func NewSimpleClientset() clientset.Interface {
	return &Clientset{*fakekube.NewSimpleClientset(), client.New()}
}

// Clientset extends Clientset
type Clientset struct {
	fakekube.Clientset
	MetaClient client.CoreInterface
}

func (c *Clientset) CertificatesV1() certificatesv1.CertificatesV1Interface {
	return &kecertificates.CertificatesV1Bridge{FakeCertificatesV1: fakecertificatesv1.FakeCertificatesV1{Fake: &c.Fake}, MetaClient: c.MetaClient}
}
