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
	pluginwatcherapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
	kubeletinternalconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager"
	klconfigmap "k8s.io/kubernetes/pkg/kubelet/configmap"
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
	serverstats "k8s.io/kubernetes/pkg/kubelet/server/stats"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
	"k8s.io/kubernetes/pkg/kubelet/stats"
	kubestatus "k8s.io/kubernetes/pkg/kubelet/status"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/pkg/kubelet/util/queue"
	"k8s.io/kubernetes/pkg/kubelet/volumemanager"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/configmap"
	"k8s.io/kubernetes/pkg/volume/downwardapi"
	"k8s.io/kubernetes/pkg/volume/emptydir"
	"k8s.io/kubernetes/pkg/volume/hostpath"
	"k8s.io/kubernetes/pkg/volume/projected"
	secretvolume "k8s.io/kubernetes/pkg/volume/secret"
	"k8s.io/kubernetes/pkg/volume/util/hostutil"
	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"
	"k8s.io/utils/mount"

	"github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis"
	edgecadvisor "github.com/kubeedge/kubeedge/edge/pkg/edged/cadvisor"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/clcm"
	edgedconfig "github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/containers"
	fakekube "github.com/kubeedge/kubeedge/edge/pkg/edged/fake"
	edgeimages "github.com/kubeedge/kubeedge/edge/pkg/edged/images"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/server"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/status"
	edgedutil "github.com/kubeedge/kubeedge/edge/pkg/edged/util"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/util/record"
	csiplugin "github.com/kubeedge/kubeedge/edge/pkg/edged/volume/csi"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	plegChannelCapacity = 1000
	plegRelistPeriod    = time.Second * 1
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
	// redirectContainerStream decide whether to redirect the container stream
	redirectContainerStream = false
	// ResolvConfDefault gives the default dns resolv configration file
	ResolvConfDefault = "/etc/resolv.conf"
)

// podReady holds the initPodReady flag and its lock
type podReady struct {
	// initPodReady is flag to check Pod ready status
	initPodReady bool
	// podReadyLock is used to guard initPodReady flag
	podReadyLock sync.RWMutex
}

// edged is the main edged implementation.
type edged struct {
	// dns config
	dnsConfigurer             *kubedns.Configurer
	hostname                  string
	namespace                 string
	nodeName                  string
	runtimeCache              kubecontainer.RuntimeCache
	interfaceName             string
	uid                       types.UID
	nodeStatusUpdateFrequency time.Duration
	registrationCompleted     bool
	containerManager          cm.ContainerManager
	containerRuntimeName      string
	concurrentConsumers       int
	// container runtime
	containerRuntime   kubecontainer.Runtime
	podCache           kubecontainer.Cache
	os                 kubecontainer.OSInterface
	resourceAnalyzer   serverstats.ResourceAnalyzer
	runtimeService     internalapi.RuntimeService
	podManager         podmanager.Manager
	pleg               pleg.PodLifecycleEventGenerator
	statusManager      kubestatus.Manager
	kubeClient         clientset.Interface
	probeManager       prober.Manager
	livenessManager    proberesults.Manager
	startupManager     proberesults.Manager
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
	hostUtil           hostutil.HostUtils
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
	// edged cgroup driver for container runtime
	cgroupDriver string
	// clusterDns dns
	clusterDNS []net.IP
	// edge node IP
	nodeIP net.IP

	// StatsProvider provides the node and the container stats.
	*stats.StatsProvider

	// cAdvisor used for container information.
	cadvisor cadvisor.Interface

	// pluginmanager runs a set of asynchronous loops that figure out which
	// plugins need to be registered/unregistered based on this node and makes it so.
	pluginManager pluginmanager.PluginManager

	recorder recordtools.EventRecorder
	enable   bool

	configMapManager klconfigmap.Manager

	// Cached MachineInfo returned by cadvisor.
	machineInfo *cadvisorapi.MachineInfo

	dockerLegacyService dockershim.DockerLegacyService
	// Optional, defaults to simple Docker implementation
	runner kubecontainer.ContainerCommandRunner
}

