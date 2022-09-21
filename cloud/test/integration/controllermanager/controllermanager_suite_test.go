package controllermanager

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

// Values of the following two variables will be linked when
// building the test binary.
var appsCRDDirectoryPath string
var envtestBinDir string

var (
	cfg       *rest.Config
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	k8sClient client.Client
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{appsCRDDirectoryPath},
		BinaryAssetsDirectory: envtestBinDir,
	}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).To(BeNil())
	Expect(cfg).NotTo(BeNil())

	By("preparing a live client")
	err = appsv1alpha1.Install(scheme.Scheme)
	Expect(err).To(BeNil())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).To(BeNil())
	Expect(k8sClient).NotTo(BeNil())

	By("starting controller manager")
	controllerManager, err := controllermanager.NewAppsControllerManager(ctx, cfg)
	Expect(err).To(BeNil())
	go func() {
		defer GinkgoRecover()
		err = controllerManager.Start(ctx)
		Expect(err).To(BeNil())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).To(BeNil())
})

func TestAppsAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeGroup Test Suite")
}
