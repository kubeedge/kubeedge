/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/kubelet.go"
and made some variant
*/

package edged

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/jsonpb"
	cadvisorapi "github.com/google/cadvisor/info/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	recordtools "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/klog"
	kubeletinternalconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	pluginwatcherapi "k8s.io/kubernetes/pkg/kubelet/apis/pluginregistration/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/dockershim"
	dockerremote "k8s.io/kubernetes/pkg/kubelet/dockershim/remote"
	"k8s.io/kubernetes/pkg/kubelet/images"
	"k8s.io/kubernetes/pkg/kubelet/kuberuntime"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
	kubedns "k8s.io/kubernetes/pkg/kubelet/network/dns"
	"k8s.io/kubernetes/pkg/kubelet/pleg"
	"k8s.io/kubernetes/pkg/kubelet/pluginmanager"
	plugincache "k8s.io/kubernetes/pkg/kubelet/pluginmanager/cache"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/kubelet/remote"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
	kubestatus "k8s.io/kubernetes/pkg/kubelet/status"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/pkg/kubelet/util/queue"
	"k8s.io/kubernetes/pkg/kubelet/volumemanager"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/configmap"
	"k8s.io/kubernetes/pkg/volume/downwardapi"
	"k8s.io/kubernetes/pkg/volume/emptydir"
	"k8s.io/kubernetes/pkg/volume/hostpath"
	"k8s.io/kubernetes/pkg/volume/projected"
	secretvolume "k8s.io/kubernetes/pkg/volume/secret"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/cadvisor"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/clcm"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/containers"
	fakekube "github.com/kubeedge/kubeedge/edge/pkg/edged/fake"
	edgeimages "github.com/kubeedge/kubeedge/edge/pkg/edged/images"
	edgepleg "github.com/kubeedge/kubeedge/edge/pkg/edged/pleg"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/server"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/status"
	edgedutil "github.com/kubeedge/kubeedge/edge/pkg/edged/util"
	utilpod "github.com/kubeedge/kubeedge/edge/pkg/edged/util/pod"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/util/record"
	csiplugin "github.com/kubeedge/kubeedge/edge/pkg/edged/volume/csi"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	plegChannelCapacity = 1000
	plegRelistPeriod    = time.Second * 1
	concurrentConsumers = 5
	backOffPeriod       = 10 * time.Second
	// MaxContainerBackOff is the max backoff period, exported for the e2e test
	MaxContainerBackOff = 300 * time.Second
	enqueueDuration     = 10 * time.Second
	// ImageGCPeriod is the period for performing image garbage collection.
	ImageGCPeriod = 5 * time.Second
	// ContainerGCPeriod is the period for performing container garbage collection.
	ContainerGCPeriod = 60 * time.Second
	// Period for performing global cleanup tasks.
	housekeepingPeriod   = time.Second * 2
	syncWorkQueuePeriod  = time.Second * 2
	minAge               = 60 * time.Second
	imageGcHighThreshold = "edged.image-gc-high-threshold"
	syncMsgRespTimeout   = 1 * time.Minute
	//DefaultRootDir give default directory
	DefaultRootDir                   = "/var/lib/edged"
	workerResyncIntervalJitterFactor = 0.5
	//EdgeController gives controller name
	EdgeController = "edgecontroller"
	//RemoteContainerRuntime give Remote container runtime name
	RemoteContainerRuntime = "remote"
	//RemoteRuntimeEndpoint gives the default endpoint for CRI runtime
	RemoteRuntimeEndpoint = "unix:///var/run/dockershim.sock"
	//MinimumEdgedMemoryCapacity gives the minimum default memory (2G) of edge
	MinimumEdgedMemoryCapacity = 2147483647
	//PodSandboxImage gives the default pause container image
	PodSandboxImage = "k8s.gcr.io/pause"
	//DockerEndpoint gives the default endpoint for docker engine
	DockerEndpoint = "unix:///var/run/docker.sock"
	//DockerShimEndpoint gives the default endpoint for Docker shim runtime
	DockerShimEndpoint = "unix:///var/run/dockershim.sock"
	//DockerShimEndpointDeprecated this is the deprecated dockershim endpoint
	DockerShimEndpointDeprecated = "/var/run/dockershim.sock"
	//DockershimRootDir givesthe default path to the dockershim root directory
	DockershimRootDir = "/var/lib/dockershim"
	//HairpinMode only use forkubenetNetworkPlugin.Currently not working
	HairpinMode = kubeletinternalconfig.HairpinVeth
	//NonMasqueradeCIDR only use forkubenetNetworkPlugin.Currently not working
	NonMasqueradeCIDR = "10.0.0.1/8"
	//cgroupName used for check if the cgroup is mounted.(default "")
	cgroupName = ""
	// PluginName gives the plugin name.(default "",use noop plugin)
	pluginName = ""
	//PluginBinDir gives the dir of cni plugin executable file
	pluginBinDir = "/opt/cni/bin"
	// PluginConfDir gives the dir of cni plugin confguration file
	pluginConfDir = "/etc/cni/net.d"
	//MTU give the default maximum transmission unit of  net interface
	mtu = 1500
	// redirectContainerStream decide whether to redirect the container stream
	redirectContainerStream = false
	// ResolvConfDefault gives the default dns resolv configration file
	ResolvConfDefault = "/etc/resolv.conf"
	// ImagePullProgressDeadlineDefault gives the default image pull progress deadline
	ImagePullProgressDeadlineDefault = 60
)

var (
	zeroDuration = metav1.Duration{}
)

// podReady holds the initPodReady flag and its lock
type podReady struct {
	// initPodReady is flag to check Pod ready status
	initPodReady bool
	// podReadyLock is used to guard initPodReady flag
	podReadyLock sync.RWMutex
}

