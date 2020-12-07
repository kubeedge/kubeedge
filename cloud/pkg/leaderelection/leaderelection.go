package leaderelection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
	config "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

func Run(cfg *config.CloudCoreConfig, readyzAdaptor *ReadyzAdaptor) {
	// To help debugging, immediately log config for LeaderElection
	klog.Infof("Config for LeaderElection : %v", *cfg.LeaderElection)
	// Init Context for leaderElection
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	// Init podReadinessGate to false at the begin of Run
	if err := TryToPatchPodReadinessGate(corev1.ConditionFalse); err != nil {
		klog.Errorf("Error init pod readinessGate: %v", err)
	}

	coreBroadcaster := record.NewBroadcaster()
	cli, err := utils.KubeClient()
	if err != nil {
		klog.Warningf("Create kube client for leaderElection failed with error: %s", err)
		return
	}
	if err = CreateNamespaceIfNeeded(cli, "kubeedge"); err != nil {
		klog.Warningf("Create Namespace kubeedge failed with error: %s", err)
		return
	}
	coreRecorder := coreBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "CloudCore"})
	leaderElectionConfig, err := makeLeaderElectionConfig(*cfg.LeaderElection, cli, coreRecorder)

	if err != nil {
		klog.Errorf("couldn't create leaderElectorConfig: %v", err)
		return
	}
	leaderElectionConfig.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			// Start all modules,
			core.StartModules()
			// Patch PodReadinessGate if program run in pod
			err := TryToPatchPodReadinessGate(corev1.ConditionTrue)
			if err != nil {
				// Terminate the program gracefully
				klog.Errorf("Error patching pod readinessGate: %v", err)
				TriggerGracefulShutdown()
			}
		},
		OnStoppedLeading: func() {
			// TODO: is it necessary to terminate the program gracefully?
			//klog.Fatalf("leaderelection lost, rudely terminate program")
			klog.Errorf("leaderelection lost, gracefully terminate program")
			// Reset PodReadinessGate to false if cloudcore stop
			err := TryToPatchPodReadinessGate(corev1.ConditionFalse)
			if err != nil {
				klog.Errorf("Error reset pod readinessGate: %v", err)
			}
			// Trigger core.GracefulShutdown()
			TriggerGracefulShutdown()
		},
	}

	leaderElector, err := leaderelection.NewLeaderElector(*leaderElectionConfig)
	// Set readyzAdaptor manually
	readyzAdaptor.SetLeaderElection(leaderElector)
	if err != nil {
		klog.Errorf("couldn't create leader elector: %v", err)
		return
	}

	// Start leaderElection until becoming leader, terminate program if leader lost or context.cancel
	go leaderElector.Run(beehiveContext.GetContext())

	// Monitor system signal and shutdown gracefully and it should be in main gorutine
	core.GracefulShutdown()
}

// makeLeaderElectionConfig builds a leader election configuration. It will
// create a new resource lock associated with the configuration.
func makeLeaderElectionConfig(config componentbaseconfig.LeaderElectionConfiguration, client clientset.Interface, recorder record.EventRecorder) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New(config.ResourceLock,
		config.ResourceNamespace,
		config.ResourceName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		return nil, fmt.Errorf("couldn't create resource lock: %v", err)
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: config.LeaseDuration.Duration,
		RenewDeadline: config.RenewDeadline.Duration,
		RetryPeriod:   config.RetryPeriod.Duration,
		WatchDog:      nil,
		Name:          "cloudcore",
	}, nil
}

// Try to patch PodReadinessGate if program runs in pod
func TryToPatchPodReadinessGate(status corev1.ConditionStatus) error {
	podname, isInPod := os.LookupEnv("CLOUDCORE_POD_NAME")
	if isInPod {
		namespace := os.Getenv("CLOUDCORE_POD_NAMESPACE")
		klog.Infof("CloudCore is running in pod %v/%v, try to patch PodReadinessGate", namespace, podname)
		//TODO: use specific clients
		cli, err := utils.KubeClient()
		if err != nil {
			return fmt.Errorf("create kube client for patching podReadinessGate failed with error: %v", err)
		}

		//Creat patchBytes
		getPod, err := cli.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
		originalJSON, err := json.Marshal(getPod)
		if err != nil {
			return fmt.Errorf("failed to marshal modified pod %q into JSON: %v", podname, err)
		}
		//Todo: Read PodReadinessGate from CloudCore configuration or env
		condition := corev1.PodCondition{Type: "kubeedge.io/CloudCoreIsLeader", Status: status}
		podutil.UpdatePodCondition(&getPod.Status, &condition)
		newJSON, err := json.Marshal(getPod)
		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, newJSON, corev1.Pod{})
		if err != nil {
			return fmt.Errorf("failed to create two way merge patch: %v", err)
		}

		var maxRetries = 3
		var isPatchSuccess = false
		for i := 1; i <= maxRetries; i++ {
			_, err = cli.CoreV1().Pods(namespace).Patch(context.Background(), podname, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
			if err == nil {
				isPatchSuccess = true
				klog.Infof("Successfully patching podReadinessGate: kubeedge.io/CloudCoreIsLeader to pod %q through apiserver", podname)
				break
			}
			if errors.IsConflict(err) {
				// If the patch failure is due to update conflict, the necessary retransmission is performed
				if i >= maxRetries {
					klog.Errorf("updateMaxRetries(%d) has reached, failed to patching podReadinessGate: kubeedge.io/CloudCoreIsLeader because of update conflict", maxRetries)
				}
				continue
			}
			break
		}
		if !isPatchSuccess {
			return err
		}
	} else {
		klog.Infoln("CloudCore is not running in pod")
	}
	return nil
}

// Trigger core.GracefulShutdown()
func TriggerGracefulShutdown() {
	if beehiveContext.GetContext().Err() != nil {
		klog.Errorln("Program is in gracefully shutdown")
		return
	}
	klog.Errorln("Trigger graceful shutdown!")
	p, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		klog.Errorf("Failed to find self process: %v", err)
	}
	err = p.Signal(os.Interrupt)
	if err != nil {
		klog.Errorf("Failed to trigger graceful shutdown: %v", err)
	}
}

func CreateNamespaceIfNeeded(cli *clientset.Clientset, ns string) error {
	c := cli.CoreV1()
	if _, err := c.Namespaces().Get(context.Background(), ns, metav1.GetOptions{}); err == nil {
		// the namespace already exists
		return nil
	}
	newNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: "",
		},
	}
	_, err := c.Namespaces().Create(context.Background(), newNs, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		err = nil
	}
	return err
}
