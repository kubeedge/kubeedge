package metaserver

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
)

var (
	gateway = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1beta1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": "gateways.networking.istio.io",
			},
			"spec": map[string]interface{}{
				"group": "networking.istio.io",
				"names": map[string]string{
					"kind":     "Gateway",
					"plural":   "gateways",
					"singular": "gateway",
				},
				"scope":   "Namespaced",
				"version": "v1alpha3",
			},
		},
	}

	serviceentry = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1beta1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name": "serviceentries.networking.istio.io",
			},
			"spec": map[string]interface{}{
				"group": "networking.istio.io",
				"names": map[string]string{
					"kind":     "ServiceEntry",
					"plural":   "serviceentries",
					"singular": "serviceentry",
				},
				"scope":   "Namespaced",
				"version": "v1alpha3",
			},
		},
	}

	se = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "ServiceEntry",
			"metadata": map[string]interface{}{
				"name":      "test-serviceentry",
				"namespace": "default",
			},
		},
	}

	gw = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "test-gateway",
				"namespace": "default",
			},
		},
	}
)

func AddCRD() error {
	err := imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), serviceentry)
	if err != nil {
		return err
	}
	err = imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), gateway)
	if err != nil {
		return err
	}
	// wait for UpdateCrdMap() to detect crd
	time.Sleep(time.Minute)
	err = imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), gw)
	if err != nil {
		return err
	}
	err = imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), se)
	if err != nil {
		return err
	}
	return nil
}
