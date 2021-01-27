package edgegateway

import (
	"context"
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/annotations/class"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/controller"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/controller/store"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/k8s"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/net/ssl"
	err1 "github.com/pkg/errors"
	err2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/cri-api/pkg/errors"
	"k8s.io/klog"
	"math/rand"
	"os"
	"time"
)

const(
	// ReadWriteByUser defines linux permission to read and write files for the owner user
	ReadWriteByUser = 0700
)

var(
	// create directories to store ingress file
	directories = []string{
		DefaultSSLDirectory,
		AuthDirectory,
	}

	// DefaultSSLDirectory defines the location where the SSL certificates will be generated
	// This directory contains all the SSL certificates that are specified in internal rules.
	// The name of each file is <namespace>-<secret name>.pem. The content is the concatenated
	// certificate and key.
	DefaultSSLDirectory = "/etc/ingress-controller/ssl"

	// AuthDirectory default directory used to store files
	// to authenticate request
	AuthDirectory = "/etc/ingress-controller/auth"
)

// discovery contains the runtime configuration for the EdgeGateway discovery service.
type discovery struct {
	// config
	config *config.Configure
	// store
	store *store.Storer
	// informer list and watch
	informer *store.Informer
}

// Start discovery server
func (d *discovery) Start(kubeClient clientset.Interface) (err error , conf *controller.Configuration) {

	klog.InitFlags(nil)

	rand.Seed(time.Now().UnixNano())

	// get parse flags into conf
	showVersion , conf , err := ParseFlags()

	if showVersion {
		os.Exit(0)
	}

	if err != nil {
		klog.Fatal(err)
	}

	// create directories to storage ingress-nginx file
	err = CreateRequiredDirectories()
	if err!=nil  {
		klog.Fatal(err)
	}

	if len(conf.DefaultService) >0  {
		err := checkService(conf.DefaultService , kubeClient )
		if err != nil {
			klog.Fatal(err)
		}
		klog.Infof("Valid default backend", "service", conf.DefaultService)
	}

	if len(conf.PublishService) > 0 {
		err := checkService(conf.PublishService, kubeClient)
		if err != nil {
			klog.Fatal(err)
		}
	}

	if conf.Namespace != "" {
		_, err = kubeClient.CoreV1().Namespaces().Get(context.TODO(),conf.Namespace, metav1.GetOptions{})
		if err != nil {
			klog.Fatalf("No namespace with name %v found: %v", conf.Namespace, err)
		}
	}

	var isNetworkingIngressAvailable bool

	isNetworkingIngressAvailable, k8s.IsIngressV1Beta1Ready, _ = k8s.NetworkingIngressAvailable(kubeClient)
	if !isNetworkingIngressAvailable {
		klog.Fatalf("ingress-nginx requires Kubernetes v1.14.0 or higher")
	}

	if k8s.IsIngressV1Beta1Ready {
		klog.Info("Enabling new internal features available since Kubernetes v1.18")
		k8s.IngressClass, err = kubeClient.NetworkingV1beta1().IngressClasses().
			Get(context.TODO(), class.IngressClass, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				if !err2.IsUnauthorized(err) && !err2.IsForbidden(err) {
					klog.Fatalf("Error searching IngressClass: %v", err)
				}

				klog.Error(err, "Searching IngressClass", "class", class.IngressClass)
			}

			klog.Warningf("No IngressClass resource with name %v found. Only annotation will be used.", class.IngressClass)

			// TODO: remove once this is fixed in client-go
			k8s.IngressClass = nil
		}

		if k8s.IngressClass != nil && k8s.IngressClass.Spec.Controller != k8s.IngressNginxController {
			klog.Errorf(`Invalid IngressClass (Spec.Controller) value "%v". Should be "%v"`, k8s.IngressClass.Spec.Controller, k8s.IngressNginxController)
			klog.Fatalf("IngressClass with name %v is not valid for ingress-nginx (invalid Spec.Controller)", class.IngressClass)
		}
	}

	// ssl cert
	conf.FakeCertificate = ssl.GetFakeSSLCert()
	klog.Infof("SSL fake certificate created", "file", conf.FakeCertificate.PemFileName)

	conf.Client = kubeClient

	err = k8s.GetIngressPod(kubeClient)
	if err != nil {
		klog.Fatalf("Unexpected error obtaining ingress-nginx pod: %v", err)
	}

	// register profiler
	if conf.EnableProfiling {
		go registerProfiler()
	}

	return err,conf
}

// CreateRequiredDirectories verifies if the required directories to
// start the ingress controller exist and creates the missing ones.
func CreateRequiredDirectories() error {
	for _, directory := range directories {
		_, err := os.Stat(directory)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(directory, ReadWriteByUser)
				if err != nil {
					return err1.Wrapf(err, "creating directory '%v'", directory)
				}

				continue
			}

			return err1.Wrapf(err, "checking directory %v", directory)
		}
	}

	return nil
}

// checkService check service
func checkService(key string, kubeClient clientset.Interface ) error {
	ns, name, err := k8s.ParseNameNS(key)
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Services(ns).Get(context.TODO(), name , metav1.GetOptions{})
	if err != nil {
		if err2.IsUnauthorized(err) || err2.IsForbidden(err) {
			return fmt.Errorf("✖ the cluster seems to be running with a restrictive Authorization mode and the internal controller does not have the required permissions to operate normally")
		}

		if err2.IsNotFound(err) {
			return fmt.Errorf("No service with name %v found in namespace %v: %v", name, ns, err)
		}

		return fmt.Errorf("Unexpected error searching service with name %v in namespace %v: %v", name, ns, err)
	}

	return nil
}