//Define edged
type edged struct {
	//dns config
	dnsConfigurer             *kubedns.Configurer
	context                   *context.Context
	hostname                  string
	namespace                 string
	nodeName                  string
	interfaceName             string
	uid                       types.UID
	nodeStatusUpdateFrequency time.Duration
	registrationCompleted     bool
	containerManager          cm.ContainerManager
	containerRuntimeName      string
	// container runtime
	containerRuntime   kubecontainer.Runtime
	podCache           kubecontainer.Cache
	os                 kubecontainer.OSInterface
	runtimeService     internalapi.RuntimeService
	podManager         podmanager.Manager
	pleg               pleg.PodLifecycleEventGenerator
	statusManager      kubestatus.Manager
	kubeClient         clientset.Interface
	probeManager       prober.Manager
	livenessManager    proberesults.Manager
	server             *server.Server
	podAdditionQueue   *workqueue.Type
	podAdditionBackoff *flowcontrol.Backoff
	podDeletionQueue   *workqueue.Type
	podDeletionBackoff *flowcontrol.Backoff
	imageGCManager     images.ImageGCManager
	containerGCManager kubecontainer.ContainerGC
	metaClient         client.CoreInterface
	volumePluginMgr    *volume.VolumePluginMgr
	mounter            mount.Interface
	volumeManager      volumemanager.VolumeManager
	rootDirectory      string
	gpuPluginEnabled   bool
	version            string
	// podReady is structure with initPodReady flag and its lock
	podReady
	// cache for secret
	secretStore    cache.Store
	configMapStore cache.Store
	workQueue      queue.WorkQueue
	clcm           clcm.ContainerLifecycleManager
	//edged cgroup driver for container runtime
	cgroupDriver string
	//clusterDns dns
	clusterDNS []net.IP
	// edge node IP
	nodeIP net.IP

	// pluginmanager runs a set of asynchronous loops that figure out which
	// plugins need to be registered/unregistered based on this node and makes it so.
	pluginManager pluginmanager.PluginManager

	recorder recordtools.EventRecorder
}

//Config defines configuration details
type Config struct {
	nodeName                  string
	nodeNamespace             string
	interfaceName             string
	memoryCapacity            int64
	nodeStatusUpdateInterval  time.Duration
	devicePluginEnabled       bool
	gpuPluginEnabled          bool
	imageGCHighThreshold      int
	imageGCLowThreshold       int
	imagePullProgressDeadline int
	MaxPerPodContainerCount   int
	DockerAddress             string
	runtimeType               string
	remoteRuntimeEndpoint     string
	remoteImageEndpoint       string
	RuntimeRequestTimeout     metav1.Duration
	PodSandboxImage           string
	cgroupDriver              string
	nodeIP                    string
	clusterDNS                string
	clusterDomain             string
}

// Register register edged
func Register() {
	edged, err := newEdged()
	if err != nil {
		klog.Errorf("init new edged error, %v", err)
		return
	}
	core.Register(edged)
}

func (e *edged) Name() string {
	return modules.EdgedModuleName
}

func (e *edged) Group() string {
	return modules.EdgedGroup
}

func (e *edged) Start(c *context.Context) {
	e.context = c
	e.metaClient = client.New(c)

	// use self defined client to replace fake kube client
	e.kubeClient = fakekube.NewSimpleClientset(e.metaClient)

	e.statusManager = status.NewManager(e.kubeClient, e.podManager, utilpod.NewPodDeleteSafety(), e.metaClient)
	if err := e.initializeModules(); err != nil {
		klog.Errorf("initialize module error: %v", err)
		os.Exit(1)
	}

	e.volumeManager = volumemanager.NewVolumeManager(
		true,
		types.NodeName(e.nodeName),
		e.podManager,
		e.statusManager,
		e.kubeClient,
		e.volumePluginMgr,
		e.containerRuntime,
		e.mounter,
		e.getPodsDir(),
		record.NewEventRecorder(),
		false,
		false,
	)
	go e.volumeManager.Run(edgedutil.NewSourcesReady(), utilwait.NeverStop)
	go utilwait.Until(e.syncNodeStatus, e.nodeStatusUpdateFrequency, utilwait.NeverStop)

	e.probeManager = prober.NewManager(e.statusManager, e.livenessManager, containers.NewContainerRunner(), kubecontainer.NewRefManager(), record.NewEventRecorder())
	e.pleg = edgepleg.NewGenericLifecycleRemote(e.containerRuntime, e.probeManager, plegChannelCapacity, plegRelistPeriod, e.podManager, e.statusManager, e.podCache, clock.RealClock{}, e.interfaceName)
	e.statusManager.Start()
	e.pleg.Start()

	e.podAddWorkerRun(concurrentConsumers)
	e.podRemoveWorkerRun(concurrentConsumers)

	housekeepingTicker := time.NewTicker(housekeepingPeriod)
	syncWorkQueueCh := time.NewTicker(syncWorkQueuePeriod)
	e.probeManager.Start()
	go e.syncLoopIteration(e.pleg.Watch(), housekeepingTicker.C, syncWorkQueueCh.C)
	go e.server.ListenAndServe()

	e.imageGCManager.Start()
	e.StartGarbageCollection()

	e.pluginManager = pluginmanager.NewPluginManager(
		e.getPluginsRegistrationDir(), /* sockDir */
		e.getPluginsDir(),             /* deprecatedSockDir */
		nil,
	)

	// Adding Registration Callback function for CSI Driver
	e.pluginManager.AddHandler(pluginwatcherapi.CSIPlugin, plugincache.PluginHandler(csiplugin.PluginHandler))
	// Start the plugin manager
	klog.Infof("starting plugin manager")
	go e.pluginManager.Run(edgedutil.NewSourcesReady(), utilwait.NeverStop)

	klog.Infof("starting syncPod")
	e.syncPod()
}

