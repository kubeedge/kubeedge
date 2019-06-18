/*
Copyright 2015 The Kubernetes Authors.

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
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
Changes done are
1. Interface containerManager is derived from "k8s.io/kubernetes/pkg/kubelet/cm/container_manager_linux.go"
   runed extra interface  and changed most of the realization
2. Struct containerManager partially derived from kubernetes/pkg/kubelet/cm/devicemanager.ManagerImpl
*/

package containers

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerstrslice "github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis/runtime/cri"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/securitycontext"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/flowcontrol"
	deviceplugin "k8s.io/kubernetes/pkg/kubelet/cm/devicemanager"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/container/testing"
	"k8s.io/kubernetes/pkg/kubelet/gpu"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/scheduler/schedulercache"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	"k8s.io/kubernetes/pkg/util/selinux"
)

//Pod details constants
const (
	// Taken from lmctfy https://github.com/google/lmctfy/blob/master/lmctfy/controllers/cpu_controller.cc
	minShares     = 2
	sharesPerCPU  = 1024
	milliCPUToCPU = 1000

	// 100000 is equivalent to 100ms
	quotaPeriod    = 100 * minQuotaPeriod
	minQuotaPeriod = 1000

	// TODO: change those label names to follow kubernetes's format
	podDeletionGracePeriodLabel    = "io.kubernetes.pod.deletionGracePeriod"
	podTerminationGracePeriodLabel = "io.kubernetes.pod.terminationGracePeriod"

	containerHashLabel                     = "io.kubernetes.container.hash"
	containerRestartCountLabel             = "io.kubernetes.container.restartCount"
	containerTerminationMessagePathLabel   = "io.kubernetes.container.terminationMessagePath"
	containerTerminationMessagePolicyLabel = "io.kubernetes.container.terminationMessagePolicy"

	KubernetesPodNameLabel       = "io.kubernetes.pod.name"
	KubernetesPodNamespaceLabel  = "io.kubernetes.pod.namespace"
	KubernetesPodUIDLabel        = "io.kubernetes.pod.uid"
	KubernetesContainerNameLabel = "io.kubernetes.container.name"

	// kubePrefix is used to identify the containers/sandboxes on the node managed by kubelet
	kubePrefix = "k8s"

	// Delimiter used to construct docker container names.
	nameDelimiter = "_"

	UnknownContainerStatuses = "UnknownContainerStatuses"
	PodCompleted             = "PodCompleted"
	ContainersNotReady       = "ContainersNotReady"
	ContainersNotInitialized = "ContainersNotInitialized"
)

// The struct containerManager partially derived from kubernetes/pkg/kubelet/cm/devicemanager.ManagerImpl
// and changed most of the realization
type containerManager struct {
	*testing.Mock
	backOff                  *flowcontrol.Backoff
	runtimeService           cri.RuntimeService
	podContainer             map[types.UID]*cri.Container
	podContainerLock         sync.RWMutex
	containerRecords         map[string]*containerRecord
	containerRecordsLock     sync.Mutex
	devicePluginManager      deviceplugin.Manager
	gpuManager               gpu.GPUManager
	defaultHostInterfaceName string
	livenessManager          proberesults.Manager
}

type containerRecord struct {
	firstDetected time.Time
	lastUsed      time.Time
	Status        kubecontainer.ContainerState
	podID         types.UID
}

type containerToKillInfo struct {
	// The spec of the container.
	container *v1.Container
	// The name of the container.
	name string
	// The message indicates why the container will be killed.
	message string
}

type podActions struct {
	// ContainersToStart keeps a list of indexes for the containers to start,
	// where the index is the index of the specific container in the pod spec (
	// pod.Spec.Containers.
	ContainersToStart      []int
	ContainersRestartCount int
	// ContainersToKill keeps a map of containers that need to be killed, note that
	// the key is the container ID of the container, while
	// the value contains necessary information to kill a container.
	ContainersToKill map[types.UID]containerToKillInfo
}

// ContainerManager partially derived from kubernetes/pkg/kubelet/cm.ContainerManager
// pruned extra interface and append our own method
type ContainerManager interface {
	Start(activePods deviceplugin.ActivePodsFunc) error
	GetDevicePluginResourceCapacity() (v1.ResourceList, []string)
	UpdatePluginResources(*schedulercache.NodeInfo, *lifecycle.PodAdmitAttributes) error
	InitPodContainer() error
	StartPod(pod *v1.Pod, runOpt *kubecontainer.RunContainerOptions) error
	UpdatePod(pod *v1.Pod) error
	TerminatePod(uid types.UID) error
	RunInContainer(id kubecontainer.ContainerID, cmd []string, timeout time.Duration) ([]byte, error)
	GetPodStatusOwn(pod *v1.Pod) (*v1.PodStatus, error)
	GetPods(all bool) ([]*kubecontainer.Pod, error)
	GarbageCollect(gcPolicy kubecontainer.ContainerGCPolicy, ready bool, evictNonDeletedPods bool) error
	GeneratePodReadyCondition(statuses []v1.ContainerStatus) v1.PodCondition
	CleanupOrphanedPod(activePods []*v1.Pod)
}

const (
	defaultGracePeriod = 30

	defaultNetWortMode dockercontainer.NetworkMode = "host"
)

//NewContainerManager initialises and returns a container manager object
func NewContainerManager(runtimeService cri.RuntimeService, livenessManager proberesults.Manager, containerBackOff *flowcontrol.Backoff, devicePluginEnabled bool, gpuManager gpu.GPUManager, interfaceName string) (ContainerManager, error) {
	var devicePluginManager deviceplugin.Manager
	var err error
	cm := &containerManager{
		Mock:                     new(testing.Mock),
		runtimeService:           runtimeService,
		backOff:                  containerBackOff,
		podContainer:             make(map[types.UID]*cri.Container),
		containerRecords:         make(map[string]*containerRecord),
		gpuManager:               gpuManager,
		defaultHostInterfaceName: interfaceName,
		livenessManager:          livenessManager,
	}
	if devicePluginEnabled {
		devicePluginManager, err = deviceplugin.NewManagerImpl()
	} else {
		devicePluginManager, err = deviceplugin.NewManagerStub()
	}

	if err != nil {
		return nil, err
	}
	cm.devicePluginManager = devicePluginManager
	return cm, nil
}

