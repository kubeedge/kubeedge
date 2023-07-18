/*
Copyright 2023 The KubeEdge Authors.

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

package csrapprover

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"net/url"
	"testing"

	capi "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/testing"
	"k8s.io/kubernetes/pkg/controller"
)

func TestHandle(t *testing.T) {
	cases := []struct {
		name   string
		csr    *capi.CertificateSigningRequest
		err    bool
		verify func(*testing.T, []testclient.Action)
	}{
		{
			name: "csr already approved or Denied",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "already-approve",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions),
				},
				Status: capi.CertificateSigningRequestStatus{Conditions: []capi.CertificateSigningRequestCondition{
					{
						Type:   capi.CertificateApproved,
						Status: corev1.ConditionTrue,
						Reason: "AutoApproved",
					},
				}},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "signerName is not kubernetes.io/kubelet-serving",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "signername-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeAPIServerClientSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
					},
					Request: csrWithOpts(kubeletServerPEMOptions),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "org is not 'system:nodes'",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "org-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions, pemOptions{org: "not-system:nodes"}),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "CN does not have system:node: prefix",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cn-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions, pemOptions{cn: "notprefixed"}),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "has an unexpected usage",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "usage-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageClientAuth,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "it does not specify any dnsNames or ipAddresses",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-dns-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions, pemOptions{ipAddresses: []net.IP{}, dnsNames: []string{}}),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "missing an expected usage",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "miss-usage-invalid",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
					},
					Request: csrWithOpts(kubeletServerPEMOptions),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 0 {
					t.Errorf("expected no client calls but got: %#v", as)
				}
			},
		},
		{
			name: "a valid metaserver CertificateSigningRequest",
			csr: &capi.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "normal-csr",
				},
				Spec: capi.CertificateSigningRequestSpec{
					SignerName: capi.KubeletServingSignerName,
					Usages: []capi.KeyUsage{
						capi.UsageDigitalSignature,
						capi.UsageKeyEncipherment,
						capi.UsageServerAuth,
					},
					Request: csrWithOpts(kubeletServerPEMOptions),
				},
			},
			verify: func(t *testing.T, as []testclient.Action) {
				if len(as) != 1 {
					t.Errorf("expected two calls but got: %#v", as)
					return
				}
				a := as[0].(testclient.UpdateActionImpl)
				if got, expected := a.Verb, "update"; got != expected {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
				if got, expected := a.Resource, (schema.GroupVersionResource{Group: "certificates.k8s.io", Version: "v1", Resource: "certificatesigningrequests"}); got != expected {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
				if got, expected := a.Subresource, "approval"; got != expected {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
				csr := a.Object.(*capi.CertificateSigningRequest)
				if len(csr.Status.Conditions) != 1 {
					t.Errorf("expected CSR to have approved condition: %#v", csr)
				}
				c := csr.Status.Conditions[0]
				if got, expected := c.Type, capi.CertificateApproved; got != expected {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
				if got, expected := c.Reason, "AutoApproved"; got != expected {
					t.Errorf("got: %v, expected: %v", got, expected)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(c.csr)
			informerFactory := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), controller.NoResyncPeriodFunc())
			approver := NewCSRApprover(client, informerFactory.Certificates().V1().CertificateSigningRequests())
			if err := approver.handle(c.csr); err != nil && !c.err {
				t.Errorf("unexpected err: %v", err)
			}
			c.verify(t, client.Actions())
		})
	}
}

var kubeletServerPEMOptions = pemOptions{
	cn:          "system:node:edge-node",
	org:         "system:nodes",
	ipAddresses: []net.IP{{127, 0, 0, 1}},
}

type pemOptions struct {
	cn             string
	org            string
	ipAddresses    []net.IP
	dnsNames       []string
	emailAddresses []string
	uris           []string
}

// overlayPEMOptions overlays one set of pemOptions on top of another to allow
// for easily overriding a single field in the options
func overlayPEMOptions(opts ...pemOptions) pemOptions {
	if len(opts) == 0 {
		return pemOptions{}
	}
	base := opts[0]
	for _, opt := range opts[1:] {
		if opt.cn != "" {
			base.cn = opt.cn
		}
		if opt.org != "" {
			base.org = opt.org
		}
		if opt.ipAddresses != nil {
			base.ipAddresses = opt.ipAddresses
		}
		if opt.dnsNames != nil {
			base.dnsNames = opt.dnsNames
		}
		if opt.emailAddresses != nil {
			base.emailAddresses = opt.emailAddresses
		}
		if opt.uris != nil {
			base.uris = opt.uris
		}
	}
	return base
}

func csrWithOpts(base pemOptions, overlays ...pemOptions) []byte {
	opts := overlayPEMOptions(append([]pemOptions{base}, overlays...)...)
	uris := make([]*url.URL, len(opts.uris))
	for i, s := range opts.uris {
		u, err := url.ParseRequestURI(s)
		if err != nil {
			panic(err)
		}
		uris[i] = u
	}
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   opts.cn,
			Organization: []string{opts.org},
		},
		IPAddresses:    opts.ipAddresses,
		DNSNames:       opts.dnsNames,
		EmailAddresses: opts.emailAddresses,
		URIs:           uris,
	}

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		panic(err)
	}

	csrPemBlock := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}

	p := pem.EncodeToMemory(csrPemBlock)
	if p == nil {
		panic("invalid pem block")
	}

	return p
}
