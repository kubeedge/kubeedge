package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	proxyproto "github.com/armon/go-proxyproto"
	"github.com/eapache/channels"
	fakekube "github.com/kubeedge/kubeedge/edge/pkg/edged/fake"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress"
	ngx_config "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller/store"
	ngx_template "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/controller/template"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/default"
	ing_net "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/net"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/net/dns"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/net/ssl"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/Ingress/task"
	adm_controller "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/admission/controller"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/mitchellh/go-ps"
	"github.com/ncabatoff/process-exporter/proc"
	err1 "github.com/pkg/errors"
	"io/ioutil"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	err2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// AuthDirectory default directory used to store files
	// to authenticate request
	AuthDirectory = "/etc/ingress-controller/auth"

	// DefaultSSLDirectory defines the location where the SSL certificates will be generated
	// This directory contains all the SSL certificates that are specified in Ingress rules.
	// The name of each file is <namespace>-<secret name>.pem. The content is the concatenated
	// certificate and key.
	DefaultSSLDirectory = "/etc/ingress-controller/ssl"

	// ReadWriteByUser defines linux permission to read and write files for the owner user
	ReadWriteByUser = 0700

	// DefaultAnnotationsPrefix defines the common prefix used in the nginx ingress controller
	DefaultAnnotationsPrefix = "nginx.ingress.kubernetes.io"

	// IngressNginxController defines the valid value of IngressClass
	// Controller field for ingress-nginx
	IngressNginxController = "k8s.io/ingress-nginx"

	// tempNginxPattern defines nginx template config
	tempNginxPattern = "nginx-cfg"

	geoIPPath   = "/etc/nginx/geoip"
	dbExtension = ".mmdb"
)

var (
	directories = []string{
		DefaultSSLDirectory,
		AuthDirectory,
	}
	// MaxmindLicenseKey maxmind license key to download databases
	MaxmindLicenseKey = ""

	// MaxmindEditionIDs maxmind editions (GeoLite2-City, GeoLite2-Country, GeoIP2-ISP, etc)
	MaxmindEditionIDs = ""

	// UpdateInterval defines the time interval, in seconds, in
	// which the status should check if an update is required.
	UpdateInterval = 60

	// DefaultClass defines the default class used in the nginx ingress controller
	DefaultClass = "nginx"

	// IngressClass sets the runtime ingress class to use
	// An empty string means accept all ingresses without
	// annotation and the ones configured with class nginx
	IngressClass = "nginx"

	// AnnotationsPrefix is the mutable attribute that the controller explicitly refers to
	AnnotationsPrefix = DefaultAnnotationsPrefix

	// IsIngressV1Beta1Ready indicates if the running Kubernetes version is at least v1.18.0
	IsIngressV1Beta1Ready bool

	// IngressPodDetails hold information about the ingress-nginx pod
	IngressPodDetails *PodInfo

	selectorLabelKeys = []string{
		"app.kubernetes.io/component",
		"app.kubernetes.io/instance",
		"app.kubernetes.io/name",
	}

	// StreamPort defines the port used by NGINX for the NGINX stream configuration socket
	StreamPort = 10247

	// StatusPort port used by NGINX for the status server
	StatusPort = 10246

	// HealthPath defines the path used to define the health check location in NGINX
	HealthPath = "/healthz"

	// HealthCheckTimeout defines the time limit in seconds for a probe to health-check-path to succeed
	HealthCheckTimeout = 10 * time.Second

	// ProfilerPort port used by the ingress controller to expose the Go Profiler when it is enabled.
	ProfilerPort = 10245

	// TemplatePath path of the NGINX template
	TemplatePath = "/etc/nginx/template/nginx.tmpl"

	// PID defines the location of the pid file used by NGINX
	PID = "/tmp/nginx.pid"
)

// PodInfo contains runtime information about the pod running the Ingres controller
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PodInfo struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