func (cm *containerManager) Start(activePods deviceplugin.ActivePodsFunc) error {
	cm.devicePluginManager.Start(deviceplugin.ActivePodsFunc(activePods), sourceImpl{})
	return nil
}

func (cm *containerManager) GetDevicePluginResourceCapacity() (v1.ResourceList, []string) {
	devicePluginCapacity, _, removedDevicePlugins := cm.devicePluginManager.GetCapacity()
	return devicePluginCapacity, removedDevicePlugins
}

func (cm *containerManager) UpdatePluginResources(nodeInfo *schedulercache.NodeInfo, attrs *lifecycle.PodAdmitAttributes) error {
	return cm.devicePluginManager.Allocate(nodeInfo, attrs)
}

// HashContainer returns the hash of the container. It is used to compare
// the running container with its desired spec.
func hashContainer(container *v1.Container) uint64 {
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *container)
	return uint64(hash.Sum32())
}

func newContainerLabels(pod *v1.Pod, container *v1.Container, restartCount int) map[string]string {
	labels := map[string]string{}
	labels[KubernetesPodNameLabel] = pod.Name
	labels[KubernetesPodNamespaceLabel] = pod.Namespace
	labels[KubernetesPodUIDLabel] = string(pod.UID)
	labels[KubernetesContainerNameLabel] = container.Name

	labels[containerHashLabel] = strconv.FormatUint(hashContainer(container), 16)
	labels[containerRestartCountLabel] = strconv.Itoa(restartCount)
	labels[containerTerminationMessagePathLabel] = container.TerminationMessagePath
	labels[containerTerminationMessagePolicyLabel] = string(container.TerminationMessagePolicy)
	return labels
}

func containerChanged(container *v1.Container, oldHash uint64) (uint64, uint64, bool) {
	expectedHash := hashContainer(container)
	return expectedHash, oldHash, oldHash != expectedHash
}

// ShouldContainerBeRestarted checks whether a container needs to be restarted.
// TODO(yifan): Think about how to refactor this.
func shouldContainerBeRestarted(container *v1.Container, pod *v1.Pod, status *cri.ContainerStatus) bool {
	// If the container was never started before, we should start it.
	// NOTE(random-liu): If all historical containers were GC'd, we'll also return true here.
	if status == nil {
		return true
	}
	// Check whether container is running
	if status.State == cri.StatusRUNNING {
		return false
	}
	// Always restart container in the unknown, or in the created state.
	if status.State == cri.StatusUNKNOWN || status.State == cri.StatusCREATED {
		return true
	}
	// Check RestartPolicy for dead container
	if pod.Spec.RestartPolicy == v1.RestartPolicyNever {
		log.LOGGER.Infof("Already ran container %q of pod %q, do nothing", container.Name, pod)
		return false
	}
	if pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
		// Check the exit code.
		if status.ExitCode == 0 {
			log.LOGGER.Infof("Already successfully ran container %q of pod %q, do nothing", container.Name, pod.Name)
			return false
		}
	}
	return true
}

func shouldRestartOnFailure(pod *v1.Pod) bool {
	return pod.Spec.RestartPolicy != v1.RestartPolicyNever
}

func (cm *containerManager) getResource(pod *v1.Pod, container *v1.Container) *kubecontainer.RunContainerOptions {
	opts := &kubecontainer.RunContainerOptions{}
	devOpts, err := cm.devicePluginManager.GetDeviceRunContainerOptions(pod, container)
	if err != nil || devOpts == nil {
		return opts
	}
	opts.Devices = append(opts.Devices, devOpts.Devices...)
	opts.Mounts = append(opts.Mounts, devOpts.Mounts...)
	opts.Envs = append(opts.Envs, devOpts.Envs...)
	return opts
}

func (cm *containerManager) getContainerRecords(id string) (*containerRecord, bool) {
	cm.containerRecordsLock.Lock()
	defer cm.containerRecordsLock.Unlock()
	record, ok := cm.containerRecords[id]
	return record, ok
}

func (cm *containerManager) addContainerRecords(id string, record *containerRecord) {
	cm.containerRecordsLock.Lock()
	defer cm.containerRecordsLock.Unlock()
	cm.containerRecords[id] = record
}

func (cm *containerManager) deleteContainerRecords(id string) {
	cm.containerRecordsLock.Lock()
	defer cm.containerRecordsLock.Unlock()
	delete(cm.containerRecords, id)
}

