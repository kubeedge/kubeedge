/*
Copyright 2022 The KubeEdge Authors.

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

package controllermanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager"
)

// Values of the following two variables will be linked when
// building the test binary.
var appsCRDDirectoryPath string
var envtestBinDir string

var (
	cfg             *rest.Config
	ctx             context.Context
	cancel          context.CancelFunc
	testEnv         *envtest.Environment
	k8sClient       client.Client
	webhookPort     int
	healthProbePort int
	webhookCertDir  string
	managerDone     chan error
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	webhookPort, err = freeWebhookTestPort()
	Expect(err).NotTo(HaveOccurred())
	healthProbePort, err = freeWebhookTestPort()
	Expect(err).NotTo(HaveOccurred())
	webhookCertDir, err = os.MkdirTemp("", "kubeedge-webhook-test-")
	Expect(err).NotTo(HaveOccurred())
	Expect(writeWebhookTestTLSCert(webhookCertDir)).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{appsCRDDirectoryPath},
		BinaryAssetsDirectory: envtestBinDir,
	}
	cfg, err = testEnv.Start()
	Expect(err).To(BeNil())
	Expect(cfg).NotTo(BeNil())

	By("preparing a live client")
	err = appsv1alpha1.Install(scheme.Scheme)
	Expect(err).To(BeNil())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).To(BeNil())
	Expect(k8sClient).NotTo(BeNil())

	By("starting controller manager with webhook server")
	healthProbeAddress := fmt.Sprintf("127.0.0.1:%d", healthProbePort)
	controllerManager, err := controllermanager.NewControllerManager(
		ctx,
		cfg,
		healthProbeAddress,
		webhookPort,
		webhookCertDir,
	)
	Expect(err).To(BeNil())

	controllerManager.GetWebhookServer().Register("/test-admission", &admission.Webhook{
		Handler: admission.HandlerFunc(func(_ context.Context, _ admission.Request) admission.Response {
			return admission.Allowed("test webhook")
		}),
	})

	managerDone = make(chan error, 1)
	go func() {
		managerDone <- controllerManager.Start(ctx)
	}()
})

var _ = AfterSuite(func() {
	By("stopping controller manager")
	cancel()

	select {
	case err := <-managerDone:
		Expect(err).NotTo(HaveOccurred())
	case <-time.After(15 * time.Second):
		Fail("controller manager did not stop within 15 seconds")
	}

	By("verifying webhook port is released")
	Eventually(func() bool {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", webhookPort))
		if err != nil {
			return false
		}
		_ = listener.Close()
		return true
	}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())

	Expect(os.RemoveAll(webhookCertDir)).To(Succeed())

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).To(BeNil())
})

func TestAppsAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeGroup Test Suite")
}
