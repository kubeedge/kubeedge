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
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/kubelet_pods.go"
and made some variant
*/

package edged

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	utilfile "k8s.io/kubernetes/pkg/util/file"
	"k8s.io/kubernetes/pkg/volume/util"
	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"
	"k8s.io/kubernetes/pkg/volume/validation"
)

const (
	etcHostsPath = "/etc/hosts"
)

// GetActivePods returns non-terminal pods
func (e *edged) GetActivePods() []*v1.Pod {
	allPods := e.podManager.GetPods()
	activePods := e.filterOutTerminatedPods(allPods)
	return activePods
}

// filterOutTerminatedPods returns the given pods which the status manager
// does not consider failed or succeeded.
func (e *edged) filterOutTerminatedPods(pods []*v1.Pod) []*v1.Pod {
	var filteredPods []*v1.Pod
	for _, p := range pods {
		if e.podIsTerminated(p) {
			continue
		}
		filteredPods = append(filteredPods, p)
	}
	return filteredPods
}

// truncatePodHostnameIfNeeded truncates the pod hostname if it's longer than 63 chars.
func truncatePodHostnameIfNeeded(podName, hostname string) (string, error) {
	// Cap hostname at 63 chars (specification is 64bytes which is 63 chars and the null terminating char).
	const hostnameMaxLen = 63
	if len(hostname) <= hostnameMaxLen {
		return hostname, nil
	}
	truncated := hostname[:hostnameMaxLen]
	log.LOGGER.Errorf("hostname for pod:%q was longer than %d. Truncated hostname to :%q", podName, hostnameMaxLen, truncated)
	// hostname should not end with '-' or '.'
	truncated = strings.TrimRight(truncated, "-.")
	if len(truncated) == 0 {
		// This should never happen.
		return "", fmt.Errorf("hostname for pod %q was invalid: %q", podName, hostname)
	}
	return truncated, nil
}

// GeneratePodHostNameAndDomain creates a hostname and domain name for a pod,
// given that pod's spec and annotations or returns an error.
func (e *edged) GeneratePodHostNameAndDomain(pod *v1.Pod) (string, string, error) {
	// TODO(vmarmol): Handle better.
	clusterDomain := "cluster"

	hostname := pod.Name
	if len(pod.Spec.Hostname) > 0 {
		hostname = pod.Spec.Hostname
	}

	hostname, err := truncatePodHostnameIfNeeded(pod.Name, hostname)
	if err != nil {
		return "", "", err
	}

	hostDomain := ""
	if len(pod.Spec.Subdomain) > 0 {
		hostDomain = fmt.Sprintf("%s.%s.svc.%s", pod.Spec.Subdomain, pod.Namespace, clusterDomain)
	}

	return hostname, hostDomain, nil
}

// Get a list of pods that have data directories.
func (e *edged) listPodsFromDisk() ([]types.UID, error) {
	podInfos, err := ioutil.ReadDir(e.getPodsDir())
	if err != nil {
		return nil, err
	}
	pods := []types.UID{}
	for i := range podInfos {
		if podInfos[i].IsDir() {
			pods = append(pods, types.UID(podInfos[i].Name()))
		}
	}
	return pods, nil
}

// hasHostNamespace returns true if hostIPC, hostNetwork, or hostPID are set to true.
func hasHostNamespace(pod *v1.Pod) bool {
	if pod.Spec.SecurityContext == nil {
		return false
	}
	return pod.Spec.HostIPC || pod.Spec.HostNetwork || pod.Spec.HostPID
}

// hasHostVolume returns true if the pod spec has a HostPath volume.
func hasHostVolume(pod *v1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		if v.HostPath != nil {
			return true
		}
	}
	return false
}

// hasNonNamespacedCapability returns true if MKNOD, SYS_TIME, or SYS_MODULE is requested for any container.
func hasNonNamespacedCapability(pod *v1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext != nil && c.SecurityContext.Capabilities != nil {
			for _, cap := range c.SecurityContext.Capabilities.Add {
				if cap == "MKNOD" || cap == "SYS_TIME" || cap == "SYS_MODULE" {
					return true
				}
			}
		}
	}

	return false
}