// the nginx controller entry function
func startNginxController() {
	klog.InitFlags(nil)

	rand.Seed(time.Now().UnixNano())

	// get parse flags into conf
	showVersion , conf , err := ingress.ParseFlags()

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

	// MetaManager client
	metaClient := client.New()

	// kubernetes client
	kubeClient := fakekube.NewSimpleClientset(metaClient)

	if kubeClient == nil {
		handleFatalInitError("change metaClient to kubeClient error")
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

	// ssl cert
	conf.FakeCertificate = ssl.GetFakeSSLCert()
	klog.Infof("SSL fake certificate created", "file", conf.FakeCertificate.PemFileName)

	// network
	var isNetworkingIngressAvailable bool

	isNetworkingIngressAvailable, IsIngressV1Beta1Ready, _ = NetworkingIngressAvailable(kubeClient)
	if !isNetworkingIngressAvailable {
		klog.Fatalf("ingress-nginx requires Kubernetes v1.14.0 or higher")
	}

	if IsIngressV1Beta1Ready {
		klog.Info("Enabling new Ingress features available since Kubernetes v1.18")
		IngressClass,err := kubeClient.NetworkingV1beta1().IngressClasses().Get(context.TODO(), IngressClass, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				if !errors.IsUnauthorized(err) && !errors.IsForbidden(err) {
					klog.Fatalf("Error searching IngressClass: %v", err)
				}

				klog.Error(err, "Searching IngressClass", "class", IngressClass)
			}

			klog.Warningf("No IngressClass resource with name %v found. Only annotation will be used.", IngressClass)

			// TODO: remove once this is fixed in client-go
			IngressClass = nil
		}

		if IngressClass != nil && IngressClass.Spec.Controller != IngressNginxController {
			klog.Errorf(`Invalid IngressClass (Spec.Controller) value "%v". Should be "%v"`, IngressClass.Spec.Controller, IngressNginxController)
			klog.Fatalf("IngressClass with name %v is not valid for ingress-nginx (invalid Spec.Controller)", IngressClass)
		}
	}

	conf.Client = kubeClient

	err = GetIngressPod(kubeClient)
	if err != nil {
		klog.Fatalf("Unexpected error obtaining ingress-nginx pod: %v", err)
	}

	// register profiler
	if conf.EnableProfiling {
		go registerProfiler()
	}

	// create nginx controller
	ngx := NewNginxController(conf)

	mux := http.NewServeMux()
	registerHealthz(HealthPath, ngx, mux)
	go startHTTPServer(conf.ListenPorts.Health, mux)
	go ngx.Start()

	handleSigterm(ngx, func(code int) {
		os.Exit(code)
	})
}

type exiter func(code int)

func handleSigterm(ngx *NginxController, exit exiter) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	klog.Info("Received SIGTERM, shutting down")

	exitCode := 0
	if err := ngx.Stop(); err != nil {
		klog.Warningf("Error during shutdown: %v", err)
		exitCode = 1
	}

	klog.Info("Handled quit, awaiting Pod deletion")
	time.Sleep(10 * time.Second)

	klog.Info("Exiting", "code", exitCode)
	exit(exitCode)
}

// NginxController describes a NGINX Ingress controller.
type NginxController struct {
	cfg *Configuration

	recorder record.EventRecorder

	syncQueue *task.Queue

	syncStatus defaults.Syncer

	syncRateLimiter flowcontrol.RateLimiter

	// stopLock is used to enforce that only a single call to Stop send at
	// a given time. We allow stopping through an HTTP endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock *sync.Mutex

	stopCh   chan struct{}
	updateCh *channels.RingChannel

	// ngxErrCh is used to detect errors with the NGINX processes
	ngxErrCh chan error

	// runningConfig contains the running configuration in the Backend
	runningConfig *ingress.Configuration

	t ngx_template.TemplateWriter

	resolver []net.IP

	isIPV6Enabled bool

	isShuttingDown bool

	Proxy *TCPProxy

	store store.Storer

	//metricCollector metric.Collector

	validationWebhookServer *http.Server

	command NginxExecTester
}

