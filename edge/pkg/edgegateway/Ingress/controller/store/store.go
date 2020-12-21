package store

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/eapache/channels"
	ingress "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/class"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/annotations/parser"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller"
	ngx_config "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller/template"
	defaults "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/net/ssl"
	err1 "github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/kubernetes/pkg/apis/networking"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sort"
	"sync"
	"time"
)


// IngressFilterFunc decides if an Ingress should be omitted or not
type IngressFilterFunc func(*ingress.Ingress) bool

// Storer is the interface that wraps the required methods to gather information
// about ingresses, services, secrets and ingress annotations.
type Storer interface {
	// GetBackendConfiguration returns the nginx configuration stored in a configmap
	GetBackendConfiguration() ngx_config.Configuration

	// GetConfigMap returns the ConfigMap matching key.
	GetConfigMap(key string) (*corev1.ConfigMap, error)

	// GetSecret returns the Secret matching key.
	GetSecret(key string) (*corev1.Secret, error)

	// GetService returns the Service matching key.
	GetService(key string) (*corev1.Service, error)

	// GetServiceEndpoints returns the Endpoints of a Service matching key.
	GetServiceEndpoints(key string) (*corev1.Endpoints, error)

	// ListIngresses returns a list of all Ingresses in the store.
	ListIngresses() []*ingress.Ingress

	// GetLocalSSLCert returns the local copy of a SSLCert
	GetLocalSSLCert(name string) (*ingress.SSLCert, error)

	// ListLocalSSLCerts returns the list of local SSLCerts
	ListLocalSSLCerts() []*ingress.SSLCert

	// GetAuthCertificate resolves a given secret name into an SSL certificate.
	// The secret must contain 3 keys named:
	//   ca.crt: contains the certificate chain used for authentication
	GetAuthCertificate(string) (*defaults.AuthSSLCert, error)

	// GetDefaultBackend returns the default backend configuration
	GetDefaultBackend() defaults.Backend

	// Run initiates the synchronization of the controllers
	Run(stopCh chan struct{})
}

// EventType type of event associated with an informer
type EventType string

const (
	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
	// ConfigurationEvent event associated when a controller configuration object is created or updated
	ConfigurationEvent EventType = "CONFIGURATION"
)

// Event holds the context of an event.
type Event struct {
	Type EventType
	Obj  interface{}
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Informer struct {
	Ingress   cache.SharedIndexInformer
	Endpoint  cache.SharedIndexInformer
	Service   cache.SharedIndexInformer
	Secret    cache.SharedIndexInformer
	ConfigMap cache.SharedIndexInformer
}

// Run initiates the synchronization of the informers against the API server.
func (i *Informer) Run(stopCh chan struct{}) {
	go i.Secret.Run(stopCh)
	go i.Endpoint.Run(stopCh)
	go i.Service.Run(stopCh)
	go i.ConfigMap.Run(stopCh)

	// wait for all involved caches to be synced before processing items
	// from the queue
	if !cache.WaitForCacheSync(stopCh,
		i.Endpoint.HasSynced,
		i.Service.HasSynced,
		i.Secret.HasSynced,
		i.ConfigMap.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	// in big clusters, deltas can keep arriving even after HasSynced
	// functions have returned 'true'
	time.Sleep(1 * time.Second)

	// we can start syncing ingress objects only after other caches are
	// ready, because ingress rules require content from other listers, and
	// 'add' events get triggered in the handlers during caches population.
	go i.Ingress.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh,
		i.Ingress.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}
}

// k8sStore internal Storer implementation using informers and thread safe stores
type k8sStore struct {
	// backendConfig contains the running configuration from the configmap
	// this is required because this rarely changes but is a very expensive
	// operation to execute in each OnUpdate invocation
	backendConfig ngx_config.Configuration

	// informer contains the cache Informers
	informers *Informer

	// listers contains the cache.Store interfaces used in the ingress controller
	listers *Lister

	// sslStore local store of SSL certificates (certificates used in ingress)
	// this is required because the certificates must be present in the
	// container filesystem
	sslStore *SSLCertTracker

	annotations annotations.Extractor

	// secretIngressMap contains information about which ingress references a
	// secret in the annotations.
	secretIngressMap ObjectRefMap

	// updateCh
	updateCh *channels.RingChannel

	// syncSecretMu protects against simultaneous invocations of syncSecret
	syncSecretMu *sync.Mutex

	// backendConfigMu protects against simultaneous read/write of backendConfig
	backendConfigMu *sync.RWMutex

	defaultSSLCertificate string
}

func (s *k8sStore) GetBackendConfiguration() ngx_config.Configuration {
	s.backendConfigMu.RLock()
	defer s.backendConfigMu.RUnlock()

	return s.backendConfig
}

// GetAuthCertificate is used by the auth-tls annotations to get a cert from a secret
func (s *k8sStore) GetAuthCertificate(name string) (*defaults.AuthSSLCert, error) {
	if _, err := s.GetLocalSSLCert(name); err != nil {
		s.syncSecret(name)
	}

	cert, err := s.GetLocalSSLCert(name)
	if err != nil {
		return nil, err
	}

	return &defaults.AuthSSLCert{
		Secret:      name,
		CAFileName:  cert.CAFileName,
		CASHA:       cert.CASHA,
		CRLFileName: cert.CRLFileName,
		CRLSHA:      cert.CRLSHA,
		PemFileName: cert.PemFileName,
	}, nil
}

// GetLocalSSLCert returns the local copy of a SSLCert
func (s *k8sStore) GetLocalSSLCert(key string) (*ingress.SSLCert, error) {
	return s.sslStore.ByKey(key)
}

// syncSecret synchronizes the content of a TLS Secret (certificate(s), secret
// key) with the filesystem. The resulting files can be used by NGINX.
func (s *k8sStore) syncSecret(key string) {
	s.syncSecretMu.Lock()
	defer s.syncSecretMu.Unlock()

	klog.V(3).InfoS("Syncing Secret", "name", key)

	cert, err := s.getPemCertificate(key)
	if err != nil {
		if !isErrSecretForAuth(err) {
			klog.Warningf("Error obtaining X.509 certificate: %v", err)
		}
		return
	}

	// create certificates and add or update the item in the store
	cur, err := s.GetLocalSSLCert(key)
	if err == nil {
		if cur.Equal(cert) {
			// no need to update
			return
		}
		klog.InfoS("Updating secret in local store", "name", key)
		s.sslStore.Update(key, cert)
		// this update must trigger an update
		// (like an update event from a change in Ingress)
		s.sendDummyEvent()
		return
	}

	klog.InfoS("Adding secret to local store", "name", key)
	s.sslStore.Add(key, cert)
	// this update must trigger an update
	// (like an update event from a change in Ingress)
	s.sendDummyEvent()
}

// sendDummyEvent sends a dummy event to trigger an update
// This is used in when a secret change
func (s *k8sStore) sendDummyEvent() {
	s.updateCh.In() <- Event{
		Type: UpdateEvent,
		Obj: &networkingv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dummy",
				Namespace: "dummy",
			},
		},
	}
}