func (e *edged) Cleanup() {
	if err := recover(); err != nil {
		klog.Errorf("edged exit with error: %v", err)
	}
	e.context.Cleanup(e.Name())
	klog.Info("edged exit!")
}

// isInitPodReady is used to safely return initPodReady flag
func (e *edged) isInitPodReady() bool {
	e.podReadyLock.RLock()
	defer e.podReadyLock.RUnlock()
	return e.initPodReady
}

// setInitPodReady is used to safely set initPodReady flag
func (e *edged) setInitPodReady(readyStatus bool) {
	e.podReadyLock.Lock()
	defer e.podReadyLock.Unlock()
	e.initPodReady = readyStatus
}

func getConfig() *Config {
	var conf Config
	var ok bool
	conf.nodeName = config.CONFIG.GetConfigurationByKey("edged.hostname-override").(string)
	conf.nodeNamespace = config.CONFIG.GetConfigurationByKey("edged.register-node-namespace").(string)
	conf.interfaceName = config.CONFIG.GetConfigurationByKey("edged.interface-name").(string)
	nodeStatusUpdateInterval := config.CONFIG.GetConfigurationByKey("edged.node-status-update-frequency").(int)
	conf.nodeStatusUpdateInterval = time.Duration(nodeStatusUpdateInterval) * time.Second
	conf.devicePluginEnabled = config.CONFIG.GetConfigurationByKey("edged.device-plugin-enabled").(bool)
	conf.gpuPluginEnabled = config.CONFIG.GetConfigurationByKey("edged.gpu-plugin-enabled").(bool)
	conf.imageGCHighThreshold = config.CONFIG.GetConfigurationByKey("edged.image-gc-high-threshold").(int)
	conf.imageGCLowThreshold = config.CONFIG.GetConfigurationByKey("edged.image-gc-low-threshold").(int)
	conf.MaxPerPodContainerCount = config.CONFIG.GetConfigurationByKey("edged.maximum-dead-containers-per-container").(int)
	if conf.DockerAddress, ok = config.CONFIG.GetConfigurationByKey("edged.docker-address").(string); !ok {
		conf.DockerAddress = DockerEndpoint
	}
	if conf.runtimeType, ok = config.CONFIG.GetConfigurationByKey("edged.runtime-type").(string); !ok {
		conf.runtimeType = RemoteContainerRuntime
	}
	if conf.cgroupDriver, ok = config.CONFIG.GetConfigurationByKey("edged.cgroup-driver").(string); !ok {
		conf.cgroupDriver = "systemd"
	}
	if conf.nodeIP, ok = config.CONFIG.GetConfigurationByKey("edged.node-ip").(string); !ok {
		conf.nodeIP = "127.0.0.1"
	}
	if conf.clusterDNS, ok = config.CONFIG.GetConfigurationByKey("edged.cluster-dns").(string); !ok {
		conf.clusterDNS = ""
	}
	if conf.clusterDomain, ok = config.CONFIG.GetConfigurationByKey("edged.cluster-domain").(string); !ok {
		conf.clusterDomain = ""
	}

	//Deal with 32-bit and 64-bit compatibility issues: issue #1070
	switch v := config.CONFIG.GetConfigurationByKey("edged.edged-memory-capacity-bytes").(type) {
	case int:
		conf.memoryCapacity = int64(v)
	case int64:
		conf.memoryCapacity = v
	default:
		panic("Invalid type for edged.edged-memory-capacity-bytes, valid types are one of [int,int64].")
	}

	if conf.memoryCapacity == 0 {
		conf.memoryCapacity = MinimumEdgedMemoryCapacity
	}
	conf.remoteRuntimeEndpoint = config.CONFIG.GetConfigurationByKey("edged.remote-runtime-endpoint").(string)
	if conf.remoteRuntimeEndpoint == "" {
		conf.remoteRuntimeEndpoint = RemoteRuntimeEndpoint
	}
	conf.remoteImageEndpoint = config.CONFIG.GetConfigurationByKey("edged.remote-image-endpoint").(string)
	if conf.RuntimeRequestTimeout == zeroDuration {
		conf.RuntimeRequestTimeout = metav1.Duration{Duration: 2 * time.Minute}
	}
	conf.PodSandboxImage = config.CONFIG.GetConfigurationByKey("edged.podsandbox-image").(string)
	if conf.PodSandboxImage == "" {
		conf.PodSandboxImage = PodSandboxImage
	}
	conf.imagePullProgressDeadline = config.CONFIG.GetConfigurationByKey("edged.image-pull-progress-deadline").(int)
	if conf.imagePullProgressDeadline == 0 {
		conf.imagePullProgressDeadline = ImagePullProgressDeadlineDefault
	}
	return &conf
}

func getRuntimeAndImageServices(remoteRuntimeEndpoint string, remoteImageEndpoint string, runtimeRequestTimeout metav1.Duration) (internalapi.RuntimeService, internalapi.ImageManagerService, error) {
	rs, err := remote.NewRemoteRuntimeService(remoteRuntimeEndpoint, runtimeRequestTimeout.Duration)
	if err != nil {
		return nil, nil, err
	}
	is, err := remote.NewRemoteImageService(remoteImageEndpoint, runtimeRequestTimeout.Duration)
	if err != nil {
		return nil, nil, err
	}
	return rs, is, err
}

