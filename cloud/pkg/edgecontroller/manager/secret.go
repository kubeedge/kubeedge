package manager

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// SecretManager manage all events of secret by SharedInformer
type SecretManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch secret change
func (sm *SecretManager) Events() chan watch.Event {
	return sm.events
}

// NewSecretManager create SecretManager by kube clientset and namespace
func NewSecretManager(kubeClient *kubernetes.Clientset, namespace string) (*SecretManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "secrets", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.SecretEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Secret{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &SecretManager{events: events}, nil
}