// ErrSecretForAuth error to indicate a secret is used for authentication
var ErrSecretForAuth = fmt.Errorf("secret is used for authentication")

func isErrSecretForAuth(e error) bool {
	return e == ErrSecretForAuth
}

// getPemCertificate receives a secret, and creates a ingress.SSLCert as return.
// It parses the secret and verifies if it's a keypair, or a 'ca.crt' secret only.
func (s *k8sStore) getPemCertificate(secretName string) (*ingress.SSLCert, error) {
	secret, err := s.listers.Secret.ByKey(secretName)
	if err != nil {
		return nil, err
	}

	cert, okcert := secret.Data[corev1.TLSCertKey]
	key, okkey := secret.Data[corev1.TLSPrivateKeyKey]
	ca := secret.Data["ca.crt"]

	crl := secret.Data["ca.crl"]

	auth := secret.Data["auth"]

	// namespace/secretName -> namespace-secretName
	nsSecName := strings.Replace(secretName, "/", "-", -1)

	var sslCert *ingress.SSLCert
	if okcert && okkey {
		if cert == nil {
			return nil, fmt.Errorf("key 'tls.crt' missing from Secret %q", secretName)
		}

		if key == nil {
			return nil, fmt.Errorf("key 'tls.key' missing from Secret %q", secretName)
		}

		sslCert, err = ssl.CreateSSLCert(cert, key, string(secret.UID))
		if err != nil {
			return nil, fmt.Errorf("unexpected error creating SSL Cert: %v", err)
		}

		if len(ca) > 0 {
			caCert, err := ssl.CheckCACert(ca)
			if err != nil {
				return nil, fmt.Errorf("parsing CA certificate: %v", err)
			}

			path, err := ssl.StoreSSLCertOnDisk(nsSecName, sslCert)
			if err != nil {
				return nil, fmt.Errorf("error while storing certificate and key: %v", err)
			}

			sslCert.PemFileName = path
			sslCert.CACertificate = caCert
			sslCert.CAFileName = path
			sslCert.CASHA = ssl.SHA1(path)

			err = ssl.ConfigureCACertWithCertAndKey(nsSecName, ca, sslCert)
			if err != nil {
				return nil, fmt.Errorf("error configuring CA certificate: %v", err)
			}

			if len(crl) > 0 {
				err = ssl.ConfigureCRL(nsSecName, crl, sslCert)
				if err != nil {
					return nil, fmt.Errorf("error configuring CRL certificate: %v", err)
				}
			}
		}

		msg := fmt.Sprintf("Configuring Secret %q for TLS encryption (CN: %v)", secretName, sslCert.CN)
		if ca != nil {
			msg += " and authentication"
		}

		if crl != nil {
			msg += " and CRL"
		}

		klog.V(3).InfoS(msg)
	} else if len(ca) > 0 {
		sslCert, err = ssl.CreateCACert(ca)
		if err != nil {
			return nil, fmt.Errorf("unexpected error creating SSL Cert: %v", err)
		}

		err = ssl.ConfigureCACert(nsSecName, ca, sslCert)
		if err != nil {
			return nil, fmt.Errorf("error configuring CA certificate: %v", err)
		}

		if len(crl) > 0 {
			err = ssl.ConfigureCRL(nsSecName, crl, sslCert)
			if err != nil {
				return nil, err
			}
		}
		// makes this secret in 'syncSecret' to be used for Certificate Authentication
		// this does not enable Certificate Authentication
		klog.V(3).InfoS("Configuring Secret for TLS authentication", "secret", secretName)
	} else {
		if auth != nil {
			return nil, ErrSecretForAuth
		}

		return nil, fmt.Errorf("secret %q contains no keypair or CA certificate", secretName)
	}

	sslCert.Name = secret.Name
	sslCert.Namespace = secret.Namespace

	// the default SSL certificate needs to be present on disk
	if secretName == s.defaultSSLCertificate {
		path, err := ssl.StoreSSLCertOnDisk(nsSecName, sslCert)
		if err != nil {
			return nil, err1.Wrap(err, "storing default SSL Certificate")
		}

		sslCert.PemFileName = path
	}

	return sslCert, nil
}