//newEdged creates new edged object and initialises it
func newEdged() (*edged, error) {
	conf := getConfig()
	backoff := flowcontrol.NewBackOff(backOffPeriod, MaxContainerBackOff)

	podManager := podmanager.NewPodManager()
	policy := images.ImageGCPolicy{
		HighThresholdPercent: conf.imageGCHighThreshold,
		LowThresholdPercent:  conf.imageGCLowThreshold,
		MinAge:               minAge,
	}
	// build new object to match interface
	recorder := record.NewEventRecorder()

	ed := &edged{
		nodeName:                  conf.nodeName,
		interfaceName:             conf.interfaceName,
		namespace:                 conf.nodeNamespace,
		gpuPluginEnabled:          conf.gpuPluginEnabled,
		cgroupDriver:              conf.cgroupDriver,
		podManager:                podManager,
		podAdditionQueue:          workqueue.New(),
		podCache:                  kubecontainer.NewCache(),
		podAdditionBackoff:        backoff,
		podDeletionQueue:          workqueue.New(),
		podDeletionBackoff:        backoff,
		kubeClient:                nil,
		nodeStatusUpdateFrequency: conf.nodeStatusUpdateInterval,
		mounter:                   mount.New(""),
		uid:                       types.UID("38796d14-1df3-11e8-8e5a-286ed488f209"),
		version:                   fmt.Sprintf("%s-kubeedge-%s", constants.CurrentSupportK8sVersion, version.Get()),
		rootDirectory:             DefaultRootDir,
		secretStore:               cache.NewStore(cache.MetaNamespaceKeyFunc),
		configMapStore:            cache.NewStore(cache.MetaNamespaceKeyFunc),
		workQueue:                 queue.NewBasicWorkQueue(clock.RealClock{}),
		nodeIP:                    net.ParseIP(conf.nodeIP),
		recorder:                  recorder,
	}

	err := ed.makePodDir()
	if err != nil {
		klog.Errorf("create pod dir [%s] failed: %v", ed.getPodsDir(), err)
		os.Exit(1)
	}

	ed.livenessManager = proberesults.NewManager()
	nodeRef := &v1.ObjectReference{
		Kind:      "Node",
		Name:      string(ed.nodeName),
		UID:       types.UID(ed.nodeName),
		Namespace: "",
	}
	statsProvider := edgeimages.NewStatsProvider()
	containerGCPolicy := kubecontainer.ContainerGCPolicy{
		MinAge:             minAge,
		MaxContainers:      -1,
		MaxPerPodContainer: conf.MaxPerPodContainerCount,
	}

	//ed.podCache = kubecontainer.NewCache()

	if conf.remoteRuntimeEndpoint != "" {
		// remoteImageEndpoint is same as remoteRuntimeEndpoint if not explicitly specified
		if conf.remoteImageEndpoint == "" {
			conf.remoteImageEndpoint = conf.remoteRuntimeEndpoint
		}
	}

	//create and start the docker shim running as a grpc server
	if conf.remoteRuntimeEndpoint == DockerShimEndpoint || conf.remoteRuntimeEndpoint == DockerShimEndpointDeprecated {
		streamingConfig := &streaming.Config{}
		DockerClientConfig := &dockershim.ClientConfig{
			DockerEndpoint:            conf.DockerAddress,
			ImagePullProgressDeadline: time.Duration(conf.imagePullProgressDeadline) * time.Second,
			EnableSleep:               true,
			WithTraceDisabled:         true,
		}

		pluginConfigs := dockershim.NetworkPluginSettings{
			HairpinMode:       kubeletinternalconfig.HairpinMode(HairpinMode),
			NonMasqueradeCIDR: NonMasqueradeCIDR,
			PluginName:        pluginName,
			PluginBinDirs:     []string{pluginBinDir},
			PluginConfDir:     pluginConfDir,
			MTU:               mtu,
		}

		redirectContainerStream := redirectContainerStream
		cgroupDriver := ed.cgroupDriver

		ds, err := dockershim.NewDockerService(DockerClientConfig, conf.PodSandboxImage, streamingConfig,
			&pluginConfigs, cgroupName, cgroupDriver, DockershimRootDir, redirectContainerStream)

		if err != nil {
			return nil, err
		}

		klog.Infof("RemoteRuntimeEndpoint: %q, remoteImageEndpoint: %q",
			conf.remoteRuntimeEndpoint, conf.remoteRuntimeEndpoint)

		klog.Info("Starting the GRPC server for the docker CRI shim.")
		server := dockerremote.NewDockerServer(conf.remoteRuntimeEndpoint, ds)
		if err := server.Start(); err != nil {
			return nil, err
		}

	}
	ed.clusterDNS = convertStrToIP(conf.clusterDNS)
	ed.dnsConfigurer = kubedns.NewConfigurer(recorder, nodeRef, ed.nodeIP, ed.clusterDNS, conf.clusterDomain, ResolvConfDefault)

	containerRefManager := kubecontainer.NewRefManager()
	httpClient := &http.Client{}
	runtimeService, imageService, err := getRuntimeAndImageServices(conf.remoteRuntimeEndpoint, conf.remoteRuntimeEndpoint, conf.RuntimeRequestTimeout)
	if err != nil {
		return nil, err
	}
	if ed.os == nil {
		ed.os = kubecontainer.RealOS{}
	}

	ed.clcm, err = clcm.NewContainerLifecycleManager(DefaultRootDir)

	var machineInfo cadvisorapi.MachineInfo
	machineInfo.MemoryCapacity = uint64(conf.memoryCapacity)
	containerRuntime, err := kuberuntime.NewKubeGenericRuntimeManager(
		recorder,
		ed.livenessManager,
		"",
		containerRefManager,
		&machineInfo,
		ed,
		ed.os,
		ed,
		httpClient,
		backoff,
		false,
		0,
		0,
		false,
		metav1.Duration{Duration: 100 * time.Millisecond},
		runtimeService,
		imageService,
		ed.clcm.InternalContainerLifecycle(),
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("New generic runtime manager failed, err: %s", err.Error())
	}

	cadvisorInterface, err := cadvisor.New("")
	containerManager, err := cm.NewContainerManager(mount.New(""),
		cadvisorInterface,
		cm.NodeConfig{
			CgroupDriver:       conf.cgroupDriver,
			SystemCgroupsName:  conf.cgroupDriver,
			KubeletCgroupsName: conf.cgroupDriver,
			ContainerRuntime:   conf.runtimeType,
			KubeletRootDir:     DefaultRootDir,
		},
		false,
		conf.devicePluginEnabled,
		recorder)
	if err != nil {
		return nil, fmt.Errorf("init container manager failed with error: %v", err)
	}
	ed.containerRuntime = containerRuntime
	ed.containerRuntimeName = RemoteContainerRuntime
	ed.containerManager = containerManager
	ed.runtimeService = runtimeService
	imageGCManager, err := images.NewImageGCManager(ed.containerRuntime, statsProvider, recorder, nodeRef, policy, conf.PodSandboxImage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize image manager: %v", err)
	}
	ed.imageGCManager = imageGCManager

	containerGCManager, err := kubecontainer.NewContainerGC(containerRuntime, containerGCPolicy, &containers.KubeSourcesReady{})
	if err != nil {
		return nil, fmt.Errorf("init Container GC Manager failed with error %s", err.Error())
	}
	ed.containerGCManager = containerGCManager
	ed.server = server.NewServer(ed.podManager)
	ed.volumePluginMgr, err = NewInitializedVolumePluginMgr(ed, ProbeVolumePlugins(""))
	if err != nil {
		return nil, fmt.Errorf("init VolumePluginMgr failed with error %s", err.Error())
	}

	return ed, nil
}

