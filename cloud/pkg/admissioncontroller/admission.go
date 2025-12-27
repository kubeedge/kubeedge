package admissioncontroller

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	v1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
)

const (
	ValidateCRDWebhookConfigName    = "kubeedge-crds-validate-webhook-configuration"
	ValidateDeviceWebhookName       = "validatedevice.kubeedge.io"
	ValidateDeviceModelWebhookName  = "validatedevicemodel.kubeedge.io"
	ValidateRuleWebhookName         = "validatedrule.kubeedge.io"
	ValidateRuleEndpointWebhookName = "validatedruleendpoint.kubeedge.io"
	ValidateNodeUpgradeWebhookName  = "validatenodeupgradejob.kubeedge.io"

	OfflineMigrationConfigName  = "mutate-offlinemigration"
	OfflineMigrationWebhookName = "mutateofflinemigration.kubeedge.io"

	MutatingAdmissionWebhookName   = "kubeedge-mutating-webhook"
	MutatingNodeUpgradeWebhookName = "mutatingnodeupgradejob.kubeedge.io"

	AutonomyLabel = "app-offline.kubeedge.io=autonomy"
)

var scheme = runtime.NewScheme()

// codecs is for retrieving serializers for the supported wire formats
// and conversion wrappers to define preferred internal and external versions.
var codecs = serializer.NewCodecFactory(scheme)

var controller = &AdmissionController{}

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddDeviceCrds(scheme))
}

// AdmissionController implements the admission webhook for validation of configuration.
type AdmissionController struct {
	Client    *kubernetes.Clientset
	CrdClient *versioned.Clientset
}

func strPtr(s string) *string { return &s }

// Run starts the webhook service
// Run starts the webhook service
func Run(opt *options.AdmissionOptions) error {
	klog.V(4).Infof("AdmissionOptions: %+v", *opt)
	klog.Info("Initializing admission webhook controller")
	
	restConfig, err := clientcmd.BuildConfigFromFlags(opt.Master, opt.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build REST config from flags: %v", err)
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kube client failed: %v", err)
	}
	vcli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create versioned client failed: %v", err)
	}

	controller.Client = cli
	controller.CrdClient = vcli
	klog.Info("Admission controller clients initialized successfully")

	caBundle, err := os.ReadFile(opt.CaCertFile)
	if err != nil {
		return fmt.Errorf("unable to read CA certificate file %s: %v", opt.CaCertFile, err)
	}
	klog.Info("CA certificate bundle loaded successfully")

	klog.Info("Registering admission webhooks")
	if err = controller.registerWebhooks(opt, caBundle); err != nil {
		return fmt.Errorf("failed to register webhooks: %v", err)
	}
	klog.Info("Webhooks registered successfully")

	http.HandleFunc("/devices", serveDevice)
	http.HandleFunc("/devicemodels", serveDeviceModel)
	http.HandleFunc("/rules", serveRule)
	http.HandleFunc("/ruleendpoints", serveRuleEndpoint)
	http.HandleFunc("/offlinemigration", serveOfflineMigration)
	http.HandleFunc("/nodeupgradejobs", serveNodeUpgradeJob)
	http.HandleFunc("/mutating/nodeupgradejobs", serveMutatingNodeUpgradeJob)

	klog.Info("Configuring TLS for webhook server")
	tlsConfig, err := configTLS(opt, restConfig)
	if err != nil {
		return fmt.Errorf("failed to configure TLS: %v", err)
	}
	klog.Info("TLS configuration completed successfully")
	
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", opt.Port),
		TLSConfig: tlsConfig,
	}

	klog.Infof("Starting admission webhook server on port %v", opt.Port)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("webhook server failed: %v", err)
	}
	
	klog.Info("Admission webhook server shutdown completed")
	return nil
}

