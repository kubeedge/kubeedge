package pleg

import (
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/pkg/kubelet/pleg"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	"k8s.io/kubernetes/pkg/kubelet/status"

	"github.com/kubeedge/kubeedge/common/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/containers"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
)

//GenericLifecycle is object for pleg lifecycle
type GenericLifecycle struct {
	pleg.GenericPLEG
	runtime      containers.ContainerManager
	relistPeriod time.Duration
	status       status.Manager
	podManager   podmanager.Manager
	probeManager prober.Manager
}

//NewGenericLifecycle creates new generic life cycle object
func NewGenericLifecycle(manager containers.ContainerManager, probeManager prober.Manager, channelCapacity int,
	relistPeriod time.Duration, podManager podmanager.Manager, statusManager status.Manager) pleg.PodLifecycleEventGenerator {
	kubeContainerManager := containers.NewKubeContainerRuntime(manager)
	genericPLEG := pleg.NewGenericPLEG(kubeContainerManager, channelCapacity, relistPeriod, nil, clock.RealClock{})
	return &GenericLifecycle{
		GenericPLEG:  *genericPLEG.(*pleg.GenericPLEG),
		relistPeriod: relistPeriod,
		runtime:      manager,
		status:       statusManager,
		podManager:   podManager,
		probeManager: probeManager,
	}
}

// Start spawns a goroutine to relist periodically.
func (gl *GenericLifecycle) Start() {
	gl.GenericPLEG.Start()
	go wait.Until(func() {
		log.LOGGER.Infof("GenericLifecycle: Relisting")
		podListPm := gl.podManager.GetPods()
		for _, pod := range podListPm {
			if err := gl.updatePodStatus(pod); err != nil {
				log.LOGGER.Errorf("update pod %s status error", pod.Name)
			}
		}
	}, gl.relistPeriod, wait.NeverStop)
}

func (gl *GenericLifecycle) updatePodStatus(pod *v1.Pod) error {
	podStatus, err := gl.runtime.GetPodStatusOwn(pod)
	newStatus := *podStatus.DeepCopy()
	gl.probeManager.UpdatePodStatus(pod.UID, &newStatus)
	newStatus.Conditions = append(newStatus.Conditions, gl.runtime.GeneratePodReadyCondition(newStatus.ContainerStatuses))
	pod.Status = newStatus
	gl.status.SetPodStatus(pod, newStatus)
	return err
}