// HasPrivilegedContainer returns true if any of the containers in the pod are privileged.
func hasPrivilegedContainer(pod *v1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext != nil &&
			c.SecurityContext.Privileged != nil &&
			*c.SecurityContext.Privileged {
			return true
		}
	}
	return false
}

// enableHostUserNamespace determines if the host user namespace should be used by the container runtime.
// Returns true if the pod is using a host pid, pic, or network namespace, the pod is using a non-namespaced
// capability, the pod contains a privileged container, or the pod has a host path volume.
//
// NOTE: when if a container shares any namespace with another container it must also share the user namespace
// or it will not have the correct capabilities in the namespace.  This means that host user namespace
// is enabled per pod, not per container.
func (e *edged) enableHostUserNamespace(pod *v1.Pod) bool {
	if hasPrivilegedContainer(pod) || hasHostNamespace(pod) ||
		hasHostVolume(pod) || hasNonNamespacedCapability(pod) {
		return true
	}
	return false
}

// podIsTerminated returns true if pod is in the terminated state ("Failed" or "Succeeded").
func (e *edged) podIsTerminated(pod *v1.Pod) bool {
	// Check the cached pod status which was set after the last sync.
	status, ok := e.statusManager.GetPodStatus(pod.UID)
	if !ok {
		// If there is no cached status, use the status from the
		// apiserver. This is useful if kubelet has recently been
		// restarted.
		status = pod.Status
	}

	return status.Phase == v1.PodFailed || status.Phase == v1.PodSucceeded || (pod.DeletionTimestamp != nil && notRunning(status.ContainerStatuses))
}

