package admissioncontroller

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	admissionregistrationv1beta1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
)

const (
	ValidateDeviceModelConfigName  = "validate-devicemodel"
	ValidateDeviceModelWebhookName = "validatedevicemodel.kubeedge.io"
)

var scheme = runtime.NewScheme()

//Codecs is for retrieving serializers for the supported wire formats
//and conversion wrappers to define preferred internal and external versions.
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1beta1.AddToScheme(scheme))
	utilruntime.Must(addDeviceCrds(scheme))
}

// TODO: move this func to apis/devices/v1alpha1/register.go
func addDeviceCrds(scheme *runtime.Scheme) error {
	// Add Device
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Device{}, &v1alpha1.DeviceList{})
	metav1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)
	// Add DeviceModel
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.DeviceModel{}, &v1alpha1.DeviceModelList{})
	metav1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)

	return nil
}

// AdmissionController implements the admission webhook for validation of configuration.
type AdmissionController struct {
	Client *kubernetes.Clientset
}

func strPtr(s string) *string { return &s }

// Run starts the webhook service
func Run(config *options.Config) {
	restConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.Kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalf("Create kube client failed with error: %v", err)
	}

	ac := AdmissionController{}
	ac.Client = cli

	caBundle, err := ioutil.ReadFile(config.CaCertFile)
	if err != nil {
		klog.Fatalf("Unable to read cacert file: %v\n", err)
	}

	//TODO: read somewhere to get what's kind of webhook is enabled, register those webhook only.
	err = ac.registerWebhooks(config, caBundle)
	if err != nil {
		klog.Fatalf("Failed to register the webhook with error: %v", err)
	}

	http.HandleFunc("/devicemodels", serveDeviceModel)

	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", config.Port),
		TLSConfig: configTLS(config, restConfig),
	}

	server.ListenAndServeTLS("", "")
}

// ConfigTLS is a helper function that generate tls certificates from directly defined tls config or kubeconfig
// These are passed in as command line for cluster certification. If tls config is passed in, we use the directly
// defined tls config, else use that defined in kubeconfig
func configTLS(config *options.Config, restConfig *restclient.Config) *tls.Config {
	if len(config.CertFile) != 0 && len(config.KeyFile) != 0 {
		sCert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			klog.Fatal(err)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}
	}

	if len(restConfig.CertData) != 0 && len(restConfig.KeyData) != 0 {
		sCert, err := tls.X509KeyPair(restConfig.CertData, restConfig.KeyData)
		if err != nil {
			klog.Fatal(err)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{sCert},
		}
	}

	klog.Fatal("tls: failed to find any tls config data")
	return &tls.Config{}
}

// Register registers the admission webhook.
func (ac *AdmissionController) registerWebhooks(c *options.Config, cabundle []byte) error {
	ignorePolicy := admissionregistrationv1beta1.Ignore
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
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"devicemodels"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: c.AdmissionServiceNamespace,
						Name:      c.AdmissionServiceName,
						Path:      strPtr("/devicemodels"),
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
	return nil
}

func registerValidateWebhook(client admissionregistrationv1beta1client.ValidatingWebhookConfigurationInterface,
	webhooks []admissionregistrationv1beta1.ValidatingWebhookConfiguration) error {
	for _, hook := range webhooks {
		existing, err := client.Get(hook.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil && existing != nil {
			existing.Webhooks = hook.Webhooks
			klog.Infof("Updating ValidatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Update(existing); err != nil {
				return err
			}
		} else {
			klog.Infof("Creating ValidatingWebhookConfiguration: %v", hook.Name)
			if _, err := client.Create(&hook); err != nil {
				return err
			}
		}
	}
	return nil
}