func (cm *containerManager) computePodActions(pod *v1.Pod) podActions {
	log.LOGGER.Infof("compute pod actions %s: %+v", pod.Name, pod)
	restartCount := -1
	changes := podActions{
		ContainersToStart: []int{},
		ContainersToKill:  make(map[types.UID]containerToKillInfo),
	}

	for idx, container := range pod.Spec.Containers {
		innerContainer, ok := cm.getContainerFromMap(pod.UID)
		if !ok {
			restartCount = 0
			log.LOGGER.Infof("no container id for pod %s, need start container", pod.Name)
			changes.ContainersToStart = append(changes.ContainersToStart, idx)
			continue
		}
		status, err := cm.runtimeService.ContainerStatus(innerContainer.ID)
		if err != nil {
			log.LOGGER.Errorf("get container status for pod %s failed: %v", pod.Name, err)
		}

		if status == nil || status.State != cri.StatusRUNNING {
			if shouldContainerBeRestarted(&container, pod, status) {
				message := fmt.Sprintf("Container %+v is dead, but RestartPolicy says that we should restart it.", container)
				log.LOGGER.Infof(message)
				changes.ContainersToStart = append(changes.ContainersToStart, idx)
			}
			continue
		}

		labels, err := cm.getPodContainerLabels(pod)
		if err != nil {
			log.LOGGER.Errorf("get container labels for pod %s failed: %v", pod.Name, err)
			continue
		}

		if labels[containerHashLabel] == "" {
			log.LOGGER.Errorf("container hash is empty for pod %s failed: %v", pod.Name, err)
			continue
		}

		hash, err := strconv.ParseUint(labels[containerHashLabel], 16, 64)
		if err != nil {
			log.LOGGER.Errorf("container hash is empty for pod %s failed: %v", pod.Name, err)
			continue
		}

		// The container is running, but kill the container if any of the following condition is met.
		reason := ""
		restart := shouldRestartOnFailure(pod)
		containerID := kubecontainer.BuildContainerID("docker", status.ID.ID)

		expectedHash, actualHash, changed := containerChanged(&container, uint64(hash))
		if changed {
			reason = fmt.Sprintf("Container spec hash changed (%d vs %d).", actualHash, expectedHash)
			// Restart regardless of the restart policy because the container
			// spec changed.
			restart = true
		} else if liveness, found := cm.livenessManager.Get(containerID); found && liveness == proberesults.Failure {
			reason = "Container failed liveness probe."
		} else {
			continue
		}

		message := reason
		if restart {
			message = fmt.Sprintf("%s. Container will be killed and recreated.", message)
			changes.ContainersToStart = append(changes.ContainersToStart, idx)
		}
		// We need to kill the container, but if we also want to restart the
		// container afterwards, make the intent clear in the message. Also do
		// not kill the entire pod since we expect container to be running eventually.
		changes.ContainersToKill[types.UID(status.ID.ID)] = containerToKillInfo{
			name:      container.Name,
			container: &pod.Spec.Containers[idx],
			message:   message,
		}
		log.LOGGER.Infof("Container %q (%q) of pod %s: %s", container.Name, status.ID.ID, pod.Name, message)
	}

	if restartCount == -1 {
		labels, err := cm.getPodContainerLabels(pod)
		if err != nil {
			log.LOGGER.Errorf("get container labels for pod %s failed: %v", pod.Name, err)
		} else {
			restartCount, err := strconv.Atoi(labels[containerRestartCountLabel])
			if err != nil {
				log.LOGGER.Errorf("Invalid restart count labbel: %v", err)
			} else {
				changes.ContainersRestartCount = restartCount + 1
			}
		}
	}

	return changes
}

func (cm *containerManager) killContainer(podID types.UID, containerID string) error {
	err := cm.runtimeService.StopContainer(containerID, defaultGracePeriod)
	if err != nil {
		// if an error returns when we stop container, maybe this container is
		// already stoped, ignore this error, try to remove
		log.LOGGER.Errorf("stop container [%s] for pod [%s], err: %v", containerID, podID, err)
	}
	err = cm.runtimeService.DeleteContainer(kubecontainer.ContainerID{ID: containerID})
	if err != nil {
		log.LOGGER.Errorf("remove container [%s] for pod [%s], err: %v", containerID, podID, err)
		return err
	}
	if podID != "" {
		cm.deleteContainerFromMap(podID)
	}
	return nil
}

func makeContainerName(pod *v1.Pod, container *v1.Container, restartCount int) string {
	return strings.Join([]string{
		kubePrefix,                      // 0
		container.Name,                  // 1:
		pod.Name,                        // 2: sandbox name
		pod.Namespace,                   // 3: sandbox namesapce
		string(pod.UID),                 // 4  sandbox uid
		fmt.Sprintf("%d", restartCount), // 5
	}, nameDelimiter)
}

func defaultMemorySwap() int64 {
	return 0
}

// milliCPUToShares converts milliCPU to CPU shares
func milliCPUToShares(milliCPU int64) int64 {
	if milliCPU == 0 {
		// Return 2 here to really match kernel default for zero milliCPU.
		return minShares
	}
	// Conceptually (milliCPU / milliCPUToCPU) * sharesPerCPU, but factored to improve rounding.
	shares := (milliCPU * sharesPerCPU) / milliCPUToCPU
	if shares < minShares {
		return minShares
	}
	return shares
}

// milliCPUToQuota converts milliCPU to CFS quota and period values
func milliCPUToQuota(milliCPU int64) (quota int64, period int64) {
	// CFS quota is measured in two values:
	//  - cfs_period_us=100ms (the amount of time to measure usage across)
	//  - cfs_quota=20ms (the amount of cpu time allowed to be used across a period)
	// so in the above example, you are limited to 20% of a single CPU
	// for multi-cpu environments, you just scale equivalent amounts
	if milliCPU == 0 {
		return
	}

	// we set the period to 100ms by default
	period = quotaPeriod

	// we then convert your milliCPU to a value normalized over a period
	quota = (milliCPU * quotaPeriod) / milliCPUToCPU

	// quota needs to be a minimum of 1ms.
	if quota < minQuotaPeriod {
		quota = minQuotaPeriod
	}

	return
}