func (e *edged) initializeModules() error {
	node, _ := e.initialNode()
	if err := e.containerManager.Start(node, e.GetActivePods, edgedutil.NewSourcesReady(), e.statusManager, e.runtimeService); err != nil {
		klog.Errorf("Failed to start device plugin manager %v", err)
		return err
	}
	return nil
}

func (e *edged) StartGarbageCollection() {
	go utilwait.Until(func() {
		err := e.imageGCManager.GarbageCollect()
		if err != nil {
			klog.Errorf("Image garbage collection failed")
		}
	}, ImageGCPeriod, utilwait.NeverStop)

	go utilwait.Until(func() {
		if e.isInitPodReady() {
			err := e.containerGCManager.GarbageCollect()
			if err != nil {
				klog.Errorf("Container garbage collection failed: %v", err)
			}
		}
	}, ContainerGCPeriod, utilwait.NeverStop)
}

func (e *edged) syncLoopIteration(plegCh <-chan *pleg.PodLifecycleEvent, housekeepingCh <-chan time.Time, syncWorkQueueCh <-chan time.Time) {

	for {
		select {
		case update := <-e.livenessManager.Updates():
			if update.Result == proberesults.Failure {
				pod, ok := e.podManager.GetPodByUID(update.PodUID)
				if !ok {
					klog.Infof("SyncLoop (container unhealthy): ignore irrelevant update: %#v", update)
					break
				}
				klog.Infof("SyncLoop (container unhealthy): %q", format.Pod(pod))
				if pod.Spec.RestartPolicy == v1.RestartPolicyNever {
					break
				}
				var containerCompleted bool
				if pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
							containerCompleted = true
							break
						}
					}
					if containerCompleted {
						break
					}
				}
				klog.Infof("Will restart pod [%s]", pod.Name)
				key := types.NamespacedName{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				}
				e.podAdditionQueue.Add(key.String())
			}
		case plegEvent := <-plegCh:
			if pod, ok := e.podManager.GetPodByUID(plegEvent.ID); ok {
				if plegEvent.Type == pleg.ContainerDied {
					if pod.Spec.RestartPolicy == v1.RestartPolicyNever {
						break
					}
					var containerCompleted bool
					if pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
						for _, containerStatus := range pod.Status.ContainerStatuses {
							if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
								containerCompleted = true
								break
							}
						}
						if containerCompleted {
							break
						}
					}
					klog.Errorf("sync loop get event container died, restart pod [%s]", pod.Name)
					key := types.NamespacedName{
						Namespace: pod.Namespace,
						Name:      pod.Name,
					}
					e.podAdditionQueue.Add(key.String())
				} else {
					klog.Infof("sync loop get event [%s], ignore it now.", plegEvent.Type)
				}
			} else {
				klog.Infof("sync loop ignore event: [%s], with pod [%s] not found", plegEvent.Type, plegEvent.ID)
			}
		case <-housekeepingCh:
			err := e.HandlePodCleanups()
			if err != nil {
				klog.Errorf("Handle Pod Cleanup Failed: %v", err)
			}
		case <-syncWorkQueueCh:
			podsToSync := e.getPodsToSync()
			if len(podsToSync) == 0 {
				break
			}
			for _, pod := range podsToSync {
				if !e.podIsTerminated(pod) {
					key := types.NamespacedName{
						Namespace: pod.Namespace,
						Name:      pod.Name,
					}
					e.podAdditionQueue.Add(key.String())
				}
			}
		}
	}
}

// NewNamespacedNameFromString parses the provided string and returns a NamespacedName
func NewNamespacedNameFromString(s string) types.NamespacedName {
	Separator := '/'
	nn := types.NamespacedName{}
	result := strings.Split(s, string(Separator))
	if len(result) == 2 {
		nn.Namespace = result[0]
		nn.Name = result[1]
	}
	return nn
}