// NewNginxController create a new nginx Ingress controller
func NewNginxController(config *Configuration) *NginxController {
	// event broadcaset
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: config.Client.CoreV1().Events(config.Namespace),
	})

	// dns解析
	h, err := dns.GetSystemNameServers()
	if err != nil {
		klog.Warningf("Error reading system nameservers: %v", err)
	}

	// 创建nginx controller
	n := &NginxController{
		isIPV6Enabled: ing_net.IsIPv6Enabled(),

		resolver:        h,
		cfg:             config,
		syncRateLimiter: flowcontrol.NewTokenBucketRateLimiter(config.SyncRateLimit, 1),

		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{
			Component: "nginx-ingress-controller",
		}),

		stopCh:   make(chan struct{}),
		updateCh: channels.NewRingChannel(1024),

		ngxErrCh: make(chan error),

		stopLock: &sync.Mutex{},

		runningConfig: new(ingress.Configuration),

		Proxy: &TCPProxy{},

		//metricCollector: mc,

		command: NewNginxCommand(),
	}

	// admission controller achieve webhook server
	if n.cfg.ValidationWebhook != "" {
		n.validationWebhookServer = &http.Server{
			Addr: config.ValidationWebhook,
			Handler: adm_controller.NewAdmissionControllerServer(&adm_controller.IngressAdmission{Checker: n}),
			TLSConfig: ssl.NewTLSListener(n.cfg.ValidationWebhookCertPath, n.cfg.ValidationWebhookKeyPath).TLSConfig(),
			// disable http/2
			// https://github.com/kubernetes/kubernetes/issues/80313
			// https://github.com/kubernetes/ingress-nginx/issues/6323#issuecomment-737239159
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
	}

	n.store = store.New(
		config.Namespace,
		config.ConfigMapName,
		config.TCPConfigMapName,
		config.UDPConfigMapName,
		config.DefaultSSLCertificate,
		config.ResyncPeriod,
		config.Client,
		n.updateCh,
		config.DisableCatchAll)

	n.syncQueue = task.NewTaskQueue(n.syncIngress)

	if config.UpdateStatus {
		n.syncStatus = defaults.NewStatusSyncer(defaults.Config{
			Client:                 config.Client,
			PublishService:         config.PublishService,
			PublishStatusAddress:   config.PublishStatusAddress,
			IngressLister:          n.store,
			UpdateStatusOnShutdown: config.UpdateStatusOnShutdown,
			UseNodeInternalIP:      config.UseNodeInternalIP,
		})
	} else {
		klog.Warning("Update of Ingress status is disabled (flag --update-status)")
	}

	onTemplateChange := func() {
		template, err := ngx_template.NewTemplate(TemplatePath)
		if err != nil {
			// this error is different from the rest because it must be clear why nginx is not working
			klog.Error(err, "Error loading new template")
			return
		}

		n.t = template
		klog.Info("New NGINX configuration template loaded")
		n.syncQueue.EnqueueTask(task.GetDummyObject("template-change"))
	}

	ngxTpl, err := ngx_template.NewTemplate(TemplatePath)
	if err != nil {
		klog.Fatalf("Invalid NGINX configuration template: %v", err)
	}

	n.t = ngxTpl

	_, err = defaults.NewFileWatcher(TemplatePath, onTemplateChange)
	if err != nil {
		klog.Fatalf("Error creating file watcher for %v: %v", TemplatePath, err)
	}

	filesToWatch := []string{}
	err = filepath.Walk("/etc/nginx/geoip/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		filesToWatch = append(filesToWatch, path)
		return nil
	})

	if err != nil {
		klog.Fatalf("Error creating file watchers: %v", err)
	}

	for _, f := range filesToWatch {
		_, err = defaults.NewFileWatcher(f, func() {
			klog.Info("File changed detected. Reloading NGINX", "path", f)
			n.syncQueue.EnqueueTask(task.GetDummyObject("file-change"))
		})
		if err != nil {
			klog.Fatalf("Error creating file watcher for %v: %v", f, err)
		}
	}

	return n
}