// configTLS is a helper function that generate tls certificates from directly defined tls config or kubeconfig
// These are passed in as command line for cluster certification. If tls config is passed in, we use the directly
// defined tls config, else use that defined in kubeconfig
func configTLS(opt *options.AdmissionOptions, restConfig *restclient.Config) (*tls.Config, error) {
	if len(opt.CertFile) != 0 && len(opt.KeyFile) != 0 {
		klog.Infof("Loading TLS certificate from files: %s, %s", opt.CertFile, opt.KeyFile)
		sCert, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair from files: %v", err)
		}

		klog.Info("TLS certificate loaded successfully from files")
		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}, nil
	}

	if len(restConfig.CertData) != 0 && len(restConfig.KeyData) != 0 {
		klog.Info("Loading TLS certificate from REST config data")
		sCert, err := tls.X509KeyPair(restConfig.CertData, restConfig.KeyData)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair from config data: %v", err)
		}

		klog.Info("TLS certificate loaded successfully from config data")
		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}, nil
	}
	
	klog.Error("No TLS configuration data found")
	return nil, errors.New("tls: failed to find any TLS configuration data, please provide certificate files or kubeconfig with certificate data")
}

// registerWebhooks registers the admission webhook.
func (ac *AdmissionController) registerWebhooks(opt *options.AdmissionOptions, cabundle []byte) error {
	klog.Info("Starting webhook registration process")
	
	ignorePolicy := admissionregistrationv1.Ignore
	failPolicy := admissionregistrationv1.Fail
	noneSideEffect := admissionregistrationv1.SideEffectClassNone

	klog.Info("Creating validating webhook configuration")
	// validating webhook configuration
	validatingWebhookConfiguration := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			// there'll be two ValidatingWebhookConfigurations, to keep compatible, we can only keep one webhook for one CRD
			Name: ValidateCRDWebhookConfigName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			// Device Validating Webhook
			{
				Name: ValidateDeviceWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"devices.kubeedge.io"},
						APIVersions: []string{"v1beta1"},
						Resources:   []string{"devices"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/devices"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy:           &ignorePolicy,
				SideEffects:             &noneSideEffect,
				AdmissionReviewVersions: []string{"v1"},
			},
			{ // Device Model Validating Webhook

				Name: ValidateDeviceModelWebhookName,
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"devices.kubeedge.io"},
						APIVersions: []string{"v1beta1"},
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
	
	klog.Infof("Registering validating webhook configuration: %s", ValidateCRDWebhookConfigName)
	if err := registerValidateWebhook(ac.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations(),
		[]admissionregistrationv1.ValidatingWebhookConfiguration{validatingWebhookConfiguration}); err != nil {
		klog.Errorf("Failed to register validating webhooks: %v", err)
		return fmt.Errorf("failed to register validating webhooks: %v", err)
	}
	klog.Info("Validating webhooks registered successfully")

	klog.Info("Creating mutating webhook configurations")
	objectSelector, err := metav1.ParseToLabelSelector(AutonomyLabel)
	if err != nil {
		klog.Errorf("Failed to parse object selector: %v", err)
		return fmt.Errorf("failed to parse object selector: %v", err)
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

	// NodeUpgradeJob mutating webhook
	nodeUpgradeJobWebhook := admissionregistrationv1.MutatingWebhook{
		Name: MutatingNodeUpgradeWebhookName,
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
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
				Path:      strPtr("/mutating/nodeupgradejobs"),
				Port:      &opt.Port,
			},
			CABundle: cabundle,
		},
		FailurePolicy:           &ignorePolicy,
		SideEffects:             &noneSideEffect,
		AdmissionReviewVersions: []string{"v1"},
	}
	
	// mutatingWebhook contains all the kubeedge related Mutating webhook
	mutatingWebhook := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: MutatingAdmissionWebhookName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			nodeUpgradeJobWebhook,
		},
	}

	klog.Infof("Registering mutating webhook configurations: %s, %s", 
		OfflineMigrationConfigName, MutatingAdmissionWebhookName)
		
	if err := registerMutatingWebhook(ac.Client.AdmissionregistrationV1().MutatingWebhookConfigurations(),
		[]admissionregistrationv1.MutatingWebhookConfiguration{offlineMigrationWebhook, mutatingWebhook}); err != nil {
		klog.Errorf("Failed to register mutating webhooks: %v", err)
		return fmt.Errorf("failed to register mutating webhooks: %v", err)
	}
	
	klog.Info("Webhook configuration completed successfully")
	return nil
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