// New creates a new object store to be used in the ingress controller
func New(
	namespace, configmap, tcp, udp, defaultSSLCertificate string,
	resyncPeriod time.Duration,
	client clientset.Interface,
	updateCh *channels.RingChannel,
	disableCatchAll bool) Storer {

	store := &k8sStore{
		informers:             &Informer{},
		listers:               &Lister{},
		sslStore:              NewSSLCertTracker(),
		updateCh:              updateCh,
		backendConfig:         ngx_config.NewDefault(),
		syncSecretMu:          &sync.Mutex{},
		backendConfigMu:       &sync.RWMutex{},
		secretIngressMap:      NewObjectRefMap(),
		defaultSSLCertificate: defaultSSLCertificate,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&clientcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(namespace),
	})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{
		Component: "nginx-ingress-controller",
	})

	// k8sStore fulfills resolver.Resolver interface
	store.annotations = annotations.NewAnnotationExtractor(store)

	store.listers.IngressWithAnnotation.Store = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	// As we currently do not filter out kubernetes objects we list, we can
	// retrieve a huge amount of data from the API server.
	// In a cluster using HELM < v3 configmaps are used to store binary data.
	// If you happen to have a lot of HELM releases in the cluster it will make
	// the memory consumption of nginx-ingress-controller explode.
	// In order to avoid that we filter out labels OWNER=TILLER.
	labelsTweakListOptionsFunc := func(options *metav1.ListOptions) {
		if len(options.LabelSelector) > 0 {
			options.LabelSelector += ",OWNER!=TILLER"
		} else {
			options.LabelSelector = "OWNER!=TILLER"
		}
	}

	// As of HELM >= v3 helm releases are stored using Secrets instead of ConfigMaps.
	// In order to avoid listing those secrets we discard type "helm.sh/release.v1"
	secretsTweakListOptionsFunc := func(options *metav1.ListOptions) {
		helmAntiSelector := fields.OneTermNotEqualSelector("type", "helm.sh/release.v1")
		baseSelector, err := fields.ParseSelector(options.FieldSelector)

		if err != nil {
			options.FieldSelector = helmAntiSelector.String()
		} else {
			options.FieldSelector = fields.AndSelectors(baseSelector, helmAntiSelector).String()
		}
	}

	// create informers factory, enable and assign required informers
	infFactory := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod,
		informers.WithNamespace(namespace),
	)

	// create informers factory for configmaps
	infFactoryConfigmaps := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(labelsTweakListOptionsFunc),
	)

	// create informers factory for secrets
	infFactorySecrets := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(secretsTweakListOptionsFunc),
	)

	store.informers.Ingress = infFactory.Networking().V1beta1().Ingresses().Informer()
	store.listers.Ingress.Store = store.informers.Ingress.GetStore()

	store.informers.Endpoint = infFactory.Core().V1().Endpoints().Informer()
	store.listers.Endpoint.Store = store.informers.Endpoint.GetStore()

	store.informers.Secret = infFactorySecrets.Core().V1().Secrets().Informer()
	store.listers.Secret.Store = store.informers.Secret.GetStore()

	store.informers.ConfigMap = infFactoryConfigmaps.Core().V1().ConfigMaps().Informer()
	store.listers.ConfigMap.Store = store.informers.ConfigMap.GetStore()

	store.informers.Service = infFactory.Core().V1().Services().Informer()
	store.listers.Service.Store = store.informers.Service.GetStore()

	ingDeleteHandler := func(obj interface{}) {
		ing, ok := toIngress(obj)
		if !ok {
			// If we reached here it means the ingress was deleted but its final state is unrecorded.
			tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
			if !ok {
				klog.ErrorS(nil, "Error obtaining object from tombstone", "key", obj)
				return
			}
			ing, ok = tombstone.Obj.(*networkingv1beta1.Ingress)
			if !ok {
				klog.Errorf("Tombstone contained object that is not an Ingress: %#v", obj)
				return
			}
		}

		if !class.IsValid(ing) {
			return
		}

		if isCatchAllIngress(ing.Spec) && disableCatchAll {
			klog.InfoS("Ignoring delete for catch-all because of --disable-catch-all", "ingress", klog.KObj(ing))
			return
		}

		store.listers.IngressWithAnnotation.Delete(ing)

		key := defaults.MetaNamespaceKey(ing)
		store.secretIngressMap.Delete(key)

		updateCh.In() <- Event{
			Type: DeleteEvent,
			Obj:  obj,
		}
	}

	ingEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ing, _ := toIngress(obj)
			if !class.IsValid(ing) {
				ingressClass, _ := parser.GetStringAnnotation(class.IngressKey, ing)
				klog.InfoS("Ignoring ingress", "ingress", klog.KObj(ing), "kubernetes.io/ingress.class", ingressClass, "ingressClassName", StringPtrDerefOr(ing.Spec.IngressClassName, ""))
				return
			}

			if isCatchAllIngress(ing.Spec) && disableCatchAll {
				klog.InfoS("Ignoring add for catch-all ingress because of --disable-catch-all", "ingress", klog.KObj(ing))
				return
			}

			recorder.Eventf(ing, corev1.EventTypeNormal, "Sync", "Scheduled for sync")

			store.syncIngress(ing)
			store.updateSecretIngressMap(ing)
			store.syncSecrets(ing)

			updateCh.In() <- Event{
				Type: CreateEvent,
				Obj:  obj,
			}
		},
		DeleteFunc: ingDeleteHandler,
		UpdateFunc: func(old, cur interface{}) {
			oldIng, _ := toIngress(old)
			curIng, _ := toIngress(cur)

			validOld := class.IsValid(oldIng)
			validCur := class.IsValid(curIng)
			if !validOld && validCur {
				if isCatchAllIngress(curIng.Spec) && disableCatchAll {
					klog.InfoS("ignoring update for catch-all ingress because of --disable-catch-all", "ingress", klog.KObj(curIng))
					return
				}

				klog.InfoS("creating ingress", "ingress", klog.KObj(curIng), "class", class.IngressKey)
				recorder.Eventf(curIng, corev1.EventTypeNormal, "Sync", "Scheduled for sync")
			} else if validOld && !validCur {
				klog.InfoS("removing ingress", "ingress", klog.KObj(curIng), "class", class.IngressKey)
				ingDeleteHandler(old)
				return
			} else if validCur && !reflect.DeepEqual(old, cur) {
				if isCatchAllIngress(curIng.Spec) && disableCatchAll {
					klog.InfoS("ignoring update for catch-all ingress and delete old one because of --disable-catch-all", "ingress", klog.KObj(curIng))
					ingDeleteHandler(old)
					return
				}

				recorder.Eventf(curIng, corev1.EventTypeNormal, "Sync", "Scheduled for sync")
			} else {
				klog.V(3).InfoS("No changes on ingress. Skipping update", "ingress", klog.KObj(curIng))
				return
			}

			store.syncIngress(curIng)
			store.updateSecretIngressMap(curIng)
			store.syncSecrets(curIng)

			updateCh.In() <- Event{
				Type: UpdateEvent,
				Obj:  cur,
			}
		},
	}

	secrEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sec := obj.(*corev1.Secret)
			key := defaults.MetaNamespaceKey(sec)

			if store.defaultSSLCertificate == key {
				store.syncSecret(store.defaultSSLCertificate)
			}

			// find references in ingresses and update local ssl certs
			if ings := store.secretIngressMap.Reference(key); len(ings) > 0 {
				klog.InfoS("Secret was added and it is used in ingress annotations. Parsing", "secret", key)
				for _, ingKey := range ings {
					ing, err := store.getIngress(ingKey)
					if err != nil {
						klog.Errorf("could not find Ingress %v in local store", ingKey)
						continue
					}
					store.syncIngress(ing)
					store.syncSecrets(ing)
				}
				updateCh.In() <- Event{
					Type: CreateEvent,
					Obj:  obj,
				}
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				sec := cur.(*corev1.Secret)
				key := defaults.MetaNamespaceKey(sec)

				if store.defaultSSLCertificate == key {
					store.syncSecret(store.defaultSSLCertificate)
				}

				// find references in ingresses and update local ssl certs
				if ings := store.secretIngressMap.Reference(key); len(ings) > 0 {
					klog.InfoS("secret was updated and it is used in ingress annotations. Parsing", "secret", key)
					for _, ingKey := range ings {
						ing, err := store.getIngress(ingKey)
						if err != nil {
							klog.ErrorS(err, "could not find Ingress in local store", "ingress", ingKey)
							continue
						}
						store.syncIngress(ing)
						store.syncSecrets(ing)
					}
					updateCh.In() <- Event{
						Type: UpdateEvent,
						Obj:  cur,
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			sec, ok := obj.(*corev1.Secret)
			if !ok {
				// If we reached here it means the secret was deleted but its final state is unrecorded.
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}

				sec, ok = tombstone.Obj.(*corev1.Secret)
				if !ok {
					return
				}
			}

			store.sslStore.Delete(defaults.MetaNamespaceKey(sec))

			key := defaults.MetaNamespaceKey(sec)

			// find references in ingresses
			if ings := store.secretIngressMap.Reference(key); len(ings) > 0 {
				klog.InfoS("secret was deleted and it is used in ingress annotations. Parsing", "secret", key)
				for _, ingKey := range ings {
					ing, err := store.getIngress(ingKey)
					if err != nil {
						klog.Errorf("could not find Ingress %v in local store", ingKey)
						continue
					}
					store.syncIngress(ing)
				}

				updateCh.In() <- Event{
					Type: DeleteEvent,
					Obj:  obj,
				}
			}
		},
	}

	epEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			updateCh.In() <- Event{
				Type: CreateEvent,
				Obj:  obj,
			}
		},
		DeleteFunc: func(obj interface{}) {
			updateCh.In() <- Event{
				Type: DeleteEvent,
				Obj:  obj,
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			oep := old.(*corev1.Endpoints)
			cep := cur.(*corev1.Endpoints)
			if !reflect.DeepEqual(cep.Subsets, oep.Subsets) {
				updateCh.In() <- Event{
					Type: UpdateEvent,
					Obj:  cur,
				}
			}
		},
	}

	// TODO: add e2e test to verify that changes to one or more configmap trigger an update
	changeTriggerUpdate := func(name string) bool {
		return name == configmap || name == tcp || name == udp
	}

	handleCfgMapEvent := func(key string, cfgMap *corev1.ConfigMap, eventName string) {
		// updates to configuration configmaps can trigger an update
		triggerUpdate := false
		if changeTriggerUpdate(key) {
			triggerUpdate = true
			recorder.Eventf(cfgMap, corev1.EventTypeNormal, eventName, fmt.Sprintf("ConfigMap %v", key))
			if key == configmap {
				store.setConfig(cfgMap)
			}
		}

		ings := store.listers.IngressWithAnnotation.List()
		for _, ingKey := range ings {
			key := defaults.MetaNamespaceKey(ingKey)
			ing, err := store.getIngress(key)
			if err != nil {
				klog.Errorf("could not find Ingress %v in local store: %v", key, err)
				continue
			}

			if parser.AnnotationsReferencesConfigmap(ing) {
				store.syncIngress(ing)
				continue
			}

			if triggerUpdate {
				store.syncIngress(ing)
			}
		}

		if triggerUpdate {
			updateCh.In() <- Event{
				Type: ConfigurationEvent,
				Obj:  cfgMap,
			}
		}
	}

	cmEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cfgMap := obj.(*corev1.ConfigMap)
			key := defaults.MetaNamespaceKey(cfgMap)
			handleCfgMapEvent(key, cfgMap, "CREATE")
		},
		UpdateFunc: func(old, cur interface{}) {
			if reflect.DeepEqual(old, cur) {
				return
			}

			cfgMap := cur.(*corev1.ConfigMap)
			key := defaults.MetaNamespaceKey(cfgMap)
			handleCfgMapEvent(key, cfgMap, "UPDATE")
		},
	}

	serviceHandler := cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, cur interface{}) {
			oldSvc := old.(*corev1.Service)
			curSvc := cur.(*corev1.Service)

			if reflect.DeepEqual(oldSvc, curSvc) {
				return
			}

			updateCh.In() <- Event{
				Type: UpdateEvent,
				Obj:  cur,
			}
		},
	}

	store.informers.Ingress.AddEventHandler(ingEventHandler)
	store.informers.Endpoint.AddEventHandler(epEventHandler)
	store.informers.Secret.AddEventHandler(secrEventHandler)
	store.informers.ConfigMap.AddEventHandler(cmEventHandler)
	store.informers.Service.AddEventHandler(serviceHandler)

	// do not wait for informers to read the configmap configuration
	ns, name, _ := controller.ParseNameNS(configmap)
	cm, err := client.CoreV1().ConfigMaps(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Unexpected error reading configuration configmap: %v", err)
	}

	store.setConfig(cm)
	return store
}

