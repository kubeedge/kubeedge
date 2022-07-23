package admissioncontroller

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

const (
	ValidateDeviceModelConfigName   = "validate-devicemodel"
	ValidateDeviceModelWebhookName  = "validatedevicemodel.kubeedge.io"
	ValidateRuleWebhookName         = "validatedrule.kubeedge.io"
	ValidateRuleEndpointWebhookName = "validatedruleendpoint.kubeedge.io"
	ValidateNodeUpgradeWebhookName  = "validatenodeupgradejob.kubeedge.io"

	OfflineMigrationConfigName  = "mutate-offlinemigration"
	OfflineMigrationWebhookName = "mutateofflinemigration.kubeedge.io"

	AutonomyLabel = "app-offline.kubeedge.io=autonomy"

	caPkiPath        = "/etc/kubeedge/ca/"
	caPkiName        = "rootCA"
	admissionPkiPath = "/etc/kubeedge/certs/"
	admissionPkiName = "admission"
)

var scheme = runtime.NewScheme()

//codecs is for retrieving serializers for the supported wire formats
//and conversion wrappers to define preferred internal and external versions.
var codecs = serializer.NewCodecFactory(scheme)

var controller = &AdmissionController{}

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddDeviceCrds(scheme))
}

// AdmissionController implements the admission webhook for validation of configuration.
type AdmissionController struct {
	Client    *kubernetes.Clientset
	CrdClient *versioned.Clientset
}

func strPtr(s string) *string { return &s }

type CertificatePath struct {
	CaCertFile string
	CertFile   string
	KeyFile    string
}

// Run starts the webhook service
func Run(opt *options.AdmissionOptions) error {
	klog.V(4).Infof("AdmissionOptions: %+v", *opt)
	restConfig, err := clientcmd.BuildConfigFromFlags(opt.Master, opt.Kubeconfig)
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kube client failed with error: %v", err)
	}
	vcli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create versioned client failed with error: %v", err)
	}

	controller.Client = cli
	controller.CrdClient = vcli

	cert, err := certificates(opt)
	if err != nil {
		return err
	}

	caBundle, err := os.ReadFile(cert.CaCertFile)
	if err != nil {
		return fmt.Errorf("unable to read cacert file: %v", err)
	}

	//TODO: read somewhere to get what's kind of webhook is enabled, register those webhook only.
	if err = controller.registerWebhooks(opt, caBundle); err != nil {
		return fmt.Errorf("failed to register the webhook with error: %v", err)
	}

	http.HandleFunc("/devicemodels", serveDeviceModel)
	http.HandleFunc("/rules", serveRule)
	http.HandleFunc("/ruleendpoints", serveRuleEndpoint)
	http.HandleFunc("/offlinemigration", serveOfflineMigration)
	http.HandleFunc("/nodeupgradejobs", serveNodeUpgradeJob)

	tlsConfig, err := configTLS(cert, restConfig)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", opt.Port),
		TLSConfig: tlsConfig,
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("start server failed with error: %v", err)
	}
	return nil
}

