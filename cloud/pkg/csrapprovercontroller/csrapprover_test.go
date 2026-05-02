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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/assert"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const metaServerServingMsg = "Auto approving MetaServer serving certificate."

func TestGetCertApprovalCondition(t *testing.T) {
	tests := []struct {
		name        string
		status      *certificatesv1.CertificateSigningRequestStatus
		wantApprove bool
		wantDeny    bool
	}{
		{
			name: "no conditions",
			status: &certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{},
			},
			wantApprove: false,
			wantDeny:    false,
		},
		{
			name: "approved only",
			status: &certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{Type: certificatesv1.CertificateApproved},
				},
			},
			wantApprove: true,
			wantDeny:    false,
		},
		{
			name: "denied only",
			status: &certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{Type: certificatesv1.CertificateDenied},
				},
			},
			wantApprove: false,
			wantDeny:    true,
		},
		{
			name: "both approved and denied",
			status: &certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{Type: certificatesv1.CertificateApproved},
					{Type: certificatesv1.CertificateDenied},
				},
			},
			wantApprove: true,
			wantDeny:    true,
		},
		{
			name: "unrelated condition types",
			status: &certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{Type: certificatesv1.CertificateFailed},
				},
			},
			wantApprove: false,
			wantDeny:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApprove, gotDeny := GetCertApprovalCondition(tt.status)
			assert.Equal(t, tt.wantApprove, gotApprove)
			assert.Equal(t, tt.wantDeny, gotDeny)
		})
	}
}

func TestAppendApprovalCondition(t *testing.T) {
	tests := []struct {
		name    string
		csr     *certificatesv1.CertificateSigningRequest
		message string
	}{
		{
			name: "empty status",
			csr: &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{},
			},
			message: metaServerServingMsg,
		},
		{
			name: "existing conditions",
			csr: &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{Type: certificatesv1.CertificateFailed},
					},
				},
			},
			message: "Auto approving another certificate.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origLen := len(tt.csr.Status.Conditions)
			appendApprovalCondition(tt.csr, tt.message)

			assert.Equal(t, origLen+1, len(tt.csr.Status.Conditions))
			lastCond := tt.csr.Status.Conditions[len(tt.csr.Status.Conditions)-1]
			assert.Equal(t, certificatesv1.CertificateApproved, lastCond.Type)
			assert.Equal(t, corev1.ConditionTrue, lastCond.Status)
			assert.Equal(t, "AutoApproved", lastCond.Reason)
			assert.Equal(t, tt.message, lastCond.Message)
		})
	}
}

func TestUsagesToSet(t *testing.T) {
	tests := []struct {
		name   string
		usages []certificatesv1.KeyUsage
		want   sets.String
	}{
		{
			name:   "empty usages",
			usages: []certificatesv1.KeyUsage{},
			want:   sets.NewString(),
		},
		{
			name:   "single usage",
			usages: []certificatesv1.KeyUsage{certificatesv1.UsageDigitalSignature},
			want:   sets.NewString(string(certificatesv1.UsageDigitalSignature)),
		},
		{
			name:   "multiple usages",
			usages: []certificatesv1.KeyUsage{certificatesv1.UsageDigitalSignature, certificatesv1.UsageKeyEncipherment},
			want:   sets.NewString(string(certificatesv1.UsageDigitalSignature), string(certificatesv1.UsageKeyEncipherment)),
		},
		{
			name:   "duplicate usages",
			usages: []certificatesv1.KeyUsage{certificatesv1.UsageDigitalSignature, certificatesv1.UsageDigitalSignature},
			want:   sets.NewString(string(certificatesv1.UsageDigitalSignature)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usagesToSet(tt.usages)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsMetaServerServingCert(t *testing.T) {
	tests := []struct {
		name       string
		signerName string
		orgs       []string
		dnsNames   []string
		ipAddrs    []string
		usages     []certificatesv1.KeyUsage
		want       bool
	}{
		{
			name:       "matching signer + valid kubelet serving CSR",
			signerName: certificatesv1.KubeletServingSignerName,
			orgs:       []string{"system:nodes"},
			dnsNames:   []string{"node1"},
			usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageServerAuth,
			},
			want: true,
		},
		{
			name:       "wrong signer name",
			signerName: "wrong.signer.name",
			orgs:       []string{"system:nodes"},
			dnsNames:   []string{"node1"},
			usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageServerAuth,
			},
			want: false,
		},
		{
			name:       "correct signer but non-kubelet CSR (missing server auth)",
			signerName: certificatesv1.KubeletServingSignerName,
			orgs:       []string{"system:nodes"},
			dnsNames:   []string{"node1"},
			usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			template := x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   "system:node:node1",
					Organization: tt.orgs,
				},
				DNSNames: tt.dnsNames,
			}
			csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
			x509cr, _ := x509.ParseCertificateRequest(csrBytes)

			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: tt.signerName,
					Usages:     tt.usages,
				},
			}

			got := isMetaServerServingCert(csr, x509cr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRecognizers(t *testing.T) {
	recs := recognizers()
	assert.Equal(t, 1, len(recs))
	assert.Equal(t, metaServerServingMsg, recs[0].successMessage)
	assert.NotNil(t, recs[0].recognize)
}
