package status

import (
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubelet/status"
)

// manager as status manager, embedded a k8s.io/kubernetes/pkg/kubelet/status.Manager
// inherit it's method but refactored Start() function to periodicity update status to IEF
type manager struct {
	status.Manager
	// TODO: consider need lock?
	podManager        podmanager.Manager
	apiStatusVersions map[types.UID]*v1.PodStatus
	metaClient        client.CoreInterface
}

//NewManager creates and returns a new manager object
func NewManager(kubeClient clientset.Interface, podManager podmanager.Manager, podDeletionSafety status.PodDeletionSafetyProvider, metaClient client.CoreInterface) status.Manager {
	kubeManager := status.NewManager(kubeClient, podManager, podDeletionSafety)
	return &manager{
		Manager:           kubeManager,
		metaClient:        metaClient,
		podManager:        podManager,
		apiStatusVersions: make(map[types.UID]*v1.PodStatus),
	}
}

const syncPeriod = 10 * time.Second

func (m *manager) Start() {
	log.LOGGER.Info("Starting to sync pod status with apiserver")
	syncTicker := time.Tick(syncPeriod)

	go wait.Forever(func() {
		select {
		case <-syncTicker:
			m.updatePodStatus()
		}
	}, 0)
}

func (m *manager) updatePodStatus() {
	for _, pod := range m.podManager.GetPods() {
		uid := pod.UID
		podStatus, ok := m.GetPodStatus(uid)
		if !ok || &podStatus == nil {
			continue
		}
		latestStatus, ok := m.apiStatusVersions[uid]
		if ok && apiequality.Semantic.DeepEqual(latestStatus, &podStatus) {
			continue
		}
		s := *podStatus.DeepCopy()
		var conditionFlag bool
		podCondition := v1.PodCondition{Type: v1.PodReady, Status: v1.ConditionFalse, Reason: "ContainersNotReady"}
		for idx, cs := range podStatus.ContainerStatuses {
			if cs.State.Running != nil && cs.State.Running.StartedAt.Unix() == 0 {
				newState := v1.ContainerState{Waiting: &v1.ContainerStateWaiting{
					Reason:  "CrashLoopBackOff",
					Message: "Container restarting in container runtime",
				}}
				s.ContainerStatuses[idx].State = newState
				conditionFlag = true
			}
		}
		var podReadyFlag bool
		if conditionFlag {
			if s.Conditions == nil {
				s.Conditions = append(s.Conditions, podCondition)
			} else {
				for index, condition := range s.Conditions {
					if condition.Type == v1.PodReady {
						s.Conditions[index].Status = v1.ConditionFalse
						s.Conditions[index].Reason = "ContainersNotReady"
						podReadyFlag = true
						break
					}
				}
				if !podReadyFlag {
					s.Conditions = append(s.Conditions, podCondition)
				}
			}
		}

		err := m.metaClient.PodStatus(pod.Namespace).Update(pod.Name, edgeapi.PodStatusRequest{UID: pod.UID, Name: pod.Name, Status: s})
		if err != nil {
			log.LOGGER.Errorf("Update pod status failed err :%v", err)
		}
		log.LOGGER.Infof("Status for pod %s updated successfully: %+v", pod.Name, podStatus)
		m.apiStatusVersions[pod.UID] = podStatus.DeepCopy()
	}
}
