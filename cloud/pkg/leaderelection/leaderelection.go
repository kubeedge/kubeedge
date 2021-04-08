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
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
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

	cli := client.GetKubeClient()
	if err := CreateNamespaceIfNeeded(cli, "kubeedge"); err != nil {
		klog.Warningf("Create Namespace kubeedge failed with error: %s", err)
		return
	}

	coreBroadcaster := record.NewBroadcaster()
	coreBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: cli.CoreV1().Events("")})
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
			informers.GetInformersManager().Start(beehiveContext.Done())

			// Patch PodReadinessGate if program run in pod
			if err := TryToPatchPodReadinessGate(corev1.ConditionTrue); err != nil {
				// Terminate the program gracefully
				klog.Errorf("Error patching pod readinessGate: %v", err)
				if err := TriggerGracefulShutdown(); err != nil {
					klog.Fatalf("failed to gracefully terminate program: %v", err)
				}
			}
		},
		OnStoppedLeading: func() {
			klog.Errorf("leaderelection lost, gracefully terminate program")

			// Reset PodReadinessGate to false if cloudcore stop
			if err := TryToPatchPodReadinessGate(corev1.ConditionFalse); err != nil {
				klog.Errorf("Error reset pod readinessGate: %v", err)
			}

			// Trigger core.GracefulShutdown()
			if err := TriggerGracefulShutdown(); err != nil {
				klog.Fatalf("failed to gracefully terminate program: %v", err)
			}
		},
	}

	leaderElector, err := leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		klog.Errorf("couldn't create leader elector: %v", err)
		return
	}
	readyzAdaptor.SetLeaderElection(leaderElector)

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
	if !isInPod {
		klog.Infoln("CloudCore is not running in pod")
		return nil
	}

	namespace := os.Getenv("CLOUDCORE_POD_NAMESPACE")
	klog.Infof("CloudCore is running in pod %s/%s, try to patch PodReadinessGate", namespace, podname)
	client := client.GetKubeClient()

	//Creat patchBytes
	getPod, err := client.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod(%s/%s): %v", namespace, podname, err)
	}
	originalJSON, err := json.Marshal(getPod)
	if err != nil {
		return fmt.Errorf("failed to marshal original pod %q into JSON: %v", podname, err)
	}

	//Todo: Read PodReadinessGate from CloudCore configuration or env
	condition := corev1.PodCondition{Type: "kubeedge.io/CloudCoreIsLeader", Status: status}
	podutil.UpdatePodCondition(&getPod.Status, &condition)
	newJSON, err := json.Marshal(getPod)
	if err != nil {
		return fmt.Errorf("failed to marshal modified pod %q into JSON: %v", podname, err)
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, newJSON, corev1.Pod{})
	if err != nil {
		return fmt.Errorf("failed to create two way merge patch: %v", err)
	}

	var maxRetries = 3
	for i := 1; i <= maxRetries; i++ {
		_, err = client.CoreV1().Pods(namespace).Patch(context.Background(), podname, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
		if err == nil {
			klog.Infof("Successfully patching podReadinessGate: kubeedge.io/CloudCoreIsLeader to pod %q through apiserver", podname)
			return nil
		}
		if !errors.IsConflict(err) {
			return err
		}

		// If the patch failure is due to update conflict, the necessary retransmission is performed
		if i >= maxRetries {
			klog.Errorf("updateMaxRetries(%d) has reached, failed to patching podReadinessGate: kubeedge.io/CloudCoreIsLeader because of update conflict", maxRetries)
		}
		continue
	}

	return err
}

// TriggerGracefulShutdown triggers core.GracefulShutdown()
func TriggerGracefulShutdown() error {
	if beehiveContext.GetContext().Err() != nil {
		klog.Infoln("Program is in gracefully shutdown")
		return nil
	}

	klog.Infoln("Trigger graceful shutdown!")
	p, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		return fmt.Errorf("Failed to find self process: %v", err)
	}

	if err := p.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("Failed to trigger graceful shutdown: %v", err)
	}
	return nil
}

func CreateNamespaceIfNeeded(cli clientset.Interface, ns string) error {
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