// makePodDataDirs creates the dirs for the pod datas.
func (e *edged) makePodDataDirs(pod *v1.Pod) error {
	uid := pod.UID
	if err := os.MkdirAll(e.getPodDir(uid), 0750); err != nil && !os.IsExist(err) {
		return err
	}
	if err := os.MkdirAll(e.getPodVolumesDir(uid), 0750); err != nil && !os.IsExist(err) {
		return err
	}
	if err := os.MkdirAll(e.getPodPluginsDir(uid), 0750); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (e *edged) makePodDir() error {
	if err := os.MkdirAll(e.getPodsDir(), 0750); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// notRunning returns true if every status is terminated or waiting, or the status list
// is empty.
func notRunning(statuses []v1.ContainerStatus) bool {
	for _, status := range statuses {
		if status.State.Terminated == nil && status.State.Waiting == nil {
			return false
		}
	}
	return true
}

func (e *edged) GenerateContainerOptions(pod *v1.Pod) (*kubecontainer.RunContainerOptions, error) {
	opts := kubecontainer.RunContainerOptions{}
	hostname, hostDomainName, err := e.GeneratePodHostNameAndDomain(pod)
	if err != nil {
		return nil, err
	}
	podName := util.GetUniquePodName(pod)
	volumes := e.volumeManager.GetMountedVolumesForPod(podName)
	for _, container := range pod.Spec.Containers {
		mounts, err := makeMounts(pod, e.getPodDir(pod.UID), &container, hostname, hostDomainName, pod.Status.PodIP, volumes)
		if err != nil {
			return nil, err
		}
		opts.Mounts = append(opts.Mounts, mounts...)
	}

	return &opts, nil
}

// makeMounts determines the mount points for the given container.
func makeMounts(pod *v1.Pod, podDir string, container *v1.Container, hostName, hostDomain, podIP string, podVolumes kubecontainer.VolumeMap) ([]kubecontainer.Mount, error) {
	// Kubernetes only mounts on /etc/hosts if:
	// - container is not an infrastructure (pause) container
	// - container is not already mounting on /etc/hosts
	// - OS is not Windows
	// Kubernetes will not mount /etc/hosts if:
	// - when the Pod sandbox is being created, its IP is still unknown. Hence, PodIP will not have been set.
	mountEtcHostsFile := len(podIP) > 0 && runtime.GOOS != "windows"
	log.LOGGER.Infof("container: %v/%v/%v podIP: %q creating hosts mount: %v", pod.Namespace, pod.Name, container.Name, podIP, mountEtcHostsFile)
	mounts := []kubecontainer.Mount{}
	for _, mount := range container.VolumeMounts {
		// do not mount /etc/hosts if container is already mounting on the path
		mountEtcHostsFile = mountEtcHostsFile && (mount.MountPath != etcHostsPath)
		vol, ok := podVolumes[mount.Name]
		if !ok || vol.Mounter == nil {
			log.LOGGER.Errorf("Mount cannot be satisfied for container %q, because the volume is missing or the volume mounter is nil: %+v", container.Name, mount)
			return nil, fmt.Errorf("cannot find volume %q to mount into container %q", mount.Name, container.Name)
		}

		relabelVolume := false
		// If the volume supports SELinux and it has not been
		// relabeled already and it is not a read-only volume,
		// relabel it and mark it as labeled
		if vol.Mounter.GetAttributes().Managed && vol.Mounter.GetAttributes().SupportsSELinux && !vol.SELinuxLabeled {
			vol.SELinuxLabeled = true
			relabelVolume = true
		}
		hostPath, err := util.GetPath(vol.Mounter)
		if err != nil {
			return nil, err
		}
		if mount.SubPath != "" {
			if filepath.IsAbs(mount.SubPath) {
				return nil, fmt.Errorf("error SubPath `%s` must not be an absolute path", mount.SubPath)
			}

			err = validation.ValidatePathNoBacksteps(mount.SubPath)
			if err != nil {
				return nil, fmt.Errorf("unable to provision SubPath `%s`: %v", mount.SubPath, err)
			}

			fileinfo, err := os.Lstat(hostPath)
			if err != nil {
				return nil, err
			}
			perm := fileinfo.Mode()

			hostPath = filepath.Join(hostPath, mount.SubPath)

			if subPathExists, err := utilfile.FileOrSymlinkExists(hostPath); err != nil {
				log.LOGGER.Errorf("Could not determine if subPath %s exists; will not attempt to change its permissions", hostPath)
			} else if !subPathExists {
				// Create the sub path now because if it's auto-created later when referenced, it may have an
				// incorrect ownership and mode. For example, the sub path directory must have at least g+rwx
				// when the pod specifies an fsGroup, and if the directory is not created here, Docker will
				// later auto-create it with the incorrect mode 0750
				if err := os.MkdirAll(hostPath, perm); err != nil {
					log.LOGGER.Errorf("failed to mkdir:%s", hostPath)
					return nil, err
				}

				// chmod the sub path because umask may have prevented us from making the sub path with the same
				// permissions as the mounter path
				if err := os.Chmod(hostPath, perm); err != nil {
					return nil, err
				}
			}
		}

		// Docker Volume Mounts fail on Windows if it is not of the form C:/
		containerPath := mount.MountPath
		if runtime.GOOS == "windows" {
			if (strings.HasPrefix(hostPath, "/") || strings.HasPrefix(hostPath, "\\")) && !strings.Contains(hostPath, ":") {
				hostPath = "c:" + hostPath
			}
		}
		if !filepath.IsAbs(containerPath) {
			containerPath = makeAbsolutePath(runtime.GOOS, containerPath)
		}

		// Extend the path according to extend type of mount volume, by appending the  pod metadata to the path.
		// TODO: this logic is added by Huawei, make sure what this for and remove it
		// extendVolumePath := volumehelper.GetExtendVolumePath(pod, container, mount.ExtendPathMode)
		// if extendVolumePath != "" {
		// 	hostPath = filepath.Join(hostPath, extendVolumePath)
		// }
		propagation, err := translateMountPropagation(mount.MountPropagation)
		if err != nil {
			return nil, err
		}
		log.LOGGER.Infof("Pod %q container %q mount %q has propagation %q", format.Pod(pod), container.Name, mount.Name, propagation)

		mounts = append(mounts, kubecontainer.Mount{
			Name:           mount.Name,
			ContainerPath:  containerPath,
			HostPath:       hostPath,
			ReadOnly:       mount.ReadOnly,
			SELinuxRelabel: relabelVolume,
			Propagation:    propagation,
		})
	}
	if mountEtcHostsFile {
		hostAliases := pod.Spec.HostAliases
		hostsMount, err := makeHostsMount(podDir, podIP, hostName, hostDomain, hostAliases, pod.Spec.HostNetwork)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, *hostsMount)
	}
	return mounts, nil
}

func makeAbsolutePath(goos, path string) string {
	if goos != "windows" {
		return "/" + path
	}
	// These are all for windows
	// If there is a colon, give up.
	if strings.Contains(path, ":") {
		return path
	}
	// If there is a slash, but no drive, add 'c:'
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return "c:" + path
	}
	// Otherwise, add 'c:\'
	return "c:\\" + path
}