func (e *edged) podAddWorkerRun(consumers int) {
	for i := 0; i < consumers; i++ {
		klog.Infof("start pod addition queue work %d", i)
		go func(i int) {
			for {
				item, quit := e.podAdditionQueue.Get()
				if quit {
					klog.Errorf("consumer: [%d], worker addition queue is shutting down!", i)
					return
				}
				namespacedName := NewNamespacedNameFromString(item.(string))
				podName := namespacedName.Name
				klog.Infof("worker [%d] get pod addition item [%s]", i, podName)
				backOffKey := fmt.Sprintf("pod_addition_worker_%s", podName)
				if e.podAdditionBackoff.IsInBackOffSinceUpdate(backOffKey, e.podAdditionBackoff.Clock.Now()) {
					klog.Errorf("consume pod addition backoff: Back-off consume pod [%s] addition  error, backoff: [%v]", podName, e.podAdditionBackoff.Get(backOffKey))
					go func() {
						klog.Infof("worker [%d] backoff pod addition item [%s] failed, re-add to queue", i, podName)
						time.Sleep(e.podAdditionBackoff.Get(backOffKey))
						e.podAdditionQueue.Add(item)
					}()
					e.podAdditionQueue.Done(item)
					continue
				}
				err := e.consumePodAddition(&namespacedName)
				if err != nil {
					if err == apis.ErrPodNotFound {
						klog.Infof("worker [%d] handle pod addition item [%s] failed with not found error.", i, podName)
						e.podAdditionBackoff.Reset(backOffKey)
					} else {
						go func() {
							klog.Errorf("worker [%d] handle pod addition item [%s] failed: %v, re-add to queue", i, podName, err)
							e.podAdditionBackoff.Next(backOffKey, e.podAdditionBackoff.Clock.Now())
							time.Sleep(enqueueDuration)
							e.podAdditionQueue.Add(item)
						}()
					}
				} else {
					e.podAdditionBackoff.Reset(backOffKey)
				}
				e.podAdditionQueue.Done(item)
			}
		}(i)
	}
}

func (e *edged) podRemoveWorkerRun(consumers int) {
	for i := 0; i < consumers; i++ {
		go func(i int) {
			for {
				item, quit := e.podDeletionQueue.Get()
				if quit {
					klog.Errorf("consumer: [%d], worker addition queue is shutting down!", i)
					return
				}
				namespacedName := NewNamespacedNameFromString(item.(string))
				podName := namespacedName.Name
				klog.Infof("consumer: [%d], worker get removed pod [%s]\n", i, podName)
				err := e.consumePodDeletion(&namespacedName)
				if err != nil {
					if err == apis.ErrContainerNotFound {
						klog.Infof("pod [%s] is not exist, with container not found error", podName)
					} else if err == apis.ErrPodNotFound {
						klog.Infof("pod [%s] is not found", podName)
					} else {
						go func(item interface{}) {
							klog.Errorf("worker remove pod [%s] failed: %v", podName, err)
							time.Sleep(2 * time.Second)
							e.podDeletionQueue.Add(item)
						}(item)
					}

				}
				e.podDeletionQueue.Done(item)
			}
		}(i)
	}
}

func (e *edged) consumePodAddition(namespacedName *types.NamespacedName) error {
	podName := namespacedName.Name
	klog.Infof("start to consume added pod [%s]", podName)
	pod, ok := e.podManager.GetPodByName(namespacedName.Namespace, podName)
	if !ok || pod.DeletionTimestamp != nil {
		return apis.ErrPodNotFound
	}

	if err := e.makePodDataDirs(pod); err != nil {
		klog.Errorf("Unable to make pod data directories for pod %q: %v", format.Pod(pod), err)
		return err
	}

	if err := e.volumeManager.WaitForAttachAndMount(pod); err != nil {
		klog.Errorf("Unable to mount volumes for pod %q: %v; skipping pod", format.Pod(pod), err)
		return err
	}

	secrets, err := e.getSecretsFromMetaManager(pod)
	if err != nil {
		return err
	}

	curPodStatus, err := e.podCache.Get(pod.GetUID())
	if err != nil {
		klog.Errorf("Pod status for %s from cache failed: %v", podName, err)
		return err
	}

	result := e.containerRuntime.SyncPod(pod, curPodStatus, secrets, e.podAdditionBackoff)
	if err := result.Error(); err != nil {
		// Do not return error if the only failures were pods in backoff
		for _, r := range result.SyncResults {
			if r.Error != kubecontainer.ErrCrashLoopBackOff && r.Error != images.ErrImagePullBackOff {
				// Do not record an event here, as we keep all event logging for sync pod failures
				// local to container runtime so we get better errors
				return err
			}
		}

		return nil
	}

	e.workQueue.Enqueue(pod.UID, utilwait.Jitter(time.Minute, workerResyncIntervalJitterFactor))
	klog.Infof("consume added pod [%s] successfully\n", podName)
	return nil
}

func (e *edged) consumePodDeletion(namespacedName *types.NamespacedName) error {
	podName := namespacedName.Name
	klog.Infof("start to consume removed pod [%s]", podName)
	pod, ok := e.podManager.GetPodByName(namespacedName.Namespace, podName)
	if !ok {
		return apis.ErrPodNotFound
	}

	podStatus, err := e.podCache.Get(pod.GetUID())
	if err != nil {
		klog.Errorf("Pod status for %s from cache failed: %v", podName, err)
		return err
	}

	err = e.containerRuntime.KillPod(pod, kubecontainer.ConvertPodStatusToRunningPod(e.containerRuntimeName, podStatus), nil)
	if err != nil {
		if err == apis.ErrContainerNotFound {
			return err
		}
		return fmt.Errorf("consume removed pod [%s] failed, %v", podName, err)
	}
	klog.Infof("consume removed pod [%s] successfully\n", podName)
	return nil
}

