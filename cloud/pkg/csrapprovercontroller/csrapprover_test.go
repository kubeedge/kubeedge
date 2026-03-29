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
*/

package csrapprovercontroller

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewCSRApprover(t *testing.T) {
	assert := assert.New(t)

	client := fake.NewSimpleClientset()

	informerFactory := informers.NewSharedInformerFactory(client, 0)
	csrInformer := informerFactory.Certificates().V1().CertificateSigningRequests()

	approver := NewCSRApprover(client, csrInformer)

	assert.NotNil(approver)
	assert.Equal(client, approver.kubeClient)
	assert.NotNil(approver.queue)
	assert.NotNil(approver.csrLister)
	assert.NotNil(approver.csrSynced)
	assert.NotEmpty(approver.recognizers)

	// Test the event handler
	csr := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr",
		},
	}
	err := csrInformer.Informer().GetStore().Add(csr)
	assert.NoError(err)
	err = csrInformer.Informer().GetIndexer().Add(csr)
	assert.NoError(err)

	assert.Equal(1, approver.queue.Len())
	item, _ := approver.queue.Get()
	assert.Equal("test-csr", item)
}

func TestEnqueueCertificateRequest(t *testing.T) {
	assert := assert.New(t)

	client := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(client, 0)
	csrInformer := informerFactory.Certificates().V1().CertificateSigningRequests()

	approver := NewCSRApprover(client, csrInformer)

	// Test case 1: Enqueue a valid CSR
	csr1 := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr-1",
		},
	}
	approver.enqueueCertificateRequest(csr1)
	assert.Equal(1, approver.queue.Len())

	// Test case 2: Enqueue another valid CSR
	csr2 := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr-2",
		},
	}
	approver.enqueueCertificateRequest(csr2)
	assert.Equal(2, approver.queue.Len())

	// Test case 3: Enqueue same CSR again
	approver.enqueueCertificateRequest(csr1)
	assert.Equal(2, approver.queue.Len())

	// Test case 4: Enqueue invalid object
	invalidObj := "not a CSR"
	assert.NotPanics(func() {
		approver.enqueueCertificateRequest(invalidObj)
	}, "Enqueueing an invalid object should not panic")
	assert.Equal(2, approver.queue.Len())

	expectedItems := map[string]bool{
		"test-csr-1": false,
		"test-csr-2": false,
	}
	for approver.queue.Len() > 0 {
		key, _ := approver.queue.Get()
		keyStr, ok := key.(string)
		assert.True(ok)
		_, exists := expectedItems[keyStr]
		assert.True(exists, "Unexpected item in queue: %s", keyStr)
		expectedItems[keyStr] = true
		approver.queue.Done(key)
	}

	for key, found := range expectedItems {
		assert.True(found, "Expected item not found in queue: %s", key)
	}
}

func TestGetCertApprovalCondition(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		status         certificatesv1.CertificateSigningRequestStatus
		expectApproved bool
		expectDenied   bool
	}{
		{
			name: "Approved CSR",
			status: certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{
						Type:   certificatesv1.CertificateApproved,
						Status: corev1.ConditionTrue,
					},
				},
			},
			expectApproved: true,
			expectDenied:   false,
		},
		{
			name: "Denied CSR",
			status: certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{
						Type:   certificatesv1.CertificateDenied,
						Status: corev1.ConditionTrue,
					},
				},
			},
			expectApproved: false,
			expectDenied:   true,
		},
		{
			name: "Pending CSR",
			status: certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{},
			},
			expectApproved: false,
			expectDenied:   false,
		},
		{
			name: "Both Approved and Denied",
			status: certificatesv1.CertificateSigningRequestStatus{
				Conditions: []certificatesv1.CertificateSigningRequestCondition{
					{
						Type:   certificatesv1.CertificateApproved,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   certificatesv1.CertificateDenied,
						Status: corev1.ConditionTrue,
					},
				},
			},
			expectApproved: true,
			expectDenied:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			approved, denied := GetCertApprovalCondition(&tc.status)
			assert.Equal(tc.expectApproved, approved, "Approved status mismatch for case: %s", tc.name)
			assert.Equal(tc.expectDenied, denied, "Denied status mismatch for case: %s", tc.name)
		})
	}
}