// syncIngress parses ingress annotations converting the value of the
// annotation to a go struct
func (s *k8sStore) syncIngress(ing *networkingv1beta1.Ingress) {
	key := defaults.MetaNamespaceKey(ing)
	klog.V(3).Infof("updating annotations information for ingress %v", key)

	copyIng := &networkingv1beta1.Ingress{}
	ing.ObjectMeta.DeepCopyInto(&copyIng.ObjectMeta)
	ing.Spec.DeepCopyInto(&copyIng.Spec)
	ing.Status.DeepCopyInto(&copyIng.Status)

	for ri, rule := range copyIng.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for pi, path := range rule.HTTP.Paths {
			if path.Path == "" {
				copyIng.Spec.Rules[ri].HTTP.Paths[pi].Path = "/"
			}
		}
	}

	SetDefaultNGINXPathType(copyIng)

	err := s.listers.IngressWithAnnotation.Update(&ingress.Ingress{
		Ingress:           *copyIng,
		ParsedAnnotations: s.annotations.Extract(ing),
	})
	if err != nil {
		klog.Error(err)
	}
}

// updateSecretIngressMap takes an Ingress and updates all Secret objects it
// references in secretIngressMap.
func (s *k8sStore) updateSecretIngressMap(ing *networkingv1beta1.Ingress) {
	key := defaults.MetaNamespaceKey(ing)
	klog.V(3).Infof("updating references to secrets for ingress %v", key)

	// delete all existing references first
	s.secretIngressMap.Delete(key)

	var refSecrets []string

	for _, tls := range ing.Spec.TLS {
		secrName := tls.SecretName
		if secrName != "" {
			secrKey := fmt.Sprintf("%v/%v", ing.Namespace, secrName)
			refSecrets = append(refSecrets, secrKey)
		}
	}

	// We can not rely on cached ingress annotations because these are
	// discarded when the referenced secret does not exist in the local
	// store. As a result, adding a secret *after* the ingress(es) which
	// references it would not trigger a resync of that secret.
	secretAnnotations := []string{
		"auth-secret",
		"auth-tls-secret",
		"proxy-ssl-secret",
		"secure-verify-ca-secret",
	}
	for _, ann := range secretAnnotations {
		secrKey, err := objectRefAnnotationNsKey(ann, ing)
		if err != nil && !defaults.IsMissingAnnotations(err) {
			klog.Errorf("error reading secret reference in annotation %q: %s", ann, err)
			continue
		}
		if secrKey != "" {
			refSecrets = append(refSecrets, secrKey)
		}
	}

	// populate map with all secret references
	s.secretIngressMap.Insert(key, refSecrets...)
}