// Start starts a new NGINX master process running in the foreground.
func (n *NginxController) Start() {
	klog.Info("Starting NGINX Ingress controller")

	n.store.Run(n.stopCh)

	//we need to use the defined ingress class to allow multiple leaders
	//in order to update information about ingress status
	electionID := fmt.Sprintf("%v-%v", n.cfg.ElectionID, DefaultClass)
	if IngressClass != "" {
		electionID = fmt.Sprintf("%v-%v", n.cfg.ElectionID, IngressClass)
	}

	setupLeaderElection(&leaderElectionConfig{
		Client:     n.cfg.Client,
		ElectionID: electionID,
		OnStartedLeading: func(stopCh chan struct{}) {
			if n.syncStatus != nil {
				go n.syncStatus.Run(stopCh)
			}

			//n.metricCollector.OnStartedLeading(electionID)
			// manually update SSL expiration metrics
			// (to not wait for a reload)
			//n.metricCollector.SetSSLExpireTime(n.runningConfig.Servers)
		},
		OnStoppedLeading: func() {
			//n.metricCollector.OnStoppedLeading(electionID)
		},
	})

	cmd := n.command.ExecCommand()

	// put NGINX in another process group to prevent it
	// to receive signals meant for the controller
	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Setpgid: true,
	//	Pgid:    0,
	//}


	if n.cfg.EnableSSLPassthrough {
		n.setupSSLProxy()
	}

	klog.Info("Starting NGINX process")
	n.start(cmd)

	go n.syncQueue.Run(time.Second, n.stopCh)
	// force initial sync
	n.syncQueue.EnqueueTask(task.GetDummyObject("initial-sync"))

	// In case of error the temporal configuration file will
	// be available up to five minutes after the error
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			err := cleanTempNginxCfg()
			if err != nil {
				klog.Error(err, "Unexpected error removing temporal configuration files")
			}
		}
	}()

	if n.validationWebhookServer != nil {
		klog.Info("Starting validation webhook", "address", n.validationWebhookServer.Addr,
			"certPath", n.cfg.ValidationWebhookCertPath, "keyPath", n.cfg.ValidationWebhookKeyPath)
		go func() {
			klog.Error(n.validationWebhookServer.ListenAndServeTLS("", ""), "Error listening for TLS connections")
		}()
	}

	for {
		select {
		case err := <-n.ngxErrCh:
			if n.isShuttingDown {
				return
			}

			// if the nginx master process dies, the workers continue to process requests
			// until the failure of the configured livenessProbe and restart of the pod.
			if ingress.IsRespawnIfRequired(err) {
				return
			}

		case event := <-n.updateCh.Out():
			if n.isShuttingDown {
				break
			}

			if evt, ok := event.(store.Event); ok {
				klog.V(3).Info("Event received", "type", evt.Type, "object", evt.Obj)
				if evt.Type == store.ConfigurationEvent {
					// TODO: is this necessary? Consider removing this special case
					n.syncQueue.EnqueueTask(task.GetDummyObject("configmap-change"))
					continue
				}

				n.syncQueue.EnqueueSkippableTask(evt.Obj)
			} else {
				klog.Warningf("Unexpected event type received %T", event)
			}
		case <-n.stopCh:
			return
		}
	}
}

// Stop gracefully stops the NGINX master process.
func (n *NginxController) Stop() error {
	n.isShuttingDown = true

	n.stopLock.Lock()
	defer n.stopLock.Unlock()

	if n.syncQueue.IsShuttingDown() {
		return fmt.Errorf("shutdown already in progress")
	}

	klog.Info("Shutting down controller queues")
	close(n.stopCh)
	go n.syncQueue.Shutdown()
	if n.syncStatus != nil {
		n.syncStatus.Shutdown()
	}

	if n.validationWebhookServer != nil {
		klog.Info("Stopping admission controller")
		err := n.validationWebhookServer.Close()
		if err != nil {
			return err
		}
	}

	// send stop signal to NGINX
	klog.Info("Stopping NGINX process")
	cmd := n.command.ExecCommand("-s", "quit")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	// wait for the NGINX process to terminate
	timer := time.NewTicker(time.Second * 1)
	for range timer.C {
		if !IsRunning() {
			klog.Info("NGINX process has stopped")
			timer.Stop()
			break
		}
	}

	return nil
}