func TestAppendApprovalCondition(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		initialCSR     *certificatesv1.CertificateSigningRequest
		message        string
		expectedLength int
	}{
		{
			name: "Append to empty conditions",
			initialCSR: &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{},
				},
			},
			message:        "Test message",
			expectedLength: 1,
		},
		{
			name: "Append to existing conditions",
			initialCSR: &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateDenied,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			message:        "test message",
			expectedLength: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appendApprovalCondition(tc.initialCSR, tc.message)

			assert.Len(tc.initialCSR.Status.Conditions, tc.expectedLength)

			lastCondition := tc.initialCSR.Status.Conditions[tc.expectedLength-1]
			assert.Equal(certificatesv1.CertificateApproved, lastCondition.Type)
			assert.Equal(corev1.ConditionTrue, lastCondition.Status)
			assert.Equal("AutoApproved", lastCondition.Reason)
			assert.Equal(tc.message, lastCondition.Message)
		})
	}
}

func TestRecognizers(t *testing.T) {
	assert := assert.New(t)

	recognizerList := recognizers()

	assert.NotEmpty(recognizerList)
	assert.Len(recognizerList, 1)

	recognizer := recognizerList[0]
	assert.Equal("Auto approving MetaServer serving certificate.", recognizer.successMessage)
}

func TestIsMetaServerServingCert(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid MetaServer serving cert
	validCSR := &certificatesv1.CertificateSigningRequest{
		Spec: certificatesv1.CertificateSigningRequestSpec{
			SignerName: certificatesv1.KubeletServingSignerName,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageServerAuth,
			},
		},
	}
	validX509CR := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "system:node:metaserver.kubeedge.io",
			Organization: []string{"system:nodes"},
		},
		DNSNames:    []string{"metaserver.kubeedge.io"},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.1")},
	}

	assert.True(isMetaServerServingCert(validCSR, validX509CR))

	// Test case 2: Invalid signer name
	invalidSignerCSR := &certificatesv1.CertificateSigningRequest{
		Spec: certificatesv1.CertificateSigningRequestSpec{
			SignerName: "invalid-signer",
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageServerAuth,
			},
		},
	}

	assert.False(isMetaServerServingCert(invalidSignerCSR, validX509CR))

	// Test case 3: Invalid usages
	invalidUsagesCSR := &certificatesv1.CertificateSigningRequest{
		Spec: certificatesv1.CertificateSigningRequestSpec{
			SignerName: certificatesv1.KubeletServingSignerName,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageClientAuth,
			},
		},
	}

	assert.False(isMetaServerServingCert(invalidUsagesCSR, validX509CR))

	// Test case 4: Invalid subject
	invalidSubjectX509CR := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "invalid:node:metaserver.kubeedge.io",
			Organization: []string{"system:nodes"},
		},
		DNSNames:    []string{"metaserver.kubeedge.io"},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.1")},
	}

	assert.False(isMetaServerServingCert(validCSR, invalidSubjectX509CR))

	// Test case 5: Missing DNS names and IP addresses
	missingDNSAndIPX509CR := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "system:node:metaserver.kubeedge.io",
			Organization: []string{"system:nodes"},
		},
	}

	assert.False(isMetaServerServingCert(validCSR, missingDNSAndIPX509CR))
}

func TestUsagesToSet(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Empty usages
	emptyUsages := []certificatesv1.KeyUsage{}
	assert.Equal(sets.NewString(), usagesToSet(emptyUsages))

	// Test case 2: Single usage
	singleUsage := []certificatesv1.KeyUsage{certificatesv1.UsageDigitalSignature}
	expectedSingle := sets.NewString(string(certificatesv1.UsageDigitalSignature))
	assert.Equal(expectedSingle, usagesToSet(singleUsage))

	// Test case 3: Multiple usages
	multipleUsages := []certificatesv1.KeyUsage{
		certificatesv1.UsageDigitalSignature,
		certificatesv1.UsageKeyEncipherment,
		certificatesv1.UsageServerAuth,
	}
	expectedMultiple := sets.NewString(
		string(certificatesv1.UsageDigitalSignature),
		string(certificatesv1.UsageKeyEncipherment),
		string(certificatesv1.UsageServerAuth),
	)
	assert.Equal(expectedMultiple, usagesToSet(multipleUsages))

	// Test case 4: Duplicate usages
	duplicateUsages := []certificatesv1.KeyUsage{
		certificatesv1.UsageDigitalSignature,
		certificatesv1.UsageKeyEncipherment,
		certificatesv1.UsageDigitalSignature,
	}
	expectedDuplicate := sets.NewString(
		string(certificatesv1.UsageDigitalSignature),
		string(certificatesv1.UsageKeyEncipherment),
	)
	assert.Equal(expectedDuplicate, usagesToSet(duplicateUsages))
}
