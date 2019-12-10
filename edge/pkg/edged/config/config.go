package config

import (
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

const (
	//DockerEndpoint gives the default endpoint for docker engine
	DockerEndpoint = "unix:///var/run/docker.sock"

	//RemoteRuntimeEndpoint gives the default endpoint for CRI runtime
	RemoteRuntimeEndpoint = "unix:///var/run/dockershim.sock"

	//RemoteContainerRuntime give Remote container runtime name
	RemoteContainerRuntime = "remote"

	//MinimumEdgedMemoryCapacity gives the minimum default memory (2G) of edge
	MinimumEdgedMemoryCapacity = 2147483647

	//PodSandboxImage gives the default pause container image
	PodSandboxImage = "k8s.gcr.io/pause"

	// ImagePullProgressDeadlineDefault gives the default image pull progress deadline
	ImagePullProgressDeadlineDefault = 60
)

var c Configure
var once sync.Once

type Configure struct {
	NodeName                  string
	NodeNamespace             string
	InterfaceName             string
	MemoryCapacity            int64
	NodeStatusUpdateInterval  time.Duration
	DevicePluginEnabled       bool
	GPUPluginEnabled          bool
	ImageGCHighThreshold      int
	ImageGCLowThreshold       int
	ImagePullProgressDeadline int
	MaxPerPodContainerCount   int
	DockerAddress             string
	RuntimeType               string
	RemoteRuntimeEndpoint     string
	RemoteImageEndpoint       string
	RuntimeRequestTimeout     metav1.Duration
	PodSandboxImage           string
	CgroupDriver              string
	NodeIP                    string
	ClusterDNS                string
	ClusterDomain             string
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		nodeStatusUpdateInterval := config.CONFIG.GetConfigurationByKey("edged.node-status-update-frequency").(int)
		dockerAddress, ok := config.CONFIG.GetConfigurationByKey("edged.docker-address").(string)
		if !ok {
			dockerAddress = DockerEndpoint
		}
		runtimeType, ok := config.CONFIG.GetConfigurationByKey("edged.runtime-type").(string)
		if !ok {
			runtimeType = RemoteContainerRuntime
		}
		cgroupDriver, ok := config.CONFIG.GetConfigurationByKey("edged.cgroup-driver").(string)
		if !ok {
			cgroupDriver = "systemd"
		}
		nodeIP, ok := config.CONFIG.GetConfigurationByKey("edged.node-ip").(string)
		if !ok {
			nodeIP = "127.0.0.1"
		}
		clusterDNS, ok := config.CONFIG.GetConfigurationByKey("edged.cluster-dns").(string)
		if !ok {
			clusterDNS = ""
		}
		clusterDomain, ok := config.CONFIG.GetConfigurationByKey("edged.cluster-domain").(string)
		if !ok {
			clusterDomain = ""
		}
		var memCapacity int64
		//Deal with 32-bit and 64-bit compatibility issues: issue #1070
		switch v := config.CONFIG.GetConfigurationByKey("edged.edged-memory-capacity-bytes").(type) {
		case int:
			memCapacity = int64(v)
		case int64:
			memCapacity = v
		default:
			panic("Invalid type for edged.edged-memory-capacity-bytes, valid types are one of [int,int64].")
		}
		if memCapacity == 0 {
			memCapacity = MinimumEdgedMemoryCapacity
		}
		remoteRuntimeEndpoint := config.CONFIG.GetConfigurationByKey("edged.remote-runtime-endpoint").(string)
		if remoteRuntimeEndpoint == "" {
			remoteRuntimeEndpoint = RemoteRuntimeEndpoint
		}

		remoteImageEndpoint := config.CONFIG.GetConfigurationByKey("edged.remote-image-endpoint").(string)
		// remoteImageEndpoint is same as remoteRuntimeEndpoint if not explicitly specified
		if remoteImageEndpoint == "" {
			remoteImageEndpoint = remoteRuntimeEndpoint
		}

		podSandboxImage := config.CONFIG.GetConfigurationByKey("edged.podsandbox-image").(string)
		if podSandboxImage == "" {
			podSandboxImage = PodSandboxImage
		}
		imagePullProgressDeadline := config.CONFIG.GetConfigurationByKey("edged.image-pull-progress-deadline").(int)
		if imagePullProgressDeadline == 0 {
			imagePullProgressDeadline = ImagePullProgressDeadlineDefault
		}
		runtimeRequestTimeout := config.CONFIG.GetConfigurationByKey("edged.runtime-request-timeout").(int)
		if runtimeRequestTimeout == 0 {
			runtimeRequestTimeout = 2
		}

		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init edged config error")
			os.Exit(1)
		}
		c = Configure{
			NodeName:                  config.CONFIG.GetConfigurationByKey("edged.hostname-override").(string),
			NodeNamespace:             config.CONFIG.GetConfigurationByKey("edged.register-node-namespace").(string),
			InterfaceName:             config.CONFIG.GetConfigurationByKey("edged.interface-name").(string),
			DevicePluginEnabled:       config.CONFIG.GetConfigurationByKey("edged.device-plugin-enabled").(bool),
			GPUPluginEnabled:          config.CONFIG.GetConfigurationByKey("edged.gpu-plugin-enabled").(bool),
			ImageGCHighThreshold:      config.CONFIG.GetConfigurationByKey("edged.image-gc-high-threshold").(int),
			ImageGCLowThreshold:       config.CONFIG.GetConfigurationByKey("edged.image-gc-low-threshold").(int),
			MaxPerPodContainerCount:   config.CONFIG.GetConfigurationByKey("edged.maximum-dead-containers-per-container").(int),
			NodeStatusUpdateInterval:  time.Duration(nodeStatusUpdateInterval) * time.Second,
			DockerAddress:             dockerAddress,
			RuntimeType:               runtimeType,
			CgroupDriver:              cgroupDriver,
			NodeIP:                    nodeIP,
			ClusterDNS:                clusterDNS,
			ClusterDomain:             clusterDomain,
			MemoryCapacity:            memCapacity,
			RemoteRuntimeEndpoint:     remoteRuntimeEndpoint,
			RemoteImageEndpoint:       remoteImageEndpoint,
			PodSandboxImage:           podSandboxImage,
			ImagePullProgressDeadline: imagePullProgressDeadline,
			RuntimeRequestTimeout: metav1.Duration{
				Duration: time.Duration(runtimeRequestTimeout) * time.Minute,
			},
		}
		klog.Infof("init edged config successfully，config info %++v", c)
	})
}
func Get() *Configure {
	return &c
}