// translateMountPropagation transforms v1.MountPropagationMode to
// runtimeapi.MountPropagation.
func translateMountPropagation(mountMode *v1.MountPropagationMode) (runtimeapi.MountPropagation, error) {
	if !utilfeature.DefaultFeatureGate.Enabled(features.MountPropagation) {
		// mount propagation is disabled, use private as in the old versions
		return runtimeapi.MountPropagation_PROPAGATION_PRIVATE, nil
	}
	switch {
	case mountMode == nil:
		// HostToContainer is the default
		return runtimeapi.MountPropagation_PROPAGATION_HOST_TO_CONTAINER, nil
	case *mountMode == v1.MountPropagationHostToContainer:
		return runtimeapi.MountPropagation_PROPAGATION_HOST_TO_CONTAINER, nil
	case *mountMode == v1.MountPropagationBidirectional:
		return runtimeapi.MountPropagation_PROPAGATION_BIDIRECTIONAL, nil
	default:
		return 0, fmt.Errorf("invalid MountPropagation mode: %q", mountMode)
	}
}

// makeHostsMount makes the mountpoint for the hosts file that the containers
// in a pod are injected with.
func makeHostsMount(podDir, podIP, hostName, hostDomainName string, hostAliases []v1.HostAlias, useHostNetwork bool) (*kubecontainer.Mount, error) {
	hostsFilePath := path.Join(podDir, "etc-hosts")
	if err := ensureHostsFile(hostsFilePath, podIP, hostName, hostDomainName, hostAliases, useHostNetwork); err != nil {
		return nil, err
	}
	return &kubecontainer.Mount{
		Name:           "k8s-managed-etc-hosts",
		ContainerPath:  etcHostsPath,
		HostPath:       hostsFilePath,
		ReadOnly:       false,
		SELinuxRelabel: true,
	}, nil
}

// ensureHostsFile ensures that the given host file has an up-to-date ip, host
// name, and domain name.
func ensureHostsFile(fileName, hostIP, hostName, hostDomainName string, hostAliases []v1.HostAlias, useHostNetwork bool) error {
	var hostsFileContent []byte
	var err error

	if useHostNetwork {
		// if Pod is using host network, read hosts file from the node's filesystem.
		// `etcHostsPath` references the location of the hosts file on the node.
		// `/etc/hosts` for *nix systems.
		hostsFileContent, err = nodeHostsFileContent(etcHostsPath, hostAliases)
		if err != nil {
			return err
		}
	} else {
		// if Pod is not using host network, create a managed hosts file with Pod IP and other information.
		hostsFileContent = managedHostsFileContent(hostIP, hostName, hostDomainName, hostAliases)
	}

	return ioutil.WriteFile(fileName, hostsFileContent, 0644)
}

// nodeHostsFileContent reads the content of node's hosts file.
func nodeHostsFileContent(hostsFilePath string, hostAliases []v1.HostAlias) ([]byte, error) {
	hostsFileContent, err := ioutil.ReadFile(hostsFilePath)
	if err != nil {
		return nil, err
	}
	hostsFileContent = append(hostsFileContent, hostsEntriesFromHostAliases(hostAliases)...)
	return hostsFileContent, nil
}

func hostsEntriesFromHostAliases(hostAliases []v1.HostAlias) []byte {
	if len(hostAliases) == 0 {
		return []byte{}
	}

	var buffer bytes.Buffer
	buffer.WriteString("\n")
	buffer.WriteString("# Entries added by HostAliases.\n")
	// write each IP/hostname pair as an entry into hosts file
	for _, hostAlias := range hostAliases {
		for _, hostname := range hostAlias.Hostnames {
			buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, hostname))
		}
	}
	return buffer.Bytes()
}