// Register register edged
func Register(e *v1alpha1.Edged) {
	edgedconfig.InitConfigure(e)
	edged, err := newEdged(e.Enable)
	if err != nil {
		klog.Errorf("init new edged error, %v", err)
		os.Exit(1)
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

//Enable indicates whether this module is enabled
func (e *edged) Enable() bool {
	return e.enable
}

func (e *edged) Start() {
	e.volumePluginMgr = NewInitializedVolumePluginMgr(e, ProbeVolumePlugins(""))

	if err := e.initializeModules(); err != nil {
		klog.Errorf("initialize module error: %v", err)
		os.Exit(1)
	}
	e.hostUtil = hostutil.NewHostUtil()

	e.configMapManager = klconfigmap.NewSimpleConfigMapManager(e.kubeClient)

	e.volumeManager = volumemanager.NewVolumeManager(
		true,
		types.NodeName(e.nodeName),
		e.podManager,
		e.statusManager,
		e.kubeClient,
		e.volumePluginMgr,
		e.containerRuntime,
		e.mounter,
		e.hostUtil,
		e.getPodsDir(),
		record.NewEventRecorder(),
		false,
		false,
		volumepathhandler.NewBlockVolumePathHandler(),
	)
	go e.volumeManager.Run(edgedutil.NewSourcesReady(), utilwait.NeverStop)
	go utilwait.Until(e.syncNodeStatus, e.nodeStatusUpdateFrequency, utilwait.NeverStop)

	e.probeManager = prober.NewManager(e.statusManager, e.livenessManager, e.startupManager, e.runner, kubecontainer.NewRefManager(), record.NewEventRecorder())
	e.pleg = pleg.NewGenericPLEG(e.containerRuntime, plegChannelCapacity, plegRelistPeriod, e.podCache, clock.RealClock{})
	e.statusManager.Start()
	e.pleg.Start()

	e.podAddWorkerRun(e.concurrentConsumers)
	e.podRemoveWorkerRun(e.concurrentConsumers)

	housekeepingTicker := time.NewTicker(housekeepingPeriod)
	syncWorkQueueCh := time.NewTicker(syncWorkQueuePeriod)
	e.probeManager.Start()
	go e.syncLoopIteration(e.pleg.Watch(), housekeepingTicker.C, syncWorkQueueCh.C)
	go e.server.ListenAndServe(e, e.resourceAnalyzer, true)

	e.imageGCManager.Start()
	e.StartGarbageCollection()

	e.pluginManager = pluginmanager.NewPluginManager(
		e.getPluginsRegistrationDir(), /* sockDir */
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

func (e *edged) cgroupRoots() []string {
	var cgroupRoots []string

	cgroupRoots = append(cgroupRoots, cm.NodeAllocatableRoot(edgedconfig.Config.CgroupRoot, edgedconfig.Config.CGroupDriver))
	kubeletCgroup, err := cm.GetKubeletContainer("")
	if err != nil {
		klog.Warningf("failed to get the edged's cgroup: %v. Edged system container metrics may be missing.", err)
	} else if kubeletCgroup != "" {
		cgroupRoots = append(cgroupRoots, kubeletCgroup)
	}

	runtimeCgroup, err := cm.GetRuntimeContainer(e.containerRuntimeName, "")
	if err != nil {
		klog.Warningf("failed to get the container runtime's cgroup: %v. Runtime system container metrics may be missing.", err)
	} else if runtimeCgroup != "" {
		// RuntimeCgroups is optional, so ignore if it isn't specified
		cgroupRoots = append(cgroupRoots, runtimeCgroup)
	}

	return cgroupRoots
}

//newEdged creates new edged object and initialises it
func newEdged(enable bool) (*edged, error) {
	backoff := flowcontrol.NewBackOff(backOffPeriod, MaxContainerBackOff)

	podManager := podmanager.NewPodManager()
	policy := images.ImageGCPolicy{
		HighThresholdPercent: int(edgedconfig.Config.ImageGCHighThreshold),
		LowThresholdPercent:  int(edgedconfig.Config.ImageGCLowThreshold),
		MinAge:               minAge,
	}
	// build new object to match interface
	recorder := record.NewEventRecorder()

	metaClient := client.New()

	ed := &edged{
		nodeName:                  edgedconfig.Config.HostnameOverride,
		interfaceName:             edgedconfig.Config.InterfaceName,
		namespace:                 edgedconfig.Config.RegisterNodeNamespace,
		containerRuntimeName:      edgedconfig.Config.RuntimeType,
		gpuPluginEnabled:          edgedconfig.Config.GPUPluginEnabled,
		cgroupDriver:              edgedconfig.Config.CGroupDriver,
		concurrentConsumers:       edgedconfig.Config.ConcurrentConsumers,
		podManager:                podManager,
		podAdditionQueue:          workqueue.New(),
		podCache:                  kubecontainer.NewCache(),
		podAdditionBackoff:        backoff,
		podDeletionQueue:          workqueue.New(),
		podDeletionBackoff:        backoff,
		metaClient:                metaClient,
		kubeClient:                fakekube.NewSimpleClientset(metaClient),
		nodeStatusUpdateFrequency: time.Duration(edgedconfig.Config.NodeStatusUpdateFrequency) * time.Second,
		mounter:                   mount.New(""),
		uid:                       types.UID("38796d14-1df3-11e8-8e5a-286ed488f209"),
		version:                   fmt.Sprintf("%s-kubeedge-%s", constants.CurrentSupportK8sVersion, version.Get()),
		rootDirectory:             DefaultRootDir,
		secretStore:               cache.NewStore(cache.MetaNamespaceKeyFunc),
		configMapStore:            cache.NewStore(cache.MetaNamespaceKeyFunc),
		workQueue:                 queue.NewBasicWorkQueue(clock.RealClock{}),
		nodeIP:                    net.ParseIP(edgedconfig.Config.NodeIP),
		recorder:                  recorder,
		enable:                    enable,
	}

	err := ed.makePodDir()
	if err != nil {
		klog.Errorf("create pod dir [%s] failed: %v", ed.getPodsDir(), err)
		os.Exit(1)
	}

	ed.livenessManager = proberesults.NewManager()
	ed.startupManager = proberesults.NewManager()

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
		MaxPerPodContainer: int(edgedconfig.Config.MaximumDeadContainersPerPod),
	}

	//create and start the docker shim running as a grpc server
	if edgedconfig.Config.RemoteRuntimeEndpoint == DockerShimEndpoint ||
		edgedconfig.Config.RemoteRuntimeEndpoint == DockerShimEndpointDeprecated {
		streamingConfig := &streaming.Config{}
		DockerClientConfig := &dockershim.ClientConfig{
			DockerEndpoint:            edgedconfig.Config.DockerAddress,
			ImagePullProgressDeadline: time.Duration(edgedconfig.Config.ImagePullProgressDeadline) * time.Second,
			EnableSleep:               true,
			WithTraceDisabled:         true,
		}

		pluginConfigs := dockershim.NetworkPluginSettings{
			HairpinMode:        kubeletinternalconfig.HairpinMode(HairpinMode),
			NonMasqueradeCIDR:  NonMasqueradeCIDR,
			PluginName:         edgedconfig.Config.NetworkPluginName,
			PluginBinDirString: edgedconfig.Config.CNIBinDir,
			PluginConfDir:      edgedconfig.Config.CNIConfDir,
			PluginCacheDir:     edgedconfig.Config.CNICacheDir,
			MTU:                int(edgedconfig.Config.NetworkPluginMTU),
		}

		redirectContainerStream := redirectContainerStream
		cgroupDriver := ed.cgroupDriver

		ds, err := dockershim.NewDockerService(DockerClientConfig,
			edgedconfig.Config.PodSandboxImage,
			streamingConfig,
			&pluginConfigs,
			cgroupName,
			cgroupDriver,
			DockershimRootDir,
			redirectContainerStream)

		if err != nil {
			return nil, err
		}

		klog.Infof("RemoteRuntimeEndpoint: %q, remoteImageEndpoint: %q",
			edgedconfig.Config.RemoteRuntimeEndpoint, edgedconfig.Config.RemoteImageEndpoint)

		klog.Info("Starting the GRPC server for the docker CRI shim.")
		server := dockerremote.NewDockerServer(edgedconfig.Config.RemoteRuntimeEndpoint, ds)
		if err := server.Start(); err != nil {
			return nil, err
		}
		// Create dockerLegacyService when the logging driver is not supported.
		supported, err := ds.IsCRISupportedLogDriver()
		if err != nil {
			return nil, err
		}
		if !supported {
			ed.dockerLegacyService = ds
		}
	}
	ed.clusterDNS = convertStrToIP(edgedconfig.Config.ClusterDNS)
	ed.dnsConfigurer = kubedns.NewConfigurer(recorder,
		nodeRef,
		ed.nodeIP,
		ed.clusterDNS,
		edgedconfig.Config.ClusterDomain,
		ResolvConfDefault)

	containerRefManager := kubecontainer.NewRefManager()
	httpClient := &http.Client{}
	runtimeService, imageService, err := getRuntimeAndImageServices(
		edgedconfig.Config.RemoteRuntimeEndpoint,
		edgedconfig.Config.RemoteImageEndpoint,
		metav1.Duration{
			Duration: time.Duration(edgedconfig.Config.RuntimeRequestTimeout) * time.Minute,
		})
	if err != nil {
		return nil, err
	}
	if ed.os == nil {
		ed.os = kubecontainer.RealOS{}
	}

	ed.clcm, err = clcm.NewContainerLifecycleManager(DefaultRootDir)

	useLegacyCadvisorStats := cadvisor.UsingLegacyCadvisorStats(edgedconfig.Config.RuntimeType, edgedconfig.Config.RemoteRuntimeEndpoint)
	if edgedconfig.Config.EnableMetrics {
		imageFsInfoProvider := cadvisor.NewImageFsInfoProvider(edgedconfig.Config.RuntimeType, edgedconfig.Config.RemoteRuntimeEndpoint)
		cadvisorInterface, err := cadvisor.New(imageFsInfoProvider, ed.rootDirectory, ed.cgroupRoots(), useLegacyCadvisorStats)
		if err != nil {
			return nil, err
		}
		ed.cadvisor = cadvisorInterface

		machineInfo, err := ed.cadvisor.MachineInfo()
		if err != nil {
			return nil, err
		}
		ed.machineInfo = machineInfo
	} else {
		cadvisorInterface, _ := edgecadvisor.New("")
		ed.cadvisor = cadvisorInterface

		var machineInfo cadvisorapi.MachineInfo
		machineInfo.MemoryCapacity = uint64(edgedconfig.Config.EdgedMemoryCapacity)
		ed.machineInfo = &machineInfo
	}

	containerRuntime, err := kuberuntime.NewKubeGenericRuntimeManager(
		recorder,
		ed.livenessManager,
		ed.startupManager,
		"",
		containerRefManager,
		ed.machineInfo,
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

	if edgedconfig.Config.CgroupsPerQOS && edgedconfig.Config.CgroupRoot == "" {
		klog.Info("--cgroups-per-qos enabled, but --cgroup-root was not specified.  defaulting to /")
		edgedconfig.Config.CgroupRoot = "/"
	}

	containerManager, err := cm.NewContainerManager(mount.New(""),
		ed.cadvisor,
		cm.NodeConfig{
			CgroupDriver:                 edgedconfig.Config.CGroupDriver,
			SystemCgroupsName:            edgedconfig.Config.SystemCgroups,
			KubeletCgroupsName:           edgedconfig.Config.EdgeCoreCgroups,
			ContainerRuntime:             edgedconfig.Config.RuntimeType,
			CgroupsPerQOS:                edgedconfig.Config.CgroupsPerQOS,
			KubeletRootDir:               DefaultRootDir,
			ExperimentalCPUManagerPolicy: string(cpumanager.PolicyNone),
			CgroupRoot:                   edgedconfig.Config.CgroupRoot,
		},
		false,
		edgedconfig.Config.DevicePluginEnabled,
		recorder)
	if err != nil {
		return nil, fmt.Errorf("init container manager failed with error: %v", err)
	}

	ed.containerRuntime = containerRuntime
	ed.runner = containerRuntime
	ed.containerManager = containerManager
	ed.runtimeService = runtimeService

	runtimeCache, err := kubecontainer.NewRuntimeCache(ed.containerRuntime)
	if err != nil {
		return nil, err
	}
	ed.runtimeCache = runtimeCache

	ed.resourceAnalyzer = serverstats.NewResourceAnalyzer(ed, edgedconfig.Config.VolumeStatsAggPeriod)

	ed.statusManager = status.NewManager(ed.kubeClient, ed.podManager, ed, ed.metaClient)

	if useLegacyCadvisorStats {
		ed.StatsProvider = stats.NewCadvisorStatsProvider(
			ed.cadvisor,
			ed.resourceAnalyzer,
			ed.podManager,
			ed.runtimeCache,
			ed.containerRuntime,
			ed.statusManager)
	} else {
		ed.StatsProvider = stats.NewCRIStatsProvider(
			ed.cadvisor,
			ed.resourceAnalyzer,
			ed.podManager,
			ed.runtimeCache,
			ed.runtimeService,
			imageService,
			stats.NewLogMetricsService(),
			kubecontainer.RealOS{})
	}

	imageGCManager, err := images.NewImageGCManager(
		ed.containerRuntime,
		statsProvider,
		recorder,
		nodeRef,
		policy,
		edgedconfig.Config.PodSandboxImage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize image manager: %v", err)
	}
	ed.imageGCManager = imageGCManager

	containerGCManager, err := kubecontainer.NewContainerGC(
		containerRuntime,
		containerGCPolicy,
		&containers.KubeSourcesReady{})
	if err != nil {
		return nil, fmt.Errorf("init Container GC Manager failed with error %s", err.Error())
	}
	ed.containerGCManager = containerGCManager
	ed.server = server.NewServer(ed.podManager)

	return ed, nil
}

func (e *edged) initializeModules() error {
	if edgedconfig.Config.EnableMetrics {
		// Start resource analyzer
		e.resourceAnalyzer.Start()

		if err := e.cadvisor.Start(); err != nil {
			// Fail kubelet and rely on the babysitter to retry starting kubelet.
			// TODO(random-liu): Add backoff logic in the babysitter
			klog.Fatalf("Failed to start cAdvisor %v", err)
		}

		// trigger on-demand stats collection once so that we have capacity information for ephemeral storage.
		// ignore any errors, since if stats collection is not successful, the container manager will fail to start below.
		e.StatsProvider.GetCgroupStats("/", true)
	}
	// Start container manager.
	node, err := e.initialNode()
	if err != nil {
		klog.Errorf("Failed to initialNode %v", err)
		return err
	}

	// containerManager must start after cAdvisor because it needs filesystem capacity information
	err = e.containerManager.Start(node, e.GetActivePods, edgedutil.NewSourcesReady(), e.statusManager, e.runtimeService)
	if err != nil {
		klog.Errorf("Failed to start container manager, err: %v", err)
		return err
	}

	return nil
}

func (e *edged) StartGarbageCollection() {
	go utilwait.Until(func() {
		err := e.imageGCManager.GarbageCollect()
		if err != nil {
			klog.Errorf("Image garbage collection failed: %v", err)
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
				if err := e.updatePodStatus(pod); err != nil {
					klog.Errorf("update pod %s status error", pod.Name)
					break
				}
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
	beehiveContext.Send(metamanager.MetaManagerModuleName, *info)
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Sync pod stop")
			return
		default:
		}
		result, err := beehiveContext.Receive(e.Name())
		if err != nil {
			klog.Errorf("failed to get pod: %v", err)
			continue
		}

		_, resType, resID, err := util.ParseResourceEdge(result.GetResource(), result.GetOperation())
		if err != nil {
			klog.Errorf("failed to parse the Resource: %v", err)
			continue
		}
		op := result.GetOperation()

		var content []byte

		switch result.Content.(type) {
		case []byte:
			content = result.GetContent().([]byte)
		default:
			content, err = json.Marshal(result.Content)
			if err != nil {
				klog.Errorf("marshal message content failed: %v", err)
				continue
			}
		}
		klog.Infof("result content is %s", result.Content)
		switch resType {
		case model.ResourceTypePod:
			if op == model.ResponseOperation && resID == "" && result.GetSource() == metamanager.MetaManagerModuleName {
				err := e.handlePodListFromMetaManager(content)
				if err != nil {
					klog.Errorf("handle podList failed: %v", err)
					continue
				}
				e.setInitPodReady(true)
			} else if op == model.ResponseOperation && resID == "" && result.GetSource() == EdgeController {
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
				resp := result.NewRespByMessage(&result, res)
				beehiveContext.SendResp(*resp)
			}
		default:
			klog.Errorf("resType is not pod or configmap or secret or volume: esType is %s", resType)
			continue
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
	containerRunningPods, err := e.containerRuntime.GetPods(false)
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
