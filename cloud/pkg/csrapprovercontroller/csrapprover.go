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
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	certificatesinformers "k8s.io/client-go/informers/certificates/v1"
	clientset "k8s.io/client-go/kubernetes"
	certificateslisters "k8s.io/client-go/listers/certificates/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/certificates"
	"k8s.io/kubernetes/pkg/controller"
)

type csrRecognizer struct {
	recognize      func(csr *certificatesv1.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool
	successMessage string
}

type CSRApprover struct {
	kubeClient clientset.Interface
	queue      workqueue.RateLimitingInterface

	csrLister certificateslisters.CertificateSigningRequestLister
	csrSynced cache.InformerSynced

	recognizers []csrRecognizer
}

// NewCSRApprover creates a new NewCSRApprover
func NewCSRApprover(client clientset.Interface, csrInformer certificatesinformers.CertificateSigningRequestInformer) *CSRApprover {
	approver := &CSRApprover{
		kubeClient: client,
		queue: workqueue.NewRateLimitingQueueWithConfig(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(200*time.Millisecond, 1000*time.Second),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		), workqueue.RateLimitingQueueConfig{Name: "certificate"}),
		recognizers: recognizers(),
	}

	_, err := csrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			csr := obj.(*certificatesv1.CertificateSigningRequest)
			klog.V(4).Infof("Adding certificate request %s", csr.Name)
			approver.enqueueCertificateRequest(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldCSR := old.(*certificatesv1.CertificateSigningRequest)
			klog.V(4).Infof("Updating certificate request %s", oldCSR.Name)
			approver.enqueueCertificateRequest(new)
		},
	})
	if err != nil {
		klog.Fatalf("new CSR approver failed, add event handler err: %v", err)
	}

	approver.csrLister = csrInformer.Lister()
	approver.csrSynced = csrInformer.Informer().HasSynced
	return approver
}

func (ap *CSRApprover) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer ap.queue.ShutDown()

	klog.Info("Start CSRApprover")
	defer klog.Info("Shut down CSRApprover")

	if !cache.WaitForNamedCacheSync("CSRApprover", stopCh, ap.csrSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(ap.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (ap *CSRApprover) enqueueCertificateRequest(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", obj, err))
		return
	}
	ap.queue.Add(key)
}

func (ap *CSRApprover) worker() {
	for ap.processNextWorkItem() {
	}
}

func (ap *CSRApprover) processNextWorkItem() bool {
	cKey, quit := ap.queue.Get()
	if quit {
		return false
	}
	defer ap.queue.Done(cKey)

	if err := ap.syncFunc(cKey.(string)); err != nil {
		ap.queue.AddRateLimited(cKey)
		utilruntime.HandleError(fmt.Errorf("sync %v failed with: %v", cKey, err))
		return true
	}

	ap.queue.Forget(cKey)
	return true
}

func (ap *CSRApprover) syncFunc(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing certificate request %s (%v)", key, time.Since(startTime))
	}()

	csr, err := ap.csrLister.Get(key)
	if apierrors.IsNotFound(err) {
		klog.Infof("csr %s has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	if len(csr.Status.Certificate) > 0 {
		return nil
	}
	csr = csr.DeepCopy()
	return ap.handle(csr)
}

func (ap *CSRApprover) handle(csr *certificatesv1.CertificateSigningRequest) error {
	if approved, denied := GetCertApprovalCondition(&csr.Status); approved || denied {
		return nil
	}
	x509cr, err := certificates.ParseCSR(csr.Spec.Request)
	if err != nil {
		return fmt.Errorf("unable to parse csr %s: %v", csr.Name, err)
	}

	for _, r := range ap.recognizers {
		if !r.recognize(csr, x509cr) {
			continue
		}

		appendApprovalCondition(csr, r.successMessage)
		_, err = ap.kubeClient.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.Background(), csr.Name, csr, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update approval for csr %s: %v", csr.Name, err)
		}
		return nil
	}
	return nil
}

func GetCertApprovalCondition(status *certificatesv1.CertificateSigningRequestStatus) (approved, denied bool) {
	for _, c := range status.Conditions {
		if c.Type == certificatesv1.CertificateApproved {
			approved = true
		}
		if c.Type == certificatesv1.CertificateDenied {
			denied = true
		}
	}
	return
}

func appendApprovalCondition(csr *certificatesv1.CertificateSigningRequest, message string) {
	csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Type:    certificatesv1.CertificateApproved,
		Status:  corev1.ConditionTrue,
		Reason:  "AutoApproved",
		Message: message,
	})
}

func recognizers() []csrRecognizer {
	return []csrRecognizer{
		{
			recognize:      isMetaServerServingCert,
			successMessage: "Auto approving MetaServer serving certificate.",
		},
	}
}

func isMetaServerServingCert(csr *certificatesv1.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool {
	if csr.Spec.SignerName != certificatesv1.KubeletServingSignerName {
		return false
	}
	return certificates.IsKubeletServingCSR(x509cr, usagesToSet(csr.Spec.Usages))
}

func usagesToSet(usages []certificatesv1.KeyUsage) sets.String {
	result := sets.NewString()
	for _, usages := range usages {
		result.Insert(string(usages))
	}
	return result
}
