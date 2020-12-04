/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package componentconfigs

import (
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/version"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubeletconfig "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"
	utilsexec "k8s.io/utils/exec"
	utilpointer "k8s.io/utils/pointer"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmapiv1beta2 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/features"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

const (
	// KubeletGroup is a pointer to the used API group name for the kubelet config
	KubeletGroup = kubeletconfig.GroupName

	// kubeletReadOnlyPort specifies the default insecure http server port
	// 0 will disable insecure http server.
	kubeletReadOnlyPort int32 = 0

	// kubeletRotateCertificates specifies the default value to enable certificate rotation
	kubeletRotateCertificates = true

	// kubeletAuthenticationAnonymousEnabled specifies the default value to disable anonymous access
	kubeletAuthenticationAnonymousEnabled = false

	// kubeletAuthenticationWebhookEnabled set the default value to enable authentication webhook
	kubeletAuthenticationWebhookEnabled = true

	// kubeletHealthzBindAddress specifies the default healthz bind address
	kubeletHealthzBindAddress = "127.0.0.1"

	// kubeletSystemdResolverConfig specifies the default resolver config when systemd service is active
	kubeletSystemdResolverConfig = "/run/systemd/resolve/resolv.conf"
)

// kubeletHandler is the handler instance for the kubelet component config
var kubeletHandler = handler{
	GroupVersion: kubeletconfig.SchemeGroupVersion,
	AddToScheme:  kubeletconfig.AddToScheme,
	CreateEmpty: func() kubeadmapi.ComponentConfig {
		return &kubeletConfig{
			configBase: configBase{
				GroupVersion: kubeletconfig.SchemeGroupVersion,
			},
		}
	},
	fromCluster: kubeletConfigFromCluster,
}

func kubeletConfigFromCluster(h *handler, clientset clientset.Interface, clusterCfg *kubeadmapi.ClusterConfiguration) (kubeadmapi.ComponentConfig, error) {
	// Read the ConfigMap from the cluster based on what version the kubelet is
	k8sVersion, err := version.ParseGeneric(clusterCfg.KubernetesVersion)
	if err != nil {
		return nil, err
	}

	configMapName := constants.GetKubeletConfigMapName(k8sVersion)
	return h.fromConfigMap(clientset, configMapName, constants.KubeletBaseConfigurationConfigMapKey, true)
}

// kubeletConfig implements the kubeadmapi.ComponentConfig interface for kubelet
type kubeletConfig struct {
	configBase
	config kubeletconfig.KubeletConfiguration
}

func (kc *kubeletConfig) DeepCopy() kubeadmapi.ComponentConfig {
	result := &kubeletConfig{}
	kc.configBase.DeepCopyInto(&result.configBase)
	kc.config.DeepCopyInto(&result.config)
	return result
}

func (kc *kubeletConfig) Marshal() ([]byte, error) {
	return kc.configBase.Marshal(&kc.config)
}

func (kc *kubeletConfig) Unmarshal(docmap kubeadmapi.DocumentMap) error {
	return kc.configBase.Unmarshal(docmap, &kc.config)
}

func (kc *kubeletConfig) Default(cfg *kubeadmapi.ClusterConfiguration, _ *kubeadmapi.APIEndpoint, nodeRegOpts *kubeadmapi.NodeRegistrationOptions) {
	const kind = "KubeletConfiguration"

	if kc.config.FeatureGates == nil {
		kc.config.FeatureGates = map[string]bool{}
	}

	// TODO: The following code should be removed after dual-stack is GA.
	// Note: The user still retains the ability to explicitly set feature-gates and that value will overwrite this base value.
	if enabled, present := cfg.FeatureGates[features.IPv6DualStack]; present {
		kc.config.FeatureGates[features.IPv6DualStack] = enabled
	}

	if kc.config.StaticPodPath == "" {
		kc.config.StaticPodPath = kubeadmapiv1beta2.DefaultManifestsDir
	} else if kc.config.StaticPodPath != kubeadmapiv1beta2.DefaultManifestsDir {
		warnDefaultComponentConfigValue(kind, "staticPodPath", kubeadmapiv1beta2.DefaultManifestsDir, kc.config.StaticPodPath)
	}

	clusterDNS := ""
	dnsIP, err := constants.GetDNSIP(cfg.Networking.ServiceSubnet, features.Enabled(cfg.FeatureGates, features.IPv6DualStack))
	if err != nil {
		clusterDNS = kubeadmapiv1beta2.DefaultClusterDNSIP
	} else {
		clusterDNS = dnsIP.String()
	}

	if kc.config.ClusterDNS == nil {
		kc.config.ClusterDNS = []string{clusterDNS}
	} else if len(kc.config.ClusterDNS) != 1 || kc.config.ClusterDNS[0] != clusterDNS {
		warnDefaultComponentConfigValue(kind, "clusterDNS", []string{clusterDNS}, kc.config.ClusterDNS)
	}

	if kc.config.ClusterDomain == "" {
		kc.config.ClusterDomain = cfg.Networking.DNSDomain
	} else if cfg.Networking.DNSDomain != "" && kc.config.ClusterDomain != cfg.Networking.DNSDomain {
		warnDefaultComponentConfigValue(kind, "clusterDomain", cfg.Networking.DNSDomain, kc.config.ClusterDomain)
	}

	// Require all clients to the kubelet API to have client certs signed by the cluster CA
	clientCAFile := filepath.Join(cfg.CertificatesDir, constants.CACertName)
	if kc.config.Authentication.X509.ClientCAFile == "" {
		kc.config.Authentication.X509.ClientCAFile = clientCAFile
	} else if kc.config.Authentication.X509.ClientCAFile != clientCAFile {
		warnDefaultComponentConfigValue(kind, "authentication.x509.clientCAFile", clientCAFile, kc.config.Authentication.X509.ClientCAFile)
	}

	if kc.config.Authentication.Anonymous.Enabled == nil {
		kc.config.Authentication.Anonymous.Enabled = utilpointer.BoolPtr(kubeletAuthenticationAnonymousEnabled)
	} else if *kc.config.Authentication.Anonymous.Enabled != kubeletAuthenticationAnonymousEnabled {
		warnDefaultComponentConfigValue(kind, "authentication.anonymous.enabled", kubeletAuthenticationAnonymousEnabled, *kc.config.Authentication.Anonymous.Enabled)
	}

	// On every client request to the kubelet API, execute a webhook (SubjectAccessReview request) to the API server
	// and ask it whether the client is authorized to access the kubelet API
	if kc.config.Authorization.Mode == "" {
		kc.config.Authorization.Mode = kubeletconfig.KubeletAuthorizationModeWebhook
	} else if kc.config.Authorization.Mode != kubeletconfig.KubeletAuthorizationModeWebhook {
		warnDefaultComponentConfigValue(kind, "authorization.mode", kubeletconfig.KubeletAuthorizationModeWebhook, kc.config.Authorization.Mode)
	}

	// Let clients using other authentication methods like ServiceAccount tokens also access the kubelet API
	if kc.config.Authentication.Webhook.Enabled == nil {
		kc.config.Authentication.Webhook.Enabled = utilpointer.BoolPtr(kubeletAuthenticationWebhookEnabled)
	} else if *kc.config.Authentication.Webhook.Enabled != kubeletAuthenticationWebhookEnabled {
		warnDefaultComponentConfigValue(kind, "authentication.webhook.enabled", kubeletAuthenticationWebhookEnabled, *kc.config.Authentication.Webhook.Enabled)
	}

	// Serve a /healthz webserver on localhost:10248 that kubeadm can talk to
	if kc.config.HealthzBindAddress == "" {
		kc.config.HealthzBindAddress = kubeletHealthzBindAddress
	} else if kc.config.HealthzBindAddress != kubeletHealthzBindAddress {
		warnDefaultComponentConfigValue(kind, "healthzBindAddress", kubeletHealthzBindAddress, kc.config.HealthzBindAddress)
	}

	if kc.config.HealthzPort == nil {
		kc.config.HealthzPort = utilpointer.Int32Ptr(constants.KubeletHealthzPort)
	} else if *kc.config.HealthzPort != constants.KubeletHealthzPort {
		warnDefaultComponentConfigValue(kind, "healthzPort", constants.KubeletHealthzPort, *kc.config.HealthzPort)
	}

	if kc.config.ReadOnlyPort != kubeletReadOnlyPort {
		warnDefaultComponentConfigValue(kind, "readOnlyPort", kubeletReadOnlyPort, kc.config.ReadOnlyPort)
	}

	// We cannot show a warning for RotateCertificates==false and we must hardcode it to true.
	// There is no way to determine if the user has set this or not, given the field is a non-pointer.
	kc.config.RotateCertificates = kubeletRotateCertificates

	// TODO: Conditionally set CgroupDriver to either `systemd` or `cgroupfs` for CRI other than Docker
	if nodeRegOpts.CRISocket == constants.DefaultDockerCRISocket {
		driver, err := kubeadmutil.GetCgroupDriverDocker(utilsexec.New())
		if err != nil {
			klog.Warningf("cannot automatically set CgroupDriver when starting the Kubelet: %v", err)
		} else {
			// if we can parse the right cgroup driver from docker info,
			// we should always override CgroupDriver here no matter user specifies this value explicitly or not
			if kc.config.CgroupDriver != "" && kc.config.CgroupDriver != driver {
				klog.Warningf("detected %q as the Docker cgroup driver, the provided value %q in %q will be overrided", driver, kc.config.CgroupDriver, kind)
			}
			kc.config.CgroupDriver = driver
		}
	}

	ok, err := isServiceActive("systemd-resolved")
	if err != nil {
		klog.Warningf("cannot determine if systemd-resolved is active: %v", err)
	}
	if ok {
		if kc.config.ResolverConfig == "" {
			kc.config.ResolverConfig = kubeletSystemdResolverConfig
		} else {
			if kc.config.ResolverConfig != kubeletSystemdResolverConfig {
				warnDefaultComponentConfigValue(kind, "resolvConf", kubeletSystemdResolverConfig, kc.config.ResolverConfig)
			}
		}
	}
}

// isServiceActive checks whether the given service exists and is running
func isServiceActive(name string) (bool, error) {
	initSystem, err := initsystem.GetInitSystem()
	if err != nil {
		return false, err
	}
	return initSystem.ServiceIsActive(name), nil
}