// get certificate path
func certificates(opt *options.AdmissionOptions) (*CertificatePath, error) {
	// if user specify the ca and certs in command line, we will use the user defined certificates
	if opt.CaCertFile != "" && opt.CertFile != "" && opt.KeyFile != "" {
		return &CertificatePath{
			CaCertFile: opt.CaCertFile,
			CertFile:   opt.CertFile,
			KeyFile:    opt.KeyFile,
		}, nil
	}

	// or use the default ca/certs stored in kubeedge-admission-secret
	// check whether kubeedge-admission-secret exist or not
	// if NOT, generate it and store it in secret
	// if exist, read certificate from it and store it to local
	_, err := controller.Client.CoreV1().Secrets(opt.AdmissionServiceNamespace).Get(context.TODO(), "kubeedge-admission-secret", metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get secret(%s/kubeedge-admission-secret): %v", opt.AdmissionServiceNamespace, err)
	}

	// If secret not found, generate certificate and store in secret
	if apierrors.IsNotFound(err) {
		// generate certificate
		klog.Infof("Generate ca and certs...")
		if err := genCerts(opt); err != nil {
			return nil, fmt.Errorf("failed to generate certs: %v", err)
		}

		keyData, err := os.ReadFile(filepath.Join(admissionPkiPath, fmt.Sprintf("%s.key", admissionPkiName)))
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %v", err)
		}
		certData, err := os.ReadFile(filepath.Join(admissionPkiPath, fmt.Sprintf("%s.crt", admissionPkiName)))
		if err != nil {
			return nil, fmt.Errorf("failed to read cert file: %v", err)
		}
		caData, err := os.ReadFile(filepath.Join(caPkiPath, fmt.Sprintf("%s.crt", caPkiName)))
		if err != nil {
			return nil, fmt.Errorf("failed to read ca cert file: %v", err)
		}

		// store certificate data in secret
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeedge-admission-secret",
				Namespace: opt.AdmissionServiceNamespace,
			},
			Data: map[string][]byte{
				"tls.key": keyData,
				"tls.crt": certData,
				"ca.crt":  caData,
			},
		}
		_, err = controller.Client.CoreV1().Secrets(opt.AdmissionServiceNamespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create secret: %v", err)
		}

		// If secret already exists, it means other replicas already generate certificates
		// we need to read certificates from secret again
		if apierrors.IsAlreadyExists(err) {
			klog.Infof("Use existed ca and cert in admission secret")
			return LoadCertificateFromSecret(opt.AdmissionServiceNamespace)
		}

		return &CertificatePath{
			CaCertFile: filepath.Join(caPkiPath, fmt.Sprintf("%s.crt", caPkiName)),
			CertFile:   filepath.Join(admissionPkiPath, fmt.Sprintf("%s.crt", admissionPkiName)),
			KeyFile:    filepath.Join(admissionPkiPath, fmt.Sprintf("%s.key", admissionPkiName)),
		}, nil
	}

	// If secret exist, read certificate from it and write to local file
	klog.Infof("Use existed ca and cert in admission secret")
	return LoadCertificateFromSecret(opt.AdmissionServiceNamespace)
}

// LoadCertificateFromSecret read certificates from secret and store them in local filesystem
// Attention: must ensure the secret already exist before calling this function
func LoadCertificateFromSecret(namespace string) (*CertificatePath, error) {
	secret, err := controller.Client.CoreV1().Secrets(namespace).Get(context.TODO(), "kubeedge-admission-secret", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret(%s/kubeedge-admission-secret): %v", namespace, err)
	}

	// prepare certs directory
	if err := os.MkdirAll(filepath.Dir(caPkiPath), os.FileMode(0755)); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(admissionPkiPath), os.FileMode(0755)); err != nil {
		return nil, err
	}
	// write certs to local filesystem
	if err := os.WriteFile(filepath.Join(admissionPkiPath, fmt.Sprintf("%s.key", admissionPkiName)), secret.Data["tls.key"], os.FileMode(0644)); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(admissionPkiPath, fmt.Sprintf("%s.crt", admissionPkiName)), secret.Data["tls.crt"], os.FileMode(0644)); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(caPkiPath, fmt.Sprintf("%s.crt", caPkiName)), secret.Data["ca.crt"], os.FileMode(0644)); err != nil {
		return nil, err
	}

	return &CertificatePath{
		CaCertFile: filepath.Join(caPkiPath, fmt.Sprintf("%s.crt", caPkiName)),
		CertFile:   filepath.Join(admissionPkiPath, fmt.Sprintf("%s.crt", admissionPkiName)),
		KeyFile:    filepath.Join(admissionPkiPath, fmt.Sprintf("%s.key", admissionPkiName)),
	}, nil
}

// configTLS is a helper function that generate tls certificates from directly defined tls config or kubeconfig
// These are passed in as command line for cluster certification. If tls config is passed in, we use the directly
// defined tls config, else use that defined in kubeconfig
func configTLS(opt *CertificatePath, restConfig *restclient.Config) (*tls.Config, error) {
	if len(opt.CertFile) != 0 && len(opt.KeyFile) != 0 {
		sCert, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
		if err != nil {
			return nil, err
		}

		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}, nil
	}

	if len(restConfig.CertData) != 0 && len(restConfig.KeyData) != 0 {
		sCert, err := tls.X509KeyPair(restConfig.CertData, restConfig.KeyData)
		if err != nil {
			return nil, err
		}

		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}, nil
	}
	return nil, errors.New("tls: failed to find any tls config data")
}