// managedHostsFileContent generates the content of the managed etc hosts based on Pod IP and other
// information.
func managedHostsFileContent(hostIP, hostName, hostDomainName string, hostAliases []v1.HostAlias) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("# Kubernetes-managed hosts file.\n")
	buffer.WriteString("127.0.0.1\tlocalhost\n")                      // ipv4 localhost
	buffer.WriteString("::1\tlocalhost ip6-localhost ip6-loopback\n") // ipv6 localhost
	buffer.WriteString("fe00::0\tip6-localnet\n")
	buffer.WriteString("fe00::0\tip6-mcastprefix\n")
	buffer.WriteString("fe00::1\tip6-allnodes\n")
	buffer.WriteString("fe00::2\tip6-allrouters\n")
	if len(hostDomainName) > 0 {
		buffer.WriteString(fmt.Sprintf("%s\t%s.%s\t%s\n", hostIP, hostName, hostDomainName, hostName))
	} else {
		buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostIP, hostName))
	}
	hostsFileContent := buffer.Bytes()
	hostsFileContent = append(hostsFileContent, hostsEntriesFromHostAliases(hostAliases)...)
	return hostsFileContent
}

// IsPodTerminated returns trus if the pod with the provided UID is in a terminated state ("Failed" or "Succeeded")
// or if the pod has been deleted or removed
func (e *edged) IsPodTerminated(uid types.UID) bool {
	pod, podFound := e.podManager.GetPodByUID(uid)
	if !podFound {
		return true
	}
	return e.podIsTerminated(pod)
}

func podIsEvicted(podStatus v1.PodStatus) bool {
	return podStatus.Phase == v1.PodFailed && podStatus.Reason == "Evicted"
}

// IsPodDeleted returns true if the pod is deleted.  For the pod to be deleted, either:
// 1. The pod object is deleted
// 2. The pod's status is evicted
// 3. The pod's deletion timestamp is set, and containers are not running
func (e *edged) IsPodDeleted(uid types.UID) bool {
	pod, podFound := e.podManager.GetPodByUID(uid)
	if !podFound {
		return true
	}
	status, statusFound := e.statusManager.GetPodStatus(pod.UID)
	if !statusFound {
		status = pod.Status
	}
	return podIsEvicted(status) || (pod.DeletionTimestamp != nil && notRunning(status.ContainerStatuses))
}

// removeOrphanedPodStatuses removes obsolete entries in podStatus where
// the pod is no longer considered bound to this node.
func (e *edged) removeOrphanedPodStatuses(pods []*v1.Pod) {
	podUIDs := make(map[types.UID]bool)
	for _, pod := range pods {
		podUIDs[pod.UID] = true
	}

	e.statusManager.RemoveOrphanedStatuses(podUIDs)
}

// GetPodCgroupParent gets pod cgroup parent from container manager.
func (e *edged) GetPodCgroupParent(pod *v1.Pod) string {
	/*pcm := e.containerManager.NewPodContainerManager()
	_, cgroupParent := pcm.GetPodContainerName(pod)
	return cgroupParent*/
	return "systemd"
}