func (e *edged) syncPod() {
	time.Sleep(10 * time.Second)

	//send msg to metamanager to get existing pods
	info := model.NewMessage("").BuildRouter(e.Name(), e.Group(), e.namespace+"/"+model.ResourceTypePod,
		model.QueryOperation)
	e.context.Send(metamanager.MetaManagerModuleName, *info)
	for {
		if request, err := e.context.Receive(e.Name()); err == nil {
			_, resType, resID, err := util.ParseResourceEdge(request.GetResource(), request.GetOperation())
			op := request.GetOperation()
			if err != nil {
				klog.Errorf("failed to parse the Resource: %v", err)
				continue
			}

			var content []byte

			switch request.Content.(type) {
			case []byte:
				content = request.GetContent().([]byte)
			default:
				content, err = json.Marshal(request.Content)
				if err != nil {
					klog.Errorf("marshal message content failed: %v", err)
					continue
				}
			}
			klog.Infof("request content is %s", request.Content)
			switch resType {
			case model.ResourceTypePod:
				if op == model.ResponseOperation && resID == "" && request.GetSource() == metamanager.MetaManagerModuleName {
					err := e.handlePodListFromMetaManager(content)
					if err != nil {
						klog.Errorf("handle podList failed: %v", err)
						continue
					}
					e.setInitPodReady(true)
				} else if op == model.ResponseOperation && resID == "" && request.GetSource() == EdgeController {
					err := e.handlePodListFromEdgeController(content)
					if err != nil {
						klog.Errorf("handle controllerPodList failed: %v", err)
						continue
					}
					e.setInitPodReady(true)
				} else {
					err := e.handlePod(op, content)
					if err != nil {
						klog.Errorf("handle pod failed: %v", err)
						continue
					}
				}
			case model.ResourceTypeConfigmap:
				if op != model.ResponseOperation {
					err := e.handleConfigMap(op, content)
					if err != nil {
						klog.Errorf("handle configMap failed: %v", err)
					}
				} else {
					klog.Infof("skip to handle configMap with type response")
					continue
				}
			case model.ResourceTypeSecret:
				if op != model.ResponseOperation {
					err := e.handleSecret(op, content)
					if err != nil {
						klog.Errorf("handle secret failed: %v", err)
					}
				} else {
					klog.Infof("skip to handle secret with type response")
					continue
				}
			case constants.CSIResourceTypeVolume:
				klog.Infof("volume operation type: %s", op)
				res, err := e.handleVolume(op, content)
				if err != nil {
					klog.Errorf("handle volume failed: %v", err)
				} else {
					resp := request.NewRespByMessage(&request, res)
					e.context.SendResp(*resp)
				}
			default:
				klog.Errorf("resType is not pod or configmap or secret: esType is %s", resType)
				continue
			}

		} else {
			klog.Errorf("failed to get pod")
		}
	}
}

func (e *edged) handleVolume(op string, content []byte) (interface{}, error) {
	switch op {
	case constants.CSIOperationTypeCreateVolume:
		return e.createVolume(content)
	case constants.CSIOperationTypeDeleteVolume:
		return e.deleteVolume(content)
	case constants.CSIOperationTypeControllerPublishVolume:
		return e.controllerPublishVolume(content)
	case constants.CSIOperationTypeControllerUnpublishVolume:
		return e.controllerUnpublishVolume(content)
	}
	return nil, nil
}

func (e *edged) createVolume(content []byte) (interface{}, error) {
	req := &csi.CreateVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal create volume req error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("start create volume: %s", req.Name)
	ctl := csiplugin.NewController()
	res, err := ctl.CreateVolume(req)
	if err != nil {
		klog.Errorf("create volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end create volume: %s result: %v", req.Name, res)
	return res, nil
}

func (e *edged) deleteVolume(content []byte) (interface{}, error) {
	req := &csi.DeleteVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal delete volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start delete volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.DeleteVolume(req)
	if err != nil {
		klog.Errorf("delete volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end delete volume: %s result: %v", req.VolumeId, res)
	return res, nil
}

func (e *edged) controllerPublishVolume(content []byte) (interface{}, error) {
	req := &csi.ControllerPublishVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal controller publish volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start controller publish volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.ControllerPublishVolume(req)
	if err != nil {
		klog.Errorf("controller publish volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end controller publish volume:: %s result: %v", req.VolumeId, res)
	return res, nil
}

func (e *edged) controllerUnpublishVolume(content []byte) (interface{}, error) {
	req := &csi.ControllerUnpublishVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal controller publish volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start controller unpublish volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.ControllerUnpublishVolume(req)
	if err != nil {
		klog.Errorf("controller unpublish volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end controller unpublish volume:: %s result: %v", req.VolumeId, res)
	return res, nil
}

func (e *edged) handlePod(op string, content []byte) (err error) {
	var pod v1.Pod
	err = json.Unmarshal(content, &pod)
	if err != nil {
		return err
	}

	switch op {
	case model.InsertOperation:
		e.addPod(&pod)
	case model.UpdateOperation:
		e.updatePod(&pod)
	case model.DeleteOperation:
		if delPod, ok := e.podManager.GetPodByName(pod.Namespace, pod.Name); ok {
			e.deletePod(delPod)
		}
	}
	return nil
}

func (e *edged) handlePodListFromMetaManager(content []byte) (err error) {
	var lists []string
	err = json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return err
	}

	for _, list := range lists {
		var pod v1.Pod
		err = json.Unmarshal([]byte(list), &pod)
		if err != nil {
			return err
		}
		e.addPod(&pod)
	}

	return nil
}

func (e *edged) handlePodListFromEdgeController(content []byte) (err error) {
	var lists []v1.Pod
	if err := json.Unmarshal(content, &lists); err != nil {
		return err
	}

	for _, list := range lists {
		e.addPod(&list)
	}

	return nil
}

func (e *edged) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.Infof("start sync addition for pod [%s]", pod.Name)
	attrs := &lifecycle.PodAdmitAttributes{}
	attrs.Pod = pod
	otherpods := e.podManager.GetPods()
	attrs.OtherPods = otherpods
	nodeInfo := schedulercache.NewNodeInfo(pod)
	e.containerManager.UpdatePluginResources(nodeInfo, attrs)
	key := types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}
	e.podManager.AddPod(pod)
	e.probeManager.AddPod(pod)
	e.podAdditionQueue.Add(key.String())
	klog.Infof("success sync addition for pod [%s]", pod.Name)
}

func (e *edged) updatePod(obj interface{}) {
	newPod := obj.(*v1.Pod)
	klog.Infof("start update pod [%s]", newPod.Name)
	key := types.NamespacedName{
		Namespace: newPod.Namespace,
		Name:      newPod.Name,
	}
	e.podManager.UpdatePod(newPod)
	e.probeManager.AddPod(newPod)
	if newPod.DeletionTimestamp == nil {
		e.podAdditionQueue.Add(key.String())
	} else {
		e.podDeletionQueue.Add(key.String())
	}
	klog.Infof("success update pod is %+v\n", newPod)
}

func (e *edged) deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.Infof("start remove pod [%s]", pod.Name)
	e.podManager.DeletePod(pod)
	e.statusManager.TerminatePod(pod)
	e.probeManager.RemovePod(pod)
	klog.Infof("success remove pod [%s]", pod.Name)
}

