package admissioncontroller

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
	OfflineMigrationConfigName      = "mutate-offlinemigration"
	OfflineMigrationWebhookName     = "mutateofflinemigration.kubeedge.io"

	AutonomyLabel = "app-offline.kubeedge.io=autonomy"
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
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1beta1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddDeviceCrds(scheme))
}

// AdmissionController implements the admission webhook for validation of configuration.
type AdmissionController struct {
	Client    *kubernetes.Clientset
	CrdClient *versioned.Clientset
}

func strPtr(s string) *string { return &s }

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

	caBundle, err := os.ReadFile(opt.CaCertFile)
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

	tlsConfig, err := configTLS(opt, restConfig)
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

// configTLS is a helper function that generate tls certificates from directly defined tls config or kubeconfig
// These are passed in as command line for cluster certification. If tls config is passed in, we use the directly
// defined tls config, else use that defined in kubeconfig
func configTLS(opt *options.AdmissionOptions, restConfig *restclient.Config) (*tls.Config, error) {
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

// registerWebhooks registers the admission webhook.
func (ac *AdmissionController) registerWebhooks(opt *options.AdmissionOptions, cabundle []byte) error {
	ignorePolicy := admissionregistrationv1beta1.Ignore
	failPolicy := admissionregistrationv1beta1.Fail
	deviceModelCRDWebhook := admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ValidateDeviceModelConfigName,
		},
		Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{
			{
				Name: ValidateDeviceModelWebhookName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"devices.kubeedge.io"},
						APIVersions: []string{"v1alpha2"},
						Resources:   []string{"devicemodels"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/devicemodels"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy: &ignorePolicy,
			},
			{
				Name: ValidateRuleWebhookName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
						admissionregistrationv1beta1.Delete,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"rules.kubeedge.io"},
						APIVersions: []string{"v1"},
						Resources:   []string{"rules"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/rules"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy: &failPolicy,
			},
			{
				Name: ValidateRuleEndpointWebhookName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
						admissionregistrationv1beta1.Delete,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"rules.kubeedge.io"},
						APIVersions: []string{"v1"},
						Resources:   []string{"ruleendpoints"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/ruleendpoints"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy: &failPolicy,
			},
		},
	}

	objectSelector, err := metav1.ParseToLabelSelector(AutonomyLabel)
	if err != nil {
		return err
	}
	offlineMigrationWebhook := admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: OfflineMigrationConfigName,
		},
		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
			{
				Name:           OfflineMigrationWebhookName,
				ObjectSelector: objectSelector,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: opt.AdmissionServiceNamespace,
						Name:      opt.AdmissionServiceName,
						Path:      strPtr("/offlinemigration"),
						Port:      &opt.Port,
					},
					CABundle: cabundle,
				},
				FailurePolicy: &ignorePolicy,
			},
		},
	}

	if err := registerValidateWebhook(ac.Client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations(),
		[]admissionregistrationv1beta1.ValidatingWebhookConfiguration{deviceModelCRDWebhook}); err != nil {
		return err
	}
	return registerMutatingWebhook(ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(),
		[]admissionregistrationv1beta1.MutatingWebhookConfiguration{offlineMigrationWebhook})
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
