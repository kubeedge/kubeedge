package v1

import (
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// CoreV1Bridge is a coreV1 bridge
type CoreV1Bridge struct {
	fakecorev1.FakeCoreV1
	MetaClient client.CoreInterface
}