func (e *edged) getSecretsFromMetaManager(pod *v1.Pod) ([]v1.Secret, error) {
	var secrets []v1.Secret
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		secret, err := e.metaClient.Secrets(e.namespace).Get(imagePullSecret.Name)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, *secret)
	}

	return secrets, nil
}

// Get pods which should be resynchronized. Currently, the following pod should be resynchronized:
//   * pod whose work is ready.
//   * internal modules that request sync of a pod.
func (e *edged) getPodsToSync() []*v1.Pod {
	allPods := e.podManager.GetPods()
	podUIDs := e.workQueue.GetWork()
	podUIDSet := sets.NewString()
	for _, podUID := range podUIDs {
		podUIDSet.Insert(string(podUID))
	}
	var podsToSync []*v1.Pod
	for _, pod := range allPods {
		if podUIDSet.Has(string(pod.UID)) {
			// The work of the pod is ready
			podsToSync = append(podsToSync, pod)
		}
	}
	return podsToSync
}

func (e *edged) handleConfigMap(op string, content []byte) (err error) {
	var configMap v1.ConfigMap
	err = json.Unmarshal(content, &configMap)
	if err != nil {
		return
	}
	_, exists, _ := e.configMapStore.Get(&configMap)
	switch op {
	case model.InsertOperation:
		err = e.configMapStore.Add(&configMap)
	case model.UpdateOperation:
		if exists {
			err = e.configMapStore.Update(&configMap)
		}
	case model.DeleteOperation:
		if exists {
			err = e.configMapStore.Delete(&configMap)
		}
	}
	if err == nil {
		klog.Infof("%s configMap [%s] for cache success.", op, configMap.Name)
	}
	return
}

func (e *edged) handleSecret(op string, content []byte) (err error) {
	var podSecret v1.Secret
	err = json.Unmarshal(content, &podSecret)
	if err != nil {
		return
	}
	_, exists, _ := e.secretStore.Get(&podSecret)
	switch op {
	case model.InsertOperation:
		err = e.secretStore.Add(&podSecret)
	case model.UpdateOperation:
		if exists {
			err = e.secretStore.Update(&podSecret)
		}
	case model.DeleteOperation:
		if exists {
			err = e.secretStore.Delete(&podSecret)
		}
	}
	if err == nil {
		klog.Infof("%s secret [%s] for cache success.", op, podSecret.Name)
	}
	return
}

// ProbeVolumePlugins collects all volume plugins into an easy to use list.
// PluginDir specifies the directory to search for additional third party
// volume plugins.
func ProbeVolumePlugins(pluginDir string) []volume.VolumePlugin {
	allPlugins := []volume.VolumePlugin{}
	hostPathConfig := volume.VolumeConfig{}
	allPlugins = append(allPlugins, configmap.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, emptydir.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, secretvolume.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, hostpath.ProbeVolumePlugins(hostPathConfig)...)
	allPlugins = append(allPlugins, csiplugin.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, downwardapi.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, projected.ProbeVolumePlugins()...)
	return allPlugins
}

func (e *edged) HandlePodCleanups() error {
	if !e.isInitPodReady() {
		return nil
	}
	pods := e.podManager.GetPods()
	containerRunningPods, err := e.containerRuntime.GetPods(true)
	if err != nil {
		return err
	}
	e.removeOrphanedPodStatuses(pods)
	err = e.cleanupOrphanedPodDirs(pods, containerRunningPods)
	if err != nil {
		return fmt.Errorf("Failed cleaning up orphaned pod directories: %s", err.Error())
	}

	return nil
}

func convertStrToIP(s string) []net.IP {
	substrs := strings.Split(s, ",")
	ips := make([]net.IP, 0)
	for _, substr := range substrs {
		if ip := net.ParseIP(substr); ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}
