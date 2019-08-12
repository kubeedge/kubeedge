package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
)

// NewCRDClient is used to create a restClient for crd
func NewCRDClient(cfg *rest.Config) (*rest.RESTClient, error) {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(AddDeviceCrds)

	err := schemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	config := *cfg
	config.APIPath = "/apis"
	config.GroupVersion = &v1alpha1.SchemeGroupVersion
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		klog.Errorf("Failed to create REST Client due to error %v", err)
		return nil, err
	}

	return client, nil
}

func AddDeviceCrds(scheme *runtime.Scheme) error {
	// Add Device
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Device{}, &v1alpha1.DeviceList{})
	v1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)
	// Add DeviceModel
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.DeviceModel{}, &v1alpha1.DeviceModelList{})
	v1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)

	return nil
}
