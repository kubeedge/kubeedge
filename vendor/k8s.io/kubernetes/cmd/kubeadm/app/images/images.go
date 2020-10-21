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
*/

package images

import (
	"fmt"

	"k8s.io/klog"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

const extraHyperKubeNote = ` The "useHyperKubeImage" field will be removed from future kubeadm config versions and possibly ignored in future releases.`

// GetGenericImage generates and returns a platform agnostic image (backed by manifest list)
func GetGenericImage(prefix, image, tag string) string {
	return fmt.Sprintf("%s/%s:%s", prefix, image, tag)
}

// GetKubernetesImage generates and returns the image for the components managed in the Kubernetes main repository,
// including the control-plane components and kube-proxy. If specified, the HyperKube image will be used.
func GetKubernetesImage(image string, cfg *kubeadmapi.ClusterConfiguration) string {
	if cfg.UseHyperKubeImage && image != constants.HyperKube {
		klog.Warningf(`WARNING: DEPRECATED use of the "hyperkube" image in place of %q.`+extraHyperKubeNote, image)
		image = constants.HyperKube
	}
	repoPrefix := cfg.GetControlPlaneImageRepository()
	kubernetesImageTag := kubeadmutil.KubernetesVersionToImageTag(cfg.KubernetesVersion)
	return GetGenericImage(repoPrefix, image, kubernetesImageTag)
}

// GetDNSImage generates and returns the image for the DNS, that can be CoreDNS or kube-dns.
// Given that kube-dns uses 3 containers, an additional imageName parameter was added
func GetDNSImage(cfg *kubeadmapi.ClusterConfiguration, imageName string) string {
	// DNS uses default image repository by default
	dnsImageRepository := cfg.ImageRepository
	// unless an override is specified
	if cfg.DNS.ImageRepository != "" {
		dnsImageRepository = cfg.DNS.ImageRepository
	}
	// DNS uses an imageTag that corresponds to the DNS version matching the Kubernetes version
	dnsImageTag := constants.GetDNSVersion(cfg.DNS.Type)

	// unless an override is specified
	if cfg.DNS.ImageTag != "" {
		dnsImageTag = cfg.DNS.ImageTag
	}
	return GetGenericImage(dnsImageRepository, imageName, dnsImageTag)
}

// GetEtcdImage generates and returns the image for etcd
func GetEtcdImage(cfg *kubeadmapi.ClusterConfiguration) string {
	// Etcd uses default image repository by default
	etcdImageRepository := cfg.ImageRepository
	// unless an override is specified
	if cfg.Etcd.Local != nil && cfg.Etcd.Local.ImageRepository != "" {
		etcdImageRepository = cfg.Etcd.Local.ImageRepository
	}
	// Etcd uses an imageTag that corresponds to the etcd version matching the Kubernetes version
	etcdImageTag := constants.DefaultEtcdVersion
	etcdVersion, warning, err := constants.EtcdSupportedVersion(constants.SupportedEtcdVersion, cfg.KubernetesVersion)
	if err == nil {
		etcdImageTag = etcdVersion.String()
	}
	if warning != nil {
		klog.Warningln(warning)
	}
	// unless an override is specified
	if cfg.Etcd.Local != nil && cfg.Etcd.Local.ImageTag != "" {
		etcdImageTag = cfg.Etcd.Local.ImageTag
	}
	return GetGenericImage(etcdImageRepository, constants.Etcd, etcdImageTag)
}

// GetControlPlaneImages returns a list of container images kubeadm expects to use on a control plane node
func GetControlPlaneImages(cfg *kubeadmapi.ClusterConfiguration) []string {
	imgs := []string{}

	// start with core kubernetes images
	if cfg.UseHyperKubeImage {
		klog.Warningln(`WARNING: DEPRECATED use of the "hyperkube" image for the Kubernetes control plane.` + extraHyperKubeNote)
		imgs = append(imgs, GetKubernetesImage(constants.HyperKube, cfg))
	} else {
		imgs = append(imgs, GetKubernetesImage(constants.KubeAPIServer, cfg))
		imgs = append(imgs, GetKubernetesImage(constants.KubeControllerManager, cfg))
		imgs = append(imgs, GetKubernetesImage(constants.KubeScheduler, cfg))
		imgs = append(imgs, GetKubernetesImage(constants.KubeProxy, cfg))
	}

	// pause is not available on the ci image repository so use the default image repository.
	imgs = append(imgs, GetPauseImage(cfg))

	// if etcd is not external then add the image as it will be required
	if cfg.Etcd.Local != nil {
		imgs = append(imgs, GetEtcdImage(cfg))
	}

	// Append the appropriate DNS images
	if cfg.DNS.Type == kubeadmapi.CoreDNS {
		imgs = append(imgs, GetDNSImage(cfg, constants.CoreDNSImageName))
	} else {
		imgs = append(imgs, GetDNSImage(cfg, constants.KubeDNSKubeDNSImageName))
		imgs = append(imgs, GetDNSImage(cfg, constants.KubeDNSSidecarImageName))
		imgs = append(imgs, GetDNSImage(cfg, constants.KubeDNSDnsMasqNannyImageName))
	}

	return imgs
}