// syncSecrets synchronizes data from all Secrets referenced by the given
// Ingress with the local store and file system.
func (s *k8sStore) syncSecrets(ing *networkingv1beta1.Ingress) {
	key := defaults.MetaNamespaceKey(ing)
	for _, secrKey := range s.secretIngressMap.ReferencedBy(key) {
		s.syncSecret(secrKey)
	}
}

// objectRefAnnotationNsKey returns an object reference formatted as a
// 'namespace/name' key from the given annotation name.
func objectRefAnnotationNsKey(ann string, ing *networkingv1beta1.Ingress) (string, error) {
	annValue, err := parser.GetStringAnnotation(ann, ing)
	if err != nil {
		return "", err
	}

	secrNs, secrName, err := cache.SplitMetaNamespaceKey(annValue)
	if secrName == "" {
		return "", err
	}

	if secrNs == "" {
		return fmt.Sprintf("%v/%v", ing.Namespace, secrName), nil
	}
	return annValue, nil
}

// Lister contains object listers (stores).
type Lister struct {
	Ingress               IngressLister
	Service               ServiceLister
	Endpoint              EndpointLister
	Secret                SecretLister
	ConfigMap             ConfigMapLister
	IngressWithAnnotation IngressWithAnnotationsLister
}

// IngressLister makes a Store that lists Ingress.
type IngressLister struct {
	cache.Store
}