// GenerateRunContainerOptions generates the RunContainerOptions, which can be used by
// the container runtime to set parameters for launching a container.
func (e *edged) GenerateRunContainerOptions(pod *v1.Pod, container *v1.Container, podIP string) (*kubecontainer.RunContainerOptions, func(), error) {
	/*opts, err := e.GenerateContainerOptions(pod)
	if err != nil {
		return nil, nil, err
	}*/
	opts := kubecontainer.RunContainerOptions{}

	hostname, hostDomainName, err := e.GeneratePodHostNameAndDomain(pod)
	if err != nil {
		return nil, nil, err
	}
	opts.Hostname = hostname
	podName := util.GetUniquePodName(pod)
	volumes := e.volumeManager.GetMountedVolumesForPod(podName)
	opts.PortMappings = kubecontainer.MakePortMappings(container)

	// TODO: remove feature gate check after no longer needed
	if utilfeature.DefaultFeatureGate.Enabled(features.BlockVolume) {
		blkutil := volumepathhandler.NewBlockVolumePathHandler()
		blkVolumes, err := e.makeBlockVolumes(pod, container, volumes, blkutil)
		if err != nil {
			return nil, nil, err
		}
		opts.Devices = append(opts.Devices, blkVolumes...)
	}

	/*envs, err := e.makeEnvironmentVariables(pod, container, podIP)
	if err != nil {
		return nil, nil, err
	}
	opts.Envs = append(opts.Envs, envs...)*/

	mounts, err := makeMounts(pod, e.getPodDir(pod.UID), container, hostname, hostDomainName, podIP, volumes)
	if err != nil {
		return nil, nil, err
	}
	opts.Mounts = append(opts.Mounts, mounts...)

	// Disabling adding TerminationMessagePath on Windows as these files would be mounted as docker volume and
	// Docker for Windows has a bug where only directories can be mounted
	if len(container.TerminationMessagePath) != 0 && runtime.GOOS != "windows" {
		p := e.getPodContainerDir(pod.UID, container.Name)
		if err := os.MkdirAll(p, 0750); err != nil {
			glog.Errorf("Error on creating %q: %v", p, err)
		} else {
			opts.PodContainerDir = p
		}
	}

	return &opts, nil, nil
}

// GetPodDNS returns DNS settings for the pod.
// This function is defined in kubecontainer.RuntimeHelper interface so we
// have to implement it.
func (e *edged) GetPodDNS(pod *v1.Pod) (*runtimeapi.DNSConfig, error) {
	dnsConfig := &runtimeapi.DNSConfig{Servers: []string{""}}
	return dnsConfig, nil
}