func (n *NginxController) start(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		klog.Fatalf("NGINX error: %v", err)
		n.ngxErrCh <- err
		return
	}

	go func() {
		n.ngxErrCh <- cmd.Wait()
	}()
}


func (n *NginxController) setupSSLProxy() {
	cfg := n.store.GetBackendConfiguration()
	sslPort := n.cfg.ListenPorts.HTTPS
	proxyPort := n.cfg.ListenPorts.SSLProxy

	klog.Info("Starting TLS proxy for SSL Pass through")
	n.Proxy = &TCPProxy{
		Default: &TCPServer{
			Hostname:      "localhost",
			IP:            "127.0.0.1",
			Port:          proxyPort,
			ProxyProtocol: true,
		},
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", sslPort))
	if err != nil {
		klog.Fatalf("%v", err)
	}

	proxyList := &proxyproto.Listener{Listener: listener, ProxyHeaderTimeout: cfg.ProxyProtocolHeaderTimeout}

	// accept TCP connections on the configured HTTPS port
	go func() {
		for {
			var conn net.Conn
			var err error

			if n.store.GetBackendConfiguration().UseProxyProtocol {
				// wrap the listener in order to decode Proxy
				// Protocol before handling the connection
				conn, err = proxyList.Accept()
			} else {
				conn, err = listener.Accept()
			}

			if err != nil {
				klog.Warningf("Error accepting TCP connection: %v", err)
				continue
			}

			klog.V(3).Info("Handling TCP connection", "remote", conn.RemoteAddr(), "local", conn.LocalAddr())
			go n.Proxy.Handle(conn)
		}
	}()
}

// clean nginx template config
func cleanTempNginxCfg() error {
	var files []string

	err := filepath.Walk(os.TempDir(), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && os.TempDir() != path {
			return filepath.SkipDir
		}

		dur, _ := time.ParseDuration("-5m")
		fiveMinutesAgo := time.Now().Add(dur)
		if strings.HasPrefix(info.Name(), tempNginxPattern) && info.ModTime().Before(fiveMinutesAgo) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}

	return nil
}

// Handler for fatal init errors. Prints a verbose error message and exits.
func handleFatalInitError(err string) {
	klog.Fatalf("Error while initiating a connection to the KubeEdge and Kubernetes API server. "+
		"This could mean the cluster is misconfigured (e.g. it has invalid API server certificates "+
		"or Service Accounts configuration). Reason: %s\n"+
		"Refer to the troubleshooting guide for more information: "+
		"https://kubernetes.github.io/ingress-nginx/troubleshooting/",
		err)
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
func checkService(key string, kubeClient kubernetes.Interface ) error {
	ns, name, err := ParseNameNS(key)
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Services(ns).Get(context.TODO(), name , metav1.GetOptions{})
	if err != nil {
		if err2.IsUnauthorized(err) || err2.IsForbidden(err) {
			return fmt.Errorf("✖ the cluster seems to be running with a restrictive Authorization mode and the Ingress controller does not have the required permissions to operate normally")
		}

		if err2.IsNotFound(err) {
			return fmt.Errorf("No service with name %v found in namespace %v: %v", name, ns, err)
		}

		return fmt.Errorf("Unexpected error searching service with name %v in namespace %v: %v", name, ns, err)
	}

	return nil
}

// ParseNameNS parses a string searching a namespace and name
func ParseNameNS(input string) (string, string, error) {
	nsName := strings.Split(input, "/")
	if len(nsName) != 2 {
		return "", "", fmt.Errorf("invalid format (namespace/name) found in '%v'", input)
	}

	return nsName[0], nsName[1], nil
}

// NetworkingIngressAvailable checks if the package "k8s.io/api/networking/v1beta1"
// is available or not and if Ingress V1 is supported (k8s >= v1.18.0)
func NetworkingIngressAvailable(client kubernetes.Interface) (bool, bool, bool) {
	// check kubernetes version to use new ingress package or not
	version114, _ := version.ParseGeneric("v1.14.0")
	version118, _ := version.ParseGeneric("v1.18.0")
	version119, _ := version.ParseGeneric("v1.19.0")

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return false, false, false
	}

	runningVersion, err := version.ParseGeneric(serverVersion.String())
	if err != nil {
		klog.Error(err, "unexpected error parsing running Kubernetes version")
		return false, false, false
	}

	return runningVersion.AtLeast(version114), runningVersion.AtLeast(version118), runningVersion.AtLeast(version119)
}

// GetIngressPod load the ingress-nginx pod
func GetIngressPod(kubeClient kubernetes.Interface) error {
	podName := os.Getenv("POD_NAME")
	podNs := os.Getenv("POD_NAMESPACE")

	if podName == "" || podNs == "" {
		return fmt.Errorf("unable to get POD information (missing POD_NAME or POD_NAMESPACE environment variable")
	}

	pod, err := kubeClient.CoreV1().Pods(podNs).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get POD information: %v", err)
	}

	labels := map[string]string{}
	for _, key := range selectorLabelKeys {
		value, ok := pod.GetLabels()[key]
		if !ok {
			return fmt.Errorf("label %v is missing. Please do not remove", key)
		}

		labels[key] = value
	}

	IngressPodDetails = &PodInfo{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
	}

	pod.ObjectMeta.DeepCopyInto(&IngressPodDetails.ObjectMeta)
	IngressPodDetails.SetLabels(labels)

	return nil
}