// ByKey returns the Ingress matching key in the local Ingress Store.
func (il IngressLister) ByKey(key string) (*networking.Ingress, error) {
	i, exists, err := il.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return i.(*networking.Ingress), nil
}

// FilterIngresses returns the list of Ingresses
func FilterIngresses(ingresses []*ingress.Ingress, filterFunc IngressFilterFunc) []*ingress.Ingress {
	afterFilter := make([]*ingress.Ingress, 0)
	for _, ingress := range ingresses {
		if !filterFunc(ingress) {
			afterFilter = append(afterFilter, ingress)
		}
	}

	sortIngressSlice(afterFilter)
	return afterFilter
}

// ServiceLister makes a Store that lists Services.
type ServiceLister struct {
	cache.Store
}

// ByKey returns the Service matching key in the local Service Store.
func (sl *ServiceLister) ByKey(key string) (*corev1.Service, error) {
	s, exists, err := sl.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return s.(*corev1.Service), nil
}

// EndpointLister makes a Store that lists Endpoints.
type EndpointLister struct {
	cache.Store
}

// ByKey returns the Endpoints of the Service matching key in the local Endpoint Store.
func (s *EndpointLister) ByKey(key string) (*corev1.Endpoints, error) {
	eps, exists, err := s.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return eps.(*corev1.Endpoints), nil
}

// SecretLister makes a Store that lists Secrets.
type SecretLister struct {
	cache.Store
}

// ByKey returns the Secret matching key in the local Secret Store.
func (sl *SecretLister) ByKey(key string) (*corev1.Secret, error) {
	s, exists, err := sl.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return s.(*corev1.Secret), nil
}

// ConfigMapLister makes a Store that lists Configmaps.
type ConfigMapLister struct {
	cache.Store
}

// ByKey returns the ConfigMap matching key in the local ConfigMap Store.
func (cml *ConfigMapLister) ByKey(key string) (*corev1.ConfigMap, error) {
	s, exists, err := cml.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return s.(*corev1.ConfigMap), nil
}