func updateCreateConfig(config *cri.ContainerConfig, container *v1.Container) {
	var cpuShares int64
	cpuRequest := container.Resources.Requests.Cpu()
	cpuLimit := container.Resources.Limits.Cpu()
	memoryLimit := container.Resources.Limits.Memory().Value()

	// If request is not specified, but limit is, we want request to default to limit.
	// API server does this for new containers, but we repeat this logic in Kubelet
	// for containers running on existing Kubernetes clusters.
	if cpuRequest.IsZero() && !cpuLimit.IsZero() {
		cpuShares = milliCPUToShares(cpuLimit.MilliValue())
	} else {
		// if cpuRequest.Amount is nil, then milliCPUToShares will return the minimal number
		// of CPU shares.
		cpuShares = milliCPUToShares(cpuRequest.MilliValue())
	}

	cpuQuota, cpuPeriod := milliCPUToQuota(cpuLimit.MilliValue())

	config.HostConfig.Resources = dockercontainer.Resources{
		Memory:     memoryLimit,
		MemorySwap: defaultMemorySwap(),
		CPUShares:  cpuShares,
		CPUQuota:   cpuQuota,
		CPUPeriod:  cpuPeriod,
	}

	return
}

func (cm *containerManager) InitPodContainer() error {
	log.LOGGER.Infof("start to init pod container map")
	containers, err := cm.runtimeService.ListContainers()
	if err != nil {
		log.LOGGER.Errorf("list container error %v", err)
		return err
	}

	for _, container := range containers {
		podID, err := cm.getPodID(container.ID)
		if err != nil {
			return err
		}
		if podID != "" {
			if oldContainer, ok := cm.getContainerFromMap(podID); !ok || oldContainer.StartAt.Before(container.StartAt) {
				cm.setContainerFromMap(podID, container)
			}
		}
	}
	return nil
}

func (cm *containerManager) getPodID(containerID string) (types.UID, error) {
	status, err := cm.runtimeService.ContainerStatus(containerID)
	if err != nil {
		log.LOGGER.Errorf("Get container status error %v", err)
		return "", err
	}
	if podID, ok := status.Labels[KubernetesPodUIDLabel]; ok {
		return types.UID(podID), nil
	}
	return "", nil
}

func (cm *containerManager) getContainer(containerID string) (*cri.Container, error) {
	containerInspect, err := cm.runtimeService.InspectContainer(containerID)
	if err != nil {
		return nil, err
	}

	container := &cri.Container{
		ID:      containerID,
		StartAt: containerInspect.Status.CreatedAt,
		Status:  containerInspect.Status.State,
	}
	return container, nil
}

// makeGPUDevices determines the devices for the given container.
// Experimental.
func (cm *containerManager) makeGPUDevices(pod *v1.Pod, container *v1.Container) ([]kubecontainer.DeviceInfo, error) {
	if container.Resources.Limits.NvidiaGPU().IsZero() {
		return nil, nil
	}

	nvidiaGPUPaths, err := cm.gpuManager.AllocateGPU(pod, container)
	if err != nil {
		return nil, err
	}
	var devices []kubecontainer.DeviceInfo
	for _, path := range nvidiaGPUPaths {
		// Devices have to be mapped one to one because of nvidia CUDA library requirements.
		devices = append(devices, kubecontainer.DeviceInfo{PathOnHost: path, PathInContainer: path, Permissions: "mrw"})
	}

	return devices, nil
}

func (cm *containerManager) GenerateRunContainerOptions(pod *v1.Pod, container *v1.Container) (*kubecontainer.RunContainerOptions, error) {
	opts := cm.getResource(pod, container)
	devices, err := cm.makeGPUDevices(pod, container)
	if err != nil {
		return nil, err
	}
	opts.Devices = append(opts.Devices, devices...)

	return opts, nil
}

func makeDevices(opts *kubecontainer.RunContainerOptions) []*cri.Device {
	devices := make([]*cri.Device, len(opts.Devices))

	for idx := range opts.Devices {
		device := opts.Devices[idx]
		devices[idx] = &cri.Device{
			HostPath:      device.PathOnHost,
			ContainerPath: device.PathInContainer,
			Permissions:   device.Permissions,
		}
	}

	return devices
}