func registerProfiler() {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/heap", pprof.Index)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Index)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Index)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
	mux.HandleFunc("/debug/pprof/block", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%v", ProfilerPort),
		Handler: mux,
	}
	klog.Fatal(server.ListenAndServe())
}

func registerHealthz(healthPath string, ic *NginxController, mux *http.ServeMux) {
	// expose health check endpoint (/healthz)
	healthz.InstallPathHandler(
		mux,
		healthPath,
		healthz.PingHealthz,
		ic,
	)
}

func startHTTPServer(port int, mux *http.ServeMux) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%v", port),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	klog.Fatal(server.ListenAndServe())
}

// Name returns the healthcheck name
func (n NginxController) Name() string {
	return "nginx-ingress-controller"
}

// Check returns if the nginx healthz endpoint is returning ok (status code 200)
func (n *NginxController) Check(_ *http.Request) error {
	if n.isShuttingDown {
		return fmt.Errorf("the ingress controller is shutting down")
	}

	// check the nginx master process is running
	fs, err := proc.NewFS("/proc", false)
	if err != nil {
		return err1.Wrap(err, "reading /proc directory")
	}

	f, err := ioutil.ReadFile(PID)
	if err != nil {
		return err1.Wrapf(err, "reading %v", PID)
	}

	pid, err := strconv.Atoi(strings.TrimRight(string(f), "\r\n"))
	if err != nil {
		return err1.Wrapf(err, "reading NGINX PID from file %v", PID)
	}

	_, err = fs.NewProc(pid)
	if err != nil {
		return err1.Wrapf(err, "checking for NGINX process with PID %v", pid)
	}

	statusCode, _, err := defaults.NewGetStatusRequest("/is-dynamic-lb-initialized")
	if err != nil {
		return err1.Wrapf(err, "checking if the dynamic load balancer started")
	}

	if statusCode != 200 {
		return fmt.Errorf("dynamic load balancer not started")
	}

	return nil
}