// IngressWithAnnotationsLister makes a Store that lists Ingress rules with annotations already parsed
type IngressWithAnnotationsLister struct {
	cache.Store
}

// ByKey returns the Ingress with annotations matching key in the local store or an error
func (il IngressWithAnnotationsLister) ByKey(key string) (*ingress.Ingress, error) {
	i, exists, err := il.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NotExistsError(key)
	}
	return i.(*ingress.Ingress), nil
}

// SSLCertTracker holds a store of referenced Secrets in Ingress rules
type SSLCertTracker struct {
	cache.ThreadSafeStore
}

// NewSSLCertTracker creates a new SSLCertTracker store
func NewSSLCertTracker() *SSLCertTracker {
	return &SSLCertTracker{
		cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{}),
	}
}

// ByKey searches for an ingress in the local ingress Store
func (s SSLCertTracker) ByKey(key string) (*ingress.SSLCert, error) {
	cert, exists := s.Get(key)
	if !exists {
		return nil, fmt.Errorf("local SSL certificate %v was not found", key)
	}
	return cert.(*ingress.SSLCert), nil
}

// ObjectRefMap is a map of references from object(s) to object (1:n). It is
// used to keep track of which data objects (Secrets) are used within Ingress
// objects.
type ObjectRefMap interface {
	Insert(consumer string, ref ...string)
	Delete(consumer string)
	Len() int
	Has(ref string) bool
	HasConsumer(consumer string) bool
	Reference(ref string) []string
	ReferencedBy(consumer string) []string
}

type objectRefMap struct {
	sync.Mutex
	v map[string]sets.String
}

// NewObjectRefMap returns a new ObjectRefMap.
func NewObjectRefMap() ObjectRefMap {
	return &objectRefMap{
		v: make(map[string]sets.String),
	}
}

// Insert adds a consumer to one or more referenced objects.
func (o *objectRefMap) Insert(consumer string, ref ...string) {
	o.Lock()
	defer o.Unlock()

	for _, r := range ref {
		if _, ok := o.v[r]; !ok {
			o.v[r] = sets.NewString(consumer)
			continue
		}
		o.v[r].Insert(consumer)
	}
}

// Delete deletes a consumer from all referenced objects.
func (o *objectRefMap) Delete(consumer string) {
	o.Lock()
	defer o.Unlock()

	for ref, consumers := range o.v {
		consumers.Delete(consumer)
		if consumers.Len() == 0 {
			delete(o.v, ref)
		}
	}
}

// Len returns the count of referenced objects.
func (o *objectRefMap) Len() int {
	return len(o.v)
}

// Has returns whether the given object is referenced by any other object.
func (o *objectRefMap) Has(ref string) bool {
	o.Lock()
	defer o.Unlock()

	if _, ok := o.v[ref]; ok {
		return true
	}
	return false
}

// HasConsumer returns whether the store contains the given consumer.
func (o *objectRefMap) HasConsumer(consumer string) bool {
	o.Lock()
	defer o.Unlock()

	for _, consumers := range o.v {
		if consumers.Has(consumer) {
			return true
		}
	}
	return false
}

// Reference returns all objects referencing the given object.
func (o *objectRefMap) Reference(ref string) []string {
	o.Lock()
	defer o.Unlock()

	consumers, ok := o.v[ref]
	if !ok {
		return make([]string, 0)
	}
	return consumers.List()
}

// ReferencedBy returns all objects referenced by the given object.
func (o *objectRefMap) ReferencedBy(consumer string) []string {
	o.Lock()
	defer o.Unlock()

	refs := make([]string, 0)
	for ref, consumers := range o.v {
		if consumers.Has(consumer) {
			refs = append(refs, ref)
		}
	}
	return refs
}

func sortIngressSlice(ingresses []*ingress.Ingress) {
	// sort Ingresses using the CreationTimestamp field
	sort.SliceStable(ingresses, func(i, j int) bool {
		ir := ingresses[i].CreationTimestamp
		jr := ingresses[j].CreationTimestamp
		if ir.Equal(&jr) {
			in := fmt.Sprintf("%v/%v", ingresses[i].Namespace, ingresses[i].Name)
			jn := fmt.Sprintf("%v/%v", ingresses[j].Namespace, ingresses[j].Name)
			klog.V(3).Infof("Ingress %v and %v have identical CreationTimestamp", in, jn)
			return in > jn
		}
		return ir.Before(&jr)
	})
}

// NotExistsError is returned when an object does not exist in a local store.
type NotExistsError string

// Error implements the error interface.
func (e NotExistsError) Error() string {
	return fmt.Sprintf("no object matching key %q in local store", string(e))
}


func toIngress(obj interface{}) (*networkingv1beta1.Ingress, bool) {
	if ing, ok := obj.(*networkingv1beta1.Ingress); ok {
		SetDefaultNGINXPathType(ing)
		return ing, true
	}

	return nil, false
}

// default path type is Prefix to not break existing definitions
var defaultPathType = networkingv1beta1.PathTypePrefix