// Make the environment variables for a pod in the given namespace.
/*func (e *edged) makeEnvironmentVariables(pod *v1.Pod, container *v1.Container, podIP string) ([]kubecontainer.EnvVar, error) {
	if pod.Spec.EnableServiceLinks == nil {
		return nil, fmt.Errorf("nil pod.spec.enableServiceLinks encountered, cannot construct envvars")
	}

	var result []kubecontainer.EnvVar
	// Note:  These are added to the docker Config, but are not included in the checksum computed
	// by kubecontainer.HashContainer(...).  That way, we can still determine whether an
	// v1.Container is already running by its hash. (We don't want to restart a container just
	// because some service changed.)
	//
	// Note that there is a race between Kubelet seeing the pod and kubelet seeing the service.
	// To avoid this users can: (1) wait between starting a service and starting; or (2) detect
	// missing service env var and exit and be restarted; or (3) use DNS instead of env vars
	// and keep trying to resolve the DNS name of the service (recommended).
	serviceEnv, err := e.getServiceEnvVarMap(pod.Namespace, *pod.Spec.EnableServiceLinks)
	if err != nil {
		return result, err
	}

	var (
		configMaps = make(map[string]*v1.ConfigMap)
		secrets    = make(map[string]*v1.Secret)
		tmpEnv     = make(map[string]string)
	)

	// Env will override EnvFrom variables.
	// Process EnvFrom first then allow Env to replace existing values.
	for _, envFrom := range container.EnvFrom {
		switch {
		case envFrom.ConfigMapRef != nil:
			cm := envFrom.ConfigMapRef
			name := cm.Name
			configMap, ok := configMaps[name]
			if !ok {
				if e.kubeClient == nil {
					return result, fmt.Errorf("Couldn't get configMap %v/%v, no kubeClient defined", pod.Namespace, name)
				}
				optional := cm.Optional != nil && *cm.Optional
				configMap, err = e.configMapManager.GetConfigMap(pod.Namespace, name)
				if err != nil {
					if errors.IsNotFound(err) && optional {
						// ignore error when marked optional
						continue
					}
					return result, err
				}
				configMaps[name] = configMap
			}

			invalidKeys := []string{}
			for k, v := range configMap.Data {
				if len(envFrom.Prefix) > 0 {
					k = envFrom.Prefix + k
				}
				if errMsgs := utilvalidation.IsEnvVarName(k); len(errMsgs) != 0 {
					invalidKeys = append(invalidKeys, k)
					continue
				}
				tmpEnv[k] = v
			}
			if len(invalidKeys) > 0 {
				sort.Strings(invalidKeys)
				e.recorder.Eventf(pod, v1.EventTypeWarning, "InvalidEnvironmentVariableNames", "Keys [%s] from the EnvFrom configMap %s/%s were skipped since they are considered invalid environment variable names.", strings.Join(invalidKeys, ", "), pod.Namespace, name)
			}
		case envFrom.SecretRef != nil:
			s := envFrom.SecretRef
			name := s.Name
			secret, ok := secrets[name]
			if !ok {
				if e.kubeClient == nil {
					return result, fmt.Errorf("Couldn't get secret %v/%v, no kubeClient defined", pod.Namespace, name)
				}
				optional := s.Optional != nil && *s.Optional
				secret, err = e.secretManager.GetSecret(pod.Namespace, name)
				if err != nil {
					if errors.IsNotFound(err) && optional {
						// ignore error when marked optional
						continue
					}
					return result, err
				}
				secrets[name] = secret
			}

			invalidKeys := []string{}
			for k, v := range secret.Data {
				if len(envFrom.Prefix) > 0 {
					k = envFrom.Prefix + k
				}
				if errMsgs := utilvalidation.IsEnvVarName(k); len(errMsgs) != 0 {
					invalidKeys = append(invalidKeys, k)
					continue
				}
				tmpEnv[k] = string(v)
			}
			if len(invalidKeys) > 0 {
				sort.Strings(invalidKeys)
				e.recorder.Eventf(pod, v1.EventTypeWarning, "InvalidEnvironmentVariableNames", "Keys [%s] from the EnvFrom secret %s/%s were skipped since they are considered invalid environment variable names.", strings.Join(invalidKeys, ", "), pod.Namespace, name)
			}
		}
	}

	// Determine the final values of variables:
	//
	// 1.  Determine the final value of each variable:
	//     a.  If the variable's Value is set, expand the `$(var)` references to other
	//         variables in the .Value field; the sources of variables are the declared
	//         variables of the container and the service environment variables
	//     b.  If a source is defined for an environment variable, resolve the source
	// 2.  Create the container's environment in the order variables are declared
	// 3.  Add remaining service environment vars
	var (
		mappingFunc = expansion.MappingFuncFor(tmpEnv, serviceEnv)
	)
	for _, envVar := range container.Env {
		runtimeVal := envVar.Value
		if runtimeVal != "" {
			// Step 1a: expand variable references
			runtimeVal = expansion.Expand(runtimeVal, mappingFunc)
		} else if envVar.ValueFrom != nil {
			// Step 1b: resolve alternate env var sources
			switch {
			case envVar.ValueFrom.FieldRef != nil:
				runtimeVal, err = e.podFieldSelectorRuntimeValue(envVar.ValueFrom.FieldRef, pod, podIP)
				if err != nil {
					return result, err
				}
			case envVar.ValueFrom.ResourceFieldRef != nil:
				defaultedPod, defaultedContainer, err := e.defaultPodLimitsForDownwardAPI(pod, container)
				if err != nil {
					return result, err
				}
				runtimeVal, err = containerResourceRuntimeValue(envVar.ValueFrom.ResourceFieldRef, defaultedPod, defaultedContainer)
				if err != nil {
					return result, err
				}
			case envVar.ValueFrom.ConfigMapKeyRef != nil:
				cm := envVar.ValueFrom.ConfigMapKeyRef
				name := cm.Name
				key := cm.Key
				optional := cm.Optional != nil && *cm.Optional
				configMap, ok := configMaps[name]
				if !ok {
					if e.kubeClient == nil {
						return result, fmt.Errorf("Couldn't get configMap %v/%v, no kubeClient defined", pod.Namespace, name)
					}
					configMap, err = e.configMapManager.GetConfigMap(pod.Namespace, name)
					if err != nil {
						if errors.IsNotFound(err) && optional {
							// ignore error when marked optional
							continue
						}
						return result, err
					}
					configMaps[name] = configMap
				}
				runtimeVal, ok = configMap.Data[key]
				if !ok {
					if optional {
						continue
					}
					return result, fmt.Errorf("Couldn't find key %v in ConfigMap %v/%v", key, pod.Namespace, name)
				}
			case envVar.ValueFrom.SecretKeyRef != nil:
				s := envVar.ValueFrom.SecretKeyRef
				name := s.Name
				key := s.Key
				optional := s.Optional != nil && *s.Optional
				secret, ok := secrets[name]
				if !ok {
					if e.kubeClient == nil {
						return result, fmt.Errorf("Couldn't get secret %v/%v, no kubeClient defined", pod.Namespace, name)
					}
					secret, err = e.secretManager.GetSecret(pod.Namespace, name)
					if err != nil {
						if errors.IsNotFound(err) && optional {
							// ignore error when marked optional
							continue
						}
						return result, err
					}
					secrets[name] = secret
				}
				runtimeValBytes, ok := secret.Data[key]
				if !ok {
					if optional {
						continue
					}
					return result, fmt.Errorf("Couldn't find key %v in Secret %v/%v", key, pod.Namespace, name)
				}
				runtimeVal = string(runtimeValBytes)
			}
		}
		// Accesses apiserver+Pods.
		// So, the master may set service env vars, or kubelet may.  In case both are doing
		// it, we delete the key from the kubelet-generated ones so we don't have duplicate
		// env vars.
		// TODO: remove this next line once all platforms use apiserver+Pods.
		delete(serviceEnv, envVar.Name)

		tmpEnv[envVar.Name] = runtimeVal
	}

	// Append the env vars
	for k, v := range tmpEnv {
		result = append(result, kubecontainer.EnvVar{Name: k, Value: v})
	}

	// Append remaining service env vars.
	for k, v := range serviceEnv {
		// Accesses apiserver+Pods.
		// So, the master may set service env vars, or kubelet may.  In case both are doing
		// it, we skip the key from the kubelet-generated ones so we don't have duplicate
		// env vars.
		// TODO: remove this next line once all platforms use apiserver+Pods.
		if _, present := tmpEnv[k]; !present {
			result = append(result, kubecontainer.EnvVar{Name: k, Value: v})
		}
	}
	return result, nil
}*/