// GeoLite2DBExists checks if the required databases for
// the GeoIP2 NGINX module are present in the filesystem
func GeoLite2DBExists() bool {
	for _, dbName := range strings.Split(MaxmindEditionIDs, ",") {
		if !fileExists(path.Join(geoIPPath, dbName+dbExtension)) {
			return false
		}
	}

	return true
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

// IsRunning returns true if a process with the name 'nginx' is found
func IsRunning() bool {
	processes, _ := ps.Processes()
	for _, p := range processes {
		if p.Executable() == "nginx" {
			return true
		}
	}

	return false
}

// generateTemplate returns the nginx configuration file content
func (n NginxController) generateTemplate(cfg ngx_config.Configuration, ingressCfg ingress.Configuration) ([]byte, error) {

	if n.cfg.EnableSSLPassthrough {
		servers := []*TCPServer{}
		for _, pb := range ingressCfg.PassthroughBackends {
			svc := pb.Service
			if svc == nil {
				klog.Warningf("Missing Service for SSL Passthrough backend %q", pb.Backend)
				continue
			}
			port, err := strconv.Atoi(pb.Port.String()) // #nosec
			if err != nil {
				for _, sp := range svc.Spec.Ports {
					if sp.Name == pb.Port.String() {
						port = int(sp.Port)
						break
					}
				}
			} else {
				for _, sp := range svc.Spec.Ports {
					if sp.Port == int32(port) {
						port = int(sp.Port)
						break
					}
				}
			}

			// TODO: Allow PassthroughBackends to specify they support proxy-protocol
			servers = append(servers, &TCPServer{
				Hostname:      pb.Hostname,
				IP:            svc.Spec.ClusterIP,
				Port:          port,
				ProxyProtocol: false,
			})
		}

		n.Proxy.ServerList = servers
	}

	// NGINX cannot resize the hash tables used to store server names. For
	// this reason we check if the current size is correct for the host
	// names defined in the Ingress rules and adjust the value if
	// necessary.
	// https://trac.nginx.org/nginx/ticket/352
	// https://trac.nginx.org/nginx/ticket/631
	var longestName int
	var serverNameBytes int

	for _, srv := range ingressCfg.Servers {
		hostnameLength := len(srv.Hostname)
		if srv.RedirectFromToWWW {
			hostnameLength += 4
		}
		if longestName < hostnameLength {
			longestName = hostnameLength
		}

		for _, alias := range srv.Aliases {
			if longestName < len(alias) {
				longestName = len(alias)
			}
		}

		serverNameBytes += hostnameLength
	}

	nameHashBucketSize := nginxHashBucketSize(longestName)
	if cfg.ServerNameHashBucketSize < nameHashBucketSize {
		klog.V(3).Info("Adjusting ServerNameHashBucketSize variable", "value", nameHashBucketSize)
		cfg.ServerNameHashBucketSize = nameHashBucketSize
	}

	serverNameHashMaxSize := nextPowerOf2(serverNameBytes)
	if cfg.ServerNameHashMaxSize < serverNameHashMaxSize {
		klog.V(3).Info("Adjusting ServerNameHashMaxSize variable", "value", serverNameHashMaxSize)
		cfg.ServerNameHashMaxSize = serverNameHashMaxSize
	}

	if cfg.MaxWorkerOpenFiles == 0 {
		// the limit of open files is per worker process
		// and we leave some room to avoid consuming all the FDs available
		wp, err := strconv.Atoi(cfg.WorkerProcesses)
		klog.V(3).Info("Worker processes", "count", wp)
		if err != nil {
			wp = 1
		}
		maxOpenFiles := (rlimitMaxNumFiles() / wp) - 1024
		klog.V(3).Info("Maximum number of open file descriptors", "value", maxOpenFiles)
		if maxOpenFiles < 1024 {
			// this means the value of RLIMIT_NOFILE is too low.
			maxOpenFiles = 1024
		}
		klog.V(3).Info("Adjusting MaxWorkerOpenFiles variable", "value", maxOpenFiles)
		cfg.MaxWorkerOpenFiles = maxOpenFiles
	}

	if cfg.MaxWorkerConnections == 0 {
		maxWorkerConnections := int(float64(cfg.MaxWorkerOpenFiles * 3.0 / 4))
		klog.V(3).Info("Adjusting MaxWorkerConnections variable", "value", maxWorkerConnections)
		cfg.MaxWorkerConnections = maxWorkerConnections
	}

	setHeaders := map[string]string{}
	if cfg.ProxySetHeaders != "" {
		cmap, err := n.store.GetConfigMap(cfg.ProxySetHeaders)
		if err != nil {
			klog.Warningf("Error reading ConfigMap %q from local store: %v", cfg.ProxySetHeaders, err)
		} else {
			setHeaders = cmap.Data
		}
	}

	addHeaders := map[string]string{}
	if cfg.AddHeaders != "" {
		cmap, err := n.store.GetConfigMap(cfg.AddHeaders)
		if err != nil {
			klog.Warningf("Error reading ConfigMap %q from local store: %v", cfg.AddHeaders, err)
		} else {
			addHeaders = cmap.Data
		}
	}

	sslDHParam := ""
	if cfg.SSLDHParam != "" {
		secretName := cfg.SSLDHParam

		secret, err := n.store.GetSecret(secretName)
		if err != nil {
			klog.Warningf("Error reading Secret %q from local store: %v", secretName, err)
		} else {
			nsSecName := strings.Replace(secretName, "/", "-", -1)
			dh, ok := secret.Data["dhparam.pem"]
			if ok {
				pemFileName, err := ssl.AddOrUpdateDHParam(nsSecName, dh)
				if err != nil {
					klog.Warningf("Error adding or updating dhparam file %v: %v", nsSecName, err)
				} else {
					sslDHParam = pemFileName
				}
			}
		}
	}

	cfg.SSLDHParam = sslDHParam

	cfg.DefaultSSLCertificate = n.getDefaultSSLCertificate()

	tc := ngx_config.TemplateConfig{
		ProxySetHeaders:          setHeaders,
		AddHeaders:               addHeaders,
		BacklogSize:              sysctlSomaxconn(),
		Backends:                 ingressCfg.Backends,
		PassthroughBackends:      ingressCfg.PassthroughBackends,
		Servers:                  ingressCfg.Servers,
		TCPBackends:              ingressCfg.TCPEndpoints,
		UDPBackends:              ingressCfg.UDPEndpoints,
		Cfg:                      cfg,
		IsIPV6Enabled:            n.isIPV6Enabled && !cfg.DisableIpv6,
		NginxStatusIpv4Whitelist: cfg.NginxStatusIpv4Whitelist,
		NginxStatusIpv6Whitelist: cfg.NginxStatusIpv6Whitelist,
		RedirectServers:          buildRedirects(ingressCfg.Servers),
		IsSSLPassthroughEnabled:  n.cfg.EnableSSLPassthrough,
		ListenPorts:              n.cfg.ListenPorts,
		PublishService:           n.GetPublishService(),
		EnableMetrics:            n.cfg.EnableMetrics,
		MaxmindEditionFiles:      n.cfg.MaxmindEditionFiles,
		HealthzURI:               nginx.HealthPath,
		MonitorMaxBatchSize:      n.cfg.MonitorMaxBatchSize,
		PID:                      nginx.PID,
		StatusPath:               nginx.StatusPath,
		StatusPort:               nginx.StatusPort,
		StreamPort:               nginx.StreamPort,
	}

	tc.Cfg.Checksum = ingressCfg.ConfigurationChecksum

	return n.t.Write(tc)
}

// testTemplate checks if the NGINX configuration inside the byte array is valid
// running the command "nginx -t" using a temporal file.
func (n NginxController) testTemplate(cfg []byte) error {
	if len(cfg) == 0 {
		return fmt.Errorf("invalid NGINX configuration (empty)")
	}
	tmpfile, err := ioutil.TempFile("", tempNginxPattern)
	if err != nil {
		return err
	}
	defer tmpfile.Close()
	err = ioutil.WriteFile(tmpfile.Name(), cfg, ReadWriteByUser)
	if err != nil {
		return err
	}
	out, err := n.command.Test(tmpfile.Name())
	if err != nil {
		// this error is different from the rest because it must be clear why nginx is not working
		oe := fmt.Sprintf(`
-------------------------------------------------------------------------------
Error: %v
%v
-------------------------------------------------------------------------------
`, err, string(out))

		return err1.New(oe)
	}

	os.Remove(tmpfile.Name())
	return nil
}