func (cm *containerManager) StartPod(pod *v1.Pod, runOpt *kubecontainer.RunContainerOptions) error {
	backOffKey := fmt.Sprintf("container_%s", pod.Name)
	if cm.backOff.IsInBackOffSinceUpdate(backOffKey, cm.backOff.Clock.Now()) {
		log.LOGGER.Errorf("container manager start pod backoff. Back-off pod start [%s] error, backoff: [%v]", pod.Name, cm.backOff.Get(backOffKey))
		return apis.ErrPodStartBackOff
	}

	podContainerChanges := cm.computePodActions(pod)
	log.LOGGER.Infof("computePodActions got %+v for pod %q", podContainerChanges, pod.Name)
	for containerID, containerInfo := range podContainerChanges.ContainersToKill {
		log.LOGGER.Infof("Killing unwanted container %q(id=%q) for pod %q", containerInfo.name, containerID, pod.Name)
		if err := cm.killContainer(pod.UID, string(containerID)); err != nil {
			log.LOGGER.Errorf("killContainer %q(id=%q) for pod %q failed: %v", containerInfo.name, containerID, pod.Name, err)
			return err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.LOGGER.Errorf("get hostname failed: %v", err)
		hostname = string(pod.UID)
	}

	for _, idx := range podContainerChanges.ContainersToStart {
		container := &pod.Spec.Containers[idx]
		log.LOGGER.Infof("create container %s in pod %v", container.Name, pod.Name)
		exposedPorts, err := makePortsAndBindings(container.Ports)
		if err != nil {
			cm.backOff.Next(backOffKey, cm.backOff.Clock.Now())
			log.LOGGER.Errorf("make container [%s] port for pod [%s] failed, %v", container.Name, pod.Name, err)
			return err
		}

		opts, err := cm.GenerateRunContainerOptions(pod, container)
		if err != nil {
			log.LOGGER.Errorf("generate pod [%s] container [%s] run container options failed, %v", pod.Name, container.Name, err)
			continue
		}

		mounts := []*kubecontainer.Mount{}
		containerMounts := cm.makeMounts(runOpt, container)
		deviceMounts := cm.makeMounts(opts, container)
		mounts = append(mounts, containerMounts...)
		mounts = append(mounts, deviceMounts...)

		envs := ConvertEnvVersion(container.Env)
		command, args := kubecontainer.ExpandContainerCommandAndArgs(container, envs)
		restartCount := podContainerChanges.ContainersRestartCount
		containerConfig := cri.ContainerConfig{
			Name: makeContainerName(pod, container, restartCount),
			Config: &dockercontainer.Config{
				Hostname:     hostname,
				Image:        container.Image,
				Env:          GenerateEnvList(container.Env),
				Labels:       newContainerLabels(pod, container, restartCount),
				ExposedPorts: exposedPorts,
				Entrypoint:   dockerstrslice.StrSlice(command),
				Cmd:          dockerstrslice.StrSlice(args),
				WorkingDir:   container.WorkingDir,
				OpenStdin:    container.Stdin,
				StdinOnce:    container.StdinOnce,
				Tty:          container.TTY,
			},
			HostConfig: &dockercontainer.HostConfig{
				NetworkMode: defaultNetWortMode,
				Binds:       GenerateMountBindings(mounts),
			},
		}

		updateCreateConfig(&containerConfig, container)

		devices := make([]dockercontainer.DeviceMapping, len(opts.Devices))
		for i, device := range opts.Devices {
			devices[i] = dockercontainer.DeviceMapping{
				PathOnHost:        device.PathOnHost,
				PathInContainer:   device.PathInContainer,
				CgroupPermissions: device.Permissions,
			}
		}
		containerConfig.HostConfig.Devices = devices

		if EnableHostUserNamespace(pod) {
			containerConfig.HostConfig.UsernsMode = dockercontainer.UsernsMode("host")
		}
		securityContext := securitycontext.NewSimpleSecurityContextProvider()
		securityContext.ModifyContainerConfig(pod, containerConfig.Config)
		securityContext.ModifyHostConfig(pod, containerConfig.HostConfig)
		containerID, err := cm.runtimeService.CreateContainer(&containerConfig)
		if err != nil {
			cm.backOff.Next(backOffKey, cm.backOff.Clock.Now())
			log.LOGGER.Errorf("start pod [%s] failed: %v ", string(pod.Name), err)
			return err
		}

		innerContainer, err := cm.getContainer(containerID)
		if err != nil {
			log.LOGGER.Errorf("get container for pod %s container id %s failed, %v", string(pod.Name), containerID, err)
		}
		cm.setContainerFromMap(pod.UID, innerContainer)

		if err = cm.runtimeService.StartContainer(containerID); err != nil {
			cm.backOff.Next(backOffKey, cm.backOff.Clock.Now())
			log.LOGGER.Errorf("start pod [%s], container id [%s] failed, with err: [%s], remove the container.", string(pod.Name), containerID, err)
			continue
		}
		log.LOGGER.Infof("start container for pod [%s] success", pod.Name)
	}
	cm.backOff.Reset(backOffKey)
	return nil
}

func (cm *containerManager) UpdatePod(pod *v1.Pod) error {
	return nil
}

func (cm *containerManager) TerminatePod(uid types.UID) error {
	container, ok := cm.getContainerFromMap(uid)
	if !ok {
		log.LOGGER.Errorf("get container id error, terminate pod [%s] not found.", uid)
		return apis.ErrContainerNotFound
	}
	err := cm.runtimeService.StopContainer(container.ID, defaultGracePeriod)
	if err != nil {
		log.LOGGER.Errorf("stop container [%s] for pod [%s], err: %v", container.ID, uid, err)
		return err
	}
	err = cm.runtimeService.DeleteContainer(kubecontainer.ContainerID{ID: container.ID})
	if err != nil {
		log.LOGGER.Errorf("remove container [%s] for pod [%s], err: %v", container.ID, uid, err)
		return err
	}
	cm.deleteContainerFromMap(uid)
	return nil
}

func (cm *containerManager) RunInContainer(id kubecontainer.ContainerID, cmd []string, timeout time.Duration) ([]byte, error) {
	return nil, nil
}

/*func (cm *containerManager) DeleteContainer(containerID string) error {

	return nil
}

func (cm *containerManager) GetContainerLog(containerID string) error {

	return nil
}

func (cm *containerManager) RemoveContainerLog(containerID string) error {

	return nil
}*/

func (cm *containerManager) getPodContainerLabels(pod *v1.Pod) (map[string]string, error) {
	container, ok := cm.getContainerFromMap(pod.UID)
	if !ok {
		return nil, fmt.Errorf("get pod [%s] status failed with container nod found error", pod.Name)
	}
	status, err := cm.runtimeService.ContainerStatus(container.ID)
	if err != nil {
		return nil, fmt.Errorf("get pod [%s] status failed with error %v", pod.Name, err)
	}
	return status.Labels, nil
}

//GetContainerStatus returns container status
func GetContainerStatus(statuses []v1.ContainerStatus, name string) (v1.ContainerStatus, bool) {
	for i := range statuses {
		if statuses[i].Name == name {
			return statuses[i], true
		}
	}
	return v1.ContainerStatus{}, false
}

func (cm *containerManager) GeneratePodReadyCondition(statuses []v1.ContainerStatus) v1.PodCondition {
	if statuses == nil {
		return v1.PodCondition{
			Type:   v1.PodReady,
			Status: v1.ConditionFalse,
			Reason: UnknownContainerStatuses,
		}
	}

	unreadyContainers := []string{}

	for _, status := range statuses {
		if !status.Ready {
			unreadyContainers = append(unreadyContainers, status.Name)
		}
	}

	unreadyMessages := []string{}
	if len(unreadyContainers) > 0 {
		unreadyMessages = append(unreadyMessages, fmt.Sprintf("containers with unready status: %s", unreadyContainers))
	}

	unreadyMessage := strings.Join(unreadyMessages, ", ")
	if unreadyMessage != "" {
		return v1.PodCondition{
			Type:    v1.PodReady,
			Status:  v1.ConditionFalse,
			Reason:  ContainersNotReady,
			Message: unreadyMessage,
		}
	}

	return v1.PodCondition{
		Type:   v1.PodReady,
		Status: v1.ConditionTrue,
	}
}

func (cm *containerManager) convertStatusToAPIStatus(pod *v1.Pod, status *cri.ContainerStatus) *v1.PodStatus {
	switch status.State {
	case cri.StatusRUNNING:
		kubeStatus := cm.toKubeContainerStatus(v1.PodRunning, status)
		return &v1.PodStatus{Phase: v1.PodRunning, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
	case cri.StatusEXITED:
		if (pod.Spec.RestartPolicy == v1.RestartPolicyNever || pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure) && status.ExitCode == 0 {
			kubeStatus := cm.toKubeContainerStatus(v1.PodSucceeded, status)
			return &v1.PodStatus{Phase: v1.PodSucceeded, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
		}
		kubeStatus := cm.toKubeContainerStatus(v1.PodFailed, status)
		return &v1.PodStatus{Phase: v1.PodFailed, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
	case cri.StatusPAUSED:
		kubeStatus := cm.toKubeContainerStatus(v1.PodUnknown, status)
		return &v1.PodStatus{Phase: v1.PodUnknown, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
	case cri.StatusCREATED:
		if (pod.Spec.RestartPolicy == v1.RestartPolicyNever || pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure) && status.ExitCode != 0 {
			kubeStatus := cm.toKubeContainerStatus(v1.PodFailed, status)
			return &v1.PodStatus{Phase: v1.PodFailed, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
		}
		kubeStatus := cm.toKubeContainerStatus(v1.PodRunning, status)
		return &v1.PodStatus{Phase: v1.PodRunning, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
	default:
		kubeStatus := cm.toKubeContainerStatus(v1.PodUnknown, status)
		return &v1.PodStatus{Phase: v1.PodUnknown, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}
	}
}

//GetPhase returns pod phase
func GetPhase(spec *v1.PodSpec, info []v1.ContainerStatus) v1.PodPhase {
	unknown := 0
	running := 0
	waiting := 0
	stopped := 0
	failed := 0
	succeeded := 0
	for _, container := range spec.Containers {
		containerStatus, ok := GetContainerStatus(info, container.Name)
		if !ok {
			unknown++
			continue
		}

		switch {
		case containerStatus.State.Running != nil:
			running++
		case containerStatus.State.Terminated != nil:
			stopped++
			if containerStatus.State.Terminated.ExitCode == 0 {
				succeeded++
			} else {
				failed++
			}
		case containerStatus.State.Waiting != nil:
			if containerStatus.LastTerminationState.Terminated != nil {
				stopped++
			} else {
				waiting++
			}
		default:
			unknown++
		}
	}
	switch {
	case waiting > 0:
		// One or more containers has not been started
		return v1.PodPending
	case running > 0 && unknown == 0:
		// All containers have been started, and at least
		// one container is running
		return v1.PodRunning
	case running == 0 && stopped > 0 && unknown == 0:
		// All containers are terminated
		if spec.RestartPolicy == v1.RestartPolicyAlways {
			// All containers are in the process of restarting
			return v1.PodRunning
		}
		if stopped == succeeded {
			// RestartPolicy is not Always, and all
			// containers are terminated in success
			return v1.PodSucceeded
		}
		if spec.RestartPolicy == v1.RestartPolicyNever {
			// RestartPolicy is Never, and all containers are
			// terminated with at least one in failure
			return v1.PodFailed
		}
		// RestartPolicy is OnFailure, and at least one in failure
		// and in the process of restarting
		return v1.PodRunning
	default:
		return v1.PodPending
	}
}

// TODO: GetPodStatusOwn is tmp, replace with GetPodStatus(uid kubetypes.UID, name, namespace string) (*kubecontainer.PodStatus, error)
func (cm *containerManager) GetPodStatusOwn(pod *v1.Pod) (*v1.PodStatus, error) {
	container, ok := cm.getContainerFromMap(pod.UID)
	if !ok {
		if pod.DeletionTimestamp == nil {
			status := &cri.ContainerStatus{ContainerStatus: kubecontainer.ContainerStatus{Reason: "ContainerCreating"}}
			kubeStatus := cm.toKubeContainerStatus(v1.PodUnknown, status)
			return &v1.PodStatus{Phase: v1.PodUnknown, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}, nil
		}
		//else
		status := &cri.ContainerStatus{ContainerStatus: kubecontainer.ContainerStatus{Reason: "Completed"}}
		kubeStatus := cm.toKubeContainerStatus(v1.PodSucceeded, status)
		return &v1.PodStatus{Phase: v1.PodSucceeded, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}, nil

	}
	status, err := cm.runtimeService.ContainerStatus(container.ID)
	if err != nil {
		status := &cri.ContainerStatus{}
		kubeStatus := cm.toKubeContainerStatus(v1.PodUnknown, status)
		return &v1.PodStatus{Phase: v1.PodUnknown, ContainerStatuses: []v1.ContainerStatus{kubeStatus}}, nil
	}

	podstatus := cm.convertStatusToAPIStatus(pod, status)
	spec := &pod.Spec
	podstatus.Phase = GetPhase(spec, podstatus.ContainerStatuses)

	hostIP, err := cm.getHostIPByInterface()
	if err != nil {
		log.LOGGER.Errorf("Cannot get host IP: %v", err)
	} else {
		podstatus.HostIP = hostIP
		if pod.Spec.HostNetwork && podstatus.PodIP == "" {
			podstatus.PodIP = hostIP
		}
	}

	return podstatus, nil
}

func (cm *containerManager) toKubeContainerStatus(phase v1.PodPhase, status *cri.ContainerStatus) v1.ContainerStatus {
	restartCount, err := strconv.Atoi(status.Labels[containerRestartCountLabel])
	if err != nil {
		restartCount = 0
	}
	kubeStatus := v1.ContainerStatus{
		Name:         status.Name,
		RestartCount: int32(restartCount),
		ImageID:      status.ImageRef,
		Image:        status.Image,
		ContainerID:  cri.DockerPrefix + status.ID.ID,
	}

	switch phase {
	case v1.PodRunning:
		kubeStatus.State.Running = &v1.ContainerStateRunning{StartedAt: metav1.Time{status.StartedAt}}
		kubeStatus.Ready = true
	case v1.PodFailed, v1.PodSucceeded:
		kubeStatus.State.Terminated = &v1.ContainerStateTerminated{
			ExitCode:    int32(status.ExitCode),
			Reason:      status.Reason,
			Message:     status.Message,
			StartedAt:   metav1.Time{status.StartedAt},
			FinishedAt:  metav1.Time{status.FinishedAt},
			ContainerID: status.ID.ID,
		}
	default:
		kubeStatus.State.Waiting = &v1.ContainerStateWaiting{
			Reason:  status.Reason,
			Message: status.Message,
		}
	}
	return kubeStatus
}

func (cm *containerManager) getHostIPByInterface() (string, error) {
	iface, err := net.InterfaceByName(cm.defaultHostInterfaceName)
	if err != nil {
		return "", fmt.Errorf("failed to get network interface: %v err:%v", cm.defaultHostInterfaceName, err)
	}
	if iface == nil {
		return "", fmt.Errorf("input iface is nil")
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no ip and mask in this network card")
}

func (cm *containerManager) GetPods(all bool) ([]*kubecontainer.Pod, error) {
	pods := make([]*kubecontainer.Pod, 0)

	uids := cm.listPodsUID()
	for _, id := range uids {
		pod := &kubecontainer.Pod{}
		pod.ID = id
		pods = append(pods, pod)
	}
	return pods, nil
}

func (cm *containerManager) setContainerFromMap(podID types.UID, container *cri.Container) {
	cm.podContainerLock.Lock()
	defer cm.podContainerLock.Unlock()

	cm.podContainer[podID] = container
}

func (cm *containerManager) deleteContainerFromMap(podID types.UID) {
	cm.podContainerLock.Lock()
	defer cm.podContainerLock.Unlock()

	delete(cm.podContainer, podID)
}

func (cm *containerManager) getContainerFromMap(podID types.UID) (*cri.Container, bool) {
	cm.podContainerLock.Lock()
	defer cm.podContainerLock.Unlock()
	container, ok := cm.podContainer[podID]
	return container, ok
}

func (cm *containerManager) listPodsUID() []types.UID {
	cm.podContainerLock.Lock()
	defer cm.podContainerLock.Unlock()

	podIDs := make([]types.UID, 0)
	for k := range cm.podContainer {
		podIDs = append(podIDs, k)
	}
	return podIDs
}

func (cm *containerManager) GarbageCollect(gcPolicy kubecontainer.ContainerGCPolicy, ready bool, evictNonDeletedPods bool) error {
	podsInUse := sets.String{}
	pods := cm.listPodsUID()
	for _, podUID := range pods {
		podsInUse.Insert(string(podUID))
	}

	err := cm.freeContainer(time.Now(), gcPolicy, podsInUse)
	if err != nil {
		return err
	}

	return nil
}

func (cm *containerManager) CleanupOrphanedPod(activePods []*v1.Pod) {
	podsInUse := sets.String{}
	for _, pod := range activePods {
		podsInUse.Insert(string(pod.UID))
	}

	uids := cm.listPodsUID()
	for _, id := range uids {
		if podsInUse.Has(string(id)) {
			continue
		} else {
			log.LOGGER.Infof("clean orphaned pod %s", id)
			if err := cm.TerminatePod(id); err != nil {
				log.LOGGER.Errorf("clean orphaned pod %s failed: %v", id, err)
			}
		}
	}
	return
}

type evictionInfo struct {
	containerID string
	containerRecord
}

type byLastUsedAndDetected []evictionInfo

func (ev byLastUsedAndDetected) Len() int      { return len(ev) }
func (ev byLastUsedAndDetected) Swap(i, j int) { ev[i], ev[j] = ev[j], ev[i] }
func (ev byLastUsedAndDetected) Less(i, j int) bool {
	if ev[i].lastUsed.Equal(ev[j].lastUsed) {
		return ev[i].firstDetected.Before(ev[j].firstDetected)
	}
	//else
	return ev[i].lastUsed.Before(ev[j].lastUsed)

}

func (cm *containerManager) freeContainer(freeTime time.Time, gcPolicy kubecontainer.ContainerGCPolicy, podsInUse sets.String) error {
	err := cm.detectContainers(freeTime, podsInUse)
	if err != nil {
		return err
	}

	containersRecords := make([]evictionInfo, 0, len(cm.containerRecords))
	for containerID, record := range cm.containerRecords {
		containersRecords = append(containersRecords, evictionInfo{
			containerID:     containerID,
			containerRecord: *record,
		})
	}
	sort.Sort(byLastUsedAndDetected(containersRecords))

	var wg sync.WaitGroup
	var deletionErrors []error
	for _, record := range containersRecords {
		log.LOGGER.Infof("Evaluating Container ID %s for possible garbage collection", record.containerID)
		if record.lastUsed.Equal(freeTime) || record.lastUsed.After(freeTime) {
			log.LOGGER.Infof("Container ID %s has lastUsed=%v which is >= freeTime=%v, not eligible for garbage collection", record.containerID, record.lastUsed, freeTime)
			continue
		}

		if freeTime.Sub(record.firstDetected) < gcPolicy.MinAge {
			log.LOGGER.Infof("Container ID %s has age %v which is less than the policy minAge of %v, not eligible for garbage collection", record.containerID, freeTime.Sub(record.firstDetected), gcPolicy.MinAge)
			continue
		}

		wg.Add(1)
		go func(ei evictionInfo) {
			defer wg.Done()
			log.LOGGER.Infof("Container GC Manager Removing container %s, container status %d", ei.containerID, ei.Status)

			err := cm.killContainer(ei.podID, ei.containerID)
			if err != nil {
				deletionErrors = append(deletionErrors, err)
				return
			}

			cm.deleteContainerRecords(ei.containerID)
		}(record)
	}

	wg.Wait()
	if len(deletionErrors) > 0 {
		return fmt.Errorf("free container failed with error: %v", deletionErrors)
	}
	return nil
}

func (cm *containerManager) detectContainers(detectTime time.Time, podsInUse sets.String) error {

	containers, err := cm.runtimeService.ListContainers()
	if err != nil {
		return err
	}
	now := time.Now()
	currentContainers := sets.NewString()

	for _, container := range containers {
		log.LOGGER.Infof("Adding container ID %s to currentContainers", container.ID)
		currentContainers.Insert(container.ID)

		record, ok := cm.getContainerRecords(container.ID)
		if !ok {
			log.LOGGER.Infof("Container ID %s is new", container.ID)
			podID, _ := cm.getPodID(container.ID)
			record = &containerRecord{
				firstDetected: container.StartAt,
				podID:         podID,
			}
			cm.addContainerRecords(container.ID, record)
		}

		if cm.isContainerUsed(record.podID, podsInUse) {
			log.LOGGER.Infof("Setting Container ID %s lastUsed to %v", container.ID, now)
			record.lastUsed = now
		}

		log.LOGGER.Infof("Container ID [%s], status is [%d], startat [%s]", container.ID, container.Status, container.StartAt)
		record.Status = container.Status
	}

	for container := range cm.containerRecords {
		if !currentContainers.Has(container) {
			log.LOGGER.Infof("Container ID %s is no longer present, removing from containerRecords", container)
			cm.deleteContainerRecords(container)
		}
	}
	return nil
}

func (cm *containerManager) isContainerUsed(podID types.UID, podsInUse sets.String) bool {
	if podID == "" {
		// if podID is nil, means this container does not in pod. it is used by others.
		return true
	}
	if _, ok := podsInUse[string(podID)]; ok {
		return true
	}
	return false
}

// only support hostNetwork. container port == host port
func makePortsAndBindings(portMappings []v1.ContainerPort) (map[nat.Port]struct{}, error) {
	exposedPorts := map[nat.Port]struct{}{}
	for _, port := range portMappings {
		exteriorPort := port.HostPort
		interiorPort := port.ContainerPort

		if exteriorPort != interiorPort || exteriorPort == 0 {
			return nil, fmt.Errorf("HostPort must be equal to ContainerPort and can not be 0")
		}
		var protocol string
		switch strings.ToUpper(string(port.Protocol)) {
		case "UDP":
			protocol = "/udp"
		case "TCP":
			protocol = "/tcp"
		default:
			log.LOGGER.Warnf("Unknown protocol %q: defaulting to TCP", port.Protocol)
			protocol = "/tcp"
		}

		dockerPort := nat.Port(strconv.Itoa(int(interiorPort)) + protocol)
		exposedPorts[dockerPort] = struct{}{}
	}
	return exposedPorts, nil
}

// makeMounts generates container volume mounts for kubelet runtime v1.
func (cm *containerManager) makeMounts(opts *kubecontainer.RunContainerOptions, container *v1.Container) []*kubecontainer.Mount {
	volumeMounts := []*kubecontainer.Mount{}

	for idx := range opts.Mounts {
		v := opts.Mounts[idx]
		selinuxRelabel := v.SELinuxRelabel && selinux.SELinuxEnabled()
		mount := &kubecontainer.Mount{
			HostPath:       v.HostPath,
			ContainerPath:  v.ContainerPath,
			ReadOnly:       v.ReadOnly,
			SELinuxRelabel: selinuxRelabel,
			Propagation:    v.Propagation,
		}

		volumeMounts = append(volumeMounts, mount)
	}

	// The reason we create and mount the log file in here (not in kubelet) is because
	// the file's location depends on the ID of the container, and we need to create and
	// mount the file before actually starting the container.
	if opts.PodContainerDir != "" && len(container.TerminationMessagePath) != 0 {
		// Because the PodContainerDir contains pod uid and container name which is unique enough,
		// here we just add a random id to make the path unique for different instances
		// of the same container.
		cid := makeUID()
		containerLogPath := filepath.Join(opts.PodContainerDir, cid)
		fs, err := os.Create(containerLogPath)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error on creating termination-log file %q: %v", containerLogPath, err))
		} else {
			fs.Close()

			// Chmod is needed because ioutil.WriteFile() ends up calling
			// open(2) to create the file, so the final mode used is "mode &
			// ~umask". But we want to make sure the specified mode is used
			// in the file no matter what the umask is.
			if err := os.Chmod(containerLogPath, 0666); err != nil {
				utilruntime.HandleError(fmt.Errorf("unable to set termination-log file permissions %q: %v", containerLogPath, err))
			}

			selinuxRelabel := selinux.SELinuxEnabled()
			volumeMounts = append(volumeMounts, &kubecontainer.Mount{
				HostPath:       containerLogPath,
				ContainerPath:  container.TerminationMessagePath,
				SELinuxRelabel: selinuxRelabel,
			})
		}
	}

	return volumeMounts
}

// makeUID returns a randomly generated string.
func makeUID() string {
	return fmt.Sprintf("%08x", rand.Uint32())
}