func genCerts(opt *options.AdmissionOptions) error {
	notAfter := time.Now().Add(time.Hour * 24 * 365 * 10).UTC()

	var kubeedgeDNS = []string{
		"localhost",
		opt.AdmissionServiceName,
		fmt.Sprintf("%s.%s", opt.AdmissionServiceName, opt.AdmissionServiceNamespace),
		fmt.Sprintf("%s.%s.svc", opt.AdmissionServiceName, opt.AdmissionServiceNamespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", opt.AdmissionServiceName, opt.AdmissionServiceNamespace),
	}

	kubeedgeIPs := []net.IP{
		net.ParseIP("127.0.0.1"),
	}

	kubeedgeAltNames := certutil.AltNames{
		DNSNames: kubeedgeDNS,
		IPs:      kubeedgeIPs,
	}

	kubeedgeCertCfg := NewCertConfig("system:admin", []string{"system:masters"}, kubeedgeAltNames, &notAfter)

	return GenCerts(kubeedgeCertCfg)
}

// registerWebhooks registers the admission webhook.
func (ac *AdmissionController) registerWebhooks(opt *options.AdmissionOptions, cabundle []byte) error {
	ignorePolicy := admissionregistrationv1.Ignore
	failPolicy := admissionregistrationv1.Fail
	noneSideEffect := admissionregistrationv1.SideEffectClassNone

	// validating webhook configuration
	validatingWebhookConfiguration := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			// TODO: this config name is not precise, think a way to change it more common, like `validate-kubeedge-crd`
			// but there'll be two ValidatingWebhookConfigurations, to keep compatible, we can only keep one webhook for one CRD
			Name: ValidateDeviceModelConfigName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			// Device Model Validating Webhook
			{
				Name: ValidateDeviceModelWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"devices.kubeedge.io"},
						APIVersions: []string{"v1alpha2"},
						Resources:   []string{"devicemodels"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/devicemodels"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &ignorePolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
			// Rule Validating Webhook
			{
				Name: ValidateRuleWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
						admissionregistrationv1.Delete,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"rules.kubeedge.io"},
						APIVersions: []string{"v1"},
						Resources:   []string{"rules"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/rules"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &failPolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
			// Rule Endpoint Validating Webhook
			{
				Name: ValidateRuleEndpointWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
						admissionregistrationv1.Delete,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"rules.kubeedge.io"},
						APIVersions: []string{"v1"},
						Resources:   []string{"ruleendpoints"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/ruleendpoints"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &failPolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
			// NodeUpgradeJob validating webhook
			{
				Name: ValidateNodeUpgradeWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
						admissionregistrationv1.Delete,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"operations.kubeedge.io"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"nodeupgradejobs"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/nodeupgradejobs"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &failPolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
	if err := registerValidateWebhook(ac.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations(),
		[]admissionregistrationv1.ValidatingWebhookConfiguration{validatingWebhookConfiguration}); err != nil {
		return err
	}

	objectSelector, err := metav1.ParseToLabelSelector(AutonomyLabel)
	if err != nil {
		return err
	}
	// offlineMigration Mutating webhook
	offlineMigrationWebhook := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: OfflineMigrationConfigName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name:           OfflineMigrationWebhookName,
				ObjectSelector: objectSelector,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/offlinemigration"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &ignorePolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}

	return registerMutatingWebhook(ac.Client.AdmissionregistrationV1().MutatingWebhookConfigurations(),
		[]admissionregistrationv1.MutatingWebhookConfiguration{offlineMigrationWebhook})
}

func (ac *AdmissionController) getRuleEndpoint(namespace, name string) (*v1.RuleEndpoint, error) {
	return ac.CrdClient.RulesV1().RuleEndpoints(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (ac *AdmissionController) listRule(namespace string) ([]v1.Rule, error) {
	rules, err := ac.CrdClient.RulesV1().Rules(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rules.Items, nil
}

func (ac *AdmissionController) listRuleEndpoint(namespace string) ([]v1.RuleEndpoint, error) {
	rules, err := ac.CrdClient.RulesV1().RuleEndpoints(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rules.Items, nil
}