// SetDefaultNGINXPathType sets a default PathType when is not defined.
func SetDefaultNGINXPathType(ing *networkingv1beta1.Ingress) {
	for _, rule := range ing.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for idx := range rule.IngressRuleValue.HTTP.Paths {
			p := &rule.IngressRuleValue.HTTP.Paths[idx]
			if p.PathType == nil {
				p.PathType = &defaultPathType
			}

			if *p.PathType == networkingv1beta1.PathTypeImplementationSpecific {
				p.PathType = &defaultPathType
			}
		}
	}
}

// isCatchAllIngress returns whether or not an ingress produces a
// catch-all server, and so should be ignored when --disable-catch-all is set
func isCatchAllIngress(spec networkingv1beta1.IngressSpec) bool {
	return spec.Backend != nil && len(spec.Rules) == 0
}

// getIngress returns the Ingress matching key.
func (s *k8sStore) getIngress(key string) (*networkingv1beta1.Ingress, error) {
	ing, err := s.listers.IngressWithAnnotation.ByKey(key)
	if err != nil {
		return nil, err
	}

	return &ing.Ingress, nil
}

func (s *k8sStore) setConfig(cmap *corev1.ConfigMap) {
	s.backendConfigMu.Lock()
	defer s.backendConfigMu.Unlock()

	if cmap == nil {
		return
	}

	s.backendConfig = template.ReadConfig(cmap.Data)
	if s.backendConfig.UseGeoIP2 && !controller.GeoLite2DBExists() {
		klog.Warning("The GeoIP2 feature is enabled but the databases are missing. Disabling")
		s.backendConfig.UseGeoIP2 = false
	}

	s.writeSSLSessionTicketKey(cmap, "/etc/nginx/tickets.key")
}

func (s *k8sStore) writeSSLSessionTicketKey(cmap *corev1.ConfigMap, fileName string) {
	ticketString := template.ReadConfig(cmap.Data).SSLSessionTicketKey
	s.backendConfig.SSLSessionTicketKey = ""

	if ticketString != "" {
		ticketBytes := base64.StdEncoding.WithPadding(base64.StdPadding).DecodedLen(len(ticketString))

		// 81 used instead of 80 because of padding
		if !(ticketBytes == 48 || ticketBytes == 81) {
			klog.Warningf("ssl-session-ticket-key must contain either 48 or 80 bytes")
		}

		decodedTicket, err := base64.StdEncoding.DecodeString(ticketString)
		if err != nil {
			klog.Errorf("unexpected error decoding ssl-session-ticket-key: %v", err)
			return
		}

		err = ioutil.WriteFile(fileName, decodedTicket, controller.ReadWriteByUser)
		if err != nil {
			klog.Errorf("unexpected error writing ssl-session-ticket-key to %s: %v", fileName, err)
			return
		}

		s.backendConfig.SSLSessionTicketKey = ticketString
	}
}

// StringPtrDerefOr dereference the string ptr and returns it if not nil,
// else returns def.
func StringPtrDerefOr(ptr *string, def string) string {
	if ptr != nil {
		return *ptr
	}
	return def
}

// GetSecret returns the Secret matching key.
func (s *k8sStore) GetSecret(key string) (*corev1.Secret, error) {
	return s.listers.Secret.ByKey(key)
}

// GetService returns the Service matching key.
func (s *k8sStore) GetService(key string) (*corev1.Service, error) {
	return s.listers.Service.ByKey(key)
}

// GetConfigMap returns the ConfigMap matching key.
func (s *k8sStore) GetConfigMap(key string) (*corev1.ConfigMap, error) {
	return s.listers.ConfigMap.ByKey(key)
}

// GetDefaultBackend returns the default backend
func (s *k8sStore) GetDefaultBackend() defaults.Backend {
	return s.GetBackendConfiguration().Backend
}

// GetServiceEndpoints returns the Endpoints of a Service matching key.
func (s *k8sStore) GetServiceEndpoints(key string) (*corev1.Endpoints, error) {
	return s.listers.Endpoint.ByKey(key)
}

// ListIngresses returns the list of Ingresses
func (s *k8sStore) ListIngresses() []*ingress.Ingress {
	// filter ingress rules
	ingresses := make([]*ingress.Ingress, 0)
	for _, item := range s.listers.IngressWithAnnotation.List() {
		ing := item.(*ingress.Ingress)
		ingresses = append(ingresses, ing)
	}

	sortIngressSlice(ingresses)

	return ingresses
}

// ListLocalSSLCerts returns the list of local SSLCerts
func (s *k8sStore) ListLocalSSLCerts() []*ingress.SSLCert {
	var certs []*ingress.SSLCert
	for _, item := range s.sslStore.List() {
		if s, ok := item.(*ingress.SSLCert); ok {
			certs = append(certs, s)
		}
	}

	return certs
}

// Run initiates the synchronization of the informers and the initial
// synchronization of the secrets.
func (s *k8sStore) Run(stopCh chan struct{}) {
	// start informers
	s.informers.Run(stopCh)
}
