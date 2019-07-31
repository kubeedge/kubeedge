package controller

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/utils"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1beta1.AddToScheme(scheme))
}

// AdmissionController implements the admission webhook for validation of configuration.
type AdmissionController struct {
	Client *kubernetes.Clientset
}

func strPtr(s string) *string { return &s }

// Register registers the admission webhook.
// FIXME: plugable?
func (ac *AdmissionController) register(WebhookName string, context *utils.CertContext) error {
	webhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{
			{
				Name: WebhookName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
						admissionregistrationv1beta1.Delete,
						admissionregistrationv1beta1.Connect,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"devices.kubeedge.io"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"devicemodels"},
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: constants.NamespaceName,
						Name:      constants.ServiceName,
						Path:      strPtr("/devicemodels"),
						Port:      pointer.Int32Ptr(constants.Port),
					},
					CABundle: context.SigningCert,
				},
			},
		},
	}

	if err := ac.Client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(webhook.Name, nil); err != nil {
		serr, ok := err.(*errors.StatusError)
		if !ok || serr.ErrStatus.Code != http.StatusNotFound {
			klog.Warningf("Could not delete existing webhook configuration: %v", err)
		}
	}

	_, err := ac.Client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(webhook)
	return err
}

// Start starts the webhook service
func (ac *AdmissionController) Start(context *utils.CertContext) {
	err := ac.register(constants.ExternalAdmissionWebhookName, context)
	if err != nil {
		klog.Fatalf("Failed to register the webhook with error: %v", err)
	}
	ac.deployService()
	tlsConfig := configTLS(context)
	http.Handle("/devicemodels", ac) //switch to something like http.HandleFunc("/devicemodels", admitDeviceModel) later.
	server := &http.Server{
		Handler:   ac,
		Addr:      fmt.Sprintf(":%v", constants.Port),
		TLSConfig: tlsConfig,
	}

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			klog.Fatalf("ListenAndServeTLS for admission webhook returned error: %v", err)
		}
	}()
}

func (ac *AdmissionController) deployService() {
	localIP := GetIPv4Addr()
	if localIP == "" {
		klog.Error("Cannot get one local valid IP")
	}
	//create service
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.NamespaceName,
			Name:      constants.ServiceName,
		},
		Spec: v1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []v1.ServicePort{
				{
					Port:       constants.Port,
					TargetPort: intstr.FromInt(constants.Port),
				},
			},
		},
	}
	_, err := ac.Client.CoreV1().Services(constants.NamespaceName).Create(service)
	if err != nil {
		klog.Fatalf("Failed to create webhook service with error: %s", err)
	}

	//create Endpoints
	endpoint := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.ServiceName,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: localIP,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port: constants.Port,
					},
				},
			},
		},
	}
	_, err = ac.Client.CoreV1().Endpoints(constants.NamespaceName).Create(endpoint)
	if err != nil {
		klog.Fatalf("Failed to create endpoints with error: %s", err)
	}
}

func (ac *AdmissionController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Fatalf("contentType=%s, expect application/json", contentType)
		return
	}

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := admissionv1beta1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := admissionv1beta1.AdmissionReview{}

	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Fatalf("decode failed with error: %v", err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		responseAdmissionReview.Response = ac.admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	klog.Infof("sending response: %v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Fatalf("cannot marshal to a valid reponse %v", err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Fatalf("cannot write reponse %v", err)
	}
}

func (ac *AdmissionController) admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	var msg string
	switch review.Request.Operation {
	case admissionv1beta1.Create, admissionv1beta1.Update, admissionv1beta1.Delete, admissionv1beta1.Connect:
		//TODO: abnormal configuration will be detected here, so far, greenlight for all of them.
		reviewResponse.Allowed = true
		log.LOGGER.Info("pass admission validation!")
	default:
		log.LOGGER.Infof("Unsupported webhook operation %v", review.Request.Operation)
		reviewResponse.Allowed = false
		msg = msg + "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func configTLS(context *utils.CertContext) *tls.Config {
	sCert, err := tls.X509KeyPair(context.Cert, context.Key)
	if err != nil {
		log.LOGGER.Fatalf("load certification failed with error: %v", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
func toAdmissionResponse(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func GetIPv4Addr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
