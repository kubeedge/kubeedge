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

// Package csrapprover implements an automated approver for KubeEdge metaserver certificates.
package csrapprover

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	capi "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
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
	capihelper "k8s.io/kubernetes/pkg/apis/certificates"
	"k8s.io/kubernetes/pkg/controller"
)

type csrRecognizer struct {
	recognize      func(csr *capi.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool
	successMessage string
}

type CSRApprover struct {
	kubeClient clientset.Interface
	queue      workqueue.RateLimitingInterface

	csrLister  certificateslisters.CertificateSigningRequestLister
	csrsSynced cache.InformerSynced

	recognizers []csrRecognizer
}

// NewCSRApprover creates a new NewCSRApprover.
func NewCSRApprover(client clientset.Interface, csrInformer certificatesinformers.CertificateSigningRequestInformer) *CSRApprover {
	cc := &CSRApprover{
		kubeClient: client,
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(200*time.Millisecond, 1000*time.Second),
			// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		), "certificate"),
		recognizers: recognizers(),
	}

	csrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			csr := obj.(*capi.CertificateSigningRequest)
			klog.V(4).Infof("Adding certificate request %s", csr.Name)
			cc.enqueueCertificateRequest(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldCSR := old.(*capi.CertificateSigningRequest)
			klog.V(4).Infof("Updating certificate request %s", oldCSR.Name)
			cc.enqueueCertificateRequest(new)
		},
	})

	cc.csrLister = csrInformer.Lister()
	cc.csrsSynced = csrInformer.Informer().HasSynced
	return cc
}

// Run the main goroutine responsible for watching and syncing jobs.
func (cc *CSRApprover) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer cc.queue.ShutDown()

	klog.Info("Starting CSRApprover")
	defer klog.Info("Shutting down CSRApprover")

	if !cache.WaitForNamedCacheSync("CSRApprover", stopCh, cc.csrsSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(cc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a thread that dequeues CSRs, handles them, and marks them done.
func (cc *CSRApprover) worker() {
	for cc.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (cc *CSRApprover) processNextWorkItem() bool {
	cKey, quit := cc.queue.Get()
	if quit {
		return false
	}
	defer cc.queue.Done(cKey)

	if err := cc.syncFunc(cKey.(string)); err != nil {
		cc.queue.AddRateLimited(cKey)
		utilruntime.HandleError(fmt.Errorf("sync %v failed with : %v", cKey, err))
		return true
	}

	cc.queue.Forget(cKey)
	return true
}

func (cc *CSRApprover) enqueueCertificateRequest(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", obj, err))
		return
	}
	cc.queue.Add(key)
}

func (cc *CSRApprover) syncFunc(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing certificate request %q (%v)", key, time.Since(startTime))
	}()
	csr, err := cc.csrLister.Get(key)
	if apierr.IsNotFound(err) {
		klog.V(3).Infof("csr has been deleted: %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	if len(csr.Status.Certificate) > 0 {
		// no need to do anything because it already has a cert
		return nil
	}

	// need to operate on a copy so we don't mutate the csr in the shared cache
	csr = csr.DeepCopy()
	return cc.handle(csr)
}

func recognizers() []csrRecognizer {
	recognizers := []csrRecognizer{
		{
			recognize:      isMetaServerServingCert,
			successMessage: "Auto approving MetaServer serving certificate.",
		},
	}
	return recognizers
}

func (cc *CSRApprover) handle(csr *capi.CertificateSigningRequest) error {
	if approved, denied := GetCertApprovalCondition(&csr.Status); approved || denied {
		return nil
	}

	x509cr, err := capihelper.ParseCSR(csr.Spec.Request)
	if err != nil {
		return fmt.Errorf("unable to parse csr %q: %v", csr.Name, err)
	}

	for _, r := range cc.recognizers {
		if !r.recognize(csr, x509cr) {
			continue
		}

		appendApprovalCondition(csr, r.successMessage)
		_, err = cc.kubeClient.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.Background(), csr.Name, csr, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("error updating approval for csr: %v", err)
		}
		return nil
	}

	return nil
}

func appendApprovalCondition(csr *capi.CertificateSigningRequest, message string) {
	csr.Status.Conditions = append(csr.Status.Conditions, capi.CertificateSigningRequestCondition{
		Type:    capi.CertificateApproved,
		Status:  corev1.ConditionTrue,
		Reason:  "AutoApproved",
		Message: message,
	})
}

func isMetaServerServingCert(csr *capi.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool {
	if csr.Spec.SignerName != capi.KubeletServingSignerName {
		return false
	}
	return capihelper.IsKubeletServingCSR(x509cr, usagesToSet(csr.Spec.Usages))
}

func usagesToSet(usages []capi.KeyUsage) sets.String {
	result := sets.NewString()
	for _, usage := range usages {
		result.Insert(string(usage))
	}
	return result
}

// GetCertApprovalCondition returns the current condition of the CSR (approved, denied)
// Source: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/certificates/certificate_controller_utils.go
func GetCertApprovalCondition(status *capi.CertificateSigningRequestStatus) (approved, denied bool) {
	for _, c := range status.Conditions {
		if c.Type == capi.CertificateApproved {
			approved = true
		}

		if c.Type == capi.CertificateDenied {
			denied = true
		}
	}

	return
}