// makeBlockVolumes maps the raw block devices specified in the path of the container
// Experimental
func (e edged) makeBlockVolumes(pod *v1.Pod, container *v1.Container, podVolumes kubecontainer.VolumeMap, blkutil volumepathhandler.BlockVolumePathHandler) ([]kubecontainer.DeviceInfo, error) {
	var devices []kubecontainer.DeviceInfo
	for _, device := range container.VolumeDevices {
		// check path is absolute
		if !filepath.IsAbs(device.DevicePath) {
			return nil, fmt.Errorf("error DevicePath `%s` must be an absolute path", device.DevicePath)
		}
		vol, ok := podVolumes[device.Name]
		if !ok || vol.BlockVolumeMapper == nil {
			glog.Errorf("Block volume cannot be satisfied for container %q, because the volume is missing or the volume mapper is nil: %+v", container.Name, device)
			return nil, fmt.Errorf("cannot find volume %q to pass into container %q", device.Name, container.Name)
		}
		// Get a symbolic link associated to a block device under pod device path
		dirPath, volName := vol.BlockVolumeMapper.GetPodDeviceMapPath()
		symlinkPath := path.Join(dirPath, volName)
		if islinkExist, checkErr := blkutil.IsSymlinkExist(symlinkPath); checkErr != nil {
			return nil, checkErr
		} else if islinkExist {
			// Check readOnly in PVCVolumeSource and set read only permission if it's true.
			permission := "mrw"
			if vol.ReadOnly {
				permission = "r"
			}
			glog.V(4).Infof("Device will be attached to container %q. Path on host: %v", container.Name, symlinkPath)
			devices = append(devices, kubecontainer.DeviceInfo{PathOnHost: symlinkPath, PathInContainer: device.DevicePath, Permissions: permission})
		}
	}

	return devices, nil
}
