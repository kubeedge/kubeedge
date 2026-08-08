package util

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/common/constants"
)

// namespaceDeleteTimeout/namespaceDeletePollInterval bound how long CleanNameSpace
// waits for a namespace to actually disappear after the delete call is accepted.
// Declared as vars (not consts) so tests can shrink them.
var (
	namespaceDeleteTimeout      = 2 * time.Minute
	namespaceDeletePollInterval = 2 * time.Second
)

func kubeConfig(kubeconfigPath string) (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(constants.DefaultKubeQPS)
	kubeConfig.Burst = int(constants.DefaultKubeBurst)
	kubeConfig.ContentType = constants.DefaultKubeContentType

	return kubeConfig, nil
}

// KubeClient from config
func KubeClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	kubeConfig, err := kubeConfig(kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("get kube config failed with error: %s", err)
	}
	return kubernetes.NewForConfig(kubeConfig)
}

// CleanNameSpace deletes the given namespace and waits for it to be fully
// removed. Namespace deletion in Kubernetes is asynchronous: the API server
// marks the namespace as Terminating and returns immediately, while the
// actual removal only completes once every object inside it (and any
// pending finalizers) is cleaned up. Returning as soon as the delete call is
// accepted, without waiting for the namespace to actually disappear, made
// `keadm reset` report success while the "kubeedge" namespace was still
// stuck in Terminating, which then caused a subsequent `keadm init`/`keadm
// join` to fail or hang while recreating it.
func (co *Common) CleanNameSpace(ns, kubeConfigPath string) error {
	cli, err := KubeClient(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	if err := cli.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = wait.PollUntilContextTimeout(context.Background(), namespaceDeletePollInterval, namespaceDeleteTimeout, true,
		func(ctx context.Context) (bool, error) {
			_, err := cli.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		},
	)
	if err != nil {
		return fmt.Errorf("namespace %s was not fully terminated within %s, it may be stuck in Terminating state "+
			"because some resources in it still have pending finalizers; please check with "+
			"`kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get -n %s --ignore-not-found` "+
			"and `kubectl describe ns %s` to find any remaining namespaced resources (including PVCs, config objects, "+
			"or custom resources not covered by `kubectl get all`), and remove any blocking finalizers manually: %v",
			ns, namespaceDeleteTimeout, ns, ns, err)
	}

	return nil
}

// IsCloudcoreContainerRunning judge whether cloudcore pod is running
func IsCloudcoreContainerRunning(ns, kubeConfigPath string) (bool, error) {
	cli, err := KubeClient(kubeConfigPath)
	if err != nil {
		return false, fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	podList, err := cli.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to query pods, error: %s", err)
	}
	if len(podList.Items) == 0 {
		return false, nil
	}
	return true, nil
}
