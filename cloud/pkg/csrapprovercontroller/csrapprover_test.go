/*
Copyright 2026 The KubeEdge Authors.

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

package csrapprovercontroller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

type csrOpts struct {
	cn       string
	orgs     []string
	dnsNames []string
	ips      []net.IP
	emails   []string
}

func makeCSRPEM(t *testing.T, o csrOpts) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:        pkix.Name{CommonName: o.cn, Organization: o.orgs},
		DNSNames:       o.dnsNames,
		IPAddresses:    o.ips,
		EmailAddresses: o.emails,
	}, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
}

func servingOpts() csrOpts {
	return csrOpts{
		cn:       "system:node:edge-node",
		orgs:     []string{"system:nodes"},
		dnsNames: []string{"edge-node"},
		ips:      []net.IP{net.ParseIP("10.0.0.1")},
	}
}

var servingUsages = []certificatesv1.KeyUsage{
	certificatesv1.UsageDigitalSignature,
	certificatesv1.UsageKeyEncipherment,
	certificatesv1.UsageServerAuth,
}

func newCSR(name string, req []byte, signer string, usages []certificatesv1.KeyUsage) *certificatesv1.CertificateSigningRequest {
	return &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:    req,
			SignerName: signer,
			Usages:     usages,
		},
	}
}

// newTestApprover builds an approver whose lister is backed by an informer
// store pre-filled with objs, so tests don't need to start informers.
func newTestApprover(t *testing.T, objs ...runtime.Object) (*CSRApprover, *fake.Clientset) {
	t.Helper()
	client := fake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(client, 0)
	csrInformer := factory.Certificates().V1().CertificateSigningRequests()
	ap := NewCSRApprover(client, csrInformer)
	for _, o := range objs {
		require.NoError(t, csrInformer.Informer().GetStore().Add(o))
	}
	t.Cleanup(ap.queue.ShutDown)
	return ap, client
}

func approvalUpdates(client *fake.Clientset) []*certificatesv1.CertificateSigningRequest {
	var out []*certificatesv1.CertificateSigningRequest
	for _, a := range client.Actions() {
		if a.GetVerb() == "update" && a.GetSubresource() == "approval" {
			out = append(out, a.(clienttesting.UpdateAction).GetObject().(*certificatesv1.CertificateSigningRequest))
		}
	}
	return out
}

func TestGetCertApprovalCondition(t *testing.T) {
	cases := []struct {
		name         string
		conds        []certificatesv1.RequestConditionType
		wantApproved bool
		wantDenied   bool
	}{
		{name: "no conditions"},
		{name: "approved", conds: []certificatesv1.RequestConditionType{certificatesv1.CertificateApproved}, wantApproved: true},
		{name: "denied", conds: []certificatesv1.RequestConditionType{certificatesv1.CertificateDenied}, wantDenied: true},
		{name: "failed only", conds: []certificatesv1.RequestConditionType{certificatesv1.CertificateFailed}},
		{name: "approved and denied", conds: []certificatesv1.RequestConditionType{certificatesv1.CertificateApproved, certificatesv1.CertificateDenied}, wantApproved: true, wantDenied: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := &certificatesv1.CertificateSigningRequestStatus{}
			for _, c := range tc.conds {
				status.Conditions = append(status.Conditions, certificatesv1.CertificateSigningRequestCondition{Type: c})
			}
			approved, denied := GetCertApprovalCondition(status)
			assert.Equal(t, tc.wantApproved, approved)
			assert.Equal(t, tc.wantDenied, denied)
		})
	}
}

func TestAppendApprovalCondition(t *testing.T) {
	csr := &certificatesv1.CertificateSigningRequest{}
	appendApprovalCondition(csr, "msg")
	require.Len(t, csr.Status.Conditions, 1)
	c := csr.Status.Conditions[0]
	assert.Equal(t, certificatesv1.CertificateApproved, c.Type)
	assert.Equal(t, corev1.ConditionTrue, c.Status)
	assert.Equal(t, "AutoApproved", c.Reason)
	assert.Equal(t, "msg", c.Message)
}

func TestUsagesToSet(t *testing.T) {
	s := usagesToSet([]certificatesv1.KeyUsage{certificatesv1.UsageServerAuth, certificatesv1.UsageServerAuth, certificatesv1.UsageDigitalSignature})
	assert.Equal(t, 2, s.Len())
	assert.True(t, s.HasAll("server auth", "digital signature"))
	assert.Equal(t, 0, usagesToSet(nil).Len())
}

func TestIsMetaServerServingCert(t *testing.T) {
	cases := []struct {
		name   string
		opts   csrOpts
		signer string
		usages []certificatesv1.KeyUsage
		want   bool
	}{
		{name: "valid kubelet serving csr", opts: servingOpts(), signer: certificatesv1.KubeletServingSignerName, usages: servingUsages, want: true},
		{name: "wrong signer", opts: servingOpts(), signer: certificatesv1.KubeAPIServerClientKubeletSignerName, usages: servingUsages},
		{name: "client auth usage", opts: servingOpts(), signer: certificatesv1.KubeletServingSignerName,
			usages: []certificatesv1.KeyUsage{certificatesv1.UsageDigitalSignature, certificatesv1.UsageClientAuth}},
		{name: "cn not a node", opts: func() csrOpts { o := servingOpts(); o.cn = "edge-node"; return o }(),
			signer: certificatesv1.KubeletServingSignerName, usages: servingUsages},
		{name: "missing system:nodes org", opts: func() csrOpts { o := servingOpts(); o.orgs = []string{"other"}; return o }(),
			signer: certificatesv1.KubeletServingSignerName, usages: servingUsages},
		{name: "no dns or ip san", opts: func() csrOpts { o := servingOpts(); o.dnsNames, o.ips = nil, nil; return o }(),
			signer: certificatesv1.KubeletServingSignerName, usages: servingUsages},
		{name: "email san not allowed", opts: func() csrOpts { o := servingOpts(); o.emails = []string{"a@b.c"}; return o }(),
			signer: certificatesv1.KubeletServingSignerName, usages: servingUsages},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csr := newCSR("csr", makeCSRPEM(t, tc.opts), tc.signer, tc.usages)
			block, _ := pem.Decode(csr.Spec.Request)
			x509cr, err := x509.ParseCertificateRequest(block.Bytes)
			require.NoError(t, err)
			assert.Equal(t, tc.want, isMetaServerServingCert(csr, x509cr))
		})
	}
}

func TestHandle(t *testing.T) {
	t.Run("approves metaserver serving csr", func(t *testing.T) {
		csr := newCSR("serving", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		require.NoError(t, ap.handle(csr.DeepCopy()))
		ups := approvalUpdates(client)
		require.Len(t, ups, 1)
		approved, _ := GetCertApprovalCondition(&ups[0].Status)
		assert.True(t, approved)
		assert.Equal(t, "Auto approving MetaServer serving certificate.", ups[0].Status.Conditions[0].Message)
	})

	t.Run("ignores csr that no recognizer matches", func(t *testing.T) {
		csr := newCSR("client", makeCSRPEM(t, servingOpts()), certificatesv1.KubeAPIServerClientSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		require.NoError(t, ap.handle(csr.DeepCopy()))
		assert.Empty(t, approvalUpdates(client))
	})

	for _, ct := range []certificatesv1.RequestConditionType{certificatesv1.CertificateApproved, certificatesv1.CertificateDenied} {
		t.Run("skips already "+string(ct), func(t *testing.T) {
			csr := newCSR("done", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
			csr.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{{Type: ct}}
			ap, client := newTestApprover(t, csr)
			require.NoError(t, ap.handle(csr.DeepCopy()))
			assert.Empty(t, approvalUpdates(client))
		})
	}

	t.Run("returns error on unparsable request", func(t *testing.T) {
		csr := newCSR("bad", []byte("not a csr"), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		err := ap.handle(csr.DeepCopy())
		assert.ErrorContains(t, err, "unable to parse csr bad")
		assert.Empty(t, approvalUpdates(client))
	})

	t.Run("returns error when approval update fails", func(t *testing.T) {
		csr := newCSR("serving", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		client.PrependReactor("update", "certificatesigningrequests", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("boom")
		})
		err := ap.handle(csr.DeepCopy())
		assert.ErrorContains(t, err, "failed to update approval for csr serving: boom")
	})
}

func TestSyncFunc(t *testing.T) {
	t.Run("deleted csr is not an error", func(t *testing.T) {
		ap, client := newTestApprover(t)
		assert.NoError(t, ap.syncFunc("missing"))
		assert.Empty(t, approvalUpdates(client))
	})

	t.Run("csr with issued certificate is skipped", func(t *testing.T) {
		csr := newCSR("issued", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
		csr.Status.Certificate = []byte("cert")
		ap, client := newTestApprover(t, csr)
		assert.NoError(t, ap.syncFunc("issued"))
		assert.Empty(t, approvalUpdates(client))
	})

	t.Run("pending csr is approved without mutating the cache", func(t *testing.T) {
		csr := newCSR("serving", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		require.NoError(t, ap.syncFunc("serving"))
		assert.Len(t, approvalUpdates(client), 1)
		cached, err := ap.csrLister.Get("serving")
		require.NoError(t, err)
		assert.Empty(t, cached.Status.Conditions, "syncFunc must DeepCopy before handling")
	})
}

func TestProcessNextWorkItem(t *testing.T) {
	t.Run("success forgets key", func(t *testing.T) {
		csr := newCSR("serving", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, client := newTestApprover(t, csr)
		ap.queue.Add("serving")
		assert.True(t, ap.processNextWorkItem())
		assert.Equal(t, 0, ap.queue.NumRequeues("serving"))
		assert.Len(t, approvalUpdates(client), 1)
	})

	t.Run("failure requeues with rate limit", func(t *testing.T) {
		csr := newCSR("bad", []byte("not a csr"), certificatesv1.KubeletServingSignerName, servingUsages)
		ap, _ := newTestApprover(t, csr)
		ap.queue.Add("bad")
		assert.True(t, ap.processNextWorkItem())
		assert.Equal(t, 1, ap.queue.NumRequeues("bad"))
	})

	t.Run("shut down queue returns false", func(t *testing.T) {
		ap, _ := newTestApprover(t)
		ap.queue.ShutDown()
		assert.False(t, ap.processNextWorkItem())
	})
}

func TestEnqueueCertificateRequest(t *testing.T) {
	ap, _ := newTestApprover(t)
	ap.enqueueCertificateRequest(newCSR("a", nil, "", nil))
	assert.Equal(t, 1, ap.queue.Len())
	key, _ := ap.queue.Get()
	assert.Equal(t, "a", key)
	ap.queue.Done(key)

	ap.enqueueCertificateRequest("not an object")
	assert.Equal(t, 0, ap.queue.Len())
}

func TestRunApprovesViaInformer(t *testing.T) {
	csr := newCSR("serving", makeCSRPEM(t, servingOpts()), certificatesv1.KubeletServingSignerName, servingUsages)
	client := fake.NewSimpleClientset(csr)
	factory := informers.NewSharedInformerFactory(client, 0)
	ap := NewCSRApprover(client, factory.Certificates().V1().CertificateSigningRequests())

	stopCh := make(chan struct{})
	done := make(chan struct{})
	factory.Start(stopCh)
	go func() {
		ap.Run(1, stopCh)
		close(done)
	}()

	assert.Eventually(t, func() bool { return len(approvalUpdates(client)) == 1 }, 5*time.Second, 20*time.Millisecond)
	close(stopCh)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after stop")
	}
}
