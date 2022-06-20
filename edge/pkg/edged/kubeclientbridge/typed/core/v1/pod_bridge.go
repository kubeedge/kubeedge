package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type PodsBridge struct {
	fakecorev1.FakePods
	ns         string
	MetaClient client.CoreInterface
}

func (c *PodsBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.Pod, err error) {
	return c.MetaClient.Pods(c.ns).Get(name)
}

func (c *PodsBridge) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Pod, err error) {
	return c.MetaClient.Pods(c.ns).Patch(name, data)
}

func (c *PodsBridge) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.MetaClient.Pods(c.ns).Delete(name, opts)
}